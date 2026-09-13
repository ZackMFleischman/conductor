package core

import (
	"encoding/json"
	"testing"
)

func TestFoundationPlainAgentDecision(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	if _, e := f.s.Store.DB.Exec("DELETE FROM workflow_policies WHERE project_id=?", f.p); e != nil {
		t.Fatal(e)
	}
	v := f.ticket("plain")
	raw, e := f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	raw, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Summary: "implemented", Evidence: "tests", QA: "inspect", Commit: "abc"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	_, e = f.s.RejectTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision, Reason: "test failed", Validation: &WorkflowInput{Commit: "abc", Evidence: "failure", Checks: map[string]string{"unit": "fail"}}})
	if e != nil {
		t.Fatal(e)
	}
	var n int
	if e = f.s.Store.DB.QueryRow("SELECT count(*) FROM ticket_decisions WHERE ticket_id=? AND outcome='rejected' AND session_id=?", v.ID, f.reviewer).Scan(&n); e != nil || n != 1 {
		t.Fatalf("decision count %d: %v", n, e)
	}
}

func TestFoundationPlainGraphAndLifecycle(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	if _, e := f.s.Store.DB.Exec("DELETE FROM workflow_policies WHERE project_id=?", f.p); e != nil {
		t.Fatal(e)
	}
	parent, dep, v := f.ticket("epic"), f.ticket("dependency"), f.ticket("work")
	edit := func(v TicketRecord, in WorkflowInput) (TicketRecord, error) {
		in.ProjectID = f.p
		in.TicketID = v.ID
		in.ExpectedRevision = v.Revision
		in.SessionID = f.reviewer
		if in.Title == "" {
			in.Title = v.Title
		}
		if in.Body == "" {
			in.Body = v.Body
		}
		in.Reason = "update graph"
		raw, e := f.s.EditTicket(f.ctx, f.req(), in)
		if e != nil {
			return v, e
		}
		var result struct {
			Ticket TicketRecord `json:"ticket"`
		}
		json.Unmarshal(raw, &result)
		return result.Ticket, nil
	}
	var e error
	v, e = edit(v, WorkflowInput{ParentID: parent.ID, Dependencies: []string{dep.ID}})
	if e != nil {
		t.Fatal(e)
	}
	if v.Metadata == nil || v.Metadata.ParentID == nil || *v.Metadata.ParentID != parent.ID {
		t.Fatalf("metadata %+v", v.Metadata)
	}
	_, e = edit(parent, WorkflowInput{ParentID: v.ID})
	f.fail("CYCLE", e)
	_, e = edit(dep, WorkflowInput{Dependencies: []string{v.ID}})
	f.fail("CYCLE", e)
	_, e = f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	f.fail("DEPENDENCY_BLOCKED", e)
	// Removing the graph is an explicit complete edit, and cycles rolled back.
	v, e = edit(v, WorkflowInput{})
	if e != nil {
		t.Fatal(e)
	}
	raw, e := f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	old := v
	raw, e = f.s.BlockTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Reason: "waiting"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	if v.State != "blocked" || v.Active {
		t.Fatal(v)
	}
	raw, e = f.s.UnblockTicket(f.ctx, f.req(), WorkflowInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision, Reason: "resolved"})
	if e != nil {
		t.Fatal(e)
	}
	var result struct {
		Ticket TicketRecord `json:"ticket"`
	}
	json.Unmarshal(raw, &result)
	v = result.Ticket
	if v.State != "ready" || v.Metadata.BlockedReason != "" {
		t.Fatal(v)
	}
	_, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: old.ClaimID, ExpectedRevision: old.Revision, Summary: "s", Evidence: "e", QA: "q"})
	f.fail("CLAIM_REVOKED", e)
	raw, e = f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	raw, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Summary: "s", Evidence: "e", QA: "q"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	raw, e = f.s.AcceptTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision, Validation: &WorkflowInput{Criteria: "done", Evidence: "verified"}})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	if v.State != "done" {
		t.Fatal(v)
	}
	shown, e := f.s.ShowTicket(f.ctx, f.p, v.ID)
	if e != nil {
		t.Fatal(e)
	}
	h := shown["history"].(map[string]any)
	if len(h["decisions"].([]map[string]any)) != 1 || len(h["submissions"].([]map[string]any)) != 1 {
		t.Fatal(h)
	}
	var n int
	if e = f.s.Store.DB.QueryRow("SELECT count(*) FROM workflow_ticket_specs WHERE project_id=?", f.p).Scan(&n); e != nil || n != 0 {
		t.Fatalf("plain policy rows %d: %v", n, e)
	}
}

