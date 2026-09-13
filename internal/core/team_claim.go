package core

import (
	"context"
	"database/sql"
)

// CheckTeamClaimTx adds coordination restrictions to foundation acquisition in
// the caller's transaction. Ordinary sessions have no coordination requirement.
// Durable launch/session records prevent historical children from becoming plain
// workers, and live launch identities close the gap before host registration.
func CheckTeamClaimTx(ctx context.Context, c *sql.Conn, projectID, ticketID, sessionID string) error {
	session, e := ReadSession(ctx, c, projectID, sessionID)
	if e != nil {
		return e
	}
	if session.DeclaredState == "stopped" {
		return Fail("SESSION_STOPPED", "session is stopped")
	}
	var runID, launchID string
	e = c.QueryRowContext(ctx, `SELECT run_id,id FROM team_launches WHERE project_id=? AND session_id=?`, projectID, sessionID).Scan(&runID, &launchID)
	if e == sql.ErrNoRows {
		// Any unresolved launch for this identity owns its registration boundary,
		// including an unknown host and a launch in a released or stopping run.
		var pending int
		e = c.QueryRowContext(ctx, `SELECT count(*) FROM team_launches WHERE project_id=? AND agent_id=? AND host_state NOT IN ('finished','stopped')`, projectID, session.AgentID).Scan(&pending)
		if e != nil {
			return e
		}
		if pending > 0 {
			return Fail("TEAM_NOT_READY", "managed identity must use its registered and acknowledged child session")
		}
		return nil
	}
	if e != nil {
		return e
	}
	r, e := readTeam(ctx, c, projectID, runID)
	if e != nil {
		return e
	}
	for _, l := range r.Launches {
		if l.ID != launchID {
			continue
		}
		if l.Role != "worker" || l.TicketID != ticketID {
			return Fail("TEAM_ASSIGNMENT", "managed execution requires the worker role and exact launch ticket")
		}
		if !r.Ready || l.HostID == "" || l.HostState != "active" || l.ObservedEpoch != r.Epoch || l.AcknowledgedEpoch != r.Epoch || l.ChallengeGeneration <= 0 || l.AcknowledgedGeneration != l.ChallengeGeneration {
			return Fail("TEAM_NOT_READY", "managed execution requires a usable ready run and current acknowledged active host/session")
		}
		var attempts int
		if e = c.QueryRowContext(ctx, `SELECT count(*) FROM claims WHERE project_id=? AND session_id=?`, projectID, sessionID).Scan(&attempts); e != nil {
			return e
		}
		if attempts > 0 {
			return Fail("CHILD_REUSED", "managed worker session has already attempted its assignment; use a fresh host/session")
		}
		return nil
	}
	return Fail("TEAM_NOT_READY", "managed launch could not be reconciled")
}
