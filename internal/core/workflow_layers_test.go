package core

import (
	"encoding/json"
	"testing"
)

func TestStandaloneDelegatedPolicy(t *testing.T) {
	f := newWorkflowFixture(t, "independent_agent")
	v := f.ticket("standalone")
	f.wf("prepare", v.ID, WorkflowInput{SessionID: f.worker, Body: "criteria and checks"})
	f.wf("authorize", v.ID, WorkflowInput{SessionID: f.worker, Reason: "standing user grant"})
	v = f.claim(v.ID)
	if !v.Active {
		t.Fatal("not claimed")
	}
	_, err := f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Commit: "abc", Summary: "done", Evidence: "unit pass", QA: "independent check"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.AcceptTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: f.get(v.ID).Revision, Validation: &WorkflowInput{Commit: "abc", Criteria: "all", Evidence: "reviewed", ContextID: "fresh", Checks: map[string]string{"unit": "pass"}}})
	if err != nil {
		t.Fatal(err)
	}
	if f.get(v.ID).State != "done" {
		t.Fatal("not accepted")
	}
	var count int
	if err = f.s.Store.DB.QueryRow("SELECT count(*) FROM team_runs").Scan(&count); err != nil || count != 0 {
		t.Fatal("unexpected team", count, err)
	}
}

func TestLateSignificantCritiqueInvalidatesGraph(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	a, b := f.ticket("active"), f.ticket("dependent")
	f.wf("edit", b.ID, WorkflowInput{Human: true, Title: b.Title, Body: b.Body, Reason: "dependency", Dependencies: []string{a.ID}})
	f.ready(a.ID)
	f.ready(b.ID)
	a = f.claim(a.ID)
	raw := f.wf("critique", a.ID, WorkflowInput{SessionID: f.reviewer, ContextID: "fresh", Body: "missing behavior", Significant: true})
	current := f.get(a.ID)
	if !current.Workflow.Paused || current.Workflow.PreparedRevision != 0 || f.get(b.ID).Workflow.AuthorizedRevision != 0 || f.get(b.ID).State != "draft" {
		t.Fatal("late finding retained approval", current, f.get(b.ID))
	}
	_, err := f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: a.ID, SessionID: f.worker, ClaimID: a.ClaimID, ExpectedRevision: current.Revision, Commit: "abc", Summary: "done", Evidence: "checks", QA: "review"})
	workflowFail(t, "WORK_PAUSED", err)
	_, err = f.s.Workflow(f.ctx, f.req(), "prepare", WorkflowInput{ProjectID: f.p, TicketID: a.ID, SessionID: f.worker, ExpectedRevision: current.Revision, Body: "plan"})
	workflowFail(t, "UNRESOLVED_FINDINGS", err)
	var result struct {
		CritiqueID string `json:"critique_id"`
	}
	json.Unmarshal(raw, &result)
	f.wf("dispose", a.ID, WorkflowInput{SessionID: f.worker, CritiqueID: result.CritiqueID, Body: "addressed"})
	f.wf("prepare", a.ID, WorkflowInput{SessionID: f.worker, Body: "revised coverage"})
	f.wf("authorize", a.ID, WorkflowInput{SessionID: f.worker, Reason: "within scope"})
	if f.get(a.ID).Workflow.Paused {
		t.Fatal("still paused")
	}
}

