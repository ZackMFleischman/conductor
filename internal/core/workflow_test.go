package core

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"testing"
)

func TestWorkflowDraftGate(t *testing.T) {
	ctx := context.Background()
	st, e := store.Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	g := gitctx.Context{CommonDir: "common", Root: "root"}
	raw, e := s.Init(ctx, g, "WF", "init")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	_, e = s.Workflow(ctx, store.Request{ID: "policy"}, "configure", WorkflowInput{ProjectID: p.ID, Human: true, Reason: "authorized", ValidationMode: "independent_agent", ExecutionMode: "delegated", PlanReview: "lightweight", Intent: "original user intent"})
	if e != nil {
		t.Fatal(e)
	}
	raw, e = s.CreateTicket(ctx, store.Request{ID: "draft"}, TicketInput{ProjectID: p.ID, Title: "draft", Body: "criteria"})
	if e != nil {
		t.Fatal(e)
	}
	var v TicketRecord
	json.Unmarshal(raw, &v)
	if v.State != "draft" {
		t.Fatalf("state=%s", v.State)
	}
}

type workflowFixture struct {
	t                              *testing.T
	ctx                            context.Context
	s                              *Service
	p                              string
	g                              gitctx.Context
	seq                            int
	worker, reviewer, otherSession string
}

func newWorkflowFixture(t *testing.T, mode string) *workflowFixture {
	t.Helper()
	st, e := store.Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { st.DB.Close() })
	f := &workflowFixture{t: t, ctx: context.Background(), s: &Service{st}, g: gitctx.Context{CommonDir: "common", Root: "root"}}
	raw, e := f.s.Init(f.ctx, f.g, "WF", "init")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	f.p = p.ID
	for _, name := range []string{"worker", "reviewer"} {
		if _, e = f.s.RegisterAgent(f.ctx, p.ID, name, "", "", name); e != nil {
			t.Fatal(e)
		}
	}
	session := func(name, key string) string {
		raw, e := f.s.StartSession(f.ctx, p.ID, name, key, f.g)
		if e != nil {
			t.Fatal(e)
		}
		var ss Session
		json.Unmarshal(raw, &ss)
		return ss.ID
	}
	f.worker = session("worker", "w")
	f.otherSession = session("worker", "w2")
	f.reviewer = session("reviewer", "r")
	f.wf("configure", "", WorkflowInput{Human: true, Reason: "user policy", Intent: "Original intent and non-goals", ValidationMode: mode, ExecutionMode: "delegated", PlanReview: "lightweight", RequiredChecks: []string{"unit"}})
	return f
}
func (f *workflowFixture) req() store.Request {
	f.seq++
	return store.Request{ID: fmt.Sprintf("req-%d", f.seq)}
}
func (f *workflowFixture) ticket(title string) TicketRecord {
	f.t.Helper()
	raw, e := f.s.CreateTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, Title: title, Body: "criteria"})
	if e != nil {
		f.t.Fatal(e)
	}
	var v TicketRecord
	json.Unmarshal(raw, &v)
	return v
}
func (f *workflowFixture) get(id string) TicketRecord {
	f.t.Helper()
	v, e := readTicket(f.ctx, f.s.Store.DB, f.p, id)
	if e != nil {
		f.t.Fatal(e)
	}
	return v
}
func (f *workflowFixture) wf(op, id string, in WorkflowInput) json.RawMessage {
	f.t.Helper()
	in.ProjectID = f.p
	in.TicketID = id
	if id != "" {
		in.ExpectedRevision = f.get(id).Revision
	}
	raw, e := f.s.Workflow(f.ctx, f.req(), op, in)
	if e != nil {
		f.t.Fatalf("%s: %v", op, e)
	}
	return raw
}
func (f *workflowFixture) ready(id string) {
	f.wf("prepare", id, WorkflowInput{Human: true, Body: "criteria => unit test, independent reviewer validates commit"})
	f.wf("authorize", id, WorkflowInput{Human: true, Reason: "within user grant"})
}
func (f *workflowFixture) claim(id string) TicketRecord {
	f.t.Helper()
	v := f.get(id)
	raw, e := f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: id, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	if e != nil {
		f.t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	return v
}
func workflowFail(t *testing.T, code string, e error) {
	t.Helper()
	if e == nil {
		t.Fatalf("expected %s", code)
	}
	f, ok := e.(*Fault)
	if !ok || f.Code != code {
		t.Fatalf("expected %s got %v", code, e)
	}
}
func TestWorkflowSpecificationGateAndIndependence(t *testing.T) {
	f := newWorkflowFixture(t, "independent_agent")
	v := f.ticket("work")
	_, e := f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: v.Revision, Location: f.g})
	workflowFail(t, "NOT_AUTHORIZED", e)
	f.ready(v.ID)
	v = f.claim(v.ID)
	_, e = f.s.NoteTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, Body: "progress"})
	if e != nil {
		t.Fatal(e)
	}
	current := f.get(v.ID)
	if current.Workflow.SpecRevision != 1 || current.Workflow.AuthorizedRevision != 1 {
		t.Fatal(current)
	}
	_, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: current.Revision, Commit: "abc", Summary: "done", Evidence: "unit pass commit abc", QA: "review criteria"})
	if e != nil {
		t.Fatal(e)
	}
	current = f.get(v.ID)
	in := TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.otherSession, ExpectedRevision: current.Revision, Validation: &WorkflowInput{Commit: "abc", Criteria: "all", Evidence: "reviewed diff and tests", ContextID: "fresh-review-context", Checks: map[string]string{"unit": "pass"}}}
	_, e = f.s.AcceptTicket(f.ctx, f.req(), in)
	workflowFail(t, "NOT_INDEPENDENT", e)
	in.SessionID = f.reviewer
	_, e = f.s.AcceptTicket(f.ctx, f.req(), in)
	if e != nil {
		t.Fatal(e)
	}
	if f.get(v.ID).State != "done" {
		t.Fatal("not accepted")
	}
	var count int
	f.s.Store.DB.QueryRow("SELECT count(*) FROM ticket_decisions WHERE ticket_id=?", v.ID).Scan(&count)
	if count != 1 {
		t.Fatal(count)
	}
}
func TestWorkflowMaterialEditPausesOwnerAndDependents(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	a, b := f.ticket("base"), f.ticket("dependent")
	f.wf("edit", b.ID, WorkflowInput{Human: true, Title: b.Title, Body: b.Body, Reason: "add prerequisite", Dependencies: []string{a.ID}})
	f.ready(a.ID)
	f.ready(b.ID)
	_, e := f.s.ClaimTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: b.ID, SessionID: f.worker, ExpectedRevision: f.get(b.ID).Revision, Location: f.g})
	workflowFail(t, "DEPENDENCY_BLOCKED", e)
	a = f.claim(a.ID)
	f.wf("edit", a.ID, WorkflowInput{Human: true, Title: a.Title, Body: "new criteria", Reason: "user amendment"})
	after := f.get(a.ID)
	if !after.Active || !after.Workflow.Paused || after.Workflow.SpecRevision != 2 || f.get(b.ID).Workflow.AuthorizedRevision != 0 {
		t.Fatal(after, f.get(b.ID))
	}
	_, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: a.ID, SessionID: f.worker, ClaimID: a.ClaimID, ExpectedRevision: after.Revision, Summary: "old", Evidence: "old", QA: "old"})
	workflowFail(t, "WORK_PAUSED", e)
	_, e = f.s.NoteTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: a.ID, SessionID: f.worker, ClaimID: a.ClaimID, Body: "checkpoint acknowledged"})
	if e != nil {
		t.Fatal(e)
	}
	f.ready(a.ID)
	if f.get(a.ID).State != "in_progress" {
		t.Fatal("owner lost")
	}
}
func TestWorkflowGraphCyclesAndPolicyFencing(t *testing.T) {
	f := newWorkflowFixture(t, "automated")
	a, b := f.ticket("a"), f.ticket("b")
	f.wf("edit", a.ID, WorkflowInput{Human: true, Title: "a", Body: "a", Reason: "dependency", Dependencies: []string{b.ID}, ParentID: b.ID})
	_, e := f.s.Workflow(f.ctx, f.req(), "edit", WorkflowInput{ProjectID: f.p, TicketID: b.ID, ExpectedRevision: f.get(b.ID).Revision, Human: true, Title: "b", Body: "b", Reason: "bad", Dependencies: []string{a.ID}})
	workflowFail(t, "CYCLE", e)
	_, e = f.s.Workflow(f.ctx, f.req(), "edit", WorkflowInput{ProjectID: f.p, TicketID: b.ID, ExpectedRevision: f.get(b.ID).Revision, Human: true, Title: "b", Body: "b", Reason: "bad", ParentID: a.ID})
	workflowFail(t, "CYCLE", e)
	if f.get(b.ID).Workflow.SpecRevision != 1 {
		t.Fatal("cycle failure changed spec")
	}
	_, e = f.s.Workflow(f.ctx, f.req(), "configure", WorkflowInput{ProjectID: f.p, SessionID: f.worker, ValidationMode: "automated", Reason: "weaken"})
	workflowFail(t, "AUTHORITY_REQUIRED", e)
	_, e = f.s.Workflow(f.ctx, f.req(), "prepare", WorkflowInput{ProjectID: f.p, TicketID: b.ID, SessionID: f.worker, ExpectedRevision: 1, Body: "self approve"})
	workflowFail(t, "AUTHORITY_REQUIRED", e)
}
func TestWorkflowBlockReleaseAndAutomatedEvidence(t *testing.T) {
	f := newWorkflowFixture(t, "automated")
	v := f.ticket("work")
	f.ready(v.ID)
	v = f.claim(v.ID)
	_, e := f.s.BlockTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Reason: "missing input"})
	if e != nil {
		t.Fatal(e)
	}
	blocked := f.get(v.ID)
	if blocked.Active || blocked.State != "blocked" {
		t.Fatal(blocked)
	}
	f.wf("unblock", v.ID, WorkflowInput{Human: true, Reason: "input supplied"})
	v = f.claim(v.ID)
	_, e = f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Commit: "abc", Summary: "done", Evidence: "checks", QA: "automatic"})
	if e != nil {
		t.Fatal(e)
	}
	in := TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: f.get(v.ID).Revision, Validation: &WorkflowInput{Commit: "abc", Criteria: "all", Evidence: "test log", Checks: map[string]string{"unit": "fail"}}}
	_, e = f.s.AcceptTicket(f.ctx, f.req(), in)
	workflowFail(t, "CHECKS_REQUIRED", e)
	in.Validation.Checks["unit"] = "pass"
	_, e = f.s.AcceptTicket(f.ctx, f.req(), in)
	if e != nil {
		t.Fatal(e)
	}
}

