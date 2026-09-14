package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/store"
	"strings"
)

// TicketMetadata is universal ticket data. WorkflowSpec is an optional policy adapter.
type TicketMetadata struct {
	SpecRevision          int     `json:"spec_revision"`
	ParentID              *string `json:"parent_id"`
	Kind                  string  `json:"kind"`
	BlockedReason         string  `json:"blocked_reason"`
	SubmittedCommit       string  `json:"submitted_commit"`
	SubmittedSpecRevision int     `json:"submitted_spec_revision"`
	LegacyHuman           bool    `json:"legacy_human"`
}

func ticketMetadata(ctx context.Context, c ticketReader, p, id string) (*TicketMetadata, error) {
	var exists int
	if e := c.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='ticket_metadata'`).Scan(&exists); e != nil {
		return nil, e
	}
	if exists == 0 {
		return nil, nil
	}
	m := &TicketMetadata{}
	e := c.QueryRowContext(ctx, `SELECT spec_revision,parent_id,kind,blocked_reason,submitted_commit,submitted_spec_revision,legacy_human FROM ticket_metadata WHERE project_id=? AND ticket_id=?`, p, id).Scan(&m.SpecRevision, &m.ParentID, &m.Kind, &m.BlockedReason, &m.SubmittedCommit, &m.SubmittedSpecRevision, &m.LegacyHuman)
	return m, e
}

func CheckTicketDependenciesTx(ctx context.Context, c *sql.Conn, p, id string) error {
	var n int
	e := c.QueryRowContext(ctx, `SELECT count(*) FROM ticket_dependencies d JOIN tickets t ON t.id=d.depends_on WHERE d.project_id=? AND d.ticket_id=? AND t.state<>'done'`, p, id).Scan(&n)
	if e != nil {
		return e
	}
	if n > 0 {
		return Fail("DEPENDENCY_BLOCKED", "prerequisites are not done")
	}
	return nil
}

// recordTicketDecisionTx writes evidence only; policy acceptance checks run separately.
func recordTicketDecisionTx(ctx context.Context, c *sql.Conn, op string, in TicketInput, v TicketRecord) error {
	var session, agent any
	if !in.Human {
		ss, e := ReadSession(ctx, c, in.ProjectID, in.SessionID)
		if e != nil {
			return e
		}
		if ss.DeclaredState == "stopped" {
			return Fail("SESSION_STOPPED", "decision session is stopped")
		}
		session = in.SessionID
		agent = ss.AgentID
	}
	m, e := ticketMetadata(ctx, c, in.ProjectID, v.ID)
	if e != nil {
		return e
	}
	if m.LegacyHuman && !in.Human {
		return Fail("HUMAN_REQUIRED", "legacy ticket requires explicit human decision")
	}
	d := in.Validation
	if d == nil {
		d = &WorkflowInput{Commit: in.Commit, Evidence: in.Evidence}
	}
	mode := "agent"
	if in.Human {
		mode = "human"
	}
	if v.Workflow != nil {
		mode = v.Workflow.ValidationMode
	}
	outcome := "accepted"
	if op == "reject" {
		outcome = "rejected"
	}
	_, e = c.ExecContext(ctx, `INSERT INTO ticket_decisions(id,project_id,ticket_id,spec_revision,mode,session_id,agent_id,context_id,commit_id,criteria,evidence,checks,created_at,outcome,reason) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, UUID(), in.ProjectID, v.ID, m.SpecRevision, mode, session, agent, d.ContextID, d.Commit, d.Criteria, d.Evidence, string(JSON(d.Checks)), Now(), outcome, in.Reason)
	return e
}

// EditTicket and UnblockTicket retain the shipped JSON specification input format.
// Configured tickets use the optional policy path; plain tickets need only attribution.
func (s *Service) EditTicket(ctx context.Context, r store.Request, in WorkflowInput) (json.RawMessage, error) {
	return s.foundationEdit(ctx, r, "edit", in)
}