func TestFoundationMetadataOnlyEditPreservesAcceptedTicket(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	if _, e := f.s.Store.DB.Exec("DELETE FROM workflow_policies WHERE project_id=?", f.p); e != nil {
		t.Fatal(e)
	}
	parent, v := f.ticket("parent"), f.ticket("accepted")
	raw, e := f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	raw, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Summary: "implemented", Evidence: "tested", QA: "inspect", Commit: "abc"})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	raw, e = f.s.AcceptTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision, Validation: &WorkflowInput{Commit: "abc", Criteria: "accepted", Evidence: "reviewed"}})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	if v.State != "done" {
		t.Fatal(v)
	}
	raw, e = f.s.EditTicketMetadata(f.ctx, f.req(), WorkflowInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision, ParentID: parent.ID, Kind: "bug", Reason: "correct grouping"})
	if e != nil {
		t.Fatal(e)
	}
	var result struct{ Ticket TicketRecord `json:"ticket"` }
	json.Unmarshal(raw, &result)
	if result.Ticket.State != "done" || result.Ticket.Metadata.ParentID == nil || *result.Ticket.Metadata.ParentID != parent.ID || result.Ticket.Metadata.Kind != "bug" {
		t.Fatal(result.Ticket)
	}
	if result.Ticket.Metadata.SubmittedCommit != "abc" || result.Ticket.Metadata.SubmittedSpecRevision != 1 {
		t.Fatal(result.Ticket.Metadata)
	}
	shown, e := f.s.ShowTicket(f.ctx, f.p, v.ID)
	if e != nil {
		t.Fatal(e)
	}
	history := shown["history"].(map[string]any)
	if len(history["submissions"].([]map[string]any)) != 1 || len(history["decisions"].([]map[string]any)) != 1 {
		t.Fatal(history)
	}
}

func TestFoundationMetadataOnlyEditRejectsGuardsWithoutChangingTicket(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	if _, e := f.s.Store.DB.Exec("DELETE FROM workflow_policies WHERE project_id=?", f.p); e != nil { t.Fatal(e) }
	parent, v := f.ticket("parent"), f.ticket("target")
	before := func() TicketRecord { return f.get(v.ID) }
	assertUnchanged := func(old TicketRecord) { got := f.get(v.ID); if got.State != old.State || got.Revision != old.Revision || got.Metadata.SpecRevision != old.Metadata.SpecRevision || got.Metadata.SubmittedCommit != old.Metadata.SubmittedCommit { t.Fatalf("changed: %#v", got) } }
	old := before()
	_, e := f.s.EditTicketMetadata(f.ctx, f.req(), WorkflowInput{ProjectID:f.p, TicketID:v.ID, SessionID:f.reviewer, ExpectedRevision:old.Revision+1, ParentID:parent.ID, Reason:"stale"})
	f.fail("REVISION_CONFLICT", e); assertUnchanged(old)
	_, e = f.s.EditTicketMetadata(f.ctx, f.req(), WorkflowInput{ProjectID:f.p, TicketID:v.ID, SessionID:f.reviewer, ExpectedRevision:old.Revision, ParentID:"missing", Reason:"missing"})
	f.fail("NOT_FOUND", e); assertUnchanged(old)
	_, e = f.s.EditTicketMetadata(f.ctx, f.req(), WorkflowInput{ProjectID:f.p, TicketID:v.ID, SessionID:f.reviewer, ExpectedRevision:old.Revision, ParentID:v.ID, Reason:"cycle"})
	f.fail("CYCLE", e); assertUnchanged(old)
	_, e = f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID:f.p, TicketID:v.ID, SessionID:f.worker, ExpectedRevision:old.Revision, Location:f.g})
	if e != nil { t.Fatal(e) }
	old = before()
	_, e = f.s.EditTicketMetadata(f.ctx, f.req(), WorkflowInput{ProjectID:f.p, TicketID:v.ID, SessionID:f.reviewer, ExpectedRevision:f.get(v.ID).Revision, ParentID:parent.ID, Reason:"active"})
	f.fail("ACTIVE_CLAIM", e); assertUnchanged(old)
	if _, e = f.s.Store.DB.Exec("UPDATE claims SET released_at=? WHERE ticket_id=?", Now(), v.ID); e != nil { t.Fatal(e) }
	if _, e = f.s.Store.DB.Exec("INSERT INTO workflow_ticket_specs(ticket_id,project_id,validation_mode,required_checks,policy_revision,execution_mode,plan_review) VALUES(?,?,?,?,?,?,?)", v.ID,f.p,"human","[]",1,"delegated","lightweight"); e != nil { t.Fatal(e) }
	_, e = f.s.EditTicketMetadata(f.ctx, f.req(), WorkflowInput{ProjectID:f.p, TicketID:v.ID, SessionID:f.reviewer, ExpectedRevision:f.get(v.ID).Revision, ParentID:parent.ID, Reason:"policy"})
	f.fail("WORKFLOW_TICKET", e); assertUnchanged(old)
}

