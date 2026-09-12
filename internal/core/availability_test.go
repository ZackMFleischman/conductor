package core

import (
	"context"
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
	"path/filepath"
	"testing"
)

func TestAgentListLabelsUnknownLiveness(t *testing.T) {
	st, e := store.Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	s := Service{st}
	ctx := context.Background()
	raw, e := s.Init(ctx, gitctx.Context{CommonDir: "p"}, "APP", "i")
	if e != nil {
		t.Fatal(e)
	}
	var p Project
	json.Unmarshal(raw, &p)
	s.RegisterAgent(ctx, p.ID, "dev", "", "", "a")
	a, e := s.ListAgents(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	raw = JSON(a[0])
	var v map[string]any
	json.Unmarshal(raw, &v)
	if v["availability"] != "registered-without-session" {
		t.Fatal(v)
	}
	s.StartSession(ctx, p.ID, "dev", "s", gitctx.Context{CommonDir: "p", Root: "w"})
	a, e = s.ListAgents(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	raw = JSON(a[0])
	json.Unmarshal(raw, &v)
	if v["availability"] != "unknown" {
		t.Fatal(v)
	}
}
