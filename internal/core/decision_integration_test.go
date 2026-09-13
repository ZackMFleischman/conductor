package core

import "testing"

func TestIndependentRejectionThroughTicketMutation(t *testing.T) {
	f := newWorkflowFixture(t, "independent_agent")
	v := f.ticket("failed implementation")
	f.ready(v.ID)
	v = f.claim(v.ID)
	_, err := f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Commit: "abc", Summary: "result", Evidence: "unit output", QA: "review"})
	if err != nil {
		t.Fatal(err)
	}
	v = f.get(v.ID)
	in := TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.otherSession, ExpectedRevision: v.Revision, Reason: "unit fails", Validation: &WorkflowInput{Commit: "abc", Criteria: "all", Evidence: "failing unit reproduction", ContextID: "fresh-validator", Checks: map[string]string{"unit": "fail"}}}
	_, err = f.s.RejectTicket(f.ctx, f.req(), in)
	workflowFail(t, "NOT_INDEPENDENT", err)
	in.SessionID = f.reviewer
	key := f.req()
	if _, err = f.s.RejectTicket(f.ctx, key, in); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.RejectTicket(f.ctx, key, in); err != nil {
		t.Fatal("rejection replay", err)
	}
	if f.get(v.ID).State != "ready" {
		t.Fatal("failed validation did not requeue")
	}
	var n int
	if err = f.s.Store.DB.QueryRow("SELECT count(*) FROM ticket_decisions WHERE ticket_id=? AND outcome='rejected' AND session_id=?", v.ID, f.reviewer).Scan(&n); err != nil || n != 1 {
		t.Fatalf("attributed decision count %d: %v", n, err)
	}
}