// EditTicketMetadata corrects parent or kind without changing a plain ticket's specification or lifecycle evidence.
func (s *Service) EditTicketMetadata(ctx context.Context, r store.Request, in WorkflowInput) (json.RawMessage, error) {
	if len(JSON(in)) > TicketTextLimit {
		return nil, Fail("USAGE", "ticket metadata body exceeds 256 KiB")
	}
	return s.Mutate(ctx, in.ProjectID, r.ID, "ticket.metadata", in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		actor, e := workflowActor(ctx, c, in, false)
		if e != nil {
			return nil, e
		}
		v, e := readTicket(ctx, c, in.ProjectID, in.TicketID)
		if e != nil {
			return nil, e
		}
		// Events store a UUID foreign key, while callers may use the supported display key.
		in.TicketID = v.ID
		if v.Workflow != nil {
			return nil, Fail("WORKFLOW_TICKET", "metadata-only edits require a plain ticket")
		}
		if v.Revision != in.ExpectedRevision {
			return nil, Fail("REVISION_CONFLICT", "ticket revision changed")
		}
		if v.Active {
			return nil, Fail("ACTIVE_CLAIM", "release ownership before editing ticket metadata")
		}
		if strings.TrimSpace(in.Reason) == "" {
			return nil, Fail("USAGE", "reason required")
		}
		var parent any
		if in.ParentID != "" {
			p, e := readTicket(ctx, c, in.ProjectID, in.ParentID)
			if e != nil {
				return nil, e
			}
			parent = p.ID
			var n int
			e = c.QueryRowContext(ctx, `WITH RECURSIVE ancestors(id) AS(SELECT ? UNION SELECT m.parent_id FROM ticket_metadata m JOIN ancestors a ON m.ticket_id=a.id WHERE m.parent_id IS NOT NULL) SELECT count(*) FROM ancestors WHERE id=?`, p.ID, v.ID).Scan(&n)
			if e != nil {
				return nil, e
			}
			if n > 0 {
				return nil, Fail("CYCLE", "parent cycle")
			}
		}
		if in.Kind == "" {
			in.Kind = v.Metadata.Kind
		}
		if _, e = c.ExecContext(ctx, `UPDATE ticket_metadata SET parent_id=?,kind=? WHERE ticket_id=?`, parent, in.Kind, v.ID); e != nil {
			return nil, e
		}
		if _, e = c.ExecContext(ctx, `UPDATE tickets SET revision=revision+1,updated_at=? WHERE id=?`, Now(), v.ID); e != nil {
			return nil, e
		}
		result, e := readTicket(ctx, c, in.ProjectID, v.ID)
		if e != nil {
			return nil, e
		}
		return workflowEvent(ctx, c, in, actor, "metadata", map[string]any{"ticket": result})
	})
}
func (s *Service) UnblockTicket(ctx context.Context, r store.Request, in WorkflowInput) (json.RawMessage, error) {
	return s.foundationEdit(ctx, r, "unblock", in)
}
func (s *Service) foundationEdit(ctx context.Context, r store.Request, op string, in WorkflowInput) (json.RawMessage, error) {
	// Route policy requests before selecting the replay operation; both paths retain workflow.* journal identity.
	v, e := readTicket(ctx, s.Store.DB, in.ProjectID, in.TicketID)
	if e != nil {
		return nil, e
	}
	if v.Workflow != nil {
		return s.Workflow(ctx, r, op, in)
	}
	if len(JSON(in)) > TicketTextLimit {
		return nil, Fail("USAGE", "ticket body exceeds 256 KiB")
	}
	actor := in.SessionID
	if in.Human {
		actor = "local-user"
	}
	return s.Mutate(ctx, in.ProjectID, r.ID, "workflow."+op, actor, in, func(c *sql.Conn) (json.RawMessage, error) {
		actor, e := workflowActor(ctx, c, in, false)
		if e != nil {
			return nil, e
		}
		v, e := readTicket(ctx, c, in.ProjectID, in.TicketID)
		if e != nil {
			return nil, e
		}
		in.TicketID = v.ID
		if v.Workflow != nil {
			return nil, Fail("POLICY_CHANGED", "retry using the current ticket policy")
		}
		if v.Revision != in.ExpectedRevision {
			return nil, Fail("REVISION_CONFLICT", "ticket revision changed")
		}
		if strings.TrimSpace(in.Reason) == "" {
			return nil, Fail("USAGE", "reason required")
		}
		if op == "unblock" {
			if v.State != "blocked" {
				return nil, Fail("INVALID_STATE", "ticket must be blocked")
			}
			if _, e = c.ExecContext(ctx, `UPDATE ticket_metadata SET blocked_reason='' WHERE ticket_id=?`, v.ID); e != nil {
				return nil, e
			}
			if _, e = c.ExecContext(ctx, `UPDATE tickets SET state='ready' WHERE id=?`, v.ID); e != nil {
				return nil, e
			}
		} else {
			if e = ValidateTicketInput("create", TicketInput{Title: in.Title, Body: in.Body}); e != nil {
				return nil, e
			}
			if v.Active {
				return nil, Fail("ACTIVE_CLAIM", "release ownership before editing a plain ticket")
			}
			if in.ValidationMode != "" || in.ExecutionMode != "" || in.PlanReview != "" || len(in.RequiredChecks) > 0 {
				return nil, Fail("WORKFLOW_DISABLED", "configure workflow before setting policy")
			}
			var parent any
			if in.ParentID != "" {
				p, e := readTicket(ctx, c, in.ProjectID, in.ParentID)
				if e != nil {
					return nil, e
				}
				parent = p.ID
				var n int
				e = c.QueryRowContext(ctx, `WITH RECURSIVE ancestors(id) AS(SELECT ? UNION SELECT m.parent_id FROM ticket_metadata m JOIN ancestors a ON m.ticket_id=a.id WHERE m.parent_id IS NOT NULL) SELECT count(*) FROM ancestors WHERE id=?`, p.ID, v.ID).Scan(&n)
				if e != nil {
					return nil, e
				}
				if n > 0 {
					return nil, Fail("CYCLE", "parent cycle")
				}
			}
			if _, e = c.ExecContext(ctx, `DELETE FROM ticket_dependencies WHERE ticket_id=?`, v.ID); e != nil {
				return nil, e
			}
			for _, dep := range in.Dependencies {
				d, e := readTicket(ctx, c, in.ProjectID, dep)
				if e != nil {
					return nil, e
				}
				var n int
				e = c.QueryRowContext(ctx, `WITH RECURSIVE deps(id) AS(SELECT ? UNION SELECT d.depends_on FROM ticket_dependencies d JOIN deps a ON d.ticket_id=a.id) SELECT count(*) FROM deps WHERE id=?`, d.ID, v.ID).Scan(&n)
				if e != nil {
					return nil, e
				}
				if n > 0 {
					return nil, Fail("CYCLE", "dependency cycle")
				}
				if _, e = c.ExecContext(ctx, `INSERT INTO ticket_dependencies VALUES(?,?,?)`, in.ProjectID, v.ID, d.ID); e != nil {
					return nil, e
				}
			}
			if in.Kind == "" {
				in.Kind = v.Metadata.Kind
			}
			if _, e = c.ExecContext(ctx, `UPDATE ticket_metadata SET spec_revision=spec_revision+1,parent_id=?,kind=? WHERE ticket_id=?`, parent, in.Kind, v.ID); e != nil {
				return nil, e
			}
			if _, e = c.ExecContext(ctx, `UPDATE tickets SET title=?,body=?,state=CASE WHEN state IN('review','done') THEN 'ready' ELSE state END WHERE id=?`, in.Title, in.Body, v.ID); e != nil {
				return nil, e
			}
			if e = InvalidateTicketPolicyTx(ctx, c, in.ProjectID, v.ID); e != nil {
				return nil, e
			}
			if _, e = c.ExecContext(ctx, `INSERT INTO ticket_versions VALUES(?,?,?,?,?,?,?)`, UUID(), in.ProjectID, v.ID, v.Metadata.SpecRevision+1, actor, string(JSON(in)), Now()); e != nil {
				return nil, e
			}
		}
		if _, e = c.ExecContext(ctx, `UPDATE tickets SET revision=revision+1,updated_at=? WHERE id=?`, Now(), v.ID); e != nil {
			return nil, e
		}
		result, e := readTicket(ctx, c, in.ProjectID, v.ID)
		if e != nil {
			return nil, e
		}
		return workflowEvent(ctx, c, in, actor, op, map[string]any{"ticket": result})
	})
}

