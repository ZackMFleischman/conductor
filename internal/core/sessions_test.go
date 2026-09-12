package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"testing"
)

func TestActiveClaimPreventsIdleStopAndDerivesBusy(t *testing.T) {
	ctx := context.Background()
	st, e := store.Open(filepath.Join(t.TempDir(), "conductor.db"), true)
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
	if _, e = s.RegisterAgent(ctx, p.ID, "dev", "", "", "agent"); e != nil {
		t.Fatal(e)
	}
	raw, e = s.StartSession(ctx, p.ID, "dev", "start", g)
	if e != nil {
		t.Fatal(e)
	}
	var session Session
	json.Unmarshal(raw, &session)
	_, e = st.Write(ctx, store.Request{ID: "fixture", Operation: "fixture", ProjectID: p.ID, ActorID: "local-user", Payload: JSON(map[string]any{})}, func(c *sql.Conn) (json.RawMessage, error) {
		_, e := c.ExecContext(ctx, "INSERT INTO tickets(id,project_id,display_key,title,body,state,created_at,updated_at) VALUES('ticket',?,'APP-1','test','body','in_progress',?,?)", p.ID, Now(), Now())
		if e != nil {
			return nil, e
		}
		_, e = c.ExecContext(ctx, "INSERT INTO claims(id,project_id,ticket_id,session_id,created_at) VALUES('claim',?,'ticket',?,?)", p.ID, session.ID, Now())
		return JSON(map[string]any{}), e
	})
	if e != nil {
		t.Fatal(e)
	}
	for _, action := range []string{"idle", "stop"} {
		if _, e = s.SessionAction(ctx, p.ID, session.ID, action, action); e == nil || e.(*Fault).Code != "ACTIVE_CLAIM" {
			t.Fatalf("%s %v", action, e)
		}
	}
	raw, e = s.SessionAction(ctx, p.ID, session.ID, "resume", "resume")
	if e != nil {
		t.Fatal(e)
	}
	json.Unmarshal(raw, &session)
	if !session.Busy || session.Claim == nil {
		t.Fatal(session)
	}
}
func TestIdentityProjectIsolation(t *testing.T) {
	ctx := context.Background()
	st, e := store.Open(filepath.Join(t.TempDir(), "conductor.db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	for _, common := range []string{"one", "two"} {
		raw, e := s.Init(ctx, gitctx.Context{CommonDir: common}, "APP", "init")
		if e != nil {
			t.Fatal(e)
		}
		var p Project
		json.Unmarshal(raw, &p)
		if _, e = s.RegisterAgent(ctx, p.ID, "same", "", "", "register"); e != nil {
			t.Fatal(e)
		}
		if _, e = s.RegisterAgent(ctx, p.ID, "none", "", "", "none"); e == nil {
			t.Fatal("reserved name")
		}
		if _, e = s.StartSession(ctx, p.ID, "unknown", "unknown", gitctx.Context{CommonDir: common}); e == nil {
			t.Fatal("unknown agent")
		}
	}
}
