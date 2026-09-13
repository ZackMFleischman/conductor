package core

import (
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"testing"
)

// Read a challenge through JSON so the regression also executes on schema 3.
func challengeAck(r TeamRun, l TeamLaunch) TeamInput {
	in := TeamInput{RunID: r.ID, SessionID: l.SessionID, ExpectedRevision: r.Revision, LaunchID: l.ID, Epoch: r.Epoch, HostID: l.HostID, Role: l.Role, AgentID: l.AgentID, Scope: r.Profile.Scope, SkillVersion: r.Profile.SkillVersion, Checkpoint: "ready"}
	var fields map[string]any
	json.Unmarshal(JSON(l), &fields)
	generation := fields["challenge_generation"]
	if generation == nil {
		generation = 1
	}
	json.Unmarshal(JSON(map[string]any{"challenge_generation": generation}), &in)
	return in
}
func TestTeamIndependentAcknowledgements(t *testing.T) {
	f := newTeamFixture(t)
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	for i := 0; i < 2; i++ {
		aid, session := f.aid, f.child
		if i == 1 {
			b, e := f.s.RegisterAgent(context.Background(), f.p, "other", "", "", "other")
			if e != nil {
				t.Fatal(e)
			}
			var a Agent
			json.Unmarshal(b, &a)
			aid = a.ID
			b, e = f.s.StartSession(context.Background(), f.p, aid, "other-session", gitctx.Context{CommonDir: "common", Root: "root"})
			if e != nil {
				t.Fatal(e)
			}
			var s Session
			json.Unmarshal(b, &s)
			session = s.ID
		}
		in := owned(r)
		in.Role = "improver"
		in.AgentID = aid
		r = f.action("launch", UUID(), in)
		l := r.Launches[len(r.Launches)-1]
		in = owned(r)
		in.LaunchID = l.ID
		in.HostID = session
		in.ChildSessionID = session
		r = f.action("register", UUID(), in)
		in = owned(r)
		in.LaunchID = l.ID
		in.HostState = "active"
		in.Evidence = "native active"
		r = f.action("observe", UUID(), in)
	}
	// Reobserve both at one epoch for the original global-epoch protocol.
	for _, l := range r.Launches {
		in := owned(r)
		in.LaunchID = l.ID
		in.HostState = "active"
		in.Evidence = "active"
		r = f.action("observe", UUID(), in)
	}
	a, b := challengeAck(r, r.Launches[0]), challengeAck(r, r.Launches[1])
	results := make(chan error, 2)
	for _, in := range []TeamInput{a, b} {
		go func(in TeamInput) {
			_, e := f.s.TeamAction(context.Background(), f.p, "ack", in.LaunchID+"-ack", in)
			results <- e
		}(in)
	}
	for i := 0; i < 2; i++ {
		if e := <-results; e != nil {
			t.Fatal(e)
		}
	}
	var e error
	r, e = f.s.GetTeam(context.Background(), f.p, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	r = f.action("ready", "ready", owned(r))
	if !r.Ready {
		t.Fatal(r.ReadinessIssues)
	}
	var writes int
	if e := f.s.Store.DB.QueryRow("SELECT count(*) FROM events WHERE kind LIKE 'team.%'").Scan(&writes); e != nil {
		t.Fatal(e)
	}
	t.Logf("two-child readiness fixture: %d team writes; 2 simultaneous own ACKs; 0 revision retries", writes)
	// A terminal non-required sibling leaves the healthy challenge and ACK intact.
	sibling := r.Launches[1]
	healthy := r.Launches[0]
	if _, e := f.s.SessionAction(context.Background(), f.p, sibling.SessionID, "stop", "stop-sibling"); e != nil {
		t.Fatal(e)
	}
	in := owned(r)
	in.LaunchID = sibling.ID
	in.HostState = "finished"
	in.Evidence = "native finished"
	r = f.action("observe", "finish-sibling", in)
	if !r.Ready || r.Launches[0].ChallengeGeneration != healthy.ChallengeGeneration {
		t.Fatal("terminal sibling invalidated healthy child", r)
	}
	old := challengeAck(r, r.Launches[0])
	for _, state := range []string{"unknown", "active"} {
		in = owned(r)
		in.LaunchID = healthy.ID
		in.HostState = state
		in.Evidence = state
		r = f.action("observe", state, in)
	}
	f.rejected("ack", "aba", "STALE_READINESS", old)
	fresh := challengeAck(r, r.Launches[0])
	r = f.action("ack", "fresh", fresh)
	in = owned(r)
	in.LaunchID = healthy.ID
	in.HostState = "active"
	in.Evidence = "same state"
	same := f.action("observe", "same", in)
	if same.Launches[0].ChallengeGeneration != r.Launches[0].ChallengeGeneration || !same.Ready {
		t.Fatal("same state invalidated readiness")
	}
	r = f.action("resume", "resume", owned(same))
	fresh.Epoch = r.Epoch // Equal positive launch generations alone must not suffice.
	f.rejected("ack", "old-observation", "STALE_READINESS", fresh)
	in = owned(r)
	in.LaunchID = healthy.ID
	in.HostState = "active"
	in.Evidence = "reconciled after resume"
	r = f.action("observe", "reobserve", in)
	fresh = challengeAck(r, r.Launches[0])
	zero := fresh
	zero.ChallengeGeneration = 0
	f.rejected("ack", "zero", "STALE_READINESS", zero)
	bad := fresh
	bad.Scope = "other"
	f.rejected("ack", "scope", "CHILD_MISMATCH", bad)
	if _, e := f.s.SessionAction(context.Background(), f.p, f.supervisor, "stop", "stop-coordinator"); e != nil {
		t.Fatal(e)
	}
	f.rejected("ack", "dead-coordinator", "SESSION_STOPPED", fresh)
	t.Log("same-snapshot ACKs: 2 successes / 0 retries; unrelated terminal transition: 1 observation / 0 sibling ACKs")
}

func TestTeamAcknowledgementSemanticGuards(t *testing.T) {
	for _, field := range []string{"host", "role", "identity", "scope", "contract", "checkpoint", "child", "coordinator", "fence", "release", "recover"} {
		t.Run(field, func(t *testing.T) {
			f := newTeamFixture(t)
			r, _ := f.claimFixture()
			in := challengeAck(r, r.Launches[0])
			code := "CHILD_MISMATCH"
			switch field {
			case "host":
				in.HostID = "wrong"
			case "role":
				in.Role = "reviewer"
			case "identity":
				in.AgentID = f.supervisor
			case "scope":
				in.Scope = "wrong"
			case "contract":
				in.SkillVersion = "wrong"
			case "checkpoint":
				in.Checkpoint = " "
			case "child", "coordinator":
				session := f.child
				if field == "coordinator" {
					session = f.supervisor
				}
				if _, e := f.s.SessionAction(context.Background(), f.p, session, "stop", "stop-session"); e != nil {
					t.Fatal(e)
				}
				code = "SESSION_STOPPED"
			case "fence":
				if _, e := f.s.Store.DB.Exec("UPDATE team_runs SET coordination_id=NULL WHERE id=?", r.ID); e != nil {
					t.Fatal(e)
				}
				code = "STALE_READINESS"
			case "release":
				v := owned(r)
				v.Reason = "release"
				f.action("release", "release", v)
				code = "STALE_READINESS"
			case "recover":
				v := owned(r)
				v.SessionID = f.replacement()
				v.Human = true
				v.Reason = "inspected prior coordinator"
				f.action("recover", "recover", v)
				code = "STALE_READINESS"
			}
			f.rejected("ack", "rejected-ack", code, in)
		})
	}
}
