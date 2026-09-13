package cli_test

import (
	"encoding/json"
	"fmt"
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

func TestTeamCLIAcknowledgementWithoutRevision(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "coord", "--request", "coord"))
	agent := testkit.MustData(t, c.Run("agent", "register", "--name", "child", "--request", "child"))
	coord := testkit.MustData(t, c.Run("session", "start", "--agent", "coord", "--request", "cs"))["session_id"].(string)
	child := testkit.MustData(t, c.Run("session", "start", "--agent", "child", "--request", "ws"))["session_id"].(string)
	file := filepath.Join(t.TempDir(), "body.json")
	write := func(v any) {
		b, _ := json.Marshal(v)
		if e := os.WriteFile(file, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	write(map[string]any{"profile": map[string]any{"scope": "scope", "approval_reference": "approved", "skill_version": "layers-v1", "host": "fixture", "host_limit": 2, "child_limit": 1, "capabilities": []string{"spawn", "status"}, "polling_seconds": 1, "delivery_policy": "worktree", "stop_conditions": "done"}})
	r := testkit.MustData(t, c.Run("team", "start", "--session", coord, "--request", "start", "--body-file", file))
	action := func(op string, body any) {
		write(body)
		r = testkit.MustData(t, c.Run("team", op, r["run_id"].(string), "--session", coord, "--coordination", r["coordination_id"].(string), "--expect-revision", fmt.Sprint(r["revision"]), "--request", op, "--body-file", file))
	}
	action("launch", map[string]any{"role": "improver", "agent_id": agent["agent_id"]})
	launch := r["launches"].([]any)[0].(map[string]any)["launch_id"]
	action("register", map[string]any{"launch_id": launch, "host_id": "host", "child_session_id": child})
	action("observe", map[string]any{"launch_id": launch, "host_state": "active", "evidence": "native active"})
	ack := map[string]any{"launch_id": launch, "epoch": r["epoch"], "host_id": "host", "agent_id": agent["agent_id"], "role": "improver", "scope": "scope", "skill_version": "layers-v1", "checkpoint": "ready"}
	write(ack)
	res := c.Run("team", "ack", r["run_id"].(string), "--session", child, "--request", "zero", "--body-file", file)
	if res["ok"] == true {
		t.Fatal("missing generation accepted")
	}
	ack["challenge_generation"] = r["launches"].([]any)[0].(map[string]any)["challenge_generation"]
	write(ack)
	testkit.MustData(t, c.Run("team", "ack", r["run_id"].(string), "--session", child, "--request", "ack", "--body-file", file))
}
