package cli

import (
	"context"
	"github.com/ZackMFleischman/conductor/internal/core"
)

func init() { Register("agent", Agent); Register("session", Session) }
func Agent(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 {
		return nil, core.Fail("USAGE", "agent subcommand required")
	}
	op := args[0]
	values := []string{}
	switch op {
	case "register":
		values = []string{"name", "role", "provider", "request"}
	case "list":
	default:
		return nil, core.Fail("USAGE", "unknown agent subcommand")
	}
	f, e := Parse(args[1:], values, nil)
	if e != nil {
		return nil, e
	}
	if e = f.NoPositionals(); e != nil {
		return nil, e
	}
	if op == "register" {
		if e = f.Require("name", "request"); e != nil {
			return nil, e
		}
	}
	s, p, _, e := OpenProject(ctx, env, f)
	if e != nil {
		return nil, e
	}
	defer s.Store.DB.Close()
	if op == "list" {
		a, e := s.ListAgents(ctx, p.ID)
		return map[string]any{"agents": a}, e
	}
	return s.RegisterAgent(ctx, p.ID, f.Values["name"], f.Values["role"], f.Values["provider"], f.Values["request"])
}
func Session(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 {
		return nil, core.Fail("USAGE", "session subcommand required")
	}
	op := args[0]
	values := []string{"session", "request"}
	if op == "start" {
		values = []string{"agent", "request"}
	} else if op != "resume" && op != "idle" && op != "stop" {
		return nil, core.Fail("USAGE", "unknown session subcommand")
	}
	f, e := Parse(args[1:], values, nil)
	if e != nil {
		return nil, e
	}
	if e = f.NoPositionals(); e != nil {
		return nil, e
	}
	if e = f.Require(values...); e != nil {
		return nil, e
	}
	s, p, g, e := OpenProject(ctx, env, f)
	if e != nil {
		return nil, e
	}
	defer s.Store.DB.Close()
	if op == "start" {
		return s.StartSession(ctx, p.ID, f.Values["agent"], f.Values["request"], g)
	}
	return s.SessionAction(ctx, p.ID, f.Values["session"], op, f.Values["request"])
}
