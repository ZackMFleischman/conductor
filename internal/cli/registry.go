package cli

import (
	"context"
	"errors"
	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"os"
	"path/filepath"
)

func init() { Register("context", Context); Register("init", Init); Register("doctor", Doctor) }
func OpenProject(ctx context.Context, env Env, f Flags) (*core.Service, core.Project, gitctx.Context, error) {
	return openProject(ctx, env, f, false)
}

// OpenProjectReadOnly resolves a registered project without database writes or schema migration.
func OpenProjectReadOnly(ctx context.Context, env Env, f Flags) (*core.Service, core.Project, gitctx.Context, error) {
	return openProject(ctx, env, f, true)
}

func openProject(ctx context.Context, env Env, f Flags, readOnly bool) (*core.Service, core.Project, gitctx.Context, error) {
	g, ge := gitctx.Resolve(env.CWD)
	if ge != nil && f.Values["project"] == "" {
		return nil, core.Project{}, g, core.Fail("NOT_FOUND", "not a registered checkout; use --project")
	}
	var s *store.Store
	var e error
	if readOnly {
		s, e = store.OpenReadOnly(filepath.Join(env.Home, "conductor.db"))
	} else {
		s, e = store.Open(filepath.Join(env.Home, "conductor.db"), false)
	}
	if e != nil {
		return nil, core.Project{}, g, e
	}
	svc := &core.Service{Store: s}
	p, e := svc.Project(ctx, f.Values["project"], g.CommonDir)
	if e != nil {
		s.DB.Close()
		return nil, p, g, e
	}
	return svc, p, g, nil
}
func Init(ctx context.Context, env Env, args []string) (any, error) {
	f, e := Parse(args, []string{"prefix", "request"}, nil)
	if e != nil {
		return nil, e
	}
	if e = f.NoPositionals(); e != nil {
		return nil, e
	}
	if e = f.Require("prefix", "request"); e != nil {
		return nil, e
	}
	g, e := gitctx.Resolve(env.CWD)
	if e != nil {
		return nil, core.Fail("USAGE", e.Error())
	}
	s, e := store.Open(filepath.Join(env.Home, "conductor.db"), true)
	if e != nil {
		return nil, e
	}
	defer s.DB.Close()
	return (&core.Service{Store: s}).Init(ctx, g, f.Values["prefix"], f.Values["request"])
}
func Context(ctx context.Context, env Env, args []string) (any, error) {
	f, e := Parse(args, []string{"session"}, []string{"if-registered"})
	if e != nil {
		return nil, e
	}
	if e = f.NoPositionals(); e != nil {
		return nil, e
	}
	path := filepath.Join(env.Home, "conductor.db")
	if _, e = os.Stat(path); os.IsNotExist(e) {
		return map[string]any{"registered": false}, nil
	} else if e != nil {
		return nil, e
	}
	s, e := store.OpenReadOnly(path)
	if e != nil {
		return nil, e
	}
	defer s.DB.Close()
	g, ge := gitctx.Resolve(env.CWD)
	if ge != nil && f.Values["project"] == "" {
		return map[string]any{"registered": false}, nil
	}
	svc := &core.Service{Store: s}
	p, e := svc.Project(ctx, f.Values["project"], g.CommonDir)
	if e != nil {
		var fault *core.Fault
		if errors.As(e, &fault) && fault.Code == "NOT_FOUND" {
			return map[string]any{"registered": false}, nil
		}
		return nil, e
	}
	v := map[string]any{"registered": true, "project": p, "project_id": p.ID}
	updates := []map[string]any{}
	rows, e := s.DB.QueryContext(ctx, "SELECT seq,id,kind,body,created_at FROM events WHERE project_id=? ORDER BY seq DESC LIMIT 20", p.ID)
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var seq int64
		var id, kind, body, at string
		if e = rows.Scan(&seq, &id, &kind, &body, &at); e != nil {
			rows.Close()
			return nil, e
		}
		updates = append(updates, map[string]any{"seq": seq, "event_id": id, "kind": kind, "body": body, "created_at": at})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	v["recent_updates"] = updates
	if id := f.Values["session"]; id != "" {
		session, e := core.ReadSession(ctx, s.DB, p.ID, id)
		if e != nil {
			return nil, e
		}
		v["session"] = session
		v["claim"] = session.Claim
		if session.Claim != nil {
			ticketID := session.Claim.(map[string]any)["ticket_id"]
			var id, key, title, body, state string
			var revision int
			var assignee *string
			e = s.DB.QueryRowContext(ctx, "SELECT id,display_key,title,body,state,revision,assigned_agent_id FROM tickets WHERE project_id=? AND id=?", p.ID, ticketID).Scan(&id, &key, &title, &body, &state, &revision, &assignee)
			if e != nil {
				return nil, e
			}
			v["ticket"] = map[string]any{"id": id, "display_key": key, "title": title, "body": body, "state": state, "revision": revision, "assigned_agent_id": assignee}
		} else {
			v["ticket"] = nil
		}
	}
	return v, nil
}
