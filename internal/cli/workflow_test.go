package cli_test

import (
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkflowCLIManagedLifecycle(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "WF", "--request", "init"))
	file := func(name, body string) string {
		p := filepath.Join(t.TempDir(), name)
		if e := os.WriteFile(p, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	policy := file("policy.json", `{"intent":"Implement a routine documented change","reason":"Explicit user opt-in","execution_mode":"human","plan_review":"lightweight","validation_mode":"automated","required_checks":["unit"]}`)
	testkit.MustData(t, c.Run("workflow", "configure", "--human", "--body-file", policy, "--request", "policy"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	ss := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "session"))
	sid := testkit.String(t, ss, "session_id")
	body := file("body.txt", "Purpose, bounded scope, acceptance criteria and validation evidence.")
	v := testkit.MustData(t, c.Run("ticket", "create", "--title", "Routine change", "--body-file", body, "--session", sid, "--request", "create"))
	id := testkit.String(t, v, "id")
	if v["state"] != "draft" {
		t.Fatal(v)
	}
	prepare := file("prepare.json", `{"body":"Criterion: unit test, worker; stage: worktree commit; evidence: test output"}`)
	testkit.MustData(t, c.Run("ticket", "prepare", id, "--human", "--body-file", prepare, "--expect-revision", "1", "--request", "prepare"))
	grant := file("grant.json", `{"reason":"User approved exact prepared scope"}`)
	testkit.MustData(t, c.Run("ticket", "authorize", id, "--human", "--body-file", grant, "--expect-revision", "2", "--request", "authorize"))
	v = testkit.MustData(t, c.Run("ticket", "claim", id, "--session", sid, "--expect-revision", "3", "--request", "claim"))
	cid := testkit.String(t, v, "claim_id")
	testkit.MustData(t, c.Run("ticket", "submit", id, "--session", sid, "--claim", cid, "--expect-revision", "4", "--commit", "abc", "--summary-file", body, "--evidence-file", body, "--qa-file", body, "--request", "submit"))
	validation := file("validation.json", `{"commit":"abc","criteria":"all listed criteria","evidence":"unit test command and transcript","checks":{"unit":"pass"}}`)
	v = testkit.MustData(t, c.Run("ticket", "accept", id, "--session", sid, "--expect-revision", "5", "--validation-file", validation, "--request", "accept"))
	if v["state"] != "done" {
		t.Fatal(v)
	}
	shown := testkit.MustData(t, c.Run("ticket", "show", id))
	if shown["workflow_history"] == nil {
		t.Fatal(shown)
	}
}
