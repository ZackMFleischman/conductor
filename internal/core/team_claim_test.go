package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/store"
)

func (f *teamFixture) ticket() TicketRecord {
	f.t.Helper()
	b, e := f.s.CreateTicket(context.Background(), store.Request{ID: UUID()}, TicketInput{ProjectID: f.p, Title: "assignment", Body: "test"})
	if e != nil {
		f.t.Fatal(e)
	}
	var ticket TicketRecord
	if e = json.Unmarshal(b, &ticket); e != nil {
		f.t.Fatal(e)
	}
	return ticket
}

func TestTeamValidatorLaunchReviewAssignedToImplementer(t *testing.T) {
	f := newTeamFixture(t)
	ticket := f.ticket()
	if _, e := f.s.Store.DB.Exec("UPDATE tickets SET state='review',assigned_agent_id=(SELECT agent_id FROM sessions WHERE id=?) WHERE id=?", f.supervisor, ticket.ID); e != nil {
		t.Fatal(e)
	}
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: f.profile()})
	in := owned(r)
	in.Role, in.AgentID, in.TicketID = "validator", f.aid, ticket.ID
	r = f.action("launch", "validator", in)
	if len(r.Launches) != 1 || r.Launches[0].TicketID != ticket.ID {
		t.Fatal(r)
	}
}

// Fixtures isolate recorded host evidence; they do not claim to inspect a process.
func (f *teamFixture) claimFixture() (TeamRun, TicketRecord) {
	f.t.Helper()
	ticket := f.ticket()
	profile := f.profile()
	profile.RequiredRoles = nil
	profile.ImprovementEnabled = false
	r := f.action("start", "start", TeamInput{SessionID: f.supervisor, Profile: profile})
	r = f.action("ready", "ready", owned(r))
	in := owned(r)
	in.Role, in.AgentID, in.TicketID = "worker", f.aid, ticket.ID
	r = f.action("launch", "launch", in)
	in = owned(r)
	in.LaunchID, in.HostID, in.ChildSessionID = r.Launches[0].ID, "host", f.child
	r = f.action("register", "register", in)
	in = owned(r)
	in.LaunchID, in.HostState, in.Evidence = r.Launches[0].ID, "active", "native host result"
	r = f.action("observe", "observe", in)
	r = f.action("ack", "ack", TeamInput{RunID: r.ID, SessionID: f.child, ExpectedRevision: r.Revision, LaunchID: r.Launches[0].ID, Epoch: r.Epoch, ChallengeGeneration: r.Launches[0].ChallengeGeneration, HostID: "host", Role: "worker", AgentID: f.aid, Scope: r.Profile.Scope, SkillVersion: r.Profile.SkillVersion, Checkpoint: "exact assignment ready"})
	r = f.action("ready", "ready-again", owned(r))
	return r, ticket
}

func (f *teamFixture) checkClaim(ticket, session string) error {
	c, e := f.s.Store.DB.Conn(context.Background())
	if e != nil {
		f.t.Fatal(e)
	}
	defer c.Close()
	return CheckTeamClaimTx(context.Background(), c, f.p, ticket, session)
}

func TestTeamClaimRestrictions(t *testing.T) {
	cases := []struct{ name, sql, code string }{
		{"valid", "", ""},
		{"wrong role", "UPDATE team_launches SET role='validator'", "TEAM_ASSIGNMENT"},
		{"improver role", "UPDATE team_launches SET role='improver'", "TEAM_ASSIGNMENT"},
		{"wrong ticket", "UPDATE team_launches SET ticket_id=NULL", "TEAM_ASSIGNMENT"},
		{"pending registration", "UPDATE team_launches SET session_id=NULL,host_id=NULL", "TEAM_NOT_READY"},
		{"wrong session", "UPDATE team_launches SET session_id=(SELECT coordinator_session_id FROM team_runs LIMIT 1)", "TEAM_NOT_READY"},
		{"zero-generation", "UPDATE team_launches SET challenge_generation=0,acknowledged_generation=0", "TEAM_NOT_READY"},
		{"wrong-generation", "UPDATE team_launches SET acknowledged_generation=challenge_generation+1", "TEAM_NOT_READY"},
		{"unacknowledged", "UPDATE team_launches SET acknowledged_epoch=0", "TEAM_NOT_READY"},
		{"stale observation", "UPDATE team_launches SET observed_epoch=0", "TEAM_NOT_READY"},
		{"unknown host", "UPDATE team_launches SET host_state='unknown'", "TEAM_NOT_READY"},
		{"finished host", "UPDATE team_launches SET host_state='finished'", "TEAM_NOT_READY"},
		{"stopping", "UPDATE team_runs SET status='stopping'", "TEAM_NOT_READY"},
		{"degraded", "UPDATE team_runs SET status='degraded'", "TEAM_NOT_READY"},
		{"starting", "UPDATE team_runs SET status='starting'", "TEAM_NOT_READY"},
		{"coordinator unavailable", "UPDATE sessions SET declared_state='stopped' WHERE id=(SELECT coordinator_session_id FROM team_runs LIMIT 1)", "TEAM_NOT_READY"},
		{"released", "UPDATE team_runs SET status='released',coordination_id=NULL", "TEAM_NOT_READY"},
		{"historical stopped run", "UPDATE team_runs SET status='stopped',coordination_id=NULL", "TEAM_NOT_READY"},
		{"stopped session", "UPDATE sessions SET declared_state='stopped' WHERE agent_id=(SELECT agent_id FROM team_launches LIMIT 1)", "SESSION_STOPPED"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f := newTeamFixture(t)
			_, ticket := f.claimFixture()
			if tt.sql != "" {
				if _, e := f.s.Store.DB.Exec(tt.sql); e != nil {
					t.Fatal(e)
				}
			}
			e := f.checkClaim(ticket.ID, f.child)
			if tt.code == "" {
				if e != nil {
					t.Fatal(e)
				}
				return
			}
			fault, ok := e.(*Fault)
			if !ok || fault.Code != tt.code {
				t.Fatalf("want %s got %v", tt.code, e)
			}
		})
	}
}

func TestTeamClaimAttemptAndOrdinaryIsolation(t *testing.T) {
	f := newTeamFixture(t)
	_, ticket := f.claimFixture()
	if e := f.checkClaim(ticket.ID, f.supervisor); e != nil {
		t.Fatalf("ordinary session restricted: %v", e)
	}
	other := f.ticket()
	if e := f.checkClaim(other.ID, f.child); e == nil {
		t.Fatal("managed worker acquired another ticket")
	}
	// A released attempt must still consume the fresh session's one-job allowance.
	if _, e := f.s.Store.DB.Exec("INSERT INTO claims(id,project_id,ticket_id,session_id,created_at,released_at) VALUES(?,?,?,?,?,?)", UUID(), f.p, ticket.ID, f.child, Now(), Now()); e != nil {
		t.Fatal(e)
	}
	e := f.checkClaim(ticket.ID, f.child)
	fault, ok := e.(*Fault)
	if !ok || fault.Code != "CHILD_REUSED" {
		t.Fatalf("released claim was reusable: %v", e)
	}
}

// Keep the guard contract tied to the existing transaction connection type.
var _ func(context.Context, *sql.Conn, string, string, string) error = CheckTeamClaimTx
