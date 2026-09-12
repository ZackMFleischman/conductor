package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/store"
	"github.com/ZackMFleischman/conductor/internal/testkit"
)

var buildOnce sync.Once
var binary string

func fixture(t *testing.T) (testkit.CLI, testkit.CLI, testkit.CLI) {
	t.Helper()
	buildOnce.Do(func() { testkit.Build(t, "../..", binary) })
	// The suite owns the binary; fixture cleanup cannot remove it.
	c := testkit.New(t)
	c.Binary = binary
	git(t, c.CWD, "-c", "user.name=Integration", "-c", "user.email=integration@example.invalid", "commit", "--allow-empty", "-qm", "fixture")
	a, b := c, c
	a.CWD = filepath.Join(t.TempDir(), "a")
	b.CWD = filepath.Join(t.TempDir(), "b")
	git(t, c.CWD, "worktree", "add", "-qb", "a", a.CWD)
	git(t, c.CWD, "worktree", "add", "-qb", "b", b.CWD)
	ok(t, c, "init", "--prefix", "IT", "--request", "init")
	return c, a, b
}
func TestMain(m *testing.M) {
	// Lifetime spans the suite, including repeated fixtures and subprocesses.
	dir, e := os.MkdirTemp("", "conductor-integration-")
	if e != nil {
		panic(e)
	}
	binary = filepath.Join(dir, "conductor.exe")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, e := c.CombinedOutput(); e != nil {
		t.Fatalf("git %v: %s %v", args, out, e)
	}
}
func ok(t *testing.T, c testkit.CLI, args ...string) map[string]any {
	t.Helper()
	v, code := c.Process(args...)
	if code != 0 {
		t.Fatalf("%v: exit %d: %v", args, code, v)
	}
	return testkit.MustData(t, v)
}
func conflict(t *testing.T, c testkit.CLI, args ...string) {
	t.Helper()
	v, code := c.Process(args...)
	if code != 3 || v["ok"] != false || v["error"].(map[string]any)["code"] == "" {
		t.Fatalf("expected structured conflict: %d %v", code, v)
	}
}
func file(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "body")
	if e := os.WriteFile(p, []byte(body), 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func session(t *testing.T, c testkit.CLI, name, key string) string {
	t.Helper()
	return testkit.String(t, ok(t, c, "session", "start", "--agent", name, "--request", key), "session_id")
}
func count(t *testing.T, c testkit.CLI, q string, args ...any) int {
	t.Helper()
	s, e := store.OpenReadOnly(filepath.Join(c.Home, "conductor.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.DB.Close()
	var n int
	if e = s.DB.QueryRow(q, args...).Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}
func TestWorkflowAndSessionHonesty(t *testing.T) {
	c, a, b := fixture(t)
	ok(t, c, "agent", "register", "--name", "worker", "--request", "agent")
	s1, s2 := session(t, a, "worker", "s1"), session(t, b, "worker", "s2")
	if s1 == s2 {
		t.Fatal("sessions reused")
	}
	body := file(t, "implementation and verification")
	ticket := ok(t, c, "ticket", "create", "--title", "workflow", "--body-file", body, "--request", "create")
	id := testkit.String(t, ticket, "id")
	claim := ok(t, a, "ticket", "claim", id, "--session", s1, "--expect-revision", "1", "--request", "claim1")
	cl := testkit.String(t, claim, "claim_id")
	conflict(t, a, "session", "stop", "--session", s1, "--request", "stop-busy")
	ok(t, c, "ticket", "release", id, "--human", "--expect-revision", "2", "--reason", "recover abandoned work", "--request", "recover")
	second := ok(t, b, "ticket", "claim", id, "--session", s2, "--expect-revision", "3", "--request", "claim2")
	cl2 := testkit.String(t, second, "claim_id")
	if cl == cl2 {
		t.Fatal("claim fencing ID reused")
	}
	conflict(t, a, "ticket", "note", id, "--session", s1, "--claim", cl, "--body-file", body, "--request", "stale")
	conflict(t, a, "ticket", "submit", id, "--session", s1, "--claim", cl, "--expect-revision", "4", "--summary-file", body, "--evidence-file", body, "--qa-file", body, "--request", "stale-submit")
	conflict(t, a, "ticket", "release", id, "--session", s1, "--claim", cl, "--expect-revision", "4", "--reason", "old owner", "--request", "stale-release")
	submit := func(rev, key, claim string) {
		ok(t, b, "ticket", "submit", id, "--session", s2, "--claim", claim, "--expect-revision", rev, "--summary-file", body, "--evidence-file", body, "--qa-file", body, "--request", key)
	}
	submit("4", "submit1", cl2)
	ok(t, c, "ticket", "reject", id, "--human", "--expect-revision", "5", "--reason", "QA uncovered defect", "--request", "reject")
	third := ok(t, b, "ticket", "claim", id, "--session", s2, "--expect-revision", "6", "--request", "claim3")
	submit("7", "submit2", testkit.String(t, third, "claim_id"))
	done := ok(t, c, "ticket", "accept", id, "--human", "--expect-revision", "8", "--request", "accept")
	if done["state"] != "done" || testkit.Number(t, done, "revision") != 9 {
		t.Fatal(done)
	}
	idle := ok(t, a, "session", "idle", "--session", s1, "--request", "idle")
	stopped := ok(t, b, "session", "stop", "--session", s2, "--request", "stop")
	for _, v := range []map[string]any{idle, stopped} {
		if v["busy"] != false || v["process_liveness"] != "unknown" || v["last_seen_at"] == "" {
			t.Fatal(v)
		}
	}
	before := ok(t, c, "agent", "list")
	after := ok(t, c, "agent", "list")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read changed session contact", before, after)
	}
	clone := c
	clone.CWD = filepath.Join(t.TempDir(), "clone")
	git(t, c.CWD, "clone", "-q", c.CWD, clone.CWD)
	if v := ok(t, clone, "context", "--if-registered"); v["registered"] != false {
		t.Fatal(v)
	}
}

func TestPackagedSetupRoundTrip(t *testing.T) {
	c, _, _ := fixture(t)
	root := t.TempDir()
	codex := filepath.Join(root, "codex")
	claude := filepath.Join(root, "claude")
	// os.UserHomeDir uses USERPROFILE on Windows and HOME on Unix.
	t.Setenv("USERPROFILE", root)
	t.Setenv("HOME", root)
	t.Setenv("CODEX_HOME", codex)
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	originals := map[string]string{
		filepath.Join(codex, "AGENTS.md"):      "Keep the existing project instructions.\n",
		filepath.Join(codex, "config.toml"):    "# preserve this comment\nmodel = \"fixture-model\"\n",
		filepath.Join(claude, "CLAUDE.md"):     "Keep the existing Claude instructions.\n",
		filepath.Join(claude, "settings.json"): `{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"echo existing"}]}]}}`,
	}
	for p, body := range originals {
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
	}
	assertOriginals := func() {
		t.Helper()
		for p, want := range originals {
			got, e := os.ReadFile(p)
			if e != nil || string(got) != want {
				t.Fatalf("existing file changed: %s %q %v", p, got, e)
			}
		}
	}
	preview := ok(t, c, "setup", "--agents", "codex,claude", "--json")
	if preview["applied"] != false {
		t.Fatal(preview)
	}
	assertOriginals()
	applied := ok(t, c, "setup", "--agents", "codex,claude", "--apply", "--json")
	if applied["access_verified"] != false {
		t.Fatal("config claimed effective host access", applied)
	}
	paths := []string{filepath.Join(root, ".agents", "skills", "conductor-work", "SKILL.md"), filepath.Join(claude, "skills", "conductor-work", "SKILL.md")}
	for _, p := range paths {
		if b, e := os.ReadFile(p); e != nil || len(b) == 0 {
			t.Fatalf("packaged skill missing: %s %v", p, e)
		}
	}
	again := ok(t, c, "setup", "--agents", "codex,claude", "--apply")
	if len(again["edits"].([]any)) != 0 {
		t.Fatal("repeat apply not idempotent", again)
	}
	ok(t, c, "setup", "--remove", "--agents", "codex,claude", "--apply")
	assertOriginals()
	for _, p := range paths {
		if _, e := os.Stat(p); !os.IsNotExist(e) {
			t.Fatalf("owned skill remains: %s %v", p, e)
		}
	}
	if n := count(t, c, "SELECT count(*) FROM projects"); n != 1 {
		t.Fatal("setup removed registry", n)
	}
}
