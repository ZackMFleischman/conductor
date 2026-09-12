package cli_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/testkit"
)

func TestProblemReplayPreservesOriginal(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	session := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "session"))
	sessionID := testkit.String(t, session, "session_id")
	bodyPath := filepath.Join(t.TempDir(), "problem.json")
	if err := os.WriteFile(bodyPath, []byte(`{"summary":"Wrong worktree","expected":"Edit the assigned checkout","actual":"Edited the primary checkout","correction":"Moved the patch to the assigned worktree"}`), 0600); err != nil {
		t.Fatal(err)
	}

	args := []string{"problem", "add", "--session", sessionID, "--body-file", bodyPath, "--request", "problem-1"}
	first := testkit.MustData(t, c.Run(args...))
	retry := testkit.MustData(t, c.Run(args...))
	if first["id"] != retry["id"] {
		t.Fatalf("retry duplicated report: %v %v", first, retry)
	}
	correctionPath := filepath.Join(t.TempDir(), "correction.md")
	if err := os.WriteFile(correctionPath, []byte("Correction: moved the patch and reran tests."), 0600); err != nil {
		t.Fatal(err)
	}
	testkit.MustData(t, c.Run("problem", "append", testkit.String(t, first, "id"),
		"--session", sessionID, "--body-file", correctionPath, "--request", "correction-1"))
	listed := testkit.MustData(t, c.Run("problem", "list"))
	items := listed["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one report: %v", items)
	}
	report := items[0].(map[string]any)
	if report["actual"] != "Edited the primary checkout" {
		t.Fatalf("original overwritten: %v", report)
	}
	if len(report["notes"].([]any)) != 1 {
		t.Fatalf("missing correction: %v", report)
	}
}

func TestProblemInputValidationAndReadOnlyList(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	session := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "session"))
	sessionID := testkit.String(t, session, "session_id")
	dir := t.TempDir()
	bad := []struct {
		name string
		body string
	}{
		{"unknown", `{"summary":"s","expected":"e","actual":"a","diagnosis":"guess"}`},
		{"missing", `{"summary":"s","actual":"a"}`},
		{"trailing", `{"summary":"s","expected":"e","actual":"a"} {}`},
		{"long-summary", `{"summary":"` + strings.Repeat("s", 4097) + `","expected":"e","actual":"a"}`},
		{"long-text", `{"summary":"s","expected":"` + strings.Repeat("e", 65537) + `","actual":"a"}`},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".json")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			result := c.Run("problem", "add", "--session", sessionID, "--body-file", path, "--request", tc.name)
			if result["ok"] != false || result["error"].(map[string]any)["code"] != "INVALID_INPUT" {
				t.Fatalf("accepted invalid body: %v", result)
			}
		})
	}
	oversized := filepath.Join(dir, "oversized.json")
	if err := os.WriteFile(oversized, []byte(strings.Repeat(" ", 256*1024+1)), 0600); err != nil {
		t.Fatal(err)
	}
	result := c.Run("problem", "add", "--session", sessionID, "--body-file", oversized, "--request", "oversized")
	if result["ok"] != false || result["error"].(map[string]any)["code"] != "INVALID_INPUT" {
		t.Fatalf("accepted oversized body: %v", result)
	}

	db, err := sql.Open("sqlite", filepath.Join(c.Home, "conductor.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if got := testkit.MustData(t, c.Run("problem", "list")); len(got["items"].([]any)) != 0 {
		t.Fatal(got)
	}
	db, err = sql.Open("sqlite", filepath.Join(c.Home, "conductor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var mode string
	if err = db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "delete" {
		t.Fatalf("list mutated journal mode: %q %v", mode, err)
	}
}