func TestStandaloneRemedyAndMilestone(t *testing.T) {
	ctx, st, svc, p, session := problemFixture(t)
	_, err := svc.AddProblem(ctx, p.ID, "p", AddProblemInput{SessionID: session.ID, Summary: "failure", Expected: "works", Actual: "fails"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := svc.BeginRetrospective(ctx, p.ID, "begin", RetrospectiveBeginInput{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	var b RetrospectiveBatch
	json.Unmarshal(raw, &b)
	raw, err = svc.DecideRetrospective(ctx, p.ID, "defer", RetrospectiveDecisionInput{SessionID: session.ID, BatchID: b.ID, GroupKey: "fix", Action: "milestone", Observation: "failure", Rationale: "delivery first", RevisitTrigger: "release", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{b.Changes[0].Seq}, Title: "Fix", Body: "repair", MilestoneID: "M1", Condition: "release validated"})
	if err != nil {
		t.Fatal(err)
	}
	var d RetrospectiveDecision
	json.Unmarshal(raw, &d)
	v, err := readTicket(ctx, st.DB, p.ID, d.TicketID)
	if err != nil {
		t.Fatal(err)
	}
	if v.State != "ready" || v.Workflow != nil || v.Metadata.ParentID == nil {
		t.Fatal(v)
	}
	c, err := st.DB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = CheckWorkflowEligibilityTx(ctx, c, p.ID, v.ID)
	c.Close()
	workflowFail(t, "DEFERRED", err)
	_, err = svc.ReleaseRetrospectiveMilestone(ctx, p.ID, "missing-scope", RetrospectiveMilestoneInput{SessionID: session.ID, MilestoneID: "M1", Evidence: "release validated"})
	if err == nil {
		t.Fatal("standalone release omitted scope and reason")
	}
	_, err = svc.ReleaseRetrospectiveMilestone(ctx, p.ID, "release", RetrospectiveMilestoneInput{SessionID: session.ID, MilestoneID: "M1", Scope: "authorized improvement", Reason: "M1 complete", Evidence: "release validated"})
	if err != nil {
		t.Fatal(err)
	}
	v, err = readTicket(ctx, st.DB, p.ID, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.State != "ready" || v.Workflow != nil {
		t.Fatal(v)
	}
	c, err = st.DB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = CheckWorkflowEligibilityTx(ctx, c, p.ID, v.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = c.QueryRowContext(ctx, "SELECT count(*) FROM workflow_policies WHERE project_id=?", p.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("policy created", count, err)
	}
}

func TestIndependentFailedValidationPolicy(t *testing.T) {
	f := newWorkflowFixture(t, "independent_agent")
	v := f.ticket("failed")
	f.ready(v.ID)
	v = f.claim(v.ID)
	_, err := f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Commit: "abc", Summary: "done", Evidence: "checks", QA: "review"})
	if err != nil {
		t.Fatal(err)
	}
	in := TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: f.get(v.ID).Revision, Reason: "unit failure", Validation: &WorkflowInput{Commit: "abc", Criteria: "all", Evidence: "unit fails", ContextID: "fresh", Checks: map[string]string{"unit": "fail"}}}
	v = f.get(v.ID)
	// The existing acceptance helper must still reject failed checks.
	c, err := f.s.Store.DB.Conn(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	err = validateWorkflowAcceptance(f.ctx, c, in, v)
	workflowFail(t, "CHECKS_REQUIRED", err)
	if err = validateWorkflowRejection(f.ctx, c, in, v); err != nil {
		t.Fatal(err)
	}
	in.SessionID = f.otherSession
	err = validateWorkflowRejection(f.ctx, c, in, v)
	workflowFail(t, "NOT_INDEPENDENT", err)
	in.SessionID = f.reviewer
	in.Validation.Commit = "other"
	err = validateWorkflowRejection(f.ctx, c, in, v)
	workflowFail(t, "STALE_EVIDENCE", err)
}

func TestPolicyCoordinatorCannotFallBackDuringShutdown(t *testing.T) {
	for _, status := range []string{"starting", "ready", "degraded", "stopping", "released", "stopped"} {
		t.Run(status, func(t *testing.T) {
			f := newWorkflowFixture(t, "human")
			v := f.ticket("policy")
			profile := &TeamProfile{Scope: "policy scope", ApprovalReference: "grant", SkillVersion: "2", Host: "test", HostLimit: 2, ChildLimit: 1, RequiredRoles: []string{}, Capabilities: []string{"status"}, PollingSeconds: 30, DeliveryPolicy: "worktree", StopConditions: "complete"}
			raw, err := f.s.TeamAction(f.ctx, f.p, "start", f.req().ID, TeamInput{SessionID: f.worker, Profile: profile})
			if err != nil {
				t.Fatal(err)
			}
			var run TeamRun
			json.Unmarshal(raw, &run)
			if _, err = f.s.Store.DB.Exec("UPDATE team_runs SET status=? WHERE id=?", status, run.ID); err != nil {
				t.Fatal(err)
			}
			in := WorkflowInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Body: "prepared scope"}
			if status != "stopped" {
				_, err = f.s.Workflow(f.ctx, f.req(), "prepare", in)
				workflowFail(t, "AUTHORITY_REQUIRED", err)
			}
			in.CoordinationID = run.CoordinationID
			_, err = f.s.Workflow(f.ctx, f.req(), "prepare", in)
			if status == "stopping" || status == "released" {
				workflowFail(t, "AUTHORITY_REQUIRED", err)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
