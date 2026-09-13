package core

import (
	"encoding/json"
	"testing"
)

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
