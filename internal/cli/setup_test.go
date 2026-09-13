package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
	for _, name := range []string{"conductor-work", "conductor-plan", "conductor-worker", "conductor-orchestrator", "conductor-retrospective", "conductor-workflow-improver"} {
		for _, skillRoot := range []string{filepath.Join(root, ".agents", "skills"), filepath.Join(root, "custom claude", "skills")} {
			if _, err := os.Stat(filepath.Join(skillRoot, name, "SKILL.md")); err != nil {
				t.Fatalf("missing installed skill %s: %v", name, err)
			}
		}
	}
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
func TestSetupOutputDoesNotExposePersonalConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("USERPROFILE", root)
	t.Setenv("HOME", root)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(root, "claude"))
	p := filepath.Join(root, "claude", "settings.json")
	os.MkdirAll(filepath.Dir(p), 0700)
	secret := "fixture-private-token-never-print"
	os.WriteFile(p, []byte(`{"env":{"TOKEN":"`+secret+`"}}`), 0600)
	for _, apply := range []bool{false, true} {
		var out bytes.Buffer
		args := []string{"setup", "--agents", "claude", "--json"}
		if apply {
			args = append(args, "--apply")
		}
		if code := Run(context.Background(), Env{CWD: root, Home: filepath.Join(root, "data"), Out: &out}, args); code != 0 {
			t.Fatal(out.String())
		}
		var result map[string]any
		if e := json.Unmarshal(out.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		edits := result["data"].(map[string]any)["edits"].([]any)
		for _, raw := range edits {
			e := raw.(map[string]any)
			for _, forbidden := range []string{"before", "after", "records", "original", "desired"} {
				if _, ok := e[forbidden]; ok {
					t.Fatalf("preview exposes private %s bytes", forbidden)
				}
			}
			if e["description"] == nil {
				t.Fatal("missing reviewable owned-change description")
			}
		}
		if bytes.Contains(out.Bytes(), []byte(secret)) || bytes.Contains(out.Bytes(), []byte(base64.StdEncoding.EncodeToString([]byte(secret)))) {
			t.Fatal("printed secret")
		}
	}
}

func TestSetupReportsWindowsApprovalWithoutExposingConfig(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows compatibility")
	}
	root := t.TempDir()
	t.Setenv("USERPROFILE", root)
	t.Setenv("HOME", root)
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	p := filepath.Join(root, "codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	before := []byte("private_token = 'never-show-this-token'\n[windows]\nsandbox = 'unelevated'\n")
	if err := os.WriteFile(p, before, 0600); err != nil {
		t.Fatal(err)
	}
	for _, apply := range []bool{false, true, true} {
		var out bytes.Buffer
		args := []string{"setup", "--agents", "codex", "--json"}
		if apply {
			args = append(args, "--apply")
		}
		if code := Run(context.Background(), Env{CWD: root, Home: filepath.Join(root, "data"), Out: &out}, args); code != 0 {
			t.Fatal(out.String())
		}
		var result map[string]any
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		data := result["data"].(map[string]any)
		access, ok := data["access"].(map[string]any)
		if !ok {
			t.Fatal("missing per-host access result")
		}
		codex := access["codex"].(map[string]any)
		if codex["mode"] != "command_approval" || codex["root_configured"] != false || codex["verified"] != false || codex["warning"] == "" {
			t.Fatalf("misleading access result: %+v", codex)
		}
		if bytes.Contains(out.Bytes(), []byte("never-show-this-token")) {
			t.Fatal("exposed personal config")
		}
		after, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("changed config: %v", err)
		}
	}
}
