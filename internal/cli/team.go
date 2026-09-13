package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/core"
	"io"
	"strconv"
)

func init() { Register("team", Team) }

// Team only records native-host effects. It never launches or kills processes.
func Team(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		return map[string]any{"commands": []string{"start", "resume", "recover", "release", "stop", "launch", "register", "ack", "observe", "ready", "show", "list", "events"}, "usage": "team ACTION [RUN_ID] --session SESSION --request KEY [--coordination TOKEN --expect-revision N --body-file JSON]; team show RUN_ID; team list; team events RUN_ID [--after SEQ]", "reference": "docs/reference/team-commands.md", "contract_version": 2}, nil
	}
	op := args[0]
	read := op == "show" || op == "list" || op == "events"
	vals := []string{"session", "request", "coordination", "expect-revision", "body-file"}
	if read {
		vals = []string{}
		if op == "events" {
			vals = []string{"after"}
		}
	}
	f, e := Parse(args[1:], vals, []string{"human", "help"})
	if e != nil {
		return nil, e
	}
	if f.Bools["help"] {
		return Team(ctx, env, []string{"--help"})
	}
	switch op {
	case "start", "list":
		if e = f.NoPositionals(); e != nil {
			return nil, e
		}
	case "resume", "recover", "release", "stop", "launch", "register", "ack", "observe", "ready", "show", "events":
		if len(f.Positionals) != 1 {
			return nil, core.Fail("USAGE", "team action requires exactly one run ID")
		}
	default:
		return nil, core.Fail("USAGE", "unknown team action")
	}
	in := core.TeamInput{}
	if !read {
		if e = f.Require("session", "request"); e != nil {
			return nil, e
		}
		if op != "start" {
			if e = f.Require("expect-revision"); e != nil {
				return nil, e
			}
		}
		if op != "start" && op != "ack" && op != "recover" {
			if e = f.Require("coordination"); e != nil {
				return nil, e
			}
		}
		if file := f.Values["body-file"]; file != "" {
			b, e := readProblemBodyFile(file)
			if e != nil {
				return nil, e
			}
			if len(bytes.TrimSpace(b)) == 0 || bytes.TrimSpace(b)[0] != '{' {
				return nil, core.Fail("INVALID_INPUT", "team body must be a JSON object")
			}
			d := json.NewDecoder(bytes.NewReader(b))
			d.DisallowUnknownFields()
			if e = d.Decode(&in); e != nil {
				return nil, core.Fail("INVALID_INPUT", "team body must contain known fields")
			}
			var extra any
			if d.Decode(&extra) != io.EOF {
				return nil, core.Fail("INVALID_INPUT", "team body must contain exactly one JSON object")
			}
			if in.RunID != "" || in.SessionID != "" || in.CoordinationID != "" || in.ExpectedRevision != 0 || in.Human {
				return nil, core.Fail("INVALID_INPUT", "run, session, coordination, revision and human authority belong in command flags")
			}
		}
		in.SessionID = f.Values["session"]
		in.CoordinationID = f.Values["coordination"]
		in.Human = f.Bools["human"]
		if op != "start" {
			in.RunID = f.Positionals[0]
			in.ExpectedRevision, e = strconv.Atoi(f.Values["expect-revision"])
			if e != nil || in.ExpectedRevision < 1 {
				return nil, core.Fail("INVALID_INPUT", "expected revision must be positive")
			}
		}
	}
	open := OpenProject
	if read {
		open = OpenProjectReadOnly
	}
	svc, p, _, e := open(ctx, env, f)
	if e != nil {
		return nil, e
	}
	defer svc.Store.DB.Close()
	switch op {
	case "show":
		return svc.GetTeam(ctx, p.ID, f.Positionals[0])
	case "list":
		runs, e := svc.ListTeams(ctx, p.ID)
		return map[string]any{"runs": runs}, e
	case "events":
		var after int64
		if raw := f.Values["after"]; raw != "" {
			after, e = strconv.ParseInt(raw, 10, 64)
			if e != nil {
				return nil, core.Fail("INVALID_INPUT", "after must be an event sequence")
			}
		}
		events, e := svc.TeamEvents(ctx, p.ID, f.Positionals[0], after)
		return map[string]any{"events": events}, e
	default:
		return svc.TeamAction(ctx, p.ID, op, f.Values["request"], in)
	}
}
