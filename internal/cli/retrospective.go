package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/ZackMFleischman/conductor/internal/core"
)

func init() { Register("retrospective", Retrospective) }

func Retrospective(ctx context.Context, env Env, args []string) (any, error) {
	if len(args) == 0 {
		return nil, core.Fail("USAGE", "retrospective subcommand required")
	}
	if args[0] == "--help" || args[0] == "help" {
		return map[string]any{"commands": []string{"begin --session S --limit 50 --request KEY", "show BATCH", "status", "list [--due] [--limit 50] [--cursor ROWID]", "decide --session S --body-file decision.json --request KEY", "commit BATCH --session S --request KEY", "decision ID", "effectiveness ID --session S --outcome improved|unchanged|worse|unknown --body-file evidence.txt --request KEY", "release-milestone --milestone ID --session S --coordination TOKEN --body-file evidence.txt --request KEY"}, "contract": "conductor-team/v2", "reference": "docs/reference/retrospective-commands.md"}, nil
	}
	op := args[0]
	var flags []string
	readOnly := false
	switch op {
	case "begin":
		flags = []string{"session", "limit", "request"}
	case "show", "decision":
		readOnly = true
	case "status":
		readOnly = true
	case "list":
		readOnly = true
		flags = []string{"limit", "cursor"}
	case "decide":
		flags = []string{"session", "body-file", "request"}
	case "commit":
		flags = []string{"session", "request"}
	case "effectiveness":
		flags = []string{"session", "outcome", "body-file", "request"}
	case "release-milestone":
		flags = []string{"session", "coordination", "milestone", "body-file", "request"}
	default:
		return nil, core.Fail("USAGE", "unknown retrospective subcommand")
	}
	var bools []string
	if op == "list" {
		bools = []string{"due"}
	}
	f, err := Parse(args[1:], flags, bools)
	if err != nil {
		return nil, err
	}
	if op == "show" || op == "decision" || op == "commit" || op == "effectiveness" {
		if len(f.Positionals) != 1 {
			return nil, core.Fail("USAGE", "exactly one ID required")
		}
	} else if err = f.NoPositionals(); err != nil {
		return nil, err
	}
	if !readOnly {
		if err = f.Require("session", "request"); err != nil {
			return nil, err
		}
	}
	if op == "decide" || op == "effectiveness" || op == "release-milestone" {
		if err = f.Require("body-file"); err != nil {
			return nil, err
		}
	}
	open := OpenProject
	if readOnly {
		open = OpenProjectReadOnly
	}
	svc, p, _, err := open(ctx, env, f)
	if err != nil {
		return nil, err
	}
	defer svc.Store.DB.Close()
	switch op {
	case "begin":
		limit := 0
		if f.Values["limit"] != "" {
			limit, err = strconv.Atoi(f.Values["limit"])
			if err != nil {
				return nil, core.Fail("INVALID_INPUT", "limit must be an integer")
			}
		}
		return svc.BeginRetrospective(ctx, p.ID, f.Values["request"], core.RetrospectiveBeginInput{SessionID: f.Values["session"], Limit: limit})
	case "show":
		return svc.GetRetrospectiveBatch(ctx, p.ID, f.Positionals[0])
	case "status":
		return svc.RetrospectiveStatus(ctx, p.ID)
	case "list":
		limit := 0
		if f.Values["limit"] != "" {
			limit, err = strconv.Atoi(f.Values["limit"])
			if err != nil {
				return nil, core.Fail("INVALID_INPUT", "limit must be an integer")
			}
		}
		return svc.ListRetrospectiveDecisions(ctx, p.ID, core.RetrospectiveListInput{Limit: limit, Cursor: f.Values["cursor"], Due: f.Bools["due"]})
	case "commit":
		return svc.CommitRetrospective(ctx, p.ID, f.Values["request"], core.RetrospectiveBatchInput{SessionID: f.Values["session"], BatchID: f.Positionals[0]})
	case "decide":
		body, err := readProblemBodyFile(f.Values["body-file"])
		if err != nil {
			return nil, err
		}
		var in core.RetrospectiveDecisionInput
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&in); err != nil {
			return nil, core.Fail("INVALID_INPUT", "decision must be a JSON object with known fields")
		}
		var trailing any
		if decoder.Decode(&trailing) != io.EOF {
			return nil, core.Fail("INVALID_INPUT", "decision requires exactly one JSON object")
		}
		if in.SessionID != "" && in.SessionID != f.Values["session"] {
			return nil, core.Fail("INVALID_INPUT", "body session_id conflicts with --session")
		}
		in.SessionID = f.Values["session"]
		return svc.DecideRetrospective(ctx, p.ID, f.Values["request"], in)
	case "decision":
		return svc.GetRetrospectiveDecision(ctx, p.ID, f.Positionals[0])
	case "effectiveness":
		body, err := readProblemBodyFile(f.Values["body-file"])
		if err != nil {
			return nil, err
		}
		return svc.AppendRetrospectiveEffectiveness(ctx, p.ID, f.Values["request"], core.RetrospectiveEffectivenessInput{SessionID: f.Values["session"], DecisionID: f.Positionals[0], Outcome: f.Values["outcome"], Evidence: string(body)})
	default:
		body, err := readProblemBodyFile(f.Values["body-file"])
		if err != nil {
			return nil, err
		}
		return svc.ReleaseRetrospectiveMilestone(ctx, p.ID, f.Values["request"], core.RetrospectiveMilestoneInput{SessionID: f.Values["session"], CoordinationID: f.Values["coordination"], MilestoneID: f.Values["milestone"], Evidence: string(body)})
	}
}
