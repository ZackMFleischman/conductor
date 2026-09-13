package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/testkit"
)

func TestRetrospectiveCLIBoundedResume(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "reviewer", "--request", "agent"))
	session := testkit.MustData(t, c.Run("session", "start", "--agent", "reviewer", "--request", "session"))
	sid := testkit.String(t, session, "session_id")
	dir := t.TempDir()
	problem := filepath.Join(dir, "problem.json")
	if err := os.WriteFile(problem, []byte(`{"summary":"setup","expected":"works","actual":"fails"}`), 0600); err != nil {
		t.Fatal(err)
	}
	testkit.MustData(t, c.Run("problem", "add", "--session", sid, "--body-file", problem, "--request", "p"))
	b := testkit.MustData(t, c.Run("retrospective", "begin", "--session", sid, "--limit", "1", "--request", "begin"))
	bid := testkit.String(t, b, "id")
	changes := b["changes"].([]any)
	in := map[string]any{"batch_id": bid, "group_key": "setup", "action": "observe", "observation": "one failure", "rationale": "need repeat", "revisit_trigger": "next session", "review_after": "2099-01-01T00:00:00Z", "event_seqs": []any{changes[0].(map[string]any)["seq"]}}
	body, _ := json.Marshal(in)
	decisionFile := filepath.Join(dir, "decision.json")
	if err := os.WriteFile(decisionFile, body, 0600); err != nil {
		t.Fatal(err)
	}
	d := testkit.MustData(t, c.Run("retrospective", "decide", "--session", sid, "--body-file", decisionFile, "--request", "decide"))
	testkit.MustData(t, c.Run("retrospective", "commit", bid, "--session", sid, "--request", "commit"))
	shown := testkit.MustData(t, c.Run("retrospective", "show", bid))
	if shown["status"] != "committed" {
		t.Fatal(shown)
	}
	detail := testkit.MustData(t, c.Run("retrospective", "decision", testkit.String(t, d, "id")))
	if detail["rationale"] != "need repeat" {
		t.Fatal(detail)
	}
	status := testkit.MustData(t, c.Run("retrospective", "status"))
	if status["pending_changes"].(float64) != 0 {
		t.Fatal(status)
	}
	listing := testkit.MustData(t, c.Run("retrospective", "list", "--limit", "1"))
	if len(listing["items"].([]any)) != 1 {
		t.Fatal(listing)
	}
}
