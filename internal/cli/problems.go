package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strconv"

	"github.com/ZackMFleischman/conductor/internal/core"
)

const problemBodyFileLimit = 256 * 1024

type problemBody struct {
	Summary    string `json:"summary"`
	Expected   string `json:"expected"`
	Actual     string `json:"actual"`
	Correction string `json:"correction,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
}

func init() { Register("problem", Problem) }

func Problem(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 {
		return nil, core.Fail("USAGE", "problem subcommand required")
	}
	op := args[0]
	var values []string
	switch op {
	case "add":
		values = []string{"session", "ticket", "body-file", "request"}
	case "append":
		values = []string{"session", "body-file", "request"}
	case "list":
		values = []string{"limit", "cursor"}
	default:
		return nil, core.Fail("USAGE", "unknown problem subcommand")
	}
	f, err := Parse(args[1:], values, nil)
	if err != nil {
		return nil, err
	}
	if op == "append" {
		if len(f.Positionals) != 1 {
			return nil, core.Fail("USAGE", "problem append requires exactly one problem ID")
		}
	} else if err = f.NoPositionals(); err != nil {
		return nil, err
	}
	if op == "add" || op == "append" {
		if err = f.Require("session", "body-file", "request"); err != nil {
			return nil, err
		}
	}
	open := OpenProject
	if op == "list" {
		open = OpenProjectReadOnly
	}
	svc, project, _, err := open(ctx, env, f)
	if err != nil {
		return nil, err
	}
	defer svc.Store.DB.Close()
	switch op {
	case "add":
		bodyBytes, e := readProblemBodyFile(f.Values["body-file"])
		if e != nil {
			return nil, e
		}
		var body problemBody
		decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&body); e != nil {
			return nil, core.Fail("INVALID_INPUT", "problem body must be a JSON object with known fields")
		}
		var trailing any
		if e = decoder.Decode(&trailing); e != io.EOF {
			return nil, core.Fail("INVALID_INPUT", "problem body must contain exactly one JSON object")
		}
		return svc.AddProblem(ctx, project.ID, f.Values["request"], core.AddProblemInput{SessionID: f.Values["session"], TicketID: f.Values["ticket"], Summary: body.Summary, Expected: body.Expected, Actual: body.Actual, Correction: body.Correction, Evidence: body.Evidence})
	case "append":
		body, e := readProblemBodyFile(f.Values["body-file"])
		if e != nil {
			return nil, e
		}
		return svc.AppendProblem(ctx, project.ID, f.Values["request"], core.AppendProblemInput{SessionID: f.Values["session"], ProblemID: f.Positionals[0], Body: string(body)})
	default:
		input := core.ListProblemsInput{Cursor: f.Values["cursor"]}
		if raw := f.Values["limit"]; raw != "" {
			input.Limit, err = strconv.Atoi(raw)
			if err != nil {
				return nil, core.Fail("INVALID_INPUT", "limit must be an integer")
			}
		}
		return svc.ListProblems(ctx, project.ID, input)
	}
}

func readProblemBodyFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, core.Fail("INVALID_INPUT", "cannot read body file")
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, problemBodyFileLimit+1))
	if err != nil {
		return nil, core.Fail("INVALID_INPUT", "cannot read body file")
	}
	if len(body) > problemBodyFileLimit {
		return nil, core.Fail("INVALID_INPUT", "problem body exceeds 256 KiB")
	}
	return body, nil
}
