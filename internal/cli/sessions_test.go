package cli_test

import (
	"database/sql"
	"github.com/ZackMFleischman/conductor/internal/testkit"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"testing"
)

func TestIdentitySessionLifecycle(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	a := testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	again := testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	if a["agent_id"] != again["agent_id"] {
		t.Fatal("replay changed identity")
	}
	bad := c.Run("agent", "register", "--name", "dev", "--request", "duplicate")
	if bad["ok"] != false {
		t.Fatal(bad)
	}
	s := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "start"))
	id := testkit.String(t, s, "session_id")
	s2 := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "start2"))
	if s["agent_id"] != s2["agent_id"] || s["session_id"] == s2["session_id"] {
		t.Fatal(s, s2)
	}
	r := testkit.MustData(t, c.Run("session", "resume", "--session", id, "--request", "resume"))
	if r["claim"] != nil {
		t.Fatal(r)
	}
	r2 := testkit.MustData(t, c.Run("session", "resume", "--session", id, "--request", "resume"))
	if r["last_seen_at"] != r2["last_seen_at"] {
		t.Fatal("replay touched contact")
	}
	testkit.MustData(t, c.Run("context", "--session", id))
	r3 := testkit.MustData(t, c.Run("session", "stop", "--session", id, "--request", "stop"))
	if r3["declared_state"] != "stopped" {
		t.Fatal(r3)
	}
	if c.Run("session", "resume", "--session", id, "--request", "again")["ok"] != false {
		t.Fatal("revived session")
	}
	testkit.MustData(t, c.Run("agent", "list"))
}
func TestCorruptAndNewerRegistryAreUnavailable(t *testing.T) {
	for _, kind := range []string{"corrupt", "newer"} {
		t.Run(kind, func(t *testing.T) {
			c := testkit.New(t)
			os.Mkdir(c.Home, 0700)
			path := filepath.Join(c.Home, "conductor.db")
			if kind == "corrupt" {
				os.WriteFile(path, []byte("not sqlite"), 0600)
			} else {
				db, e := sql.Open("sqlite", path)
				if e != nil {
					t.Fatal(e)
				}
				db.Exec("PRAGMA user_version=999")
				db.Close()
			}
			v := c.Run("context", "--if-registered")
			if v["ok"] != false || v["error"].(map[string]any)["code"] != "REGISTRY_UNAVAILABLE" {
				t.Fatal(v)
			}
		})
	}
}
func TestDoctorLeavesNoBusinessStore(t *testing.T) {
	c := testkit.New(t)
	d := testkit.MustData(t, c.Run("doctor", "--probe-write"))
	for k, v := range d["checks"].(map[string]any) {
		if v != true {
			t.Fatalf("%s: %v", k, v)
		}
	}
	files, e := os.ReadDir(c.Home)
	if e != nil || len(files) != 0 {
		t.Fatal(files, e)
	}
}
func TestUnknownAndExtraFlagsFail(t *testing.T) {
	c := testkit.New(t)
	for _, args := range [][]string{{"context", "unexpected"}, {"context", "--bogus"}, {"session", "start", "--unknown"}, {"doctor"}} {
		if c.Run(args...)["ok"] != false {
			t.Fatal(args)
		}
	}
}
