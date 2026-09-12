package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	problemBodyLimit = 256 * 1024
	problemTextLimit = 65536
	problemListLimit = 100
)

type AddProblemInput struct {
	SessionID  string `json:"session_id"`
	TicketID   string `json:"ticket_id,omitempty"`
	Summary    string `json:"summary"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	Correction string `json:"correction,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
}

type AppendProblemInput struct {
	SessionID string `json:"session_id"`
	ProblemID string `json:"problem_id"`
	Body      string `json:"body"`
}

type ListProblemsInput struct {
	Limit  int
	Cursor string
}

type ProblemNote struct {
	Seq       int64  `json:"seq"`
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type ProblemReport struct {
	ID             string        `json:"id"`
	ProjectID      string        `json:"project_id"`
	DisplayKey     string        `json:"display_key"`
	TicketID       *string       `json:"ticket_id"`
	SessionID      string        `json:"session_id"`
	Summary        string        `json:"summary"`
	Expected       string        `json:"expected"`
	Actual         string        `json:"actual"`
	Correction     string        `json:"correction"`
	Evidence       string        `json:"evidence"`
	CreatedAt      string        `json:"created_at"`
	Notes          []ProblemNote `json:"notes"`
	NotesTruncated bool          `json:"notes_truncated"`
}

type ProblemPage struct {
	Items      []ProblemReport `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

func validateProblemText(name, value string, required bool, limit int) error {
	if required && strings.TrimSpace(value) == "" {
		return Fail("INVALID_INPUT", name+" is required")
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > limit {
		return Fail("INVALID_INPUT", fmt.Sprintf("%s must contain at most %d characters", name, limit))
	}
	return nil
}

func validateProblemSession(ctx context.Context, c *sql.Conn, project, sessionID string) error {
	if sessionID == "" {
		return Fail("INVALID_INPUT", "session is required")
	}
	session, err := ReadSession(ctx, c, project, sessionID)
	if err != nil {
		return err
	}
	if session.DeclaredState == "stopped" {
		return Fail("SESSION_STOPPED", "session is stopped")
	}
	return nil
}

func (s *Service) AddProblem(ctx context.Context, project, request string, input AddProblemInput) (json.RawMessage, error) {
	for _, v := range []struct {
		name, value string
		required    bool
		limit       int
	}{
		{"summary", input.Summary, true, 4096},
		{"expected", input.Expected, true, problemTextLimit},
		{"actual", input.Actual, true, problemTextLimit},
		{"correction", input.Correction, false, problemTextLimit},
		{"evidence", input.Evidence, false, problemTextLimit},
	} {
		if err := validateProblemText(v.name, v.value, v.required, v.limit); err != nil {
			return nil, err
		}
	}
	body := struct {
		Summary, Expected, Actual, Correction, Evidence string
	}{input.Summary, input.Expected, input.Actual, input.Correction, input.Evidence}
	if len(JSON(body)) > problemBodyLimit {
		return nil, Fail("INVALID_INPUT", "problem body exceeds 256 KiB")
	}
	return s.Mutate(ctx, project, request, "problem.add", input.SessionID, input, func(c *sql.Conn) (json.RawMessage, error) {
		if err := validateProblemSession(ctx, c, project, input.SessionID); err != nil {
			return nil, err
		}
		var ticket *string
		if input.TicketID != "" {
			var id string
			err := c.QueryRowContext(ctx, "SELECT id FROM tickets WHERE project_id=? AND id=?", project, input.TicketID).Scan(&id)
			if err == sql.ErrNoRows {
				err = c.QueryRowContext(ctx, "SELECT id FROM tickets WHERE project_id=? AND display_key=?", project, input.TicketID).Scan(&id)
			}
			if err == sql.ErrNoRows {
				return nil, Fail("NOT_FOUND", "ticket not found")
			}
			if err != nil {
				return nil, err
			}
			ticket = &id
		}
		var prefix string
		var number int64
		if err := c.QueryRowContext(ctx, "SELECT prefix,next_problem FROM projects WHERE id=?", project).Scan(&prefix, &number); err != nil {
			return nil, err
		}
		if _, err := c.ExecContext(ctx, "UPDATE projects SET next_problem=next_problem+1 WHERE id=?", project); err != nil {
			return nil, err
		}
		report := ProblemReport{ID: UUID(), ProjectID: project, DisplayKey: fmt.Sprintf("%s-P%d", prefix, number), TicketID: ticket, SessionID: input.SessionID, Summary: input.Summary, Expected: input.Expected, Actual: input.Actual, Correction: input.Correction, Evidence: input.Evidence, CreatedAt: Now(), Notes: []ProblemNote{}}
		if _, err := c.ExecContext(ctx, "INSERT INTO problems(id,project_id,display_key,ticket_id,session_id,summary,expected,actual,correction,evidence,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)", report.ID, project, report.DisplayKey, ticket, report.SessionID, report.Summary, report.Expected, report.Actual, report.Correction, report.Evidence, report.CreatedAt); err != nil {
			return nil, err
		}
		if _, err := c.ExecContext(ctx, "INSERT INTO events(id,project_id,ticket_id,problem_id,actor_id,kind,payload,created_at) VALUES(?,?,?,?,?,?,?,?)", UUID(), project, ticket, report.ID, input.SessionID, "problem.add", string(JSON(input)), report.CreatedAt); err != nil {
			return nil, err
		}
		if _, err := c.ExecContext(ctx, "UPDATE sessions SET last_seen_at=? WHERE project_id=? AND id=?", report.CreatedAt, project, input.SessionID); err != nil {
			return nil, err
		}
		return JSON(report), nil
	})
}

func (s *Service) AppendProblem(ctx context.Context, project, request string, input AppendProblemInput) (json.RawMessage, error) {
	if strings.TrimSpace(input.ProblemID) == "" {
		return nil, Fail("INVALID_INPUT", "problem is required")
	}
	if err := validateProblemText("body", input.Body, true, problemTextLimit); err != nil {
		return nil, err
	}
	if len([]byte(input.Body)) > problemBodyLimit {
		return nil, Fail("INVALID_INPUT", "problem body exceeds 256 KiB")
	}
	return s.Mutate(ctx, project, request, "problem.append", input.SessionID, input, func(c *sql.Conn) (json.RawMessage, error) {
		if err := validateProblemSession(ctx, c, project, input.SessionID); err != nil {
			return nil, err
		}
		var ticket sql.NullString
		if err := c.QueryRowContext(ctx, "SELECT ticket_id FROM problems WHERE project_id=? AND id=?", project, input.ProblemID).Scan(&ticket); err == sql.ErrNoRows {
			return nil, Fail("NOT_FOUND", "problem not found")
		} else if err != nil {
			return nil, err
		}
		note := ProblemNote{ID: UUID(), SessionID: input.SessionID, Body: input.Body, CreatedAt: Now()}
		result, err := c.ExecContext(ctx, "INSERT INTO events(id,project_id,ticket_id,problem_id,actor_id,kind,body,payload,created_at) VALUES(?,?,?,?,?,'problem.append',?,'{}',?)", note.ID, project, ticket, input.ProblemID, input.SessionID, note.Body, note.CreatedAt)
		if err != nil {
			return nil, err
		}
		note.Seq, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
		if _, err = c.ExecContext(ctx, "UPDATE sessions SET last_seen_at=? WHERE project_id=? AND id=?", note.CreatedAt, project, input.SessionID); err != nil {
			return nil, err
		}
		return JSON(note), nil
	})
}

func (s *Service) ListProblems(ctx context.Context, project string, input ListProblemsInput) (ProblemPage, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > problemListLimit {
		return ProblemPage{}, Fail("INVALID_INPUT", "limit must be between 1 and 100")
	}
	var cursor int64
	var err error
	if input.Cursor != "" {
		cursor, err = strconv.ParseInt(input.Cursor, 10, 64)
		if err != nil || cursor < 0 {
			return ProblemPage{}, Fail("INVALID_INPUT", "invalid cursor")
		}
	}
	rows, err := s.Store.DB.QueryContext(ctx, "SELECT rowid,id,project_id,display_key,ticket_id,session_id,summary,expected,actual,correction,evidence,created_at FROM problems WHERE project_id=? AND rowid>? ORDER BY rowid LIMIT ?", project, cursor, limit+1)
	if err != nil {
		return ProblemPage{}, err
	}
	type item struct {
		rowid  int64
		report ProblemReport
	}
	items := []item{}
	for rows.Next() {
		var v item
		var ticket sql.NullString
		if err = rows.Scan(&v.rowid, &v.report.ID, &v.report.ProjectID, &v.report.DisplayKey, &ticket, &v.report.SessionID, &v.report.Summary, &v.report.Expected, &v.report.Actual, &v.report.Correction, &v.report.Evidence, &v.report.CreatedAt); err != nil {
			rows.Close()
			return ProblemPage{}, err
		}
		if ticket.Valid {
			v.report.TicketID = &ticket.String
		}
		v.report.Notes = []ProblemNote{}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return ProblemPage{}, err
	}
	rows.Close()
	page := ProblemPage{Items: []ProblemReport{}}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	for _, v := range items {
		notes, truncated, e := s.problemNotes(ctx, project, v.report.ID)
		if e != nil {
			return ProblemPage{}, e
		}
		v.report.Notes = notes
		v.report.NotesTruncated = truncated
		page.Items = append(page.Items, v.report)
	}
	if more {
		page.NextCursor = strconv.FormatInt(items[len(items)-1].rowid, 10)
	}
	return page, nil
}

func (s *Service) problemNotes(ctx context.Context, project, problem string) ([]ProblemNote, bool, error) {
	rows, err := s.Store.DB.QueryContext(ctx, "SELECT seq,id,actor_id,body,created_at FROM events WHERE project_id=? AND problem_id=? AND kind='problem.append' ORDER BY seq DESC LIMIT 101", project, problem)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	notes := []ProblemNote{}
	for rows.Next() {
		var note ProblemNote
		if err = rows.Scan(&note.Seq, &note.ID, &note.SessionID, &note.Body, &note.CreatedAt); err != nil {
			return nil, false, err
		}
		notes = append(notes, note)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	truncated := len(notes) > 100
	if truncated {
		notes = notes[:100]
	}
	for left, right := 0, len(notes)-1; left < right; left, right = left+1, right-1 {
		notes[left], notes[right] = notes[right], notes[left]
	}
	return notes, truncated, nil
}
