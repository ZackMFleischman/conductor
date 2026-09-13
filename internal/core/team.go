package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

// TeamProfile is immutable. A new grant/configuration requires a reconciled new run.
type TeamProfile struct {
	Scope              string   `json:"scope"`
	ApprovalReference  string   `json:"approval_reference"`
	SkillVersion       string   `json:"skill_version"`
	Host               string   `json:"host"`
	HostLimit          int      `json:"host_limit"`
	ChildLimit         int      `json:"child_limit"`
	RequiredRoles      []string `json:"required_roles"`
	ImprovementEnabled bool     `json:"improvement_enabled"`
	Capabilities       []string `json:"capabilities"`
	PollingSeconds     int      `json:"polling_seconds"`
	DeliveryPolicy     string   `json:"delivery_policy"`
	StopConditions     string   `json:"stop_conditions"`
}
type TeamInput struct {
	RunID            string       `json:"run_id,omitempty"`
	SessionID        string       `json:"session_id"`
	CoordinationID   string       `json:"coordination_id,omitempty"`
	ExpectedRevision int          `json:"expected_revision,omitempty"`
	Profile          *TeamProfile `json:"profile,omitempty"`
	LaunchID         string       `json:"launch_id,omitempty"`
	Role             string       `json:"role,omitempty"`
	AgentID          string       `json:"agent_id,omitempty"`
	TicketID         string       `json:"ticket_id,omitempty"`
	HostID           string       `json:"host_id,omitempty"`
	ChildSessionID   string       `json:"child_session_id,omitempty"`
	HostState        string       `json:"host_state,omitempty"`
	Epoch            int          `json:"epoch,omitempty"`
	Scope            string       `json:"scope,omitempty"`
	SkillVersion     string       `json:"skill_version,omitempty"`
	Checkpoint       string       `json:"checkpoint,omitempty"`
	Evidence         string       `json:"evidence,omitempty"`
	Reason           string       `json:"reason,omitempty"`
	Human            bool         `json:"human,omitempty"`
}
type TeamLaunch struct {
	ID                string `json:"launch_id"`
	Role              string `json:"role"`
	AgentID           string `json:"agent_id"`
	TicketID          string `json:"ticket_id,omitempty"`
	HostID            string `json:"host_id,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
	HostState         string `json:"host_state"`
	ObservedEpoch     int    `json:"observed_epoch"`
	AcknowledgedEpoch int    `json:"acknowledged_epoch"`
	Checkpoint        string `json:"checkpoint"`
	Evidence          string `json:"evidence"`
}
type TeamRun struct {
	ID                   string       `json:"run_id"`
	ProjectID            string       `json:"project_id"`
	CoordinatorSessionID string       `json:"coordinator_session_id"`
	CoordinationID       string       `json:"coordination_id,omitempty"`
	Status               string       `json:"status"`
	Revision             int          `json:"revision"`
	Epoch                int          `json:"epoch"`
	Profile              TeamProfile  `json:"profile"`
	Launches             []TeamLaunch `json:"launches"`
	Ready                bool         `json:"ready"`
	ReadinessIssues      []string     `json:"readiness_issues"`
	OccupiedSlots        int          `json:"occupied_slots"`
}
type teamReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// RequireCoordinatorTx is for policy operations that already bind an immutable
// session authority. New coordination operations also require the UUID fence.
func RequireCoordinatorTx(ctx context.Context, c *sql.Conn, p, session string) error {
	if e := validateProblemSession(ctx, c, p, session); e != nil {
		return e
	}
	var id string
	e := c.QueryRowContext(ctx, "SELECT id FROM team_runs WHERE project_id=? AND coordinator_session_id=? AND coordination_id IS NOT NULL AND status NOT IN ('stopping','stopped','released')", p, session).Scan(&id)
	if e == sql.ErrNoRows {
		return Fail("COORDINATION_REVOKED", "session does not hold active project coordination")
	}
	return e
}
func CheckTeamCoordinatorTx(ctx context.Context, c *sql.Conn, p, session, fence string) error {
	if e := validateProblemSession(ctx, c, p, session); e != nil {
		return e
	}
	if fence == "" {
		return Fail("COORDINATION_REVOKED", "coordination token required")
	}
	var id string
	e := c.QueryRowContext(ctx, "SELECT id FROM team_runs WHERE project_id=? AND coordinator_session_id=? AND coordination_id=?", p, session, fence).Scan(&id)
	if e == sql.ErrNoRows {
		return Fail("COORDINATION_REVOKED", "coordination token is no longer held")
	}
	return e
}
func validTeamRole(v string) bool {
	switch v {
	case "improver", "planner", "reviewer", "validator", "worker", "observer":
		return true
	}
	return false
}
func validateTeamProfile(p *TeamProfile) error {
	if p == nil || strings.TrimSpace(p.Scope) == "" || strings.TrimSpace(p.ApprovalReference) == "" || p.SkillVersion == "" || p.Host == "" || p.DeliveryPolicy == "" || p.StopConditions == "" {
		return Fail("INVALID_INPUT", "scope, approval reference, skill version, host, delivery policy and stop conditions are required")
	}
	if p.HostLimit < 1 || p.HostLimit > 1000 || p.ChildLimit < 0 || p.ChildLimit >= p.HostLimit || p.PollingSeconds < 1 || p.PollingSeconds > 3600 {
		return Fail("INVALID_INPUT", "limits must include the coordinator and a bounded polling interval")
	}
	roles := map[string]bool{}
	for _, role := range p.RequiredRoles {
		if !validTeamRole(role) || roles[role] {
			return Fail("INVALID_INPUT", "invalid or duplicate required role")
		}
		roles[role] = true
	}
	if p.ImprovementEnabled && !roles["improver"] {
		return Fail("INVALID_INPUT", "improvement-enabled runs require improver readiness")
	}
	return nil
}
func readTeam(ctx context.Context, c teamReader, p, id string) (TeamRun, error) {
	r := TeamRun{Launches: []TeamLaunch{}, ReadinessIssues: []string{}}
	var profile string
	var fence sql.NullString
	e := c.QueryRowContext(ctx, "SELECT id,project_id,coordinator_session_id,coordination_id,status,revision,epoch,profile FROM team_runs WHERE project_id=? AND id=?", p, id).Scan(&r.ID, &r.ProjectID, &r.CoordinatorSessionID, &fence, &r.Status, &r.Revision, &r.Epoch, &profile)
	if e == sql.ErrNoRows {
		return r, Fail("NOT_FOUND", "team run not found")
	}
	if e != nil {
		return r, e
	}
	r.CoordinationID = fence.String
	if e = json.Unmarshal([]byte(profile), &r.Profile); e != nil {
		return r, e
	}
	rows, e := c.QueryContext(ctx, "SELECT id,role,agent_id,COALESCE(ticket_id,''),COALESCE(host_id,''),COALESCE(session_id,''),host_state,observed_epoch,acknowledged_epoch,checkpoint,evidence FROM team_launches WHERE project_id=? AND run_id=? ORDER BY created_at,id", p, id)
	if e != nil {
		return r, e
	}
	for rows.Next() {
		var l TeamLaunch
		if e = rows.Scan(&l.ID, &l.Role, &l.AgentID, &l.TicketID, &l.HostID, &l.SessionID, &l.HostState, &l.ObservedEpoch, &l.AcknowledgedEpoch, &l.Checkpoint, &l.Evidence); e != nil {
			rows.Close()
			return r, e
		}
		r.Launches = append(r.Launches, l)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return r, e
	}
	// Liveness is host evidence supplied by the caller, never inferred from age.
	coord, e := ReadSession(ctx, c, p, r.CoordinatorSessionID)
	if e != nil {
		return r, e
	}
	r.OccupiedSlots = 1
	if r.CoordinationID == "" || coord.DeclaredState == "stopped" {
		r.ReadinessIssues = append(r.ReadinessIssues, "coordinator unavailable")
	}
	caps := map[string]bool{}
	for _, v := range r.Profile.Capabilities {
		caps[v] = true
	}
	if !caps["spawn"] || !caps["status"] {
		r.ReadinessIssues = append(r.ReadinessIssues, "native spawn/status capabilities unavailable")
	}
	roles := map[string]bool{}
	for _, l := range r.Launches {
		if l.HostState == "finished" || l.HostState == "stopped" {
			continue
		}
		r.OccupiedSlots++
		healthy := l.HostState == "active" && l.ObservedEpoch == r.Epoch && l.AcknowledgedEpoch == r.Epoch && l.SessionID != ""
		if l.SessionID != "" {
			s, e := ReadSession(ctx, c, p, l.SessionID)
			if e != nil {
				return r, e
			}
			healthy = healthy && s.DeclaredState != "stopped"
		}
		if healthy {
			roles[l.Role] = true
		} else {
			r.ReadinessIssues = append(r.ReadinessIssues, "unreconciled launch "+l.ID)
		}
	}
	for _, role := range r.Profile.RequiredRoles {
		if !roles[role] {
			r.ReadinessIssues = append(r.ReadinessIssues, "required role not ready: "+role)
		}
	}
	if r.Status == "stopping" || r.Status == "stopped" || r.Status == "released" {
		r.ReadinessIssues = append(r.ReadinessIssues, "run is "+r.Status)
	}
	r.Ready = r.Status == "ready" && len(r.ReadinessIssues) == 0
	return r, nil
}
func (s *Service) GetTeam(ctx context.Context, p, id string) (TeamRun, error) {
	return readTeam(ctx, s.Store.DB, p, id)
}

func (s *Service) TeamAction(ctx context.Context, p, op, request string, in TeamInput) (json.RawMessage, error) {
	switch op {
	case "start", "resume", "recover", "release", "stop", "launch", "register", "ack", "observe", "ready":
	default:
		return nil, Fail("USAGE", "unknown team action")
	}
	if len(JSON(in)) > 256*1024 {
		return nil, Fail("INVALID_INPUT", "team input exceeds 256 KiB")
	}
	if op == "start" {
		if e := validateTeamProfile(in.Profile); e != nil {
			return nil, e
		}
	} else if in.Profile != nil {
		return nil, Fail("INVALID_INPUT", "run profile is immutable")
	}
	return s.Mutate(ctx, p, request, "team."+op, in.SessionID, in, func(c *sql.Conn) (json.RawMessage, error) {
		if e := validateProblemSession(ctx, c, p, in.SessionID); e != nil {
			return nil, e
		}
		id := in.RunID
		if op == "start" {
			var count int
			if e := c.QueryRowContext(ctx, "SELECT count(*) FROM team_runs WHERE project_id=? AND status<>'stopped'", p).Scan(&count); e != nil {
				return nil, e
			}
			if count > 0 {
				return nil, Fail("COORDINATION_HELD", "reconcile the existing project run before starting another")
			}
			id = UUID()
			_, e := c.ExecContext(ctx, "INSERT INTO team_runs(id,project_id,coordinator_session_id,coordination_id,status,revision,epoch,profile,created_at) VALUES(?,?,?,?,'starting',1,1,?,?)", id, p, in.SessionID, UUID(), string(JSON(in.Profile)), Now())
			if e != nil {
				return nil, e
			}
		} else {
			r, e := readTeam(ctx, c, p, id)
			if e != nil {
				return nil, e
			}
			if op != "recover" && op != "ack" {
				if e = CheckTeamCoordinatorTx(ctx, c, p, in.SessionID, in.CoordinationID); e != nil {
					return nil, e
				}
				if in.CoordinationID != r.CoordinationID || in.SessionID != r.CoordinatorSessionID {
					return nil, Fail("COORDINATION_REVOKED", "token belongs to another run")
				}
			}
			if in.ExpectedRevision != r.Revision {
				return nil, Fail("REVISION_CONFLICT", "team revision changed; inspect and reconcile")
			}
			if r.Status == "stopped" {
				return nil, Fail("TEAM_STOPPED", "stopped run cannot resume")
			}
			switch op {
			case "recover":
				if !in.Human || strings.TrimSpace(in.Reason) == "" {
					return nil, Fail("RECOVERY_REQUIRED", "explicit human recovery requires inspection evidence in reason")
				}
				var used int
				if e = c.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM team_runs WHERE project_id=? AND coordinator_session_id=?)+(SELECT count(*) FROM team_launches WHERE project_id=? AND session_id=?)+(SELECT count(*) FROM claims WHERE project_id=? AND session_id=?)", p, in.SessionID, p, in.SessionID, p, in.SessionID).Scan(&used); e != nil {
					return nil, e
				}
				if used > 0 {
					return nil, Fail("CHILD_REUSED", "recovery requires a fresh replacement session with no earlier coordination, managed launch or execution claim")
				}
				_, e = c.ExecContext(ctx, "UPDATE team_runs SET coordinator_session_id=?,coordination_id=?,epoch=epoch+1,status='starting' WHERE id=?", in.SessionID, UUID(), id)
			case "resume":
				_, e = c.ExecContext(ctx, "UPDATE team_runs SET epoch=epoch+1,status='starting' WHERE id=?", id)
			case "release":
				if strings.TrimSpace(in.Reason) == "" {
					return nil, Fail("INVALID_INPUT", "release requires a checkpoint/reason")
				}
				_, e = c.ExecContext(ctx, "UPDATE team_runs SET coordination_id=NULL,status='released',epoch=epoch+1 WHERE id=?", id)
			case "stop":
				if strings.TrimSpace(in.Reason) == "" {
					return nil, Fail("INVALID_INPUT", "stop requires a checkpoint/reason")
				}
				terminal := true
				for _, l := range r.Launches {
					if l.HostState != "finished" && l.HostState != "stopped" {
						terminal = false
					}
					if l.SessionID != "" {
						s, e := ReadSession(ctx, c, p, l.SessionID)
						if e != nil {
							return nil, e
						}
						if s.Busy || s.DeclaredState != "stopped" {
							terminal = false
						}
					}
				}
				if terminal {
					_, e = c.ExecContext(ctx, "UPDATE team_runs SET status='stopped',coordination_id=NULL,epoch=epoch+1 WHERE id=?", id)
				} else {
					_, e = c.ExecContext(ctx, "UPDATE team_runs SET status='stopping',epoch=epoch+1 WHERE id=?", id)
				}
			case "ready":
				if len(r.ReadinessIssues) > 0 {
					return nil, Fail("TEAM_NOT_READY", strings.Join(r.ReadinessIssues, "; "))
				}
				_, e = c.ExecContext(ctx, "UPDATE team_runs SET status='ready' WHERE id=?", id)
			case "launch":
				e = launchTeamTx(ctx, c, p, r, in)
			case "register", "observe", "ack":
				e = updateTeamChildTx(ctx, c, p, r, op, in)
			}
			if e != nil {
				return nil, e
			}
			if _, e = c.ExecContext(ctx, "UPDATE team_runs SET revision=revision+1 WHERE id=?", id); e != nil {
				return nil, e
			}
		}
		r, e := readTeam(ctx, c, p, id)
		if e != nil {
			return nil, e
		}
		payload := map[string]any{"run_id": id, "input": in, "revision": r.Revision, "epoch": r.Epoch}
		_, e = c.ExecContext(ctx, "INSERT INTO events(id,project_id,actor_id,kind,body,payload,created_at) VALUES(?,?,?,?,?,?,?)", UUID(), p, in.SessionID, "team."+op, in.Reason, string(JSON(payload)), Now())
		if e != nil {
			return nil, e
		}
		return JSON(r), nil
	})
}

func launchTeamTx(ctx context.Context, c *sql.Conn, p string, r TeamRun, in TeamInput) error {
	caps := map[string]bool{}
	for _, v := range r.Profile.Capabilities {
		caps[v] = true
	}
	if !caps["spawn"] || !caps["status"] {
		return Fail("HOST_CAPABILITY", "native spawn and status capabilities are required before launching")
	}
	if r.Status == "stopping" || r.Status == "released" {
		return Fail("TEAM_NOT_READY", "run does not permit dispatch")
	}
	if !validTeamRole(in.Role) || in.AgentID == "" {
		return Fail("INVALID_INPUT", "launch requires role and stable agent ID")
	}
	if r.OccupiedSlots >= r.Profile.HostLimit || r.OccupiedSlots-1 >= r.Profile.ChildLimit {
		return Fail("CAPACITY_EXCEEDED", "active, starting and unknown launches consume capacity")
	}
	var n int
	if e := c.QueryRowContext(ctx, "SELECT count(*) FROM agents WHERE project_id=? AND id=?", p, in.AgentID).Scan(&n); e != nil {
		return e
	}
	if n != 1 {
		return Fail("NOT_FOUND", "launch agent not found")
	}
	if e := c.QueryRowContext(ctx, "SELECT count(*) FROM team_launches WHERE project_id=? AND agent_id=? AND host_state NOT IN ('finished','stopped')", p, in.AgentID).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return Fail("ACTIVE_CHILD", "reconcile existing identity before replacement")
	}
	if in.Role == "worker" {
		if !r.Ready {
			return Fail("TEAM_NOT_READY", "implementation dispatch requires a ready run")
		}
		if in.TicketID == "" {
			return Fail("INVALID_INPUT", "worker requires exact ticket ID")
		}
	}
	var ticket any
	if in.TicketID != "" {
		var state string
		var assigned sql.NullString
		e := c.QueryRowContext(ctx, "SELECT state,assigned_agent_id FROM tickets WHERE project_id=? AND id=?", p, in.TicketID).Scan(&state, &assigned)
		if e == sql.ErrNoRows {
			return Fail("NOT_FOUND", "launch ticket not found")
		}
		if e != nil {
			return e
		}
		if state != "ready" || (assigned.Valid && assigned.String != in.AgentID) {
			return Fail("TICKET_INELIGIBLE", "ticket must be ready and assigned to this identity or unassigned")
		}
		if in.Role == "worker" {
			if e := CheckWorkflowEligibilityTx(ctx, c, p, in.TicketID); e != nil {
				return e
			}
		}
		ticket = in.TicketID
	}
	_, e := c.ExecContext(ctx, "INSERT INTO team_launches(id,project_id,run_id,role,agent_id,ticket_id,host_state,created_at) VALUES(?,?,?,?,?,?,'starting',?)", UUID(), p, r.ID, in.Role, in.AgentID, ticket, Now())
	return e
}
func updateTeamChildTx(ctx context.Context, c *sql.Conn, p string, r TeamRun, op string, in TeamInput) error {
	var l *TeamLaunch
	for i := range r.Launches {
		if r.Launches[i].ID == in.LaunchID {
			l = &r.Launches[i]
			break
		}
	}
	if l == nil {
		return Fail("NOT_FOUND", "launch not found in this run")
	}
	if l.HostState == "finished" || l.HostState == "stopped" {
		return Fail("CHILD_TERMINAL", "terminal child cannot be reused")
	}
	switch op {
	case "register":
		if in.HostID == "" || in.ChildSessionID == "" {
			return Fail("INVALID_INPUT", "registration requires host ID and fresh session")
		}
		child, e := ReadSession(ctx, c, p, in.ChildSessionID)
		if e != nil {
			return e
		}
		if child.DeclaredState == "stopped" {
			return Fail("SESSION_STOPPED", "child session is stopped")
		}
		if child.AgentID != l.AgentID || child.ID == r.CoordinatorSessionID {
			return Fail("CHILD_MISMATCH", "child identity/session does not match intent")
		}
		if l.HostID != "" || l.SessionID != "" {
			if l.HostID == in.HostID && l.SessionID == in.ChildSessionID {
				return nil
			}
			return Fail("CHILD_MISMATCH", "launch registration cannot be replaced")
		}
		var attempts int
		if e = c.QueryRowContext(ctx, "SELECT count(*) FROM claims WHERE project_id=? AND session_id=?", p, in.ChildSessionID).Scan(&attempts); e != nil {
			return e
		}
		if attempts > 0 {
			return Fail("CHILD_REUSED", "fresh child session must not have prior execution claims")
		}
		var n int
		if e = c.QueryRowContext(ctx, "SELECT count(*) FROM team_launches WHERE project_id=? AND (host_id=? OR session_id=?)", p, in.HostID, in.ChildSessionID).Scan(&n); e != nil {
			return e
		}
		if n > 0 {
			return Fail("CHILD_REUSED", "host context and session must be fresh")
		}
		_, e = c.ExecContext(ctx, "UPDATE team_launches SET host_id=?,session_id=? WHERE id=?", in.HostID, in.ChildSessionID, l.ID)
		return e
	case "ack":
		if r.CoordinationID == "" || r.Status == "stopping" || in.Epoch != r.Epoch {
			return Fail("STALE_READINESS", "readiness challenge is no longer current")
		}
		if in.SessionID != l.SessionID || in.HostID != l.HostID || in.AgentID != l.AgentID || in.Role != l.Role || in.Scope != r.Profile.Scope || in.SkillVersion != r.Profile.SkillVersion || strings.TrimSpace(in.Checkpoint) == "" {
			return Fail("CHILD_MISMATCH", "acknowledgement must match run, role, identity, registered host/session, scope and skill version with a checkpoint")
		}
		_, e := c.ExecContext(ctx, "UPDATE team_launches SET acknowledged_epoch=?,checkpoint=? WHERE id=?", r.Epoch, in.Checkpoint, l.ID)
		return e
	case "observe":
		if strings.TrimSpace(in.Evidence) == "" {
			return Fail("INVALID_INPUT", "native host observation evidence is required")
		}
		switch in.HostState {
		case "active":
			if l.HostID == "" {
				return Fail("CHILD_MISMATCH", "register host before observing active")
			}
		case "unknown", "finished", "stopped":
		default:
			return Fail("INVALID_INPUT", "host state must be active, unknown, finished or stopped")
		}
		if in.HostState == "finished" || in.HostState == "stopped" {
			if l.SessionID != "" {
				s, e := ReadSession(ctx, c, p, l.SessionID)
				if e != nil {
					return e
				}
				if s.Busy || s.DeclaredState != "stopped" {
					return Fail("ACTIVE_CHILD_SESSION", "checkpoint/release claims and stop child session before reclaiming capacity")
				}
			}
		}
		ack := l.AcknowledgedEpoch
		epoch := r.Epoch
		if in.HostState != l.HostState {
			ack = 0
			epoch++
			// A host transition invalidates prior readiness challenges. Preserve
			// stopping state; reconciliation never silently restarts dispatch.
			if _, e := c.ExecContext(ctx, "UPDATE team_runs SET epoch=?,status=CASE WHEN status='ready' THEN 'degraded' ELSE status END WHERE id=?", epoch, r.ID); e != nil {
				return e
			}
		}
		_, e := c.ExecContext(ctx, "UPDATE team_launches SET host_state=?,observed_epoch=?,acknowledged_epoch=?,evidence=? WHERE id=?", in.HostState, epoch, ack, in.Evidence, l.ID)
		return e
	}
	return nil
}

func (s *Service) ListTeams(ctx context.Context, p string) ([]TeamRun, error) {
	rows, e := s.Store.DB.QueryContext(ctx, "SELECT id FROM team_runs WHERE project_id=? ORDER BY created_at DESC LIMIT 100", p)
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
	out := []TeamRun{}
	for _, id := range ids {
		r, e := s.GetTeam(ctx, p, id)
		if e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, nil
}
func (s *Service) TeamEvents(ctx context.Context, p, id string, after int64) ([]map[string]any, error) {
	if after < 0 {
		return nil, Fail("INVALID_INPUT", "event cursor must be nonnegative")
	}
	if _, e := s.GetTeam(ctx, p, id); e != nil {
		return nil, e
	}
	rows, e := s.Store.DB.QueryContext(ctx, "SELECT seq,kind,actor_id,payload,created_at FROM events WHERE project_id=? AND kind LIKE 'team.%' AND json_extract(payload,'$.run_id')=? AND seq>? ORDER BY seq LIMIT 100", p, id, after)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var seq int64
		var kind, actor, payload, created string
		if e = rows.Scan(&seq, &kind, &actor, &payload, &created); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"seq": seq, "kind": kind, "actor_id": actor, "payload": json.RawMessage(payload), "created_at": created})
	}
	return out, rows.Err()
}
