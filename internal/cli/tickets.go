package cli

import (
	"context"
	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/store"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func init() { Register("ticket", Ticket) }
func Ticket(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 {
		return nil, core.Fail("USAGE", "ticket subcommand required")
	}
	op := args[0]
	if op == "edit" || op == "prepare" || op == "authorize" || op == "critique" || op == "dispose" || op == "unblock" {
		return WorkflowTicket(ctx, env, args)
	}
	values := []string{}
	bools := []string{"help"}
	switch op {
	case "create":
		values = []string{"title", "body-file", "assigned-to", "request", "session"}
	case "list":
		values = []string{"state", "assigned-to", "limit", "cursor"}
	case "show":
	case "assign":
		values = []string{"to", "expect-revision", "request"}
	case "claim":
		values = []string{"session", "expect-revision", "request"}
	case "note":
		values = []string{"session", "claim", "body-file", "request"}
	case "submit":
		values = []string{"session", "claim", "expect-revision", "summary-file", "evidence-file", "qa-file", "request", "commit"}
	case "accept":
		values = []string{"expect-revision", "request", "session", "validation-file"}
		bools = append(bools, "human")
	case "reject":
		values = []string{"expect-revision", "reason", "request", "session", "validation-file"}
		bools = append(bools, "human")
	case "release", "block":
		values = []string{"session", "claim", "expect-revision", "reason", "request"}
		bools = append(bools, "human")
	default:
		return nil, core.Fail("USAGE", "unknown ticket subcommand")
	}
	f, e := Parse(args[1:], values, bools)
	if e != nil {
		return nil, e
	}
	if f.Bools["help"] {
		return map[string]any{"command": "ticket " + op, "value_flags": values, "boolean_flags": bools, "limits": "title: 300 Unicode characters; each UTF-8 body: 256 KiB; request: 1–200 bytes; list: 1–100 (default 20)", "ownership": "note/submit/owner release require session and claim; accept/reject require a live session or --human; legacy tickets require --human; all changes except notes require --expect-revision"}, nil
	}
	in := core.TicketInput{Commit: f.Values["commit"], Title: f.Values["title"], AssignedAgentID: f.Values["assigned-to"], SessionID: f.Values["session"], ClaimID: f.Values["claim"], Reason: f.Values["reason"], Human: f.Bools["human"]}
	if op == "create" || op == "list" {
		if e = f.NoPositionals(); e != nil {
			return nil, e
		}
	} else {
		if len(f.Positionals) != 1 {
			return nil, core.Fail("USAGE", "exactly one ticket ID required")
		}
		in.TicketID = f.Positionals[0]
	}
	if op != "list" && op != "show" {
		if e = f.Require("request"); e != nil {
			return nil, e
		}
		if e = core.ValidateRequest(f.Values["request"]); e != nil {
			return nil, e
		}
	}
	if f.Values["expect-revision"] != "" {
		in.ExpectedRevision, e = strconv.Atoi(f.Values["expect-revision"])
		if e != nil {
			return nil, core.Fail("USAGE", "invalid expected revision")
		}
	}
	read := func(flag string) (string, error) {
		p := f.Values[flag]
		if p == "" {
			return "", core.Fail("USAGE", "--"+flag+" required")
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(env.CWD, p)
		}
		file, e := os.Open(p)
		if e != nil {
			return "", core.Fail("USAGE", "cannot read --"+flag)
		}
		defer file.Close()
		b, e := io.ReadAll(io.LimitReader(file, core.TicketTextLimit+1))
		if e != nil {
			return "", core.Fail("USAGE", "cannot read --"+flag)
		}
		if len(b) > core.TicketTextLimit {
			return "", core.Fail("USAGE", "body exceeds 256 KiB")
		}
		return string(b), nil
	}
	if op == "create" || op == "note" {
		in.Body, e = read("body-file")
		if e != nil {
			return nil, e
		}
	}
	if op == "submit" {
		in.Summary, e = read("summary-file")
		if e != nil {
			return nil, e
		}
		in.Evidence, e = read("evidence-file")
		if e != nil {
			return nil, e
		}
		in.QA, e = read("qa-file")
		if e != nil {
			return nil, e
		}
	}
	if (op == "accept" || op == "reject") && f.Values["validation-file"] != "" {
		d, err := readWorkflowBody(env, f.Values["validation-file"])
		if err != nil {
			return nil, err
		}
		in.Validation = &d
	}
	if op == "assign" {
		if e = f.Require("to"); e != nil {
			return nil, e
		}
		in.AssignedAgentID = f.Values["to"]
	}
	if op != "list" && op != "show" {
		if e = core.ValidateTicketInput(op, in); e != nil {
			return nil, e
		}
	}
	limit := 20
	if f.Values["limit"] != "" {
		limit, e = strconv.Atoi(f.Values["limit"])
		if e != nil || limit < 1 || limit > 100 {
			return nil, core.Fail("USAGE", "limit must be 1–100")
		}
	}
	open := OpenProject
	if op == "list" || op == "show" {
		open = OpenProjectReadOnly
	}
	s, p, g, e := open(ctx, env, f)
	if e != nil {
		return nil, e
	}
	defer s.Store.DB.Close()
	in.ProjectID = p.ID
	if op == "claim" {
		in.Location = g
	}
	r := store.Request{ID: f.Values["request"]}
	switch op {
	case "list":
		return s.ListTickets(ctx, p.ID, f.Values["state"], f.Values["assigned-to"], f.Values["cursor"], limit)
	case "show":
		return s.ShowTicket(ctx, p.ID, in.TicketID)
	case "create":
		return s.CreateTicket(ctx, r, in)
	case "assign":
		return s.AssignTicket(ctx, r, in)
	case "claim":
		return s.ClaimTicket(ctx, r, in)
	case "note":
		return s.NoteTicket(ctx, r, in)
	case "submit":
		return s.SubmitTicket(ctx, r, in)
	case "accept":
		return s.AcceptTicket(ctx, r, in)
	case "reject":
		return s.RejectTicket(ctx, r, in)
	case "block":
		return s.BlockTicket(ctx, r, in)
	case "release":
		return s.ReleaseTicket(ctx, r, in)
	}
	panic("unreachable")
}
