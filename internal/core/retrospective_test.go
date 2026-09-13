package core

import (
	"encoding/json"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"sync"
	"testing"
)

func TestRetrospectiveCoverageAndOldFollowup(t *testing.T) {
	ctx, _, svc, p, session := problemFixture(t)
	raw, err := svc.AddProblem(ctx, p.ID, "p", AddProblemInput{SessionID: session.ID, Summary: "incident", Expected: "works", Actual: "fails"})
	if err != nil {
		t.Fatal(err)
	}
	report := decodeProblem(t, raw)
	raw, err = svc.BeginRetrospective(ctx, p.ID, "begin", RetrospectiveBeginInput{SessionID: session.ID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	var batch RetrospectiveBatch
	json.Unmarshal(raw, &batch)
	if len(batch.Changes) != 1 || batch.Changes[0].ProblemID != report.ID {
		t.Fatalf("batch: %+v", batch)
	}
	if _, err = svc.CommitRetrospective(ctx, p.ID, "early", RetrospectiveBatchInput{SessionID: session.ID, BatchID: batch.ID}); err == nil {
		t.Fatal("uncovered batch committed")
	}
	raw, err = svc.BeginRetrospective(ctx, p.ID, "restart", RetrospectiveBeginInput{SessionID: session.ID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	var resumed RetrospectiveBatch
	json.Unmarshal(raw, &resumed)
	if resumed.ID != batch.ID {
		t.Fatal("restart created competing batch")
	}
	decision := RetrospectiveDecisionInput{SessionID: session.ID, BatchID: batch.ID, GroupKey: "setup", Action: "observe", Observation: "one failure", Rationale: "need recurrence", RevisitTrigger: "another incident", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{batch.Changes[0].Seq}}
	if _, err = svc.DecideRetrospective(ctx, p.ID, "decision", decision); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.CommitRetrospective(ctx, p.ID, "commit", RetrospectiveBatchInput{SessionID: session.ID, BatchID: batch.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AppendProblem(ctx, p.ID, "append", AppendProblemInput{SessionID: session.ID, ProblemID: report.ID, Body: "recurred"}); err != nil {
		t.Fatal(err)
	}
	raw, err = svc.BeginRetrospective(ctx, p.ID, "next", RetrospectiveBeginInput{SessionID: session.ID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(raw, &resumed)
	if len(resumed.Changes) != 1 || resumed.Changes[0].Kind != "problem.append" || resumed.Changes[0].ProblemID != report.ID || resumed.FromSeq != batch.ThroughSeq {
		t.Fatalf("old followup missing: %+v", resumed)
	}
}

func TestRetrospectiveRemedyRetryGroupingRecurrence(t *testing.T) {
	ctx, st, svc, p, session := problemFixture(t)
	if _, err := svc.Workflow(ctx, store.Request{ID: "policy"}, "configure", WorkflowInput{ProjectID: p.ID, Human: true, Intent: "Improve setup", Reason: "test", ExecutionMode: "delegated", PlanReview: "lightweight", ValidationMode: "independent_agent"}); err != nil {
		t.Fatal(err)
	}
	makeDecision := func(n int) (RetrospectiveDecision, RetrospectiveDecisionInput) {
		t.Helper()
		if _, err := svc.AddProblem(ctx, p.ID, fmt.Sprint("p", n), AddProblemInput{SessionID: session.ID, Summary: "setup failed", Expected: "works", Actual: "fails"}); err != nil {
			t.Fatal(err)
		}
		raw, err := svc.BeginRetrospective(ctx, p.ID, fmt.Sprint("begin", n), RetrospectiveBeginInput{SessionID: session.ID})
		if err != nil {
			t.Fatal(err)
		}
		var b RetrospectiveBatch
		json.Unmarshal(raw, &b)
		in := RetrospectiveDecisionInput{SessionID: session.ID, BatchID: b.ID, GroupKey: "setup", Action: "now", Observation: "failed", Rationale: "repeat", RevisitTrigger: "next run", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{b.Changes[0].Seq}, Title: "Fix setup", Body: "Reproduce then repair setup"}
		raw, err = svc.DecideRetrospective(ctx, p.ID, fmt.Sprint("decision", n), in)
		if err != nil {
			t.Fatal(err)
		}
		var d RetrospectiveDecision
		json.Unmarshal(raw, &d)
		retry, err := svc.DecideRetrospective(ctx, p.ID, fmt.Sprint("decision", n), in)
		if err != nil || string(retry) != string(raw) {
			t.Fatalf("retry %s %v", retry, err)
		}
		if _, err = svc.CommitRetrospective(ctx, p.ID, fmt.Sprint("commit", n), RetrospectiveBatchInput{SessionID: session.ID, BatchID: b.ID}); err != nil {
			t.Fatal(err)
		}
		return d, in
	}
	first, _ := makeDecision(1)
	second, _ := makeDecision(2)
	if first.TicketID == "" || first.TicketID != second.TicketID {
		t.Fatalf("duplicate remedy: %+v %+v", first, second)
	}
	var count int
	st.DB.QueryRow("SELECT count(*) FROM tickets").Scan(&count)
	if count != 2 {
		t.Fatalf("expected epic plus remedy, got %d", count)
	}
	if _, err := st.DB.Exec("UPDATE tickets SET state='done' WHERE id=?", first.TicketID); err != nil {
		t.Fatal(err)
	}
	third, _ := makeDecision(3)
	if third.TicketID == first.TicketID || third.RecurrenceOf == nil || *third.RecurrenceOf != second.ID {
		t.Fatalf("recurrence suppressed %+v", third)
	}
	st.DB.QueryRow("SELECT count(*) FROM tickets").Scan(&count)
	if count != 3 {
		t.Fatalf("epic duplicated: %d", count)
	}
	var state string
	st.DB.QueryRow("SELECT state FROM tickets WHERE id=?", third.TicketID).Scan(&state)
	if state != "draft" {
		t.Fatalf("remedy automatically ready: %s", state)
	}
}

func TestRetrospectiveMilestoneBlocksEligibility(t *testing.T) {
	ctx, st, svc, p, session := problemFixture(t)
	if _, err := svc.Workflow(ctx, store.Request{ID: "policy"}, "configure", WorkflowInput{ProjectID: p.ID, Human: true, Intent: "Improve setup", Reason: "test", ExecutionMode: "delegated", PlanReview: "lightweight", ValidationMode: "independent_agent"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddProblem(ctx, p.ID, "p", AddProblemInput{SessionID: session.ID, Summary: "minor", Expected: "e", Actual: "a"}); err != nil {
		t.Fatal(err)
	}
	raw, err := svc.BeginRetrospective(ctx, p.ID, "begin", RetrospectiveBeginInput{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	var b RetrospectiveBatch
	json.Unmarshal(raw, &b)
	raw, err = svc.DecideRetrospective(ctx, p.ID, "defer", RetrospectiveDecisionInput{SessionID: session.ID, BatchID: b.ID, GroupKey: "minor", Action: "milestone", Observation: "minor failure", Rationale: "delivery first", RevisitTrigger: "milestone or deadline", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{b.Changes[0].Seq}, Title: "Improve setup", Body: "Repair setup after milestone", MilestoneID: "M1", Condition: "release validated"})
	if err != nil {
		t.Fatal(err)
	}
	var d RetrospectiveDecision
	json.Unmarshal(raw, &d)
	if _, err = st.DB.Exec("UPDATE workflow_ticket_specs SET prepared_revision=spec_revision,authorized_revision=spec_revision WHERE ticket_id=?", d.TicketID); err != nil {
		t.Fatal(err)
	}
	conn, err := st.DB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = CheckWorkflowEligibilityTx(ctx, conn, p.ID, d.TicketID)
	conn.Close()
	if err == nil {
		t.Fatal("deferred ticket eligible")
	}
	if _, err = svc.ReleaseRetrospectiveMilestone(ctx, p.ID, "unauthorized", RetrospectiveMilestoneInput{SessionID: session.ID, MilestoneID: "M1", Evidence: "worker claims done"}); err == nil {
		t.Fatal("worker released scheduling gate")
	}
	profile := &TeamProfile{Scope: "authorized improvement", ApprovalReference: "test grant", SkillVersion: "2", Host: "test", HostLimit: 2, ChildLimit: 1, RequiredRoles: []string{}, Capabilities: []string{"status"}, PollingSeconds: 30, DeliveryPolicy: "worktree", StopConditions: "complete"}
	raw, err = svc.TeamAction(ctx, p.ID, "start", "team", TeamInput{SessionID: session.ID, Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	var run TeamRun
	json.Unmarshal(raw, &run)
	if _, err = svc.ReleaseRetrospectiveMilestone(ctx, p.ID, "wrong-token", RetrospectiveMilestoneInput{SessionID: session.ID, CoordinationID: "wrong", MilestoneID: "M1", Evidence: "validated"}); err == nil {
		t.Fatal("wrong coordination token accepted")
	}
	if _, err = svc.ReleaseRetrospectiveMilestone(ctx, p.ID, "release", RetrospectiveMilestoneInput{SessionID: session.ID, CoordinationID: run.CoordinationID, MilestoneID: "M1", Evidence: "release validation evidence"}); err != nil {
		t.Fatal(err)
	}
	var prepared, authorized int
	var state string
	if err = st.DB.QueryRow("SELECT t.state,s.prepared_revision,s.authorized_revision FROM tickets t JOIN workflow_ticket_specs s ON t.id=s.ticket_id WHERE t.id=?", d.TicketID).Scan(&state, &prepared, &authorized); err != nil {
		t.Fatal(err)
	}
	if state != "draft" || prepared != 0 || authorized != 0 {
		t.Fatalf("milestone automatically authorized: %s %d %d", state, prepared, authorized)
	}
}

func TestRetrospectiveIndependentConnectionsDeduplicateRemedy(t *testing.T) {
	ctx, st, svc, p, session := problemFixture(t)
	if _, err := svc.Workflow(ctx, store.Request{ID: "policy"}, "configure", WorkflowInput{ProjectID: p.ID, Human: true, Intent: "Improve", Reason: "test", ExecutionMode: "delegated", PlanReview: "lightweight", ValidationMode: "independent_agent"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := svc.AddProblem(ctx, p.ID, fmt.Sprint("p", i), AddProblemInput{SessionID: session.ID, Summary: "setup", Expected: "e", Actual: "a"}); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := svc.BeginRetrospective(ctx, p.ID, "begin", RetrospectiveBeginInput{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	var b RetrospectiveBatch
	json.Unmarshal(raw, &b)
	var seq int
	var name, path string
	if err = st.DB.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	other, err := store.Open(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer other.DB.Close()
	services := []Service{svc, {Store: other}}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := services[i].DecideRetrospective(ctx, p.ID, fmt.Sprint("d", i), RetrospectiveDecisionInput{SessionID: session.ID, BatchID: b.ID, GroupKey: "setup", Action: "now", Observation: "failed", Rationale: "repair", RevisitTrigger: "repeat", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{b.Changes[i].Seq}, Title: "Fix setup", Body: "Repair"})
			results <- e
		}(i)
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	var tickets, decisions int
	if err = st.DB.QueryRow("SELECT count(*) FROM tickets").Scan(&tickets); err != nil {
		t.Fatal(err)
	}
	if err = st.DB.QueryRow("SELECT count(*) FROM retrospective_decisions").Scan(&decisions); err != nil {
		t.Fatal(err)
	}
	if tickets != 2 || decisions != 2 {
		t.Fatalf("concurrent processing duplicated remedy: tickets=%d decisions=%d", tickets, decisions)
	}
}

func TestRetrospectiveRejectsForeignCitationsAndStoppedSessions(t *testing.T) {
	ctx, st, svc, p, session := problemFixture(t)
	raw, err := svc.Init(ctx, gitctx.Context{CommonDir: "elsewhere", Root: "other"}, "OTH", "other-init")
	if err != nil {
		t.Fatal(err)
	}
	var other Project
	json.Unmarshal(raw, &other)
	if _, err = svc.RegisterAgent(ctx, other.ID, "other", "", "", "other-agent"); err != nil {
		t.Fatal(err)
	}
	raw, err = svc.StartSession(ctx, other.ID, "other", "other-session", gitctx.Context{CommonDir: "elsewhere", Root: "other"})
	if err != nil {
		t.Fatal(err)
	}
	var otherSession Session
	json.Unmarshal(raw, &otherSession)
	if _, err = svc.AddProblem(ctx, other.ID, "other-problem", AddProblemInput{SessionID: otherSession.ID, Summary: "other", Expected: "e", Actual: "a"}); err != nil {
		t.Fatal(err)
	}
	var foreignSeq int64
	st.DB.QueryRow("SELECT seq FROM events WHERE project_id=? AND kind='problem.add'", other.ID).Scan(&foreignSeq)
	raw, err = svc.BeginRetrospective(ctx, p.ID, "begin", RetrospectiveBeginInput{SessionID: session.ID})
	if err != nil {
		t.Fatal(err)
	}
	var b RetrospectiveBatch
	json.Unmarshal(raw, &b)
	_, err = svc.DecideRetrospective(ctx, p.ID, "bad", RetrospectiveDecisionInput{SessionID: session.ID, BatchID: b.ID, GroupKey: "foreign", Action: "observe", Observation: "o", Rationale: "r", RevisitTrigger: "t", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{foreignSeq}})
	if err == nil {
		t.Fatal("foreign citation accepted")
	}
	if _, err = svc.BeginRetrospective(ctx, p.ID, "foreign-session", RetrospectiveBeginInput{SessionID: otherSession.ID}); err == nil {
		t.Fatal("foreign session accepted")
	}
	if _, err = st.DB.Exec("UPDATE sessions SET declared_state='stopped' WHERE id=?", session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.CommitRetrospective(ctx, p.ID, "stopped", RetrospectiveBatchInput{SessionID: session.ID, BatchID: b.ID}); err == nil {
		t.Fatal("stopped session committed")
	}
}

func TestRetrospectiveConcurrentCoverageAndEffectiveness(t *testing.T) {
	ctx, st, svc, p, session := problemFixture(t)
	for i := 0; i < 3; i++ {
		if _, err := svc.AddProblem(ctx, p.ID, fmt.Sprint("p", i), AddProblemInput{SessionID: session.ID, Summary: "failure", Expected: "works", Actual: "fails"}); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := svc.BeginRetrospective(ctx, p.ID, "begin", RetrospectiveBeginInput{SessionID: session.ID, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	var b RetrospectiveBatch
	json.Unmarshal(raw, &b)
	if len(b.Changes) != 2 {
		t.Fatal(b)
	}
	in := RetrospectiveDecisionInput{SessionID: session.ID, BatchID: b.ID, GroupKey: "same", Action: "observe", Observation: "failures", Rationale: "check", RevisitTrigger: "repeat", ReviewAfter: "2099-01-01T00:00:00Z", EventSeqs: []int64{b.Changes[0].Seq, b.Changes[1].Seq}}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := svc.DecideRetrospective(ctx, p.ID, fmt.Sprint("decide", i), in)
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successful competing decisions: %d", success)
	}
	var id string
	if err = st.DB.QueryRow("SELECT id FROM retrospective_decisions").Scan(&id); err != nil {
		t.Fatal(err)
	}
	eff := RetrospectiveEffectivenessInput{SessionID: session.ID, DecisionID: id, Outcome: "worse", Evidence: "fresh repeat failed"}
	if _, err = svc.AppendRetrospectiveEffectiveness(ctx, p.ID, "eff", eff); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AppendRetrospectiveEffectiveness(ctx, p.ID, "eff", eff); err != nil {
		t.Fatal(err)
	}
	result, err := svc.GetRetrospectiveDecision(ctx, p.ID, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Effectiveness) != 1 || result.Effectiveness[0].Outcome != "worse" {
		t.Fatal(result)
	}
	if _, err = svc.CommitRetrospective(ctx, p.ID, "commit", RetrospectiveBatchInput{SessionID: session.ID, BatchID: b.ID}); err != nil {
		t.Fatal(err)
	}
	raw, err = svc.BeginRetrospective(ctx, p.ID, "next", RetrospectiveBeginInput{SessionID: session.ID, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(raw, &b)
	if len(b.Changes) != 1 {
		t.Fatalf("pagination dropped report %+v", b)
	}
}
