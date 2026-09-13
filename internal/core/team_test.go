package core

import (
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"sync"
	"testing"
)

type teamFixture struct {
	t                         *testing.T
	s                         *Service
	p, supervisor, child, aid string
	n                         int
}

func newTeamFixture(t *testing.T) *teamFixture {
	t.Helper()
	st, e := store.Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { st.DB.Close() })
	f := &teamFixture{t: t, s: &Service{st}}
	ctx := context.Background()
	g := gitctx.Context{CommonDir: "common", Root: "root"}
	b, e := f.s.Init(ctx, g, "APP", "init")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(b, &p)
	f.p = p.ID
	for _, name := range []string{"coordinator", "child"} {
		b, e = f.s.RegisterAgent(ctx, f.p, name, "", "", name)
		if e != nil {
			t.Fatal(e)
		}
		var a Agent
		json.Unmarshal(b, &a)
		b, e = f.s.StartSession(ctx, f.p, a.ID, name+"session", g)
		if e != nil {
			t.Fatal(e)
		}
		var s Session
		json.Unmarshal(b, &s)
		if name == "child" {
			f.child = s.ID
			f.aid = a.ID
		} else {
			f.supervisor = s.ID
		}
	}
	return f
}
func (f *teamFixture) profile() *TeamProfile {
	return &TeamProfile{Scope: "approved fixture", ApprovalReference: "user request fixture", SkillVersion: "2", Host: "native-test", HostLimit: 3, ChildLimit: 2, RequiredRoles: []string{"improver"}, ImprovementEnabled: true, Capabilities: []string{"spawn", "status"}, PollingSeconds: 30, DeliveryPolicy: "worktree", StopConditions: "scope accepted"}
}
func (f *teamFixture) action(op, key string, in TeamInput) TeamRun {
	f.t.Helper()
	b, e := f.s.TeamAction(context.Background(), f.p, op, key, in)
	if e != nil {
		f.t.Fatal(op, e)
	}
	var r TeamRun
	if e = json.Unmarshal(b, &r); e != nil {
		f.t.Fatal(e)
	}
	return r
}
func (f *teamFixture) rejected(op, key, code string, in TeamInput) {
	f.t.Helper()
	_, e := f.s.TeamAction(context.Background(), f.p, op, key, in)
	v, ok := e.(*Fault)
	if !ok || v.Code != code {
		f.t.Fatalf("%s expected %s got %v", op, code, e)
	}
}
func owned(r TeamRun) TeamInput {
	return TeamInput{RunID: r.ID, SessionID: r.CoordinatorSessionID, CoordinationID: r.CoordinationID, ExpectedRevision: r.Revision}
}
func (f *teamFixture) replacement() string {
	f.t.Helper()
	b, e := f.s.StartSession(context.Background(), f.p, "coordinator", UUID(), gitctx.Context{CommonDir: "common", Root: "root"})
	if e != nil {
		f.t.Fatal(e)
	}
	var s Session
	json.Unmarshal(b, &s)
	return s.ID
}
func TestTeamExclusiveCoordinatorRace(t *testing.T) {
	f := newTeamFixture(t)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, sess := range []string{f.supervisor, f.child} {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			_, e := f.s.TeamAction(context.Background(), f.p, "start", s, TeamInput{SessionID: s, Profile: f.profile()})
			errors <- e
		}(sess)
	}
	wg.Wait()
	close(errors)
	ok := 0
	for e := range errors {
		if e == nil {
			ok++
		} else if v, k := e.(*Fault); !k || v.Code != "COORDINATION_HELD" {
			t.Fatal(e)
		}
	}
	if ok != 1 {
		t.Fatal(ok)
	}
}
func TestTeamLaunchReadinessRecovery(t *testing.T) {
	f := newTeamFixture(t)
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	initial := owned(r)
	f.rejected("ready", "early", "TEAM_NOT_READY", owned(r))
	in := owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	r = f.action("launch", "launch", in)
	launch := r.Launches[0].ID
	// The same request returns the same durable launch even after the revision moves.
	replay := f.action("launch", "launch", in)
	if replay.Launches[0].ID != launch {
		t.Fatal("duplicate launch")
	}
	in = owned(r)
	in.LaunchID = launch
	in.HostID = "fresh-host"
	in.ChildSessionID = f.child
	r = f.action("register", "register", in)
	in = owned(r)
	in.LaunchID = launch
	in.HostState = "active"
	in.Evidence = "native host running"
	r = f.action("observe", "observe", in)
	ack := TeamInput{RunID: r.ID, SessionID: f.child, ExpectedRevision: r.Revision, LaunchID: launch, Epoch: r.Epoch, HostID: "fresh-host", Role: "improver", AgentID: f.aid, Scope: r.Profile.Scope, SkillVersion: "2", Checkpoint: "watching reports"}
	r = f.action("ack", "ack", ack)
	r = f.action("ready", "ready", owned(r))
	if !r.Ready {
		t.Fatal(r)
	}
	r = f.action("resume", "resume", owned(r))
	if r.Ready {
		t.Fatal("resume trusted stale readiness")
	}
	ack.ExpectedRevision = r.Revision
	f.rejected("ack", "old-ack", "STALE_READINESS", ack)
	f.rejected("resume", "stale", "REVISION_CONFLICT", initial)
	in = owned(r)
	in.Reason = "inspected old host and files; checkpoint requested"
	in.Human = true
	in.SessionID = f.replacement()
	r = f.action("recover", "recover", in)
	in = owned(r)
	in.SessionID = f.supervisor
	in.CoordinationID = initial.CoordinationID
	f.rejected("resume", "old-owner", "COORDINATION_REVOKED", in)
}
func TestTeamUnknownCapacityAndStoppedSession(t *testing.T) {
	f := newTeamFixture(t)
	p := f.profile()
	p.HostLimit = 2
	p.ChildLimit = 1
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: p})
	in := owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	r = f.action("launch", "launch", in)
	in = owned(r)
	in.LaunchID = r.Launches[0].ID
	in.HostState = "unknown"
	in.Evidence = "host response lost"
	r = f.action("observe", "unknown", in)
	in = owned(r)
	in.Role = "reviewer"
	in.AgentID = f.aid
	f.rejected("launch", "over", "CAPACITY_EXCEEDED", in)
	_, e := f.s.SessionAction(context.Background(), f.p, f.supervisor, "stop", "session-stop")
	if e != nil {
		t.Fatal(e)
	}
	f.rejected("ready", "dead-owner", "SESSION_STOPPED", owned(r))
	r, e = f.s.GetTeam(context.Background(), f.p, r.ID)
	if e != nil || r.Ready {
		t.Fatal(r, e)
	}
}
func TestTeamHostChangeRequiresNewChallenge(t *testing.T) {
	f := newTeamFixture(t)
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	in := owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	r = f.action("launch", "launch", in)
	lid := r.Launches[0].ID
	in = owned(r)
	in.LaunchID = lid
	in.ChildSessionID = f.child
	in.HostID = "host"
	r = f.action("register", "register", in)
	in = owned(r)
	in.LaunchID = lid
	in.HostState = "active"
	in.Evidence = "native tool reports active"
	r = f.action("observe", "observe", in)
	ack := TeamInput{RunID: r.ID, SessionID: f.child, ExpectedRevision: r.Revision, LaunchID: lid, Epoch: r.Epoch, HostID: "host", Role: "improver", AgentID: f.aid, Scope: r.Profile.Scope, SkillVersion: "2", Checkpoint: "ready"}
	r = f.action("ack", "ack", ack)
	r = f.action("ready", "ready", owned(r))
	in = owned(r)
	in.LaunchID = lid
	in.HostState = "unknown"
	in.Evidence = "host tool unavailable"
	r = f.action("observe", "unknown", in)
	if r.Ready {
		t.Fatal("unknown host ready")
	}
	ack.ExpectedRevision = r.Revision
	f.rejected("ack", "obsolete", "STALE_READINESS", ack)
}
func TestTeamStopPreservesClaimsAndReclaimsOnlyStoppedChildren(t *testing.T) {
	f := newTeamFixture(t)
	r, ticket := f.claimFixture()
	lid := r.Launches[0].ID
	ctx := context.Background()
	in := owned(r)
	b, e := f.s.ClaimTicket(ctx, store.Request{ID: "claim"}, TicketInput{ProjectID: f.p, TicketID: ticket.ID, SessionID: f.child, ExpectedRevision: ticket.Revision, Location: gitctx.Context{CommonDir: "common", Root: "root"}})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(b, &ticket)
	in = owned(r)
	in.Reason = "checkpoint requested from host"
	r = f.action("stop", "stop", in)
	if r.Status != "stopping" || r.CoordinationID == "" {
		t.Fatal(r)
	}
	in = owned(r)
	in.LaunchID = lid
	in.HostState = "finished"
	in.Evidence = "host says completed"
	f.rejected("observe", "early-finish", "ACTIVE_CHILD_SESSION", in)
	session, e := ReadSession(ctx, f.s.Store.DB, f.p, f.child)
	if e != nil || !session.Busy {
		t.Fatal("claim lost", session, e)
	}
	_, e = f.s.ReleaseTicket(ctx, store.Request{ID: "release-ticket"}, TicketInput{ProjectID: f.p, TicketID: ticket.ID, SessionID: f.child, ClaimID: ticket.ClaimID, ExpectedRevision: ticket.Revision, Reason: "checkpoint"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.s.SessionAction(ctx, f.p, f.child, "stop", "stop-child")
	if e != nil {
		t.Fatal(e)
	}
	r = f.action("observe", "finish", in)
	in = owned(r)
	in.Reason = "children confirmed stopped"
	r = f.action("stop", "complete", in)
	if r.Status != "stopped" || r.CoordinationID != "" {
		t.Fatal(r)
	}
	f.rejected("recover", "revive", "TEAM_STOPPED", TeamInput{RunID: r.ID, SessionID: f.supervisor, ExpectedRevision: r.Revision, Human: true, Reason: "cannot revive stopped run"})
}
func TestTeamReleaseFenceAndProfileCannotChange(t *testing.T) {
	f := newTeamFixture(t)
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	in := owned(r)
	in.Profile = f.profile()
	in.Profile.Scope = "different"
	f.rejected("resume", "scope-change", "INVALID_INPUT", in)
	in = owned(r)
	in.Reason = "checkpoint"
	r = f.action("release", "release", in)
	old := in
	old.ExpectedRevision = r.Revision
	f.rejected("resume", "released", "COORDINATION_REVOKED", old)
	f.rejected("start", "duplicate", "COORDINATION_HELD", TeamInput{SessionID: f.child, Profile: f.profile()})
	in = TeamInput{RunID: r.ID, SessionID: f.supervisor, ExpectedRevision: r.Revision}
	f.rejected("recover", "unapproved", "RECOVERY_REQUIRED", in)
	in.Human = true
	in.Reason = "inspected old supervisor, children and worktrees"
	f.rejected("recover", "same-session", "CHILD_REUSED", in)
	in.SessionID = f.replacement()
	r = f.action("recover", "recover", in)
	if r.Profile.Scope != "approved fixture" || r.CoordinationID == old.CoordinationID {
		t.Fatal(r)
	}
}
func TestTeamRegistrationRejectsReusedHostAndSession(t *testing.T) {
	f := newTeamFixture(t)
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	in := owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	r = f.action("launch", "launch", in)
	lid := r.Launches[0].ID
	in = owned(r)
	in.LaunchID = lid
	in.ChildSessionID = f.child
	in.HostID = "host"
	r = f.action("register", "register", in)
	in.ExpectedRevision = r.Revision
	r = f.action("register", "register-again", in)
	if len(r.Launches) != 1 {
		t.Fatal(r)
	}
	ctx := context.Background()
	_, e := f.s.SessionAction(ctx, f.p, f.child, "stop", "stop-child")
	if e != nil {
		t.Fatal(e)
	}
	in = owned(r)
	in.LaunchID = lid
	in.HostState = "stopped"
	in.Evidence = "native tool confirmed stopped"
	r = f.action("observe", "stopped", in)
	b, e := f.s.StartSession(ctx, f.p, f.aid, "new-child", gitctx.Context{CommonDir: "common", Root: "new-root"})
	if e != nil {
		t.Fatal(e)
	}
	var child Session
	json.Unmarshal(b, &child)
	in = owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	r = f.action("launch", "replacement", in)
	in = owned(r)
	in.LaunchID = r.Launches[1].ID
	in.ChildSessionID = child.ID
	in.HostID = "host"
	f.rejected("register", "reused-host", "CHILD_REUSED", in)
	in.HostID = "new-host"
	r = f.action("register", "new-host", in)
	in = owned(r)
	in.LaunchID = r.Launches[1].ID
	in.ChildSessionID = f.child
	in.HostID = "host"
	f.rejected("register", "stopped-session", "SESSION_STOPPED", in)
}
func TestTeamRegistrationRequiresUnusedExecutionSession(t *testing.T) {
	f := newTeamFixture(t)
	ctx := context.Background()
	b, e := f.s.CreateTicket(ctx, store.Request{ID: "ticket"}, TicketInput{ProjectID: f.p, Title: "old attempt", Body: "test"})
	if e != nil {
		t.Fatal(e)
	}
	var ticket TicketRecord
	json.Unmarshal(b, &ticket)
	b, e = f.s.ClaimTicket(ctx, store.Request{ID: "claim"}, TicketInput{ProjectID: f.p, TicketID: ticket.ID, SessionID: f.child, ExpectedRevision: ticket.Revision, Location: gitctx.Context{CommonDir: "common", Root: "root"}})
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(b, &ticket)
	_, e = f.s.ReleaseTicket(ctx, store.Request{ID: "release"}, TicketInput{ProjectID: f.p, TicketID: ticket.ID, SessionID: f.child, ClaimID: ticket.ClaimID, ExpectedRevision: ticket.Revision, Reason: "previous work"})
	if e != nil {
		t.Fatal(e)
	}
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	in := owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	r = f.action("launch", "launch", in)
	in = owned(r)
	in.LaunchID = r.Launches[0].ID
	in.HostID = "new-host"
	in.ChildSessionID = f.child
	f.rejected("register", "old-attempt", "CHILD_REUSED", in)
}
func TestTeamUnsupportedHostCannotLaunch(t *testing.T) {
	f := newTeamFixture(t)
	p := f.profile()
	p.Capabilities = []string{"status"}
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: p})
	in := owned(r)
	in.Role = "improver"
	in.AgentID = f.aid
	f.rejected("launch", "unsupported", "HOST_CAPABILITY", in)
}
