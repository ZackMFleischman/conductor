package cli_test

import (
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"os"
	"path/filepath"
	"testing"
)

func TestFoundationCLIPlainLifecycle(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "F", "--request", "init"))
	file := func(name, body string) string {
		p := filepath.Join(t.TempDir(), name)
		if e := os.WriteFile(p, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	for _, name := range []string{"worker", "reviewer"} {
		testkit.MustData(t, c.Run("agent", "register", "--name", name, "--request", name))
	}
	session := func(name string) string {
		return testkit.String(t, testkit.MustData(t, c.Run("session", "start", "--agent", name, "--request", name+"-session")), "session_id")
	}
	worker, reviewer := session("worker"), session("reviewer")
	body := file("body.md", "# Scope\nComplete the documented criteria.")
	v := testkit.MustData(t, c.Run("ticket", "create", "--title", "Plain", "--body-file", body, "--request", "create"))
	id := testkit.String(t, v, "id")
	if v["state"] != "ready" || v["workflow"] != nil {
		t.Fatal(v)
	}
	edit := file("edit.json", `{"title":"Updated","body":"Updated scope","kind":"improvement","reason":"clarify"}`)
	testkit.MustData(t, c.Run("ticket", "edit", id, "--session", reviewer, "--expect-revision", "1", "--body-file", edit, "--request", "edit"))
	v = testkit.MustData(t, c.Run("ticket", "claim", id, "--session", worker, "--expect-revision", "2", "--request", "claim"))
	claim := testkit.String(t, v, "claim_id")
	testkit.MustData(t, c.Run("ticket", "block", id, "--session", worker, "--claim", claim, "--expect-revision", "3", "--reason", "waiting", "--request", "block"))
	unblock := file("unblock.json", `{"reason":"resolved"}`)
	testkit.MustData(t, c.Run("ticket", "unblock", id, "--session", reviewer, "--expect-revision", "4", "--body-file", unblock, "--request", "unblock"))
	v = testkit.MustData(t, c.Run("ticket", "claim", id, "--session", worker, "--expect-revision", "5", "--request", "reclaim"))
	claim = testkit.String(t, v, "claim_id")
	testkit.MustData(t, c.Run("ticket", "submit", id, "--session", worker, "--claim", claim, "--expect-revision", "6", "--summary-file", body, "--evidence-file", body, "--qa-file", body, "--request", "submit"))
	validation := file("fail.json", `{"criteria":"requirement","evidence":"failed test","checks":{"unit":"fail"}}`)
	v = testkit.MustData(t, c.Run("ticket", "reject", id, "--session", reviewer, "--expect-revision", "7", "--reason", "regression", "--validation-file", validation, "--request", "reject"))
	if v["state"] != "ready" {
		t.Fatal(v)
	}
	shown := testkit.MustData(t, c.Run("ticket", "show", id))
	if shown["metadata"] == nil || shown["history"] == nil || shown["workflow"] != nil {
		t.Fatal(shown)
	}
}

func TestFoundationCLIMetadataOnlyEdit(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "F", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "reviewer", "--request", "reviewer"))
	session := testkit.String(t, testkit.MustData(t, c.Run("session", "start", "--agent", "reviewer", "--request", "reviewer-session")), "session_id")
	file := func(name, body string) string {
		p := filepath.Join(t.TempDir(), name)
		if e := os.WriteFile(p, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	body := file("body.md", "# Scope\nComplete the documented criteria.")
	parent := testkit.String(t, testkit.MustData(t, c.Run("ticket", "create", "--title", "Parent", "--body-file", body, "--request", "parent")), "id")
	target := testkit.MustData(t, c.Run("ticket", "create", "--title", "Target", "--body-file", body, "--request", "target"))
	metadata := file("metadata.json", `{"parent_id":"`+parent+`","kind":"bug","reason":"correct grouping"}`)
	updated := testkit.MustData(t, c.Run("ticket", "metadata", testkit.String(t, target, "id"), "--session", session, "--expect-revision", "1", "--body-file", metadata, "--request", "metadata"))
	ticket := updated["ticket"].(map[string]any)
	if ticket["state"] != "ready" {
		t.Fatal(ticket)
	}
	meta := ticket["metadata"].(map[string]any)
	if meta["parent_id"] != parent || meta["kind"] != "bug" || meta["spec_revision"].(float64) != 1 {
		t.Fatal(meta)
	}
}