func TestFoundationLegacyAndStoppedDecisions(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	f.s.Store.DB.Exec("DELETE FROM workflow_policies WHERE project_id=?", f.p)
	v := f.ticket("legacy")
	if _, e := f.s.Store.DB.Exec("UPDATE tickets SET state='review' WHERE id=?; UPDATE ticket_metadata SET legacy_human=1 WHERE ticket_id=?", v.ID, v.ID); e != nil {
		t.Fatal(e)
	}
	_, e := f.s.AcceptTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision})
	f.fail("HUMAN_REQUIRED", e)
	_, e = f.s.RejectTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision, Reason: "failed"})
	f.fail("HUMAN_REQUIRED", e)
	if _, e = f.s.Store.DB.Exec("UPDATE ticket_metadata SET legacy_human=0 WHERE ticket_id=?", v.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.SessionAction(f.ctx, f.p, f.reviewer, "stop", "stop-reviewer"); e != nil {
		t.Fatal(e)
	}
	_, e = f.s.AcceptTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.reviewer, ExpectedRevision: v.Revision})
	f.fail("SESSION_STOPPED", e)
}
func (f *workflowFixture) fail(code string, e error) {
	f.t.Helper()
	fault, ok := e.(*Fault)
	if !ok || fault.Code != code {
		f.t.Fatalf("expected %s, got %v", code, e)
	}
}

func TestFoundationMixedPolicyDependencyEdits(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	gated := f.ticket("gated")
	if _, e := f.s.Store.DB.Exec("DELETE FROM workflow_policies WHERE project_id=?", f.p); e != nil {
		t.Fatal(e)
	}
	plain := f.ticket("plain")
	f.wf("edit", gated.ID, WorkflowInput{Human: true, Title: gated.Title, Body: gated.Body, Reason: "depends on ordinary work", Dependencies: []string{plain.ID}})
	f.ready(gated.ID)
	_, e := f.s.EditTicket(f.ctx, f.req(), WorkflowInput{ProjectID: f.p, TicketID: plain.ID, SessionID: f.reviewer, ExpectedRevision: plain.Revision, Title: plain.Title, Body: "material change", Reason: "scope"})
	if e != nil {
		t.Fatal(e)
	}
	if got := f.get(gated.ID); got.State != "draft" || got.Workflow.PreparedRevision != 0 || got.Workflow.AuthorizedRevision != 0 {
		t.Fatal(got)
	}
	// Reverse edge: an ordinary dependent remains ready after its gated prerequisite changes.
	f.wf("edit", gated.ID, WorkflowInput{Human: true, Title: gated.Title, Body: gated.Body, Reason: "clear dependency"})
	plain = f.get(plain.ID)
	_, e = f.s.EditTicket(f.ctx, f.req(), WorkflowInput{ProjectID: f.p, TicketID: plain.ID, SessionID: f.reviewer, ExpectedRevision: plain.Revision, Title: plain.Title, Body: plain.Body, Reason: "depends on gated", Dependencies: []string{gated.ID}})
	if e != nil {
		t.Fatal(e)
	}
	f.wf("edit", gated.ID, WorkflowInput{Human: true, Title: gated.Title, Body: "changed again", Reason: "scope"})
	if got := f.get(plain.ID); got.State != "ready" || got.Workflow != nil {
		t.Fatal(got)
	}
}
