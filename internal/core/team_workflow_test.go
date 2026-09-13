package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
)

func TestManagedClaimProductionFenceAndReplay(t *testing.T) {
	f := newTeamFixture(t)
	_, ticket := f.claimFixture()
	ctx := context.Background()
	wrong := f.ticket()
	in := TicketInput{ProjectID: f.p, TicketID: wrong.ID, SessionID: f.child, ExpectedRevision: wrong.Revision, Location: gitctx.Context{CommonDir: "common", Root: "worker"}}
	_, err := f.s.ClaimTicket(ctx, store.Request{ID: "wrong-ticket"}, in)
	workflowFail(t, "TEAM_ASSIGNMENT", err)
	in.TicketID, in.ExpectedRevision = ticket.ID, ticket.Revision
	raw, err := f.s.ClaimTicket(ctx, store.Request{ID: "one-attempt"}, in)
	if err != nil {
		t.Fatal(err)
	}
	var claimed TicketRecord
	if err = json.Unmarshal(raw, &claimed); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.ClaimTicket(ctx, store.Request{ID: "one-attempt"}, in); err != nil {
		t.Fatal("identical replay failed", err)
	}
	_, err = f.s.ReleaseTicket(ctx, store.Request{ID: "release-attempt"}, TicketInput{ProjectID: f.p, TicketID: ticket.ID, SessionID: f.child, ClaimID: claimed.ClaimID, ExpectedRevision: claimed.Revision, Reason: "checkpoint"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err = f.s.ClaimTicket(ctx, store.Request{ID: "one-attempt"}, in)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &claimed); err != nil {
		t.Fatal(err)
	}
	if claimed.Active {
		t.Fatal("historical inactive claim replay appeared active")
	}
	current, err := readTicket(ctx, f.s.Store.DB, f.p, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	in.ExpectedRevision = current.Revision
	_, err = f.s.ClaimTicket(ctx, store.Request{ID: "second-attempt"}, in)
	workflowFail(t, "CHILD_REUSED", err)
}

func TestTeamWorkerLaunchChecksWorkflowDependencies(t *testing.T) {
	f := newWorkflowFixture(t, "independent_agent")
	prerequisite := f.ticket("prerequisite")
	work := f.ticket("dependent work")
	f.wf("edit", work.ID, WorkflowInput{Human: true, Title: work.Title, Body: work.Body, Dependencies: []string{prerequisite.ID}, Reason: "requires prerequisite"})
	f.ready(work.ID)
	profile := &TeamProfile{Scope: "dependency fixture", ApprovalReference: "test", SkillVersion: "team-v1", Host: "native-test", HostLimit: 2, ChildLimit: 1, Capabilities: []string{"spawn", "status"}, PollingSeconds: 30, DeliveryPolicy: "worktree", StopConditions: "one test"}
	b, err := f.s.TeamAction(f.ctx, f.p, "start", "start-team", TeamInput{SessionID: f.reviewer, Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	var run TeamRun
	if err = json.Unmarshal(b, &run); err != nil {
		t.Fatal(err)
	}
	b, err = f.s.TeamAction(f.ctx, f.p, "ready", "ready-team", owned(run))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &run); err != nil {
		t.Fatal(err)
	}
	session, err := ReadSession(f.ctx, f.s.Store.DB, f.p, f.worker)
	if err != nil {
		t.Fatal(err)
	}
	in := owned(run)
	in.Role, in.AgentID, in.TicketID = "worker", session.AgentID, work.ID
	_, err = f.s.TeamAction(f.ctx, f.p, "launch", "launch-work", in)
	workflowFail(t, "DEPENDENCY_BLOCKED", err)
}
