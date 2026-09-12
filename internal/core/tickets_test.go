package core

import (
	"context"

	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestTicketFencingRecoveryAndRollback(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "db")
	st, e := store.Open(path, true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	g := gitctx.Context{CommonDir: "common", Root: "root"}
	raw, e := s.Init(ctx, g, "APP", "init")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	s.RegisterAgent(ctx, p.ID, "dev", "", "", "agent")
	session := func(key string) string {
		raw, e := s.StartSession(ctx, p.ID, "dev", key, g)
		if e != nil {
			t.Fatal(e)
		}
		var v Session
		json.Unmarshal(raw, &v)
		return v.ID
	}
	a, b := session("a"), session("b")
	create := func(key string) TicketRecord {
		raw, e := s.CreateTicket(ctx, store.Request{ID: key}, TicketInput{ProjectID: p.ID, Title: key, Body: "description"})
		if e != nil {
			t.Fatal(e)
		}
		var v TicketRecord
		json.Unmarshal(raw, &v)
		return v
	}
	first, second := create("first"), create("second")
	claim := TicketInput{ProjectID: p.ID, TicketID: first.ID, SessionID: a, ExpectedRevision: 1, Location: g}
	raw, e = s.ClaimTicket(ctx, store.Request{ID: "claim"}, claim)
	if e != nil {
		t.Fatal(e)
	}
	var owner TicketRecord
	json.Unmarshal(raw, &owner)
	fail := func(code string, e error) {
		t.Helper()
		if e == nil {
			t.Fatalf("expected %s", code)
		}
		f, ok := e.(*Fault)
		if !ok || f.Code != code {
			t.Fatalf("expected %s got %v", code, e)
		}
	}
	_, e = s.ClaimTicket(ctx, store.Request{ID: "busy"}, TicketInput{ProjectID: p.ID, TicketID: second.ID, SessionID: a, ExpectedRevision: 1, Location: g})
	fail("ACTIVE_CLAIM", e)
	_, e = s.AssignTicket(ctx, store.Request{ID: "assign"}, TicketInput{ProjectID: p.ID, TicketID: first.ID, AssignedAgentID: "none", ExpectedRevision: 2})
	fail("ACTIVE_CLAIM", e)
	for _, action := range []string{"idle", "stop"} {
		_, e = s.SessionAction(ctx, p.ID, a, action, action)
		fail("ACTIVE_CLAIM", e)
	}
	var count int
	st.DB.QueryRow("SELECT count(*) FROM events WHERE ticket_id=?", first.ID).Scan(&count)
	v, _ := readTicket(ctx, st.DB, p.ID, first.ID)
	if count != 2 || v.Revision != 2 {
		t.Fatal(count, v)
	}
	_, e = s.ClaimTicket(ctx, store.Request{ID: "independent"}, TicketInput{ProjectID: p.ID, TicketID: second.ID, SessionID: b, ExpectedRevision: 1, Location: g})
	if e != nil {
		t.Fatal(e)
	}
	raw, e = s.ReleaseTicket(ctx, store.Request{ID: "recover"}, TicketInput{ProjectID: p.ID, TicketID: first.ID, ExpectedRevision: 2, Reason: "abandoned", Human: true})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &v)
	s.SessionAction(ctx, p.ID, a, "resume", "resume")
	stale := TicketInput{ProjectID: p.ID, TicketID: first.ID, SessionID: a, ClaimID: owner.ClaimID, Body: "stale", ExpectedRevision: 2, Summary: "s", Evidence: "e", QA: "q"}
	_, e = s.NoteTicket(ctx, store.Request{ID: "stale-note"}, stale)
	fail("CLAIM_REVOKED", e)
	_, e = s.SubmitTicket(ctx, store.Request{ID: "stale-submit"}, stale)
	fail("CLAIM_REVOKED", e)
	raw, e = s.ClaimTicket(ctx, store.Request{ID: "claim"}, claim)
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &owner)
	if owner.Active {
		t.Fatal(owner)
	}
	// Inject an event failure after claim insertion/ticket update: all business and journal writes roll back.
	_, e = st.DB.Exec(`CREATE TRIGGER fail_ticket_event BEFORE INSERT ON events WHEN NEW.kind='ticket.claim' BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.ClaimTicket(ctx, store.Request{ID: "rollback"}, TicketInput{ProjectID: p.ID, TicketID: first.ID, SessionID: a, ExpectedRevision: 3, Location: g})
	if e == nil {
		t.Fatal("expected injected failure")
	}
	st.DB.QueryRow("SELECT count(*) FROM claims WHERE ticket_id=? AND released_at IS NULL", first.ID).Scan(&count)
	v, _ = readTicket(ctx, st.DB, p.ID, first.ID)
	if count != 0 || v.Revision != 3 {
		t.Fatal(count, v)
	}
	st.DB.QueryRow("SELECT count(*) FROM requests WHERE request_id='rollback'").Scan(&count)
	if count != 0 {
		t.Fatal(count)
	}
}
func TestTicketConcurrentSameIdentityClaim(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "db")
	st, e := store.Open(path, true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	g := gitctx.Context{CommonDir: "common", Root: "root"}
	raw, e := s.Init(ctx, g, "APP", "init")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	s.RegisterAgent(ctx, p.ID, "dev", "", "", "agent")
	ids := []string{}
	for _, key := range []string{"a", "b"} {
		raw, e = s.StartSession(ctx, p.ID, "dev", key, g)
		if e != nil {
			t.Fatal(e)
		}
		var v Session
		json.Unmarshal(raw, &v)
		ids = append(ids, v.ID)
	}
	raw, e = s.CreateTicket(ctx, store.Request{ID: "create"}, TicketInput{ProjectID: p.ID, Title: "one", Body: "body"})
	if e != nil {
		t.Fatal(e)
	}
	var ticket TicketRecord
	json.Unmarshal(raw, &ticket)
	other, e := store.Open(path, false)
	if e != nil {
		t.Fatal(e)
	}
	defer other.DB.Close()
	services := []*Service{&s, {Store: other}}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := range services {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = services[i].ClaimTicket(ctx, store.Request{ID: ids[i]}, TicketInput{ProjectID: p.ID, TicketID: ticket.ID, SessionID: ids[i], ExpectedRevision: 1, Location: g})
		}(i)
	}
	close(start)
	wg.Wait()
	wins := 0
	for _, e := range errs {
		if e == nil {
			wins++
		} else if f, ok := e.(*Fault); !ok || f.Code != "REVISION_CONFLICT" {
			t.Fatal(e)
		}
	}
	if wins != 1 {
		t.Fatal(errs)
	}
	var n int
	st.DB.QueryRow("SELECT count(*) FROM events WHERE ticket_id=?", ticket.ID).Scan(&n)
	if n != 2 {
		t.Fatal(n)
	}
}
func TestTicketInputLimits(t *testing.T) {
	valid := TicketInput{Title: strings.Repeat("界", 300), Body: strings.Repeat("x", TicketTextLimit)}
	if e := ValidateTicketInput("create", valid); e != nil {
		t.Fatal(e)
	}
	for _, in := range []TicketInput{{Title: strings.Repeat("界", 301), Body: "b"}, {Title: "x", Body: strings.Repeat("x", TicketTextLimit+1)}, {Title: "x", Body: string([]byte{255})}, {Title: " ", Body: "b"}} {
		if e := ValidateTicketInput("create", in); e == nil {
			t.Fatal("accepted invalid input")
		}
	}
	if e := ValidateTicketInput("submit", TicketInput{SessionID: "s", ExpectedRevision: 1, Summary: "s", Evidence: "e"}); e == nil {
		t.Fatal("missing QA accepted")
	}
	if e := ValidateTicketInput("release", TicketInput{Human: true, SessionID: "s", ExpectedRevision: 1, Reason: "r"}); e == nil {
		t.Fatal("mixed owner/human")
	}

}
func TestTicketMutationsUpdateSessionContactAtomically(t *testing.T) {
	ctx := context.Background()
	st, e := store.Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	g := gitctx.Context{CommonDir: "common", Root: "original", Head: "original-head"}
	raw, e := s.Init(ctx, g, "APP", "init")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	if _, e = s.RegisterAgent(ctx, p.ID, "dev", "", "", "agent"); e != nil {
		t.Fatal(e)
	}
	raw, e = s.StartSession(ctx, p.ID, "dev", "session", g)
	if e != nil {
		t.Fatal(e)
	}
	var session Session
	json.Unmarshal(raw, &session)
	raw, e = s.CreateTicket(ctx, store.Request{ID: "create"}, TicketInput{ProjectID: p.ID, Title: "test", Body: "body"})
	if e != nil {
		t.Fatal(e)
	}
	var ticket TicketRecord
	json.Unmarshal(raw, &ticket)
	branch := "feature"
	g.Root = "other-worktree"
	g.Branch = &branch
	g.Head = "new-head"
	in := TicketInput{ProjectID: p.ID, TicketID: ticket.ID, SessionID: session.ID, ExpectedRevision: 1, Location: g}
	contact := func() Session {
		t.Helper()
		v, e := ReadSession(ctx, st.DB, p.ID, session.ID)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	prior := session.LastSeenAt
	raw, e = s.ClaimTicket(ctx, store.Request{ID: "claim"}, in)
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &ticket)
	current := contact()
	if current.LastSeenAt == prior {
		t.Error("claim did not update contact")
	}
	if current.WorktreeRoot != g.Root || current.Head != g.Head || current.Branch == nil || *current.Branch != branch {
		t.Errorf("claim did not update session location: %+v", current)
	}
	prior = current.LastSeenAt
	if _, e = s.ClaimTicket(ctx, store.Request{ID: "claim"}, in); e != nil {
		t.Fatal(e)
	}
	if contact().LastSeenAt != prior {
		t.Error("claim replay changed contact")
	}
	in.ClaimID = ticket.ClaimID
	in.Body = "note"
	in.ExpectedRevision = 2
	raw, e = s.NoteTicket(ctx, store.Request{ID: "note"}, in)
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &ticket)
	current = contact()
	if current.LastSeenAt == prior {
		t.Error("note did not update contact")
	}
	prior = current.LastSeenAt
	if _, e = s.NoteTicket(ctx, store.Request{ID: "note"}, in); e != nil {
		t.Fatal(e)
	}
	if contact().LastSeenAt != prior {
		t.Error("note replay changed contact")
	}
	in.Summary = "summary"
	in.Evidence = "evidence"
	in.QA = "manual QA"
	in.ExpectedRevision = 2
	if _, e = s.SubmitTicket(ctx, store.Request{ID: "bad-revision"}, in); e == nil {
		t.Fatal("expected conflict")
	}
	if contact().LastSeenAt != prior {
		t.Error("failed submit changed contact")
	}
	in.ExpectedRevision = 3
	if _, e = st.DB.Exec(`CREATE TRIGGER fail_submit BEFORE INSERT ON events WHEN NEW.kind='ticket.submit' BEGIN SELECT RAISE(ABORT,'injected'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.SubmitTicket(ctx, store.Request{ID: "rollback"}, in); e == nil {
		t.Fatal("expected event failure")
	}
	if contact().LastSeenAt != prior {
		t.Error("rolled back submit changed contact")
	}
	if _, e = st.DB.Exec("DROP TRIGGER fail_submit"); e != nil {
		t.Fatal(e)
	}
	raw, e = s.SubmitTicket(ctx, store.Request{ID: "submit"}, in)
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &ticket)
	current = contact()
	if current.LastSeenAt == prior {
		t.Error("submit did not update contact")
	}
	prior = current.LastSeenAt
	if _, e = s.SubmitTicket(ctx, store.Request{ID: "submit"}, in); e != nil {
		t.Fatal(e)
	}
	if contact().LastSeenAt != prior {
		t.Error("submit replay changed contact")
	}
	if _, e = s.RejectTicket(ctx, store.Request{ID: "reject"}, TicketInput{ProjectID: p.ID, TicketID: ticket.ID, ExpectedRevision: 4, Human: true, Reason: "rework"}); e != nil {
		t.Fatal(e)
	}
	if contact().LastSeenAt != prior {
		t.Error("human reject changed session contact")
	}
	in.ExpectedRevision = 5
	raw, e = s.ClaimTicket(ctx, store.Request{ID: "reclaim"}, in)
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &ticket)
	prior = contact().LastSeenAt
	in.ClaimID = ticket.ClaimID
	in.ExpectedRevision = 6
	in.Reason = "checkpoint"
	if _, e = s.ReleaseTicket(ctx, store.Request{ID: "release"}, in); e != nil {
		t.Fatal(e)
	}
	current = contact()
	if current.LastSeenAt == prior {
		t.Error("owner release did not update contact")
	}
	prior = current.LastSeenAt
	if _, e = s.ReleaseTicket(ctx, store.Request{ID: "release"}, in); e != nil {
		t.Fatal(e)
	}
	if contact().LastSeenAt != prior {
		t.Error("release replay changed contact")
	}
}
