package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/store"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func init() { Register("workflow", Workflow) }
func readWorkflowBody(env Env, path string) (core.WorkflowInput, error) {
	var in core.WorkflowInput
	if path == "" {
		return in, core.Fail("USAGE", "--body-file required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(env.CWD, path)
	}
	f, e := os.Open(path)
	if e != nil {
		return in, core.Fail("USAGE", "cannot read body file")
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, core.TicketTextLimit+1))
	if e != nil || len(b) > core.TicketTextLimit {
		return in, core.Fail("USAGE", "body file exceeds limit or cannot be read")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&in); e != nil {
		return in, core.Fail("USAGE", "invalid workflow JSON body")
	}
	var trailing any
	if e = d.Decode(&trailing); e != io.EOF {
		return in, core.Fail("USAGE", "workflow body must contain one JSON object")
	}
	return in, nil
}
func Workflow(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 {
		return nil, core.Fail("USAGE", "workflow subcommand required")
	}
	op := args[0]
	if op != "configure" && op != "show" && op != "intent" && op != "amend" {
		return nil, core.Fail("USAGE", "unknown workflow subcommand")
	}
	f, e := Parse(args[1:], []string{"body-file", "request", "session", "expect-revision"}, []string{"human", "help"})
	if e != nil {
		return nil, e
	}
	if f.Bools["help"] {
		return map[string]any{"command": "workflow " + op, "usage": "workflow configure|amend --human --body-file JSON --request KEY; workflow show|intent", "contract": "v2; configure requires policy expected_revision (0 initially), reason, original intent initially, execution_mode, plan_review, validation_mode, required_checks"}, nil
	}
	if e = f.NoPositionals(); e != nil {
		return nil, e
	}
	in := core.WorkflowInput{}
	read := op == "show" || op == "intent"
	if !read {
		if e = f.Require("request"); e != nil {
			return nil, e
		}
		if e = core.ValidateRequest(f.Values["request"]); e != nil {
			return nil, e
		}
		in, e = readWorkflowBody(env, f.Values["body-file"])
		if e != nil {
			return nil, e
		}
		in.Human = f.Bools["human"]
		in.SessionID = f.Values["session"]
		if f.Values["expect-revision"] != "" {
			in.ExpectedRevision, e = strconv.Atoi(f.Values["expect-revision"])
			if e != nil {
				return nil, core.Fail("USAGE", "invalid revision")
			}
		}
	}
	open := OpenProject
	if read {
		open = OpenProjectReadOnly
	}
	s, p, _, e := open(ctx, env, f)
	if e != nil {
		return nil, e
	}
	defer s.Store.DB.Close()
	if read {
		return s.ShowWorkflow(ctx, p.ID)
	}
	in.ProjectID = p.ID
	return s.Workflow(ctx, store.Request{ID: f.Values["request"]}, op, in)
}
func WorkflowTicket(ctx context.Context, env Env, args []string) (any, error) {
	op := args[0]
	f, e := Parse(args[1:], []string{"body-file", "session", "expect-revision", "request", "coordination"}, []string{"human", "help"})
	if e != nil {
		return nil, e
	}
	if f.Bools["help"] {
		return map[string]any{"command": "ticket " + op, "usage": "ticket " + op + " ID --body-file JSON --expect-revision N --request KEY (--human | --session S --coordination TOKEN)", "contract": "v2; critique uses session and fresh context_id; edit/unblock work on plain tickets with a live session; planning mutations follow configured authority"}, nil
	}
	if len(f.Positionals) != 1 {
		return nil, core.Fail("USAGE", "exactly one ticket ID required")
	}
	if e = f.Require("request", "expect-revision"); e != nil {
		return nil, e
	}
	if e = core.ValidateRequest(f.Values["request"]); e != nil {
		return nil, e
	}
	in, e := readWorkflowBody(env, f.Values["body-file"])
	if e != nil {
		return nil, e
	}
	in.TicketID = f.Positionals[0]
	in.SessionID = f.Values["session"]
	in.CoordinationID = f.Values["coordination"]
	in.Human = f.Bools["human"]
	in.ExpectedRevision, e = strconv.Atoi(f.Values["expect-revision"])
	if e != nil {
		return nil, core.Fail("USAGE", "invalid revision")
	}
	s, p, _, e := OpenProject(ctx, env, f)
	if e != nil {
		return nil, e
	}
	defer s.Store.DB.Close()
	in.ProjectID = p.ID
	if op == "metadata" {
		return s.EditTicketMetadata(ctx, store.Request{ID: f.Values["request"]}, in)
	}
	if op == "edit" {
		return s.EditTicket(ctx, store.Request{ID: f.Values["request"]}, in)
	}
	if op == "unblock" {
		return s.UnblockTicket(ctx, store.Request{ID: f.Values["request"]}, in)
	}
	return s.Workflow(ctx, store.Request{ID: f.Values["request"]}, op, in)
}
