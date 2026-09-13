package store

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func schemaThreeFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "db")
	db, e := sql.Open("sqlite", p)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(schema); e != nil {
		t.Fatal(e)
	}
	for _, m := range orderedMigrations {
		if m.version > 3 {
			break
		}
		for _, name := range m.files {
			b, e := migrations.ReadFile("migrations/" + name)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = db.Exec(string(b)); e != nil {
				t.Fatal(e)
			}
		}
	}
	if _, e = db.Exec("PRAGMA user_version=3"); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestChallengeMigrationDefaults(t *testing.T) {
	p := schemaThreeFixture(t)
	db, e := sql.Open("sqlite", p)
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(`INSERT INTO projects(id,common_dir,prefix) VALUES('p','p','P'); INSERT INTO agents(id,project_id,name) VALUES('a','p','a'); INSERT INTO sessions(id,project_id,agent_id,declared_state,last_seen_at,worktree_root) VALUES('s','p','a','active','now','root'); INSERT INTO team_runs(id,project_id,coordinator_session_id,coordination_id,status,revision,epoch,profile,created_at) VALUES('r','p','s','f','ready',1,7,'{}','now'); INSERT INTO team_launches(id,project_id,run_id,role,agent_id,host_state,observed_epoch,acknowledged_epoch,created_at) VALUES('l','p','r','improver','a','active',7,7,'now');`)
	db.Close()
	if e != nil {
		t.Fatal(e)
	}
	s, e := Open(p, false)
	if e != nil {
		t.Fatal(e)
	}
	defer s.DB.Close()
	var c, a, o, k int
	if e = s.DB.QueryRow("SELECT challenge_generation,acknowledged_generation,observed_epoch,acknowledged_epoch FROM team_launches").Scan(&c, &a, &o, &k); e != nil || c != 0 || a != 0 || o != 7 || k != 7 {
		t.Fatal(c, a, o, k, e)
	}
}
func TestFutureSchemaAtMigrationLock(t *testing.T) {
	p := schemaThreeFixture(t)
	s, e := open(p, false, func() {
		db, e := sql.Open("sqlite", p)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		if _, e = db.Exec(fmt.Sprintf("PRAGMA user_version=%d", SchemaVersion+1)); e != nil {
			t.Fatal(e)
		}
	})
	if s != nil {
		s.DB.Close()
	}
	if e == nil || !strings.Contains(e.Error(), "unsupported schema version") {
		t.Fatalf("intervening future upgrade accepted: %v", e)
	}
	for _, op := range []func(string) (*Store, error){func(p string) (*Store, error) { return Open(p, false) }, OpenReadOnly} {
		s, e = op(p)
		if s != nil {
			s.DB.Close()
		}
		if e == nil {
			t.Fatal("future schema accepted")
		}
	}
}

// The explicit fixture binary is never pointed at the user's registry.
func TestChallengeLegacyBinaryFence(t *testing.T) {
	binary := os.Getenv("CONDUCTOR_LEGACY_BINARY")
	if binary == "" {
		t.Skip("set CONDUCTOR_LEGACY_BINARY to a schema-3 CLI for the adoption compatibility probe")
	}
	home := t.TempDir()
	p := filepath.Join(home, "conductor.db")
	s, e := Open(p, true)
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Close()
	for _, args := range [][]string{{"team", "list"}, {"agent", "register", "--name", "blocked", "--request", "blocked"}} {
		command := append([]string{"--home", home, "--project", "fixture"}, args...)
		command = append(command, "--json")
		out, e := exec.Command(binary, command...).CombinedOutput()
		if e == nil || !strings.Contains(string(out), "unsupported schema version 4") {
			t.Fatalf("legacy access not rejected: %v %s", e, out)
		}
		t.Logf("legacy schema-3 access rejected schema 4: %s", out)
	}
}
