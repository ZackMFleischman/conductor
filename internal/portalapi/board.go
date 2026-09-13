package portalapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"sort"
	"strings"
)

// DBReader reads portal snapshots from an existing current-schema registry.
type DBReader struct{ DB *sql.DB }

var _ Reader = DBReader{}

// beginSnapshot verifies the schema on the same transaction as all data reads.
// OpenReadOnly accepts older schemas for other consumers; this API needs v3.
func (r DBReader) beginSnapshot(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	var version int
	if err = tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err == nil && version != store.SchemaVersion {
		err = ErrUnsupportedSchema
	}
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r DBReader) Projects(ctx context.Context) ([]Project, error) {
	tx, err := r.beginSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	projects, err := readProjects(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return projects, nil
}

func readProjects(ctx context.Context, tx *sql.Tx) ([]Project, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,prefix FROM projects ORDER BY prefix,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err = rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

type storedBoardTicket struct {
	ticket Ticket
	parent sql.NullString
}

func (r DBReader) Board(ctx context.Context, projectID string) (Board, error) {
	tx, err := r.beginSnapshot(ctx)
	if err != nil {
		return Board{}, err
	}
	defer tx.Rollback()
	b := Board{Tickets: make([]Ticket, 0)}
	var commonDir string
	err = tx.QueryRowContext(ctx, `SELECT id,prefix,common_dir FROM projects WHERE id=?`, projectID).Scan(&b.Project.ID, &b.Project.Name, &commonDir)
	if errors.Is(err, sql.ErrNoRows) {
		return Board{}, ErrNotFound
	}
	if err != nil {
		return Board{}, err
	}
	// A standard registered repository has a .git common directory. Never
	// guess a serving root for bare repositories or unavailable registrations.
	if filepath.IsAbs(commonDir) && strings.EqualFold(filepath.Base(commonDir), ".git") {
		b.Project.root = filepath.Dir(commonDir)
	}
	tickets, order, err := readBoardTickets(ctx, tx, projectID)
	if err != nil {
		return Board{}, err
	}
	if err = readBoardDependencies(ctx, tx, projectID, tickets); err != nil {
		return Board{}, err
	}
	if err = readBoardDeferrals(ctx, tx, projectID, tickets); err != nil {
		return Board{}, err
	}
	var watermark int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0) FROM events WHERE project_id=?`, projectID).Scan(&watermark); err != nil {
		return Board{}, err
	}
	if err = tx.Commit(); err != nil {
		return Board{}, err
	}
	// End the transaction before graph traversal, encoding, or any caller I/O.
	for _, id := range order {
		if err = ctx.Err(); err != nil {
			return Board{}, err
		}
		stored := tickets[id]
		seen := map[string]bool{id: true}
		for parent := stored.parent; parent.Valid; {
			ancestor, ok := tickets[parent.String]
			if !ok || seen[parent.String] {
				return Board{}, ErrInvalidData
			}
			seen[parent.String] = true
			stored.ticket.Ancestors = append(stored.ticket.Ancestors, Reference{ID: ancestor.ticket.ID, Key: ancestor.ticket.Key, Title: ancestor.ticket.Title})
			parent = ancestor.parent
		}
		if stored.ticket.Status == "done" {
			stored.ticket.Blockers = make([]Blocker, 0)
		} else if stored.ticket.Status == "ready" && len(stored.ticket.Blockers) > 0 {
			stored.ticket.Status = "blocked"
		}
		sort.Slice(stored.ticket.Blockers, func(i, j int) bool {
			a, b := stored.ticket.Blockers[i], stored.ticket.Blockers[j]
			if a.TicketKey != b.TicketKey {
				return a.TicketKey < b.TicketKey
			}
			return a.Reason < b.Reason
		})
		stored.ticket.Attachments = localAttachments(b.Project.root, projectID, stored.ticket)
		b.Tickets = append(b.Tickets, stored.ticket)
	}
	// Revision is derived only from durable observations, never a read timestamp.
	encoded, err := json.Marshal(struct {
		Board     Board `json:"board"`
		Watermark int64 `json:"watermark"`
	}{b, watermark})
	if err != nil {
		return Board{}, err
	}
	digest := sha256.Sum256(encoded)
	b.Revision = hex.EncodeToString(digest[:])
	return b, nil
}

func readBoardTickets(ctx context.Context, tx *sql.Tx, projectID string) (map[string]*storedBoardTicket, []string, error) {
	// Join by identity first and validate each recorded project explicitly. A
	// project-filtered INNER JOIN would silently hide missing/cross-project data.
	rows, err := tx.QueryContext(ctx, `SELECT t.id,t.display_key,t.title,t.body,t.state,t.assigned_agent_id,
		m.project_id,m.spec_revision,m.parent_id,m.blocked_reason,
		a.project_id,a.name,s.project_id,s.prepared_revision,s.authorized_revision,s.paused,
		COALESCE(m.kind,'implementation'),t.summary,t.evidence,t.qa,t.created_at,t.updated_at
		FROM tickets t
		LEFT JOIN ticket_metadata m ON m.ticket_id=t.id
		LEFT JOIN agents a ON a.id=t.assigned_agent_id
		LEFT JOIN workflow_ticket_specs s ON s.ticket_id=t.id
		WHERE t.project_id=? ORDER BY t.created_at,t.id`, projectID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	tickets := make(map[string]*storedBoardTicket)
	order := make([]string, 0)
	for rows.Next() {
		v := &storedBoardTicket{ticket: Ticket{Ancestors: make([]Reference, 0), Blockers: make([]Blocker, 0)}}
		var assignment, metadataProject, blockedReason, agentProject, agentName, policyProject sql.NullString
		var specRevision, prepared, authorized, paused sql.NullInt64
		if err = rows.Scan(&v.ticket.ID, &v.ticket.Key, &v.ticket.Title, &v.ticket.Description, &v.ticket.Status, &assignment,
			&metadataProject, &specRevision, &v.parent, &blockedReason, &agentProject, &agentName, &policyProject, &prepared, &authorized, &paused,
			&v.ticket.Kind, &v.ticket.Summary, &v.ticket.Evidence, &v.ticket.QA, &v.ticket.CreatedAt, &v.ticket.UpdatedAt); err != nil {
			return nil, nil, err
		}
		if !metadataProject.Valid || metadataProject.String != projectID || !specRevision.Valid || specRevision.Int64 < 1 || !blockedReason.Valid {
			return nil, nil, ErrInvalidData
		}
		if assignment.Valid {
			if !agentProject.Valid || agentProject.String != projectID || !agentName.Valid {
				return nil, nil, ErrInvalidData
			}
			v.ticket.Assignee = &Assignee{ID: assignment.String, Name: agentName.String}
		}
		if blockedReason.String != "" {
			v.ticket.Blockers = append(v.ticket.Blockers, Blocker{Reason: blockedReason.String})
		}
		switch v.ticket.Status {
		case "ready", "in_progress", "review", "done":
		case "blocked":
			if blockedReason.String == "" {
				v.ticket.Blockers = append(v.ticket.Blockers, Blocker{Reason: "Blocked; details unavailable"})
			}
		case "draft":
			v.ticket.Status = "blocked"
			v.ticket.Blockers = append(v.ticket.Blockers, Blocker{Reason: "Awaiting preparation and authorization"})
		default:
			return nil, nil, ErrInvalidData
		}
		if policyProject.Valid {
			if policyProject.String != projectID || !prepared.Valid || !authorized.Valid || !paused.Valid || (paused.Int64 != 0 && paused.Int64 != 1) {
				return nil, nil, ErrInvalidData
			}
			if paused.Int64 == 1 {
				v.ticket.Blockers = append(v.ticket.Blockers, Blocker{Reason: "Work paused"})
			}
			if prepared.Int64 != specRevision.Int64 {
				v.ticket.Blockers = append(v.ticket.Blockers, Blocker{Reason: "Awaiting preparation for current specification"})
			}
			if authorized.Int64 != specRevision.Int64 {
				v.ticket.Blockers = append(v.ticket.Blockers, Blocker{Reason: "Awaiting authorization for current specification"})
			}
		}
		tickets[v.ticket.ID] = v
		order = append(order, v.ticket.ID)
	}
	return tickets, order, rows.Err()
}

func readBoardDependencies(ctx context.Context, tx *sql.Tx, projectID string, tickets map[string]*storedBoardTicket) error {
	rows, err := tx.QueryContext(ctx, `SELECT project_id,ticket_id,depends_on FROM ticket_dependencies
		WHERE project_id=? OR ticket_id IN (SELECT id FROM tickets WHERE project_id=?)`, projectID, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var project, id, dependencyID string
		if err = rows.Scan(&project, &id, &dependencyID); err != nil {
			return err
		}
		ticket, ticketOK := tickets[id]
		dependency, dependencyOK := tickets[dependencyID]
		if project != projectID || !ticketOK || !dependencyOK {
			return ErrInvalidData
		}
		if dependency.ticket.Status != "done" {
			ticket.ticket.Blockers = append(ticket.ticket.Blockers, Blocker{Reason: "Waiting for " + dependency.ticket.Key + ": " + dependency.ticket.Title, TicketKey: dependency.ticket.Key})
		}
	}
	return rows.Err()
}

func readBoardDeferrals(ctx context.Context, tx *sql.Tx, projectID string, tickets map[string]*storedBoardTicket) error {
	rows, err := tx.QueryContext(ctx, `SELECT project_id,ticket_id,milestone_id,condition_text FROM retrospective_deferrals
		WHERE released_at IS NULL AND (project_id=? OR ticket_id IN (SELECT id FROM tickets WHERE project_id=?))`, projectID, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var project, id, milestone, condition string
		if err = rows.Scan(&project, &id, &milestone, &condition); err != nil {
			return err
		}
		ticket, ok := tickets[id]
		if project != projectID || !ok {
			return ErrInvalidData
		}
		reason := "Deferred until milestone " + milestone
		if condition != "" {
			reason += ": " + condition
		}
		ticket.ticket.Blockers = append(ticket.ticket.Blockers, Blocker{Reason: reason})
	}
	return rows.Err()
}
