package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"strings"
	"unicode/utf8"
)

const TicketTextLimit = 256 * 1024

// TicketInput is the complete replay payload; location is observed before entering the transaction.
type TicketInput struct {
	Commit                                                      string
	ProjectID, TicketID, SessionID, ClaimID                     string
	ExpectedRevision                                            int
	Title, Body, AssignedAgentID, Summary, Evidence, QA, Reason string
	Human                                                       bool
	Validation                                                  *WorkflowInput
	Location                                                    gitctx.Context
}
type CreateTicketInput = TicketInput
type AssignTicketInput = TicketInput
type ClaimTicketInput = TicketInput
type NoteTicketInput = TicketInput
type SubmitTicketInput = TicketInput
type AcceptTicketInput = TicketInput
type RejectTicketInput = TicketInput
type ReleaseTicketInput = TicketInput
type TicketRecord struct {
	ID              string        `json:"id"`
	ProjectID       string        `json:"project_id"`
	DisplayKey      string        `json:"display_key"`
	Title           string        `json:"title"`
	Body            string        `json:"body"`
	State           string        `json:"state"`
	Revision        int           `json:"revision"`
	AssignedAgentID *string       `json:"assigned_agent_id"`
	Summary         string        `json:"summary"`
	Evidence        string        `json:"evidence"`
	QA              string        `json:"qa"`
	CreatedAt       string        `json:"created_at"`
	UpdatedAt       string        `json:"updated_at"`
	ClaimID         string        `json:"claim_id,omitempty"`
	Active          bool          `json:"active"`
	EventID         string        `json:"event_id,omitempty"`
	Warnings        []string      `json:"warnings,omitempty"`
	Workflow        *WorkflowSpec `json:"workflow,omitempty"`
}
type ticketReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readTicket(ctx context.Context, c ticketReader, p, id string) (TicketRecord, error) {
	var v TicketRecord
	e := c.QueryRowContext(ctx, `SELECT id,project_id,display_key,title,body,state,revision,assigned_agent_id,summary,evidence,qa,created_at,updated_at,EXISTS(SELECT 1 FROM claims WHERE ticket_id=tickets.id AND released_at IS NULL) FROM tickets WHERE project_id=? AND (id=? OR display_key=?) ORDER BY CASE WHEN id=? THEN 0 ELSE 1 END LIMIT 1`, p, id, id, id).Scan(&v.ID, &v.ProjectID, &v.DisplayKey, &v.Title, &v.Body, &v.State, &v.Revision, &v.AssignedAgentID, &v.Summary, &v.Evidence, &v.QA, &v.CreatedAt, &v.UpdatedAt, &v.Active)
	if e == sql.ErrNoRows {
		return v, Fail("NOT_FOUND", "ticket not found")
	}
	if e == nil {
		v.Workflow, e = workflowSpec(ctx, c, p, v.ID)
	}
	return v, e
}
func ticketAgent(ctx context.Context, c ticketReader, p, id string) (*string, error) {
	if id == "" || id == "none" {
		return nil, nil
	}
	var aid string
	e := c.QueryRowContext(ctx, `SELECT id FROM agents WHERE project_id=? AND (id=? OR name=?) ORDER BY CASE WHEN id=? THEN 0 ELSE 1 END LIMIT 1`, p, id, id, id).Scan(&aid)
	if e == sql.ErrNoRows {
		return nil, Fail("NOT_FOUND", "agent not found")
	}
	return &aid, e
}
func ValidateTicketInput(op string, in TicketInput) error {
	for _, b := range []string{in.Body, in.Summary, in.Evidence, in.QA, in.Reason} {
		if len(b) > TicketTextLimit || !utf8.ValidString(b) {
			return Fail("USAGE", "text must be valid UTF-8 and at most 256 KiB")
		}
	}
	required := func(v string) bool { return strings.TrimSpace(v) != "" }
	if op == "create" && (!required(in.Title) || utf8.RuneCountInString(in.Title) > 300 || !utf8.ValidString(in.Title) || !required(in.Body)) {
		return Fail("USAGE", "title requires 1–300 Unicode characters and description is required")
	}
	if op != "create" && op != "note" && in.ExpectedRevision < 1 {
		return Fail("USAGE", "positive expected revision required")
	}
	if op == "note" && !required(in.Body) {
		return Fail("USAGE", "note body required")
	}
	if op == "submit" && (!required(in.Summary) || !required(in.Evidence) || !required(in.QA)) {
		return Fail("USAGE", "summary, evidence and manual QA steps required")
	}
	if (op == "release" || op == "reject" || op == "block") && !required(in.Reason) {
		return Fail("USAGE", "reason required")
	}
	if op == "reject" && !in.Human {
		return Fail("USAGE", "explicit --human required")
	}
	if in.Human && (in.SessionID != "" || in.ClaimID != "") {
		return Fail("USAGE", "human and owner modes are mutually exclusive")
	}
	if (op == "claim" || op == "note" || op == "submit" || ((op == "release" || op == "block") && !in.Human)) && in.SessionID == "" {
		return Fail("USAGE", "session required")
	}
	return nil
}
func (s *Service) ticketMutation(ctx context.Context, r store.Request, op string, in TicketInput) (json.RawMessage, error) {
	if e := ValidateTicketInput(op, in); e != nil {
		return nil, e
	}
	actor := "local-user"
	if in.SessionID != "" {
		actor = in.SessionID
	}
	raw, e := s.Mutate(ctx, in.ProjectID, r.ID, "ticket."+op, actor, in, func(c *sql.Conn) (json.RawMessage, error) {
		var v TicketRecord
		var e error
		var eventBody string
		if op == "create" {
			if in.SessionID != "" {
				ss, err := ReadSession(ctx, c, in.ProjectID, in.SessionID)
				if err != nil {
					return nil, err
				}
				if ss.DeclaredState == "stopped" {
					return nil, Fail("SESSION_STOPPED", "session is stopped")
				}
			}
			var enabled int
			if e = c.QueryRowContext(ctx, `SELECT count(*) FROM workflow_policies WHERE project_id=?`, in.ProjectID).Scan(&enabled); e != nil {
				return nil, e
			}
			if enabled > 0 {
				v, e = CreateWorkflowDraftTx(ctx, c, in.ProjectID, actor, in.Title, in.Body, "", "implementation")
				if e != nil {
					return nil, e
				}
				aid, e := ticketAgent(ctx, c, in.ProjectID, in.AssignedAgentID)
				if e != nil {
					return nil, e
				}
				v.AssignedAgentID = aid
				if _, e = c.ExecContext(ctx, `UPDATE tickets SET assigned_agent_id=? WHERE id=?`, aid, v.ID); e != nil {
					return nil, e
				}
			} else {
				aid, e := ticketAgent(ctx, c, in.ProjectID, in.AssignedAgentID)
				if e != nil {
					return nil, e
				}
				var prefix string
				var n int
				if e = c.QueryRowContext(ctx, "SELECT prefix,next_ticket FROM projects WHERE id=?", in.ProjectID).Scan(&prefix, &n); e != nil {
					return nil, e
				}
				v = TicketRecord{ID: UUID(), ProjectID: in.ProjectID, DisplayKey: fmt.Sprintf("%s-%d", prefix, n), Title: in.Title, Body: in.Body, State: "ready", Revision: 1, AssignedAgentID: aid, CreatedAt: Now(), UpdatedAt: Now()}
				if _, e = c.ExecContext(ctx, "UPDATE projects SET next_ticket=next_ticket+1 WHERE id=?", in.ProjectID); e != nil {
					return nil, e
				}
				_, e = c.ExecContext(ctx, `INSERT INTO tickets(id,project_id,display_key,title,body,assigned_agent_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.ProjectID, v.DisplayKey, v.Title, v.Body, v.AssignedAgentID, v.CreatedAt, v.UpdatedAt)
				if e != nil {
					return nil, e
				}
			}
		} else {
			v, e = readTicket(ctx, c, in.ProjectID, in.TicketID)
			if e != nil {
				return nil, e
			}
			// Fence owner requests before revision checking, including stale owners after recovery.
			if op == "note" || op == "submit" || ((op == "release" || op == "block") && !in.Human) {
				var found string
				e = c.QueryRowContext(ctx, `SELECT c.id FROM claims c JOIN sessions s ON s.id=c.session_id WHERE c.project_id=? AND c.ticket_id=? AND c.session_id=? AND c.id=? AND c.released_at IS NULL AND s.declared_state<>'stopped'`, in.ProjectID, v.ID, in.SessionID, in.ClaimID).Scan(&found)
				if e == sql.ErrNoRows {
					return nil, Fail("CLAIM_REVOKED", "claim is missing, inactive, or belongs to another owner")
				}
				if e != nil {
					return nil, e
				}
			}
			if op != "note" && v.Revision != in.ExpectedRevision {
				return nil, &Fault{Code: "REVISION_CONFLICT", Message: "ticket revision changed", Details: map[string]any{"expected": in.ExpectedRevision, "actual": v.Revision}}
			}
			switch op {
			case "assign":
				var n int
				if e = c.QueryRowContext(ctx, "SELECT count(*) FROM claims WHERE ticket_id=? AND released_at IS NULL", v.ID).Scan(&n); e != nil {
					return nil, e
				}
				if n > 0 {
					return nil, Fail("ACTIVE_CLAIM", "release ownership before assignment")
				}
				v.AssignedAgentID, e = ticketAgent(ctx, c, in.ProjectID, in.AssignedAgentID)
				if e != nil {
					return nil, e
				}
			case "claim":
				if e = CheckWorkflowEligibilityTx(ctx, c, in.ProjectID, v.ID); e != nil {
					return nil, e
				}
				session, e := ReadSession(ctx, c, in.ProjectID, in.SessionID)
				if e != nil {
					return nil, e
				}
				if session.DeclaredState == "stopped" {
					return nil, Fail("SESSION_STOPPED", "session is stopped")
				}
				if v.AssignedAgentID != nil && *v.AssignedAgentID != session.AgentID {
					return nil, Fail("ASSIGNMENT_CONFLICT", "ticket assigned to another agent")
				}
				if session.Busy {
					return nil, Fail("ACTIVE_CLAIM", "session already owns a ticket")
				}
				if v.State != "ready" {
					return nil, Fail("INVALID_STATE", "ticket must be ready")
				}
				var common string
				if e = c.QueryRowContext(ctx, "SELECT common_dir FROM projects WHERE id=?", in.ProjectID).Scan(&common); e != nil {
					return nil, e
				}
				if common != in.Location.CommonDir {
					return nil, Fail("PROJECT_MISMATCH", "claim location belongs to another project")
				}
				v.ClaimID = UUID()
				v.Active = true
				v.State = "in_progress"
				if in.Location.Primary {
					v.Warnings = []string{"PRIMARY_CHECKOUT: use an isolated worktree for implementation"}
				}
				_, e = c.ExecContext(ctx, `INSERT INTO claims(id,project_id,ticket_id,session_id,created_at,worktree_root,branch,head) VALUES(?,?,?,?,?,?,?,?)`, v.ClaimID, in.ProjectID, v.ID, in.SessionID, Now(), in.Location.Root, in.Location.Branch, in.Location.Head)
				if e != nil {
					return nil, e
				}
			case "note", "submit", "release", "block":
				if v.State != "in_progress" {
					return nil, Fail("INVALID_STATE", "ticket must be in progress")
				}
				if op == "note" {
					eventBody = in.Body
				} else {
					v.State = "ready"
					if v.Workflow != nil && (v.Workflow.Paused || v.Workflow.PreparedRevision != v.Workflow.SpecRevision || v.Workflow.AuthorizedRevision != v.Workflow.SpecRevision) {
						v.State = "draft"
					}
					v.Active = false
					eventBody = in.Reason
					if op == "submit" {
						if e = CheckWorkflowEligibilityTx(ctx, c, in.ProjectID, v.ID); e != nil {
							return nil, e
						}
						sp, e := workflowSpec(ctx, c, in.ProjectID, v.ID)
						if e != nil {
							return nil, e
						}
						if sp != nil {
							if strings.TrimSpace(in.Commit) == "" {
								return nil, Fail("COMMIT_REQUIRED", "managed submission requires tested commit")
							}
							if _, e = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET submitted_commit=?,submitted_spec_revision=spec_revision WHERE ticket_id=?`, in.Commit, v.ID); e != nil {
								return nil, e
							}
						}
						v.State = "review"
						v.Summary = in.Summary
						v.Evidence = in.Evidence
						v.QA = in.QA
						eventBody = in.Summary
					}
					if op == "block" {
						sp, e := workflowSpec(ctx, c, in.ProjectID, v.ID)
						if e != nil {
							return nil, e
						}
						if sp == nil {
							return nil, Fail("LEGACY_TICKET", "block requires workflow ticket")
						}
						v.State = "blocked"
						if _, e = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET blocked_reason=? WHERE ticket_id=?`, in.Reason, v.ID); e != nil {
							return nil, e
						}
					}
					if _, e = c.ExecContext(ctx, "UPDATE claims SET released_at=? WHERE ticket_id=? AND released_at IS NULL", Now(), v.ID); e != nil {
						return nil, e
					}
				}
			case "accept", "reject":
				if v.State != "review" {
					return nil, Fail("INVALID_STATE", "ticket must be in review")
				}
				if op == "accept" {
					if e = validateWorkflowAcceptance(ctx, c, in, v); e != nil {
						return nil, e
					}
				}
				v.State = "done"
				if op == "reject" {
					v.State = "ready"
					v.Active = false
					eventBody = in.Reason
				}
			}
			v.Revision++
			v.UpdatedAt = Now()
			_, e = c.ExecContext(ctx, `UPDATE tickets SET state=?,revision=?,assigned_agent_id=?,summary=?,evidence=?,qa=?,updated_at=? WHERE id=?`, v.State, v.Revision, v.AssignedAgentID, v.Summary, v.Evidence, v.QA, v.UpdatedAt, v.ID)
			if e != nil {
				return nil, e
			}
		}
		// Session observations are business writes: replay skips this callback and
		// any later event/journal failure rolls the observation back as well.
		if in.SessionID != "" {
			if op == "claim" {
				_, e = c.ExecContext(ctx, "UPDATE sessions SET last_seen_at=?,worktree_root=?,branch=?,head=? WHERE project_id=? AND id=?", Now(), in.Location.Root, in.Location.Branch, in.Location.Head, in.ProjectID, in.SessionID)
			} else {
				_, e = c.ExecContext(ctx, "UPDATE sessions SET last_seen_at=? WHERE project_id=? AND id=?", Now(), in.ProjectID, in.SessionID)
			}
			if e != nil {
				return nil, e
			}
		}
		v.EventID = UUID()
		v.Workflow, e = workflowSpec(ctx, c, in.ProjectID, v.ID)
		if e != nil {
			return nil, e
		}
		_, e = c.ExecContext(ctx, `INSERT INTO events(id,project_id,ticket_id,actor_id,kind,body,payload,created_at) VALUES(?,?,?,?,?,?,?,?)`, v.EventID, in.ProjectID, v.ID, actor, "ticket."+op, eventBody, string(JSON(in)), Now())
		return JSON(v), e
	})
	if e != nil {
		return nil, e
	}
	if op == "claim" {
		var v TicketRecord
		if e = json.Unmarshal(raw, &v); e != nil {
			return nil, e
		}
		var n int
		if e = s.Store.DB.QueryRowContext(ctx, "SELECT count(*) FROM claims WHERE id=? AND released_at IS NULL", v.ClaimID).Scan(&n); e != nil {
			return nil, e
		}
		v.Active = n == 1
		return JSON(v), nil
	}
	return raw, nil
}
func (s *Service) CreateTicket(c context.Context, r store.Request, i CreateTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "create", i)
}
func (s *Service) AssignTicket(c context.Context, r store.Request, i AssignTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "assign", i)
}
func (s *Service) ClaimTicket(c context.Context, r store.Request, i ClaimTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "claim", i)
}
func (s *Service) NoteTicket(c context.Context, r store.Request, i NoteTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "note", i)
}
func (s *Service) SubmitTicket(c context.Context, r store.Request, i SubmitTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "submit", i)
}
func (s *Service) AcceptTicket(c context.Context, r store.Request, i AcceptTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "accept", i)
}
func (s *Service) RejectTicket(c context.Context, r store.Request, i RejectTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "reject", i)
}
func (s *Service) ReleaseTicket(c context.Context, r store.Request, i ReleaseTicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "release", i)
}
func (s *Service) ShowTicket(ctx context.Context, p, id string) (map[string]any, error) {
	v, e := readTicket(ctx, s.Store.DB, p, id)
	if e != nil {
		return nil, e
	}
	var result map[string]any
	json.Unmarshal(JSON(v), &result)
	rows, e := s.Store.DB.QueryContext(ctx, "SELECT id,seq,kind,body,actor_id,created_at FROM events WHERE project_id=? AND ticket_id=? ORDER BY seq DESC LIMIT 21", p, v.ID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	events := []map[string]any{}
	for rows.Next() {
		var id, kind, body, actor, created string
		var seq int64
		if e = rows.Scan(&id, &seq, &kind, &body, &actor, &created); e != nil {
			return nil, e
		}
		events = append(events, map[string]any{"id": id, "seq": seq, "kind": kind, "body": body, "actor_id": actor, "created_at": created})
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	result["events_truncated"] = len(events) > 20
	if len(events) > 20 {
		events = events[:20]
	}
	result["events"] = events
	if v.Workflow != nil {
		history, err := s.workflowTicketHistory(ctx, p, v.ID)
		if err != nil {
			return nil, err
		}
		result["workflow_history"] = history
	}
	return result, nil
}
func (s *Service) ListTickets(ctx context.Context, p, state, assigned, cursor string, limit int) (map[string]any, error) {
	if limit < 1 || limit > 100 {
		return nil, Fail("USAGE", "limit must be 1–100")
	}
	if state != "" && state != "ready" && state != "in_progress" && state != "review" && state != "done" && state != "draft" && state != "blocked" {
		return nil, Fail("USAGE", "invalid ticket state")
	}
	query := "SELECT id FROM tickets WHERE project_id=? AND id>?"
	args := []any{p, cursor}
	if state != "" {
		query += " AND state=?"
		args = append(args, state)
	}
	if assigned != "" {
		aid, e := ticketAgent(ctx, s.Store.DB, p, assigned)
		if e != nil {
			return nil, e
		}
		if aid == nil {
			query += " AND assigned_agent_id IS NULL"
		} else {
			query += " AND assigned_agent_id=?"
			args = append(args, *aid)
		}
	}
	query += " ORDER BY id LIMIT ?"
	args = append(args, limit+1)
	rows, e := s.Store.DB.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	truncated := len(ids) > limit
	next := ""
	if truncated {
		ids = ids[:limit]
		next = ids[len(ids)-1]
	}
	items := []TicketRecord{}
	for _, id := range ids {
		v, e := readTicket(ctx, s.Store.DB, p, id)
		if e != nil {
			return nil, e
		}
		items = append(items, v)
	}
	return map[string]any{"tickets": items, "truncated": truncated, "next_cursor": next}, nil
}

func (s *Service) BlockTicket(c context.Context, r store.Request, i TicketInput) (json.RawMessage, error) {
	return s.ticketMutation(c, r, "block", i)
}
