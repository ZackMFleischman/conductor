package core

import (
	"context"
	"database/sql"
	"strings"
)

func validateWorkflowAcceptance(ctx context.Context, c *sql.Conn, in TicketInput, v TicketRecord) error {
	sp, e := workflowSpec(ctx, c, in.ProjectID, v.ID)
	if e != nil {
		return e
	}
	if sp == nil {
		if !in.Human {
			return Fail("HUMAN_REQUIRED", "legacy ticket requires explicit human acceptance")
		}
		return nil
	}
	if e = CheckWorkflowEligibilityTx(ctx, c, in.ProjectID, v.ID); e != nil {
		return e
	}
	if in.Validation == nil {
		return Fail("VALIDATION_REQUIRED", "structured validation evidence required")
	}
	d := in.Validation
	var submitted string
	var submittedRevision int
	if e = c.QueryRowContext(ctx, `SELECT submitted_commit,submitted_spec_revision FROM workflow_ticket_specs WHERE ticket_id=?`, v.ID).Scan(&submitted, &submittedRevision); e != nil {
		return e
	}
	if d.Commit != submitted || submittedRevision != sp.SpecRevision {
		return Fail("STALE_EVIDENCE", "validation must match submitted commit and current specification")
	}
	if strings.TrimSpace(d.Commit) == "" || strings.TrimSpace(d.Criteria) == "" || strings.TrimSpace(d.Evidence) == "" {
		return Fail("VALIDATION_REQUIRED", "tested commit, criteria and evidence required")
	}
	for _, name := range sp.RequiredChecks {
		if d.Checks[name] != "pass" {
			return Fail("CHECKS_REQUIRED", "required named check missing or not pass: "+name)
		}
	}
	var session, agent any
	switch sp.ValidationMode {
	case "human":
		if !in.Human {
			return Fail("HUMAN_REQUIRED", "human acceptance required")
		}
	case "independent_agent":
		if in.Human || in.SessionID == "" || d.ContextID == "" {
			return Fail("INDEPENDENT_VALIDATION_REQUIRED", "independent validator session and fresh host context required")
		}
		ss, e := ReadSession(ctx, c, in.ProjectID, in.SessionID)
		if e != nil {
			return e
		}
		if ss.DeclaredState == "stopped" {
			return Fail("SESSION_STOPPED", "validator session is stopped")
		}
		var n int
		e = c.QueryRowContext(ctx, `SELECT count(*) FROM claims c JOIN sessions s ON s.id=c.session_id WHERE c.ticket_id=? AND s.agent_id=?`, v.ID, ss.AgentID).Scan(&n)
		if e != nil {
			return e
		}
		if n > 0 {
			return Fail("NOT_INDEPENDENT", "validator identity participated in an implementation attempt")
		}
		session = in.SessionID
		agent = ss.AgentID
	case "automated":
		if len(sp.RequiredChecks) == 0 {
			return Fail("CHECKS_REQUIRED", "automated validation requires named checks")
		}
		if !in.Human {
			ss, e := ReadSession(ctx, c, in.ProjectID, in.SessionID)
			if e != nil {
				return e
			}
			if ss.DeclaredState == "stopped" {
				return Fail("SESSION_STOPPED", "session is stopped")
			}
			session = in.SessionID
			agent = ss.AgentID
		}
	default:
		return Fail("INVALID_POLICY", "unknown result validation mode")
	}
	_, e = c.ExecContext(ctx, `INSERT INTO workflow_validations VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, UUID(), in.ProjectID, v.ID, sp.SpecRevision, sp.ValidationMode, session, agent, d.ContextID, d.Commit, d.Criteria, d.Evidence, string(JSON(d.Checks)), Now())
	return e
}
