package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"strings"
)

type Agent struct {
	ID           string    `json:"agent_id"`
	ProjectID    string    `json:"project_id"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Provider     string    `json:"provider"`
	Sessions     []Session `json:"sessions"`
	Availability string    `json:"availability"`
}
type Session struct {
	ID              string  `json:"session_id"`
	ProjectID       string  `json:"project_id"`
	AgentID         string  `json:"agent_id"`
	DeclaredState   string  `json:"declared_state"`
	LastSeenAt      string  `json:"last_seen_at"`
	WorktreeRoot    string  `json:"worktree_root"`
	Branch          *string `json:"branch"`
	Head            string  `json:"head"`
	Busy            bool    `json:"busy"`
	Claim           any     `json:"claim"`
	ProcessLiveness string  `json:"process_liveness"`
}

func (s *Service) RegisterAgent(ctx context.Context, p, name, role, provider, request string) (json.RawMessage, error) {
	if strings.TrimSpace(name) == "" || name == "none" || len(name) > 128 {
		return nil, Fail("USAGE", "invalid or reserved agent name")
	}
	return s.Mutate(ctx, p, request, "agent.register", "local-user", map[string]string{"name": name, "role": role, "provider": provider}, func(c *sql.Conn) (json.RawMessage, error) {
		var id string
		e := c.QueryRowContext(ctx, "SELECT id FROM agents WHERE project_id=? AND name=?", p, name).Scan(&id)
		if e == nil {
			return nil, Fail("AGENT_EXISTS", "agent name already exists")
		}
		if e != sql.ErrNoRows {
			return nil, e
		}
		a := Agent{ID: UUID(), ProjectID: p, Name: name, Role: role, Provider: provider, Sessions: []Session{}}
		_, e = c.ExecContext(ctx, "INSERT INTO agents(id,project_id,name,role,provider) VALUES(?,?,?,?,?)", a.ID, p, name, role, provider)
		return JSON(a), e
	})
}
func ReadSession(ctx context.Context, c interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, p, id string) (Session, error) {
	var s Session
	s.ProcessLiveness = "unknown"
	e := c.QueryRowContext(ctx, "SELECT id,project_id,agent_id,declared_state,last_seen_at,worktree_root,branch,head FROM sessions WHERE project_id=? AND id=?", p, id).Scan(&s.ID, &s.ProjectID, &s.AgentID, &s.DeclaredState, &s.LastSeenAt, &s.WorktreeRoot, &s.Branch, &s.Head)
	if e == sql.ErrNoRows {
		return s, Fail("NOT_FOUND", "session not found")
	}
	if e != nil {
		return s, e
	}
	var claim, ticket string
	e = c.QueryRowContext(ctx, "SELECT id,ticket_id FROM claims WHERE project_id=? AND session_id=? AND released_at IS NULL", p, id).Scan(&claim, &ticket)
	if e == nil {
		s.Busy = true
		s.Claim = map[string]any{"claim_id": claim, "ticket_id": ticket}
	} else if e != sql.ErrNoRows {
		return s, e
	}
	return s, nil
}
func (s *Service) StartSession(ctx context.Context, p, agent, request string, g gitctx.Context) (json.RawMessage, error) {
	return s.Mutate(ctx, p, request, "session.start", "local-user", map[string]any{"agent": agent, "location": g}, func(c *sql.Conn) (json.RawMessage, error) {
		var aid, common string
		e := c.QueryRowContext(ctx, "SELECT id FROM agents WHERE project_id=? AND (id=? OR name=?)", p, agent, agent).Scan(&aid)
		if e == sql.ErrNoRows {
			return nil, Fail("NOT_FOUND", "agent not found")
		}
		if e != nil {
			return nil, e
		}
		if e = c.QueryRowContext(ctx, "SELECT common_dir FROM projects WHERE id=?", p).Scan(&common); e != nil {
			return nil, e
		}
		if common != g.CommonDir {
			return nil, Fail("PROJECT_MISMATCH", "worktree belongs to another project")
		}
		id := UUID()
		_, e = c.ExecContext(ctx, "INSERT INTO worktrees(id,project_id,root,branch,head) VALUES(?,?,?,?,?) ON CONFLICT(project_id,root) DO UPDATE SET branch=excluded.branch,head=excluded.head", UUID(), p, g.Root, g.Branch, g.Head)
		if e != nil {
			return nil, e
		}
		_, e = c.ExecContext(ctx, "INSERT INTO sessions(id,project_id,agent_id,last_seen_at,worktree_root,branch,head) VALUES(?,?,?,?,?,?,?)", id, p, aid, Now(), g.Root, g.Branch, g.Head)
		if e != nil {
			return nil, e
		}
		v, e := ReadSession(ctx, c, p, id)
		return JSON(v), e
	})
}
func (s *Service) SessionAction(ctx context.Context, p, id, action, request string) (json.RawMessage, error) {
	if action != "resume" && action != "idle" && action != "stop" {
		return nil, Fail("USAGE", "unknown session action")
	}
	return s.Mutate(ctx, p, request, "session."+action, id, map[string]string{"session_id": id}, func(c *sql.Conn) (json.RawMessage, error) {
		v, e := ReadSession(ctx, c, p, id)
		if e != nil {
			return nil, e
		}
		if v.DeclaredState == "stopped" {
			return nil, Fail("SESSION_STOPPED", "session is stopped")
		}
		if v.Busy && action != "resume" {
			return nil, Fail("ACTIVE_CLAIM", "release the active claim first")
		}
		state := v.DeclaredState
		if action == "idle" {
			state = "idle"
		}
		if action == "stop" {
			state = "stopped"
		}
		_, e = c.ExecContext(ctx, "UPDATE sessions SET declared_state=?,last_seen_at=? WHERE id=?", state, Now(), id)
		if e != nil {
			return nil, e
		}
		v, e = ReadSession(ctx, c, p, id)
		return JSON(v), e
	})
}
func (s *Service) ListAgents(ctx context.Context, p string) ([]Agent, error) {
	rows, e := s.Store.DB.QueryContext(ctx, "SELECT id,project_id,name,role,provider FROM agents WHERE project_id=? ORDER BY name", p)
	if e != nil {
		return nil, e
	}
	a := []Agent{}
	for rows.Next() {
		var v Agent
		if e = rows.Scan(&v.ID, &v.ProjectID, &v.Name, &v.Role, &v.Provider); e != nil {
			rows.Close()
			return nil, e
		}
		v.Sessions = []Session{}
		v.Availability = "registered-without-session"
		a = append(a, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for i := range a {
		rows, e = s.Store.DB.QueryContext(ctx, "SELECT id FROM sessions WHERE project_id=? AND agent_id=? ORDER BY id", p, a[i].ID)
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
		for _, id := range ids {
			a[i].Availability = "unknown"
			v, e := ReadSession(ctx, s.Store.DB, p, id)
			if e != nil {
				return nil, e
			}
			a[i].Sessions = append(a[i].Sessions, v)
		}
		for _, v := range a[i].Sessions {
			if v.Busy {
				a[i].Availability = "busy"
				break
			}
		}
	}
	return a, nil
}
