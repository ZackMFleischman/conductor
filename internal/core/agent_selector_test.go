package core

import (
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"testing"
)

func TestAgentUUIDNameCollision(t *testing.T) {
	ctx := context.Background()
	st, e := store.Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	g := gitctx.Context{CommonDir: "common", Root: "root"}
	raw, e := s.Init(ctx, g, "APP", "i")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	raw, e = s.RegisterAgent(ctx, p.ID, "original", "", "", "a")
	if e != nil {
		t.Fatal(e)
	}
	var a Agent
	json.Unmarshal(raw, &a)
	t.Run("reserve UUID-shaped names", func(t *testing.T) {
		if _, e = s.RegisterAgent(ctx, p.ID, a.ID, "", "", "collision"); e == nil {
			t.Fatal("accepted agent name equal to existing UUID")
		}
	})
	t.Run("legacy exact ID wins", func(t *testing.T) {
		_, e = st.DB.Exec("DELETE FROM agents WHERE project_id=? AND name=?", p.ID, a.ID)
		if e != nil {
			t.Fatal(e)
		}
		_, e = st.DB.Exec("INSERT INTO agents(id,project_id,name) VALUES(?,?,?)", UUID(), p.ID, a.ID)
		if e != nil {
			t.Fatal(e)
		}
		raw, e = s.StartSession(ctx, p.ID, a.ID, "start", g)
		if e != nil {
			t.Fatal(e)
		}
		var session Session
		json.Unmarshal(raw, &session)
		if session.AgentID != a.ID {
			t.Fatalf("selector %s bound to %s", a.ID, session.AgentID)
		}
	})
}
