package cli_test

import (
	"encoding/json"
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"os"
	"path/filepath"
	"testing"
)

func TestTeamCLIContractAndReadOnly(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("team", "--help"))
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "coord", "--request", "agent"))
	s := testkit.MustData(t, c.Run("session", "start", "--agent", "coord", "--request", "session"))
	file := filepath.Join(t.TempDir(), "body.json")
	body := map[string]any{"profile": map[string]any{"scope": "CLI fixture", "approval_reference": "user authorization fixture", "skill_version": "2", "host": "native-test", "host_limit": 2, "child_limit": 1, "required_roles": []string{}, "capabilities": []string{"spawn", "status"}, "polling_seconds": 30, "delivery_policy": "worktree", "stop_conditions": "fixture complete"}}
	b, _ := json.Marshal(body)
	if e := os.WriteFile(file, b, 0600); e != nil {
		t.Fatal(e)
	}
	r := testkit.MustData(t, c.Run("team", "start", "--session", s["session_id"].(string), "--body-file", file, "--request", "start"))
	testkit.MustData(t, c.Run("team", "show", r["run_id"].(string)))
	testkit.MustData(t, c.Run("team", "list"))
	events := testkit.MustData(t, c.Run("team", "events", r["run_id"].(string)))
	if len(events["events"].([]any)) != 1 {
		t.Fatal(events)
	}
	if e := os.WriteFile(file, []byte(`{"surprise":true}`), 0600); e != nil {
		t.Fatal(e)
	}
	res := c.Run("team", "start", "--session", s["session_id"].(string), "--body-file", file, "--request", "bad")
	if res["ok"] == true {
		t.Fatal("unknown field accepted")
	}
}