func TestWorkflowSubmissionBindsTestedCommit(t *testing.T) {
	f := newWorkflowFixture(t, "automated")
	v := f.ticket("work")
	f.ready(v.ID)
	v = f.claim(v.ID)
	_, e := f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Summary: "done", Evidence: "checks", QA: "automatic"})
	workflowFail(t, "COMMIT_REQUIRED", e)
}
func TestWorkflowLegacyReadWithoutWorkflowTables(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	if _, e := f.s.Store.DB.Exec(`DELETE FROM workflow_policies`); e != nil {
		t.Fatal(e)
	}
	v := f.ticket("legacy")
	if _, e := f.s.Store.DB.Exec(`DROP TABLE workflow_ticket_specs`); e != nil {
		t.Fatal(e)
	}
	_, e := f.s.ShowTicket(f.ctx, f.p, v.ID)
	if e != nil {
		t.Fatal(e)
	}
}
func TestWorkflowCritiqueDispositionsAndImmutableIntent(t *testing.T) {
	f := newWorkflowFixture(t, "human")
	f.wf("configure", "", WorkflowInput{Human: true, ExpectedRevision: 1, Reason: "require preparation review", Intent: "cannot overwrite original", ValidationMode: "human", ExecutionMode: "human", PlanReview: "independent_agent"})
	v := f.ticket("reviewed proposal")
	_, e := f.s.Workflow(f.ctx, f.req(), "prepare", WorkflowInput{ProjectID: f.p, TicketID: v.ID, ExpectedRevision: 1, Human: true, Body: "coverage"})
	workflowFail(t, "REVIEW_REQUIRED", e)
	raw := f.wf("critique", v.ID, WorkflowInput{SessionID: f.reviewer, Body: "missing edge case", ContextID: "fresh-planning-review", Significant: true})
	var q struct {
		CritiqueID string `json:"critique_id"`
	}
	json.Unmarshal(raw, &q)
	_, e = f.s.Workflow(f.ctx, f.req(), "prepare", WorkflowInput{ProjectID: f.p, TicketID: v.ID, ExpectedRevision: f.get(v.ID).Revision, Human: true, Body: "coverage"})
	workflowFail(t, "UNRESOLVED_FINDINGS", e)
	f.wf("dispose", v.ID, WorkflowInput{Human: true, CritiqueID: q.CritiqueID, Body: "accepted explicitly as out of scope under original intent"})
	f.ready(v.ID)
	f.wf("amend", "", WorkflowInput{Human: true, Intent: "explicit user amendment"})
	w, e := f.s.ShowWorkflow(f.ctx, f.p)
	if e != nil {
		t.Fatal(e)
	}
	intents := w["intents"].([]map[string]string)
	if len(intents) != 2 || intents[0]["body"] != "Original intent and non-goals" {
		t.Fatal(intents)
	}
	show, e := f.s.ShowTicket(f.ctx, f.p, v.ID)
	if e != nil || show["workflow_history"] == nil {
		t.Fatal(show, e)
	}
}
func TestWorkflowAcceptanceRollbackAndReplay(t *testing.T) {
	f := newWorkflowFixture(t, "automated")
	v := f.ticket("work")
	f.ready(v.ID)
	v = f.claim(v.ID)
	_, e := f.s.SubmitTicket(f.ctx, f.req(), TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ClaimID: v.ClaimID, ExpectedRevision: v.Revision, Commit: "abc", Summary: "done", Evidence: "checks", QA: "automatic"})
	if e != nil {
		t.Fatal(e)
	}
	in := TicketInput{ProjectID: f.p, TicketID: v.ID, SessionID: f.worker, ExpectedRevision: f.get(v.ID).Revision, Validation: &WorkflowInput{Commit: "wrong", Criteria: "all", Evidence: "unit transcript", Checks: map[string]string{"unit": "pass"}}}
	_, e = f.s.AcceptTicket(f.ctx, f.req(), in)
	workflowFail(t, "STALE_EVIDENCE", e)
	in.Validation.Commit = "abc"
	_, e = f.s.Store.DB.Exec(`CREATE TRIGGER validation_event_failure BEFORE INSERT ON events WHEN NEW.kind='ticket.accept' BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if e != nil {
		t.Fatal(e)
	}
	r := f.req()
	_, e = f.s.AcceptTicket(f.ctx, r, in)
	if e == nil {
		t.Fatal("expected injected failure")
	}
	var n int
	f.s.Store.DB.QueryRow(`SELECT count(*) FROM ticket_decisions`).Scan(&n)
	if n != 0 || f.get(v.ID).State != "review" {
		t.Fatal("partial acceptance")
	}
	f.s.Store.DB.Exec(`DROP TRIGGER validation_event_failure`)
	raw, e := f.s.AcceptTicket(f.ctx, r, in)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := f.s.AcceptTicket(f.ctx, r, in)
	if e != nil || string(raw) != string(replay) {
		t.Fatal("replay differs", e)
	}
	f.s.Store.DB.QueryRow(`SELECT count(*) FROM ticket_decisions`).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
