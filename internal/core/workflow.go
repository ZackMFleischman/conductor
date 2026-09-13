package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/store"
	"strings"
)

type WorkflowInput struct {
	ProjectID        string            `json:"-"`
	TicketID         string            `json:"ticket_id,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	Human            bool              `json:"-"`
	ExpectedRevision int               `json:"expected_revision"`
	CoordinationID   string            `json:"coordination_id,omitempty"`
	Intent           string            `json:"intent,omitempty"`
	Reason           string            `json:"reason,omitempty"`
	ValidationMode   string            `json:"validation_mode,omitempty"`
	ExecutionMode    string            `json:"execution_mode,omitempty"`
	PlanReview       string            `json:"plan_review,omitempty"`
	RequiredChecks   []string          `json:"required_checks,omitempty"`
	Title            string            `json:"title,omitempty"`
	Body             string            `json:"body,omitempty"`
	ParentID         string            `json:"parent_id,omitempty"`
	Kind             string            `json:"kind,omitempty"`
	Dependencies     []string          `json:"dependencies,omitempty"`
	ContextID        string            `json:"context_id,omitempty"`
	Significant      bool              `json:"significant,omitempty"`
	CritiqueID       string            `json:"critique_id,omitempty"`
	Commit           string            `json:"commit,omitempty"`
	Criteria         string            `json:"criteria,omitempty"`
	Evidence         string            `json:"evidence,omitempty"`
	Checks           map[string]string `json:"checks,omitempty"`
}
type WorkflowSpec struct {
	SpecRevision       int      `json:"spec_revision"`
	ParentID           *string  `json:"parent_id"`
	Kind               string   `json:"kind"`
	ValidationMode     string   `json:"validation_mode"`
	RequiredChecks     []string `json:"required_checks"`
	PolicyRevision     int      `json:"policy_revision"`
	ExecutionMode      string   `json:"execution_mode"`
	PlanReview         string   `json:"plan_review"`
	PreparedRevision   int      `json:"prepared_revision"`
	AuthorizedRevision int      `json:"authorized_revision"`
	Paused             bool     `json:"paused"`
	BlockedReason      string   `json:"blocked_reason"`
}

func workflowSpec(ctx context.Context, c ticketReader, p, id string) (*WorkflowSpec, error) {
	var exists int
	if e := c.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='workflow_ticket_specs'`).Scan(&exists); e != nil {
		return nil, e
	}
	if exists == 0 {
		return nil, nil
	}
	v := &WorkflowSpec{}
	var checks string
	e := c.QueryRowContext(ctx, `SELECT spec_revision,parent_id,kind,validation_mode,required_checks,policy_revision,execution_mode,plan_review,prepared_revision,authorized_revision,paused,blocked_reason FROM workflow_ticket_specs WHERE project_id=? AND ticket_id=?`, p, id).Scan(&v.SpecRevision, &v.ParentID, &v.Kind, &v.ValidationMode, &checks, &v.PolicyRevision, &v.ExecutionMode, &v.PlanReview, &v.PreparedRevision, &v.AuthorizedRevision, &v.Paused, &v.BlockedReason)
	if e == sql.ErrNoRows {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal([]byte(checks), &v.RequiredChecks)
	return v, e
}
func workflowActor(ctx context.Context, c *sql.Conn, in WorkflowInput, coordinator bool) (string, error) {
	if in.Human {
		if in.SessionID != "" {
			return "", Fail("USAGE", "human and session modes are mutually exclusive")
		}
		return "local-user", nil
	}
	ss, e := ReadSession(ctx, c, in.ProjectID, in.SessionID)
	if e != nil {
		return "", e
	}
	if ss.DeclaredState == "stopped" {
		return "", Fail("SESSION_STOPPED", "session is stopped")
	}
	if coordinator {
		var n int
		e = c.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='team_runs'`).Scan(&n)
		if e != nil {
			return "", e
		}
		if n == 0 {
			return "", Fail("AUTHORITY_REQUIRED", "active team coordinator required")
		}
		e = c.QueryRowContext(ctx, `SELECT count(*) FROM team_runs WHERE project_id=? AND coordinator_session_id=? AND coordination_id=? AND coordination_id IS NOT NULL AND status NOT IN ('stopped','released')`, in.ProjectID, in.SessionID, in.CoordinationID).Scan(&n)
		if e != nil {
			return "", e
		}
		if n != 1 {
			return "", Fail("AUTHORITY_REQUIRED", "active team coordination token required")
		}
	}
	return in.SessionID, nil
}
func CreateWorkflowDraftTx(ctx context.Context, c *sql.Conn, project, session, title, body, parentID, kind string) (TicketRecord, error) {
	var v TicketRecord
	if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
		return v, Fail("USAGE", "title and body required")
	}
	var prefix string
	var n int
	if e := c.QueryRowContext(ctx, `SELECT prefix,next_ticket FROM projects WHERE id=?`, project).Scan(&prefix, &n); e != nil {
		return v, e
	}
	v = TicketRecord{ID: UUID(), ProjectID: project, DisplayKey: fmt.Sprintf("%s-%d", prefix, n), Title: title, Body: body, State: "draft", Revision: 1, CreatedAt: Now(), UpdatedAt: Now()}
	if _, e := c.ExecContext(ctx, `INSERT INTO tickets(id,project_id,display_key,title,body,state,created_at,updated_at) VALUES(?,?,?,?,?,'draft',?,?)`, v.ID, project, v.DisplayKey, title, body, v.CreatedAt, v.UpdatedAt); e != nil {
		return v, e
	}
	if _, e := c.ExecContext(ctx, `UPDATE projects SET next_ticket=next_ticket+1 WHERE id=?`, project); e != nil {
		return v, e
	}
	if kind == "" {
		kind = "implementation"
	}
	var parent any
	if parentID != "" {
		p, e := readTicket(ctx, c, project, parentID)
		if e != nil {
			return v, e
		}
		parent = p.ID
	}
	res, e := c.ExecContext(ctx, `INSERT INTO workflow_ticket_specs(ticket_id,project_id,parent_id,kind,validation_mode,required_checks,policy_revision,execution_mode,plan_review) SELECT ?,?,?,?,validation_mode,required_checks,revision,execution_mode,plan_review FROM workflow_policies WHERE project_id=?`, v.ID, project, parent, kind, project)
	if e != nil {
		return v, e
	}
	nrows, _ := res.RowsAffected()
	if nrows != 1 {
		return v, Fail("WORKFLOW_DISABLED", "configure workflow before creating drafts")
	}
	_, e = c.ExecContext(ctx, `INSERT INTO workflow_versions VALUES(?,?,?,?,?,?,?)`, UUID(), project, v.ID, 1, session, string(JSON(v)), Now())
	return v, e
}
func CheckWorkflowEligibilityTx(ctx context.Context, c *sql.Conn, p, id string) error {
	v, e := workflowSpec(ctx, c, p, id)
	if e != nil || v == nil {
		return e
	}
	if v.Paused || v.BlockedReason != "" {
		return Fail("WORK_PAUSED", "ticket is paused or blocked")
	}
	if v.PreparedRevision != v.SpecRevision || v.AuthorizedRevision != v.SpecRevision {
		return Fail("NOT_AUTHORIZED", "current specification must be prepared and authorized")
	}
	var n int
	e = c.QueryRowContext(ctx, `SELECT count(*) FROM workflow_dependencies d JOIN tickets t ON t.id=d.depends_on WHERE d.project_id=? AND d.ticket_id=? AND t.state<>'done'`, p, id).Scan(&n)
	if e != nil {
		return e
	}
	if n > 0 {
		return Fail("DEPENDENCY_BLOCKED", "prerequisites are not done")
	}
	e = c.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE name='retrospective_deferrals' AND type='table'`).Scan(&n)
	if e != nil {
		return e
	}
	if n > 0 {
		if e = c.QueryRowContext(ctx, `SELECT count(*) FROM retrospective_deferrals WHERE project_id=? AND ticket_id=? AND released_at IS NULL`, p, id).Scan(&n); e != nil {
			return e
		}
		if n > 0 {
			return Fail("DEFERRED", "ticket remains deferred")
		}
	}
	return nil
}
func (s *Service) Workflow(ctx context.Context, r store.Request, op string, in WorkflowInput) (json.RawMessage, error) {
	if len(JSON(in)) > TicketTextLimit {
		return nil, Fail("USAGE", "workflow body exceeds 256 KiB")
	}
	actor := in.SessionID
	if in.Human {
		actor = "local-user"
	}
	return s.Mutate(ctx, in.ProjectID, r.ID, "workflow."+op, actor, in, func(c *sql.Conn) (json.RawMessage, error) {
		if op == "configure" {
			if !in.Human || in.SessionID != "" {
				return nil, Fail("AUTHORITY_REQUIRED", "policy configuration requires explicit local human authority")
			}
			if in.Reason == "" {
				return nil, Fail("USAGE", "reason required")
			}
			if in.ExecutionMode != "human" && in.ExecutionMode != "delegated" {
				return nil, Fail("USAGE", "execution_mode must be human or delegated")
			}
			if in.PlanReview != "lightweight" && in.PlanReview != "independent_agent" {
				return nil, Fail("USAGE", "plan_review must be lightweight or independent_agent")
			}
			if in.ValidationMode != "human" && in.ValidationMode != "independent_agent" && in.ValidationMode != "automated" {
				return nil, Fail("USAGE", "invalid validation_mode")
			}
			if in.ValidationMode == "automated" && len(in.RequiredChecks) == 0 {
				return nil, Fail("USAGE", "automated mode requires named checks")
			}
			var rev int
			e := c.QueryRowContext(ctx, `SELECT revision FROM workflow_policies WHERE project_id=?`, in.ProjectID).Scan(&rev)
			if e != nil && e != sql.ErrNoRows {
				return nil, e
			}
			if rev != in.ExpectedRevision {
				return nil, Fail("REVISION_CONFLICT", "policy revision changed")
			}
			if rev == 0 && strings.TrimSpace(in.Intent) == "" {
				return nil, Fail("USAGE", "original intent required")
			}
			_, e = c.ExecContext(ctx, `INSERT INTO workflow_policies VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(project_id) DO UPDATE SET revision=excluded.revision,execution_mode=excluded.execution_mode,plan_review=excluded.plan_review,validation_mode=excluded.validation_mode,required_checks=excluded.required_checks,authority=excluded.authority,reason=excluded.reason,created_at=excluded.created_at`, in.ProjectID, rev+1, in.ExecutionMode, in.PlanReview, in.ValidationMode, string(JSON(in.RequiredChecks)), actor, in.Reason, Now())
			if e != nil {
				return nil, e
			}
			if rev == 0 {
				_, e = c.ExecContext(ctx, `INSERT INTO workflow_intents VALUES(?,?,?,?,?,?)`, UUID(), in.ProjectID, "original", in.Intent, actor, Now())
				if e != nil {
					return nil, e
				}
			}
			return workflowEvent(ctx, c, in, actor, op, map[string]any{"policy_revision": rev + 1})
		}
		if op == "amend" {
			if !in.Human || in.SessionID != "" || strings.TrimSpace(in.Intent) == "" {
				return nil, Fail("AUTHORITY_REQUIRED", "user amendment requires human authority and intent")
			}
			_, e := c.ExecContext(ctx, `INSERT INTO workflow_intents VALUES(?,?,?,?,?,?)`, UUID(), in.ProjectID, "amendment", in.Intent, actor, Now())
			if e != nil {
				return nil, e
			}
			return workflowEvent(ctx, c, in, actor, op, in)
		}
		coordinator := op != "critique"
		a, e := workflowActor(ctx, c, in, coordinator)
		if e != nil {
			return nil, e
		}
		actor = a
		v, e := readTicket(ctx, c, in.ProjectID, in.TicketID)
		if e != nil {
			return nil, e
		}
		in.TicketID = v.ID
		if v.Revision != in.ExpectedRevision {
			return nil, Fail("REVISION_CONFLICT", "ticket revision changed")
		}
		sp, e := workflowSpec(ctx, c, in.ProjectID, v.ID)
		if e != nil {
			return nil, e
		}
		if sp == nil {
			return nil, Fail("LEGACY_TICKET", "legacy ticket retains human workflow")
		}
		switch op {
		case "edit":
			if in.Title == "" || in.Body == "" {
				return nil, Fail("USAGE", "edit requires complete title/body/specification")
			}
			if in.Reason == "" {
				return nil, Fail("USAGE", "material change summary required")
			}
			if in.ValidationMode == "" {
				in.ValidationMode = sp.ValidationMode
				in.RequiredChecks = sp.RequiredChecks
			}
			if in.ValidationMode != sp.ValidationMode || string(JSON(in.RequiredChecks)) != string(JSON(sp.RequiredChecks)) {
				if !in.Human {
					return nil, Fail("AUTHORITY_REQUIRED", "validation policy changes require human authority")
				}
			}
			if in.ValidationMode != "human" && in.ValidationMode != "independent_agent" && in.ValidationMode != "automated" {
				return nil, Fail("USAGE", "invalid validation mode")
			}
			if in.ValidationMode == "automated" && len(in.RequiredChecks) == 0 {
				return nil, Fail("USAGE", "automated mode requires checks")
			}
			var parent any
			if in.ParentID != "" {
				p, e := readTicket(ctx, c, in.ProjectID, in.ParentID)
				if e != nil {
					return nil, e
				}
				parent = p.ID
				var n int
				e = c.QueryRowContext(ctx, `WITH RECURSIVE ancestors(id) AS(SELECT ? UNION SELECT s.parent_id FROM workflow_ticket_specs s JOIN ancestors a ON s.ticket_id=a.id WHERE s.parent_id IS NOT NULL) SELECT count(*) FROM ancestors WHERE id=?`, p.ID, v.ID).Scan(&n)
				if e != nil {
					return nil, e
				}
				if n > 0 {
					return nil, Fail("CYCLE", "parent cycle")
				}
			}
			if _, e = c.ExecContext(ctx, `DELETE FROM workflow_dependencies WHERE ticket_id=?`, v.ID); e != nil {
				return nil, e
			}
			for _, dep := range in.Dependencies {
				d, e := readTicket(ctx, c, in.ProjectID, dep)
				if e != nil {
					return nil, e
				}
				var n int
				e = c.QueryRowContext(ctx, `WITH RECURSIVE deps(id) AS(SELECT ? UNION SELECT d.depends_on FROM workflow_dependencies d JOIN deps a ON d.ticket_id=a.id) SELECT count(*) FROM deps WHERE id=?`, d.ID, v.ID).Scan(&n)
				if e != nil {
					return nil, e
				}
				if n > 0 {
					return nil, Fail("CYCLE", "dependency cycle")
				}
				if _, e = c.ExecContext(ctx, `INSERT INTO workflow_dependencies VALUES(?,?,?)`, in.ProjectID, v.ID, d.ID); e != nil {
					return nil, e
				}
			}
			if in.Kind == "" {
				in.Kind = sp.Kind
			}
			_, e = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET spec_revision=spec_revision+1,parent_id=?,kind=?,validation_mode=?,required_checks=? WHERE ticket_id=?`, parent, in.Kind, in.ValidationMode, string(JSON(in.RequiredChecks)), v.ID)
			if e != nil {
				return nil, e
			}
			_, e = c.ExecContext(ctx, `UPDATE tickets SET title=?,body=? WHERE id=?`, in.Title, in.Body, v.ID)
			if e != nil {
				return nil, e
			}
			_, e = c.ExecContext(ctx, `WITH RECURSIVE affected(id) AS(SELECT ? UNION SELECT d.ticket_id FROM workflow_dependencies d JOIN affected a ON d.depends_on=a.id) UPDATE workflow_ticket_specs SET prepared_revision=0,authorized_revision=0,paused=CASE WHEN EXISTS(SELECT 1 FROM claims WHERE claims.ticket_id=workflow_ticket_specs.ticket_id AND released_at IS NULL) THEN 1 ELSE 0 END WHERE ticket_id IN(SELECT id FROM affected)`, v.ID)
			if e != nil {
				return nil, e
			}
			_, e = c.ExecContext(ctx, `WITH RECURSIVE affected(id) AS(SELECT ? UNION SELECT d.ticket_id FROM workflow_dependencies d JOIN affected a ON d.depends_on=a.id) UPDATE tickets SET state=CASE WHEN state IN('ready','review','done') THEN 'draft' ELSE state END,revision=revision+CASE WHEN id=? THEN 0 ELSE 1 END,updated_at=? WHERE project_id=? AND id IN(SELECT id FROM affected)`, v.ID, v.ID, Now(), in.ProjectID)
			if e != nil {
				return nil, e
			}
			_, e = c.ExecContext(ctx, `INSERT INTO workflow_versions VALUES(?,?,?,?,?,?,?)`, UUID(), in.ProjectID, v.ID, sp.SpecRevision+1, actor, string(JSON(in)), Now())
			if e != nil {
				return nil, e
			}
		case "critique":
			if in.Human || in.ContextID == "" || in.Body == "" {
				return nil, Fail("USAGE", "critique requires reviewer session, fresh context_id and body")
			}
			ss, e := ReadSession(ctx, c, in.ProjectID, in.SessionID)
			if e != nil {
				return nil, e
			}
			var n int
			e = c.QueryRowContext(ctx, `SELECT count(*) FROM claims c JOIN sessions s ON s.id=c.session_id WHERE c.ticket_id=? AND s.agent_id=?`, v.ID, ss.AgentID).Scan(&n)
			if e != nil {
				return nil, e
			}
			if n > 0 {
				return nil, Fail("NOT_INDEPENDENT", "implementer cannot critique own work")
			}
			if e = c.QueryRowContext(ctx, `SELECT count(*) FROM workflow_versions v JOIN sessions s ON s.id=v.actor_id WHERE v.ticket_id=? AND s.agent_id=?`, v.ID, ss.AgentID).Scan(&n); e != nil {
				return nil, e
			}
			if n > 0 {
				return nil, Fail("NOT_INDEPENDENT", "proposal author cannot supply its independent critique")
			}
			id := UUID()
			_, e = c.ExecContext(ctx, `INSERT INTO workflow_critiques VALUES(?,?,?,?,?,?,?,?,?,?)`, id, in.ProjectID, v.ID, sp.SpecRevision, in.SessionID, ss.AgentID, in.ContextID, in.Significant, in.Body, Now())
			if e != nil {
				return nil, e
			}
			in.CritiqueID = id
		case "dispose":
			if in.CritiqueID == "" || in.Body == "" {
				return nil, Fail("USAGE", "critique_id and disposition body required")
			}
			var n int
			e = c.QueryRowContext(ctx, `SELECT count(*) FROM workflow_critiques WHERE id=? AND project_id=? AND ticket_id=? AND spec_revision=?`, in.CritiqueID, in.ProjectID, v.ID, sp.SpecRevision).Scan(&n)
			if e != nil {
				return nil, e
			}
			if n != 1 {
				return nil, Fail("NOT_FOUND", "current critique not found")
			}
			_, e = c.ExecContext(ctx, `INSERT INTO workflow_dispositions VALUES(?,?,?,?)`, in.CritiqueID, actor, in.Body, Now())
			if e != nil {
				return nil, e
			}
		case "prepare":
			if in.Body == "" {
				return nil, Fail("USAGE", "preparation snapshot and validation coverage required in body")
			}
			var n int
			e = c.QueryRowContext(ctx, `SELECT count(*) FROM workflow_critiques q LEFT JOIN workflow_dispositions d ON d.critique_id=q.id WHERE q.ticket_id=? AND q.spec_revision=? AND q.significant=1 AND d.critique_id IS NULL`, v.ID, sp.SpecRevision).Scan(&n)
			if e != nil {
				return nil, e
			}
			if n > 0 {
				return nil, Fail("UNRESOLVED_FINDINGS", "significant critiques require dispositions")
			}
			if sp.PlanReview == "independent_agent" {
				e = c.QueryRowContext(ctx, `SELECT count(*) FROM workflow_critiques WHERE ticket_id=? AND spec_revision=?`, v.ID, sp.SpecRevision).Scan(&n)
				if e != nil {
					return nil, e
				}
				if n == 0 {
					return nil, Fail("REVIEW_REQUIRED", "current specification requires independent critique")
				}
			}
			_, e = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET prepared_revision=spec_revision WHERE ticket_id=?`, v.ID)
			if e != nil {
				return nil, e
			}
		case "authorize":
			if sp.PreparedRevision != sp.SpecRevision {
				return nil, Fail("NOT_PREPARED", "prepare current specification first")
			}
			if sp.ExecutionMode == "human" && !in.Human {
				return nil, Fail("HUMAN_REQUIRED", "execution authorization retained by user")
			}
			if in.Reason == "" {
				return nil, Fail("USAGE", "execution grant reason required")
			}
			_, e = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET authorized_revision=spec_revision,paused=0 WHERE ticket_id=?`, v.ID)
			if e != nil {
				return nil, e
			}
			_, e = c.ExecContext(ctx, `UPDATE tickets SET state=CASE WHEN state='draft' THEN 'ready' ELSE state END WHERE id=?`, v.ID)
			if e != nil {
				return nil, e
			}
		case "unblock":
			if in.Reason == "" {
				return nil, Fail("USAGE", "unblock reason required")
			}
			_, e = c.ExecContext(ctx, `UPDATE workflow_ticket_specs SET blocked_reason='' WHERE ticket_id=?`, v.ID)
			if e != nil {
				return nil, e
			}
			_, e = c.ExecContext(ctx, `UPDATE tickets SET state=CASE WHEN EXISTS(SELECT 1 FROM workflow_ticket_specs WHERE ticket_id=? AND prepared_revision=spec_revision AND authorized_revision=spec_revision) THEN 'ready' ELSE 'draft' END WHERE id=? AND state='blocked'`, v.ID, v.ID)
			if e != nil {
				return nil, e
			}
		default:
			return nil, Fail("USAGE", "unknown workflow operation")
		}
		_, e = c.ExecContext(ctx, `UPDATE tickets SET revision=revision+1,updated_at=? WHERE id=?`, Now(), v.ID)
		if e != nil {
			return nil, e
		}
		result, e := readTicket(ctx, c, in.ProjectID, v.ID)
		if e != nil {
			return nil, e
		}
		return workflowEvent(ctx, c, in, actor, op, map[string]any{"ticket": result, "critique_id": in.CritiqueID})
	})
}
func workflowEvent(ctx context.Context, c *sql.Conn, in WorkflowInput, actor, op string, result any) (json.RawMessage, error) {
	var ticket any
	if in.TicketID != "" {
		ticket = in.TicketID
	}
	_, e := c.ExecContext(ctx, `INSERT INTO events(id,project_id,ticket_id,actor_id,kind,body,payload,created_at) VALUES(?,?,?,?,?,?,?,?)`, UUID(), in.ProjectID, ticket, actor, "workflow."+op, in.Reason, string(JSON(in)), Now())
	return JSON(result), e
}
func (s *Service) ShowWorkflow(ctx context.Context, p string) (map[string]any, error) {
	var exists int
	if e := s.Store.DB.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE name='workflow_policies' AND type='table'`).Scan(&exists); e != nil {
		return nil, e
	}
	if exists == 0 {
		return map[string]any{"enabled": false}, nil
	}
	var rev int
	var execution, review, validation, checks, authority, reason string
	e := s.Store.DB.QueryRowContext(ctx, `SELECT revision,execution_mode,plan_review,validation_mode,required_checks,authority,reason FROM workflow_policies WHERE project_id=?`, p).Scan(&rev, &execution, &review, &validation, &checks, &authority, &reason)
	if e == sql.ErrNoRows {
		return map[string]any{"enabled": false}, nil
	}
	if e != nil {
		return nil, e
	}
	rows, e := s.Store.DB.QueryContext(ctx, `SELECT id,kind,body,actor_id,created_at FROM workflow_intents WHERE project_id=? ORDER BY created_at,id`, p)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	intents := []map[string]string{}
	for rows.Next() {
		var id, kind, body, actor, at string
		if e = rows.Scan(&id, &kind, &body, &actor, &at); e != nil {
			return nil, e
		}
		intents = append(intents, map[string]string{"id": id, "kind": kind, "body": body, "actor_id": actor, "created_at": at})
	}
	return map[string]any{"enabled": true, "revision": rev, "execution_mode": execution, "plan_review": review, "validation_mode": validation, "required_checks": json.RawMessage(checks), "authority": authority, "reason": reason, "intents": intents}, rows.Err()
}

// Bounded durable inputs and decisions let a fresh host reconstruct the ticket.
func (s *Service) workflowTicketHistory(ctx context.Context, p, id string) (map[string]any, error) {
	queries := map[string]string{
		"versions":     `SELECT id,spec_revision,actor_id,payload,created_at FROM workflow_versions WHERE project_id=? AND ticket_id=? ORDER BY spec_revision DESC LIMIT 101`,
		"critiques":    `SELECT q.id,q.spec_revision,q.session_id,q.agent_id,q.context_id,q.significant,q.body,q.created_at,d.actor_id AS disposition_actor,d.body AS disposition FROM workflow_critiques q LEFT JOIN workflow_dispositions d ON q.id=d.critique_id WHERE q.project_id=? AND q.ticket_id=? ORDER BY q.created_at DESC LIMIT 101`,
		"validations":  `SELECT id,spec_revision,mode,session_id,agent_id,context_id,commit_id,criteria,evidence,checks,created_at FROM workflow_validations WHERE project_id=? AND ticket_id=? ORDER BY created_at DESC LIMIT 101`,
		"dependencies": `SELECT t.id,t.display_key,t.state FROM workflow_dependencies d JOIN tickets t ON t.id=d.depends_on WHERE d.project_id=? AND d.ticket_id=? ORDER BY t.id LIMIT 101`,
	}
	result := map[string]any{}
	for key, query := range queries {
		rows, e := s.Store.DB.QueryContext(ctx, query, p, id)
		if e != nil {
			return nil, e
		}
		cols, e := rows.Columns()
		if e != nil {
			rows.Close()
			return nil, e
		}
		items := []map[string]any{}
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if e = rows.Scan(ptrs...); e != nil {
				rows.Close()
				return nil, e
			}
			item := map[string]any{}
			for i, col := range cols {
				item[col] = values[i]
			}
			items = append(items, item)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		truncated := len(items) > 100
		if truncated {
			items = items[:100]
		}
		result[key] = items
		result[key+"_truncated"] = truncated
	}
	return result, nil
}
