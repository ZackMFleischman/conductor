package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupCLI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("USERPROFILE", root)
	t.Setenv("HOME", root)
	t.Setenv("CODEX_HOME", filepath.Join(root, "custom codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(root, "custom claude"))
	home := filepath.Join(root, "data store")
	var out bytes.Buffer
	run := func(args ...string) map[string]any {
		t.Helper()
		out.Reset()
		if code := Run(context.Background(), Env{CWD: root, Home: home, Out: &out}, args); code != 0 {
			t.Fatalf("exit %d: %s", code, &out)
		}
		var result map[string]any
		if e := json.Unmarshal(out.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		return result
	}
	run("setup", "--agents", "codex,claude", "--json")
	if _, e := os.Stat(filepath.Join(root, "custom codex", "AGENTS.md")); !os.IsNotExist(e) {
		t.Fatal("preview wrote instructions")
	}
	run("setup", "--agents", "codex,claude", "--apply", "--json")
	b, e := os.ReadFile(filepath.Join(root, ".agents", "skills", "conductor-work", "SKILL.md"))
	if e != nil || !bytes.Contains(b, []byte("name: conductor-work")) {
		t.Fatalf("packaged skill missing: %v", e)
	}
	if _, e = os.Stat(filepath.Join(home, "conductor.db")); !os.IsNotExist(e) {
		t.Fatal("setup initialized tracker")
	}
	run("setup", "--remove", "--agents", "codex,claude", "--apply", "--json")
	if _, e = os.Stat(filepath.Join(root, ".agents", "skills", "conductor-work", "SKILL.md")); !os.IsNotExist(e) {
		t.Fatal("owned skill remains")
	}
}
func TestSetupRejectsUsage(t *testing.T) {
	for _, args := range [][]string{{"setup"}, {"setup", "--agents", "other"}, {"setup", "--agents", "codex", "extra"}, {"setup", "--agents", "codex", "--unknown"}, {"setup", "--agents", "codex", "--project", "x"}} {
		var out bytes.Buffer
		if code := Run(context.Background(), Env{CWD: t.TempDir(), Home: t.TempDir(), Out: &out}, args); code != 2 {
			t.Fatalf("%v: code %d: %s", args, code, &out)
		}
	}
}
