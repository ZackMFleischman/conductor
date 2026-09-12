package cli_test

import (
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestWrongAssigneeCannotClaim(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	for _, n := range []string{"frontend-1", "backend-1"} {
		testkit.MustData(t, c.Run("agent", "register", "--name", n, "--request", n))
	}
	s := testkit.MustData(t, c.Run("session", "start", "--agent", "backend-1", "--request", "session"))
	p := filepath.Join(t.TempDir(), "body.md")
	if e := os.WriteFile(p, []byte("Implement the assigned change."), 0600); e != nil {
		t.Fatal(e)
	}
	ticket := testkit.MustData(t, c.Run("ticket", "create", "--title", "Button", "--body-file", p, "--assigned-to", "frontend-1", "--request", "create"))
	got := c.Run("ticket", "claim", testkit.String(t, ticket, "id"), "--session", testkit.String(t, s, "session_id"), "--expect-revision", strconv.Itoa(int(testkit.Number(t, ticket, "revision"))), "--request", "claim")
	if got["ok"] != false {
		t.Fatal(got)
	}
	if got["error"].(map[string]any)["code"] != "ASSIGNMENT_CONFLICT" {
		t.Fatal(got)
	}
}

func TestTicketLifecycleAndReplay(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	s := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "session"))
	sid := testkit.String(t, s, "session_id")
	p := filepath.Join(t.TempDir(), "body")
	os.WriteFile(p, []byte("Verified changes and manual steps."), 0600)
	v := testkit.MustData(t, c.Run("ticket", "create", "--title", "Lifecycle", "--body-file", p, "--request", "create"))
	id := testkit.String(t, v, "id")
	rev := func() string { return strconv.Itoa(int(testkit.Number(t, v, "revision"))) }
	claimArgs := []string{"ticket", "claim", id, "--session", sid, "--expect-revision", rev(), "--request", "claim"}
	v = testkit.MustData(t, c.Run(claimArgs...))
	cid := testkit.String(t, v, "claim_id")
	v = testkit.MustData(t, c.Run("ticket", "note", id, "--session", sid, "--claim", cid, "--body-file", p, "--request", "note"))
	submit := []string{"ticket", "submit", id, "--session", sid, "--claim", cid, "--expect-revision", rev(), "--summary-file", p, "--evidence-file", p, "--qa-file", p, "--request", "submit"}
	v = testkit.MustData(t, c.Run(submit...))
	if v["state"] != "review" {
		t.Fatal(v)
	}
	again := testkit.MustData(t, c.Run(submit...))
	if again["revision"] != v["revision"] {
		t.Fatal(again)
	}
	old := testkit.MustData(t, c.Run(claimArgs...))
	if old["active"] != false {
		t.Fatal(old)
	}
	v = testkit.MustData(t, c.Run("ticket", "reject", id, "--human", "--expect-revision", rev(), "--reason", "Rework", "--request", "reject"))
	v = testkit.MustData(t, c.Run("ticket", "claim", id, "--session", sid, "--expect-revision", rev(), "--request", "claim2"))
	cid2 := testkit.String(t, v, "claim_id")
	if cid == cid2 {
		t.Fatal("reused claim")
	}
	stale := c.Run("ticket", "note", id, "--session", sid, "--claim", cid, "--body-file", p, "--request", "stale")
	if stale["ok"] != false || stale["error"].(map[string]any)["code"] != "CLAIM_REVOKED" {
		t.Fatal(stale)
	}
	v = testkit.MustData(t, c.Run("ticket", "submit", id, "--session", sid, "--claim", cid2, "--expect-revision", rev(), "--summary-file", p, "--evidence-file", p, "--qa-file", p, "--request", "submit2"))
	v = testkit.MustData(t, c.Run("ticket", "accept", id, "--human", "--expect-revision", rev(), "--request", "accept"))
	if v["state"] != "done" {
		t.Fatal(v)
	}
	shown := testkit.MustData(t, c.Run("ticket", "show", id))
	if len(shown["events"].([]any)) != 8 {
		t.Fatal(shown)
	}
}
func TestTicketAssignmentListsAndOwnerRelease(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	s := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "session"))
	sid := testkit.String(t, s, "session_id")
	p := filepath.Join(t.TempDir(), "body")
	os.WriteFile(p, []byte("body"), 0600)
	v := testkit.MustData(t, c.Run("ticket", "create", "--title", "one", "--body-file", p, "--request", "create"))
	id := testkit.String(t, v, "id")
	testkit.MustData(t, c.Run("ticket", "create", "--title", "two", "--body-file", p, "--request", "create2"))
	v = testkit.MustData(t, c.Run("ticket", "assign", id, "--to", "dev", "--expect-revision", "1", "--request", "assign"))
	if v["assigned_agent_id"] == nil {
		t.Fatal(v)
	}
	listed := testkit.MustData(t, c.Run("ticket", "list", "--assigned-to", "dev"))
	if len(listed["tickets"].([]any)) != 1 {
		t.Fatal(listed)
	}
	listed = testkit.MustData(t, c.Run("ticket", "list", "--limit", "1"))
	if listed["truncated"] != true {
		t.Fatal(listed)
	}
	next := testkit.MustData(t, c.Run("ticket", "list", "--limit", "1", "--cursor", testkit.String(t, listed, "next_cursor")))
	if next["truncated"] != false || len(next["tickets"].([]any)) != 1 {
		t.Fatal(next)
	}
	v = testkit.MustData(t, c.Run("ticket", "claim", id, "--session", sid, "--expect-revision", "2", "--request", "claim"))
	cid := testkit.String(t, v, "claim_id")
	v = testkit.MustData(t, c.Run("ticket", "release", id, "--session", sid, "--claim", cid, "--expect-revision", "3", "--reason", "checkpoint", "--request", "release"))
	if v["state"] != "ready" {
		t.Fatal(v)
	}
	v = testkit.MustData(t, c.Run("ticket", "assign", id, "--to", "none", "--expect-revision", "4", "--request", "unassign"))
	if v["assigned_agent_id"] != nil {
		t.Fatal(v)
	}
	for _, args := range [][]string{{"ticket", "accept", id, "--expect-revision", "5", "--request", "accept"}, {"ticket", "create", "--title", "x", "--body-file", p, "--request", string(make([]byte, 201))}, {"ticket", "list", "--limit", "101"}} {
		if got := c.Run(args...); got["ok"] != false {
			t.Fatal(got)
		}
	}
}
