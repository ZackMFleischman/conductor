package portalapi

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
)

const ticketNotesPageSize = 20

var ErrInvalidNotesCursor = errors.New("invalid ticket notes cursor")

type TicketNote struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type TicketNotesPage struct {
	TicketID   string       `json:"ticketId"`
	Notes      []TicketNote `json:"notes"`
	NextCursor string       `json:"nextCursor,omitempty"`
	Total      int          `json:"total"`
}

type TicketNotesReader interface {
	TicketNotes(context.Context, string, string, string) (TicketNotesPage, error)
}

// TicketNotes presents append-only notes, not the ticket's routine state events.
// The sequence boundary keeps older-page requests stable across new appends.
func (r DBReader) TicketNotes(ctx context.Context, projectID, ticketID, before string) (TicketNotesPage, error) {
	var boundary int64
	if before != "" {
		raw, err := base64.RawURLEncoding.DecodeString(before)
		if err != nil {
			return TicketNotesPage{}, ErrInvalidNotesCursor
		}
		boundary, err = strconv.ParseInt(string(raw), 10, 64)
		if err != nil || boundary <= 0 {
			return TicketNotesPage{}, ErrInvalidNotesCursor
		}
	}
	tx, err := r.beginSnapshot(ctx)
	if err != nil {
		return TicketNotesPage{}, err
	}
	defer tx.Rollback()
	var found string
	err = tx.QueryRowContext(ctx, `SELECT id FROM tickets WHERE project_id=? AND id=?`, projectID, ticketID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return TicketNotesPage{}, ErrNotFound
	}
	if err != nil {
		return TicketNotesPage{}, err
	}
	page := TicketNotesPage{TicketID: ticketID, Notes: make([]TicketNote, 0)}
	err = tx.QueryRowContext(ctx, `SELECT count(*) FROM events WHERE project_id=? AND ticket_id=? AND kind='ticket.note'`, projectID, ticketID).Scan(&page.Total)
	if err != nil {
		return TicketNotesPage{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT e.seq,e.id,COALESCE(a.name,'Unknown author'),e.body,e.created_at
 FROM events e LEFT JOIN sessions s ON s.project_id=e.project_id AND s.id=e.actor_id
 LEFT JOIN agents a ON a.project_id=s.project_id AND a.id=s.agent_id
 WHERE e.project_id=? AND e.ticket_id=? AND e.kind='ticket.note' AND (?=0 OR e.seq<?)
 ORDER BY e.seq DESC LIMIT ?`, projectID, ticketID, boundary, boundary, ticketNotesPageSize+1)
	if err != nil {
		return TicketNotesPage{}, err
	}
	var last int64
	for rows.Next() {
		var note TicketNote
		var seq int64
		if err = rows.Scan(&seq, &note.ID, &note.Author, &note.Body, &note.CreatedAt); err != nil {
			rows.Close()
			return TicketNotesPage{}, err
		}
		if len(page.Notes) == ticketNotesPageSize {
			page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(last, 10)))
			break
		}
		page.Notes = append(page.Notes, note)
		last = seq
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return TicketNotesPage{}, err
	}
	rows.Close()
	if err = tx.Commit(); err != nil {
		return TicketNotesPage{}, err
	}
	return page, nil
}

func (s *server) ticketNotes(w http.ResponseWriter, r *http.Request) {
	reader, ok := s.reader.(TicketNotesReader)
	if !ok {
		apiError(w, 404, "NOT_FOUND", "Ticket comments are unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.options.ReadTimeout)
	defer cancel()
	page, err := reader.TicketNotes(ctx, r.PathValue("project"), r.PathValue("ticket"), r.URL.Query().Get("before"))
	if errors.Is(err, ErrInvalidNotesCursor) {
		apiError(w, 400, "INVALID_CURSOR", "Invalid ticket comments cursor")
		return
	}
	if errors.Is(err, ErrNotFound) {
		apiError(w, 404, "NOT_FOUND", "Ticket not found")
		return
	}
	if err != nil {
		readError(w, err)
		return
	}
	apiJSON(w, 200, page)
}