// InvalidateTicketPolicyTx invalidates optional planning approvals downstream of a
// material foundation edit. Plain dependents never enter an unclaimable draft state.
func InvalidateTicketPolicyTx(ctx context.Context, c *sql.Conn, p, id string) error {
	if _, e := c.ExecContext(ctx, `WITH RECURSIVE affected(id) AS(SELECT ? UNION SELECT d.ticket_id FROM ticket_dependencies d JOIN affected a ON d.depends_on=a.id) UPDATE workflow_ticket_specs SET prepared_revision=0,authorized_revision=0,paused=CASE WHEN EXISTS(SELECT 1 FROM claims WHERE claims.ticket_id=workflow_ticket_specs.ticket_id AND released_at IS NULL) THEN 1 ELSE 0 END WHERE project_id=? AND ticket_id IN(SELECT id FROM affected)`, id, p); e != nil {
		return e
	}
	_, e := c.ExecContext(ctx, `WITH RECURSIVE affected(id) AS(SELECT ? UNION SELECT d.ticket_id FROM ticket_dependencies d JOIN affected a ON d.depends_on=a.id) UPDATE tickets SET state=CASE WHEN state IN('ready','review','done') THEN 'draft' ELSE state END,revision=revision+CASE WHEN id=? THEN 0 ELSE 1 END,updated_at=? WHERE project_id=? AND id IN(SELECT id FROM affected) AND EXISTS(SELECT 1 FROM workflow_ticket_specs s WHERE s.ticket_id=tickets.id)`, id, id, Now(), p)
	return e
}
