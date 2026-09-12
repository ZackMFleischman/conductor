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

func TestProblemAddAcceptsTicketDisplayKeyAndUUIDWithoutChangingRevision(t *testing.T) {
	c := testkit.New(t)
	project := testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "init"))
	projectID := testkit.String(t, project, "project_id")
	testkit.MustData(t, c.Run("agent", "register", "--name", "dev", "--request", "agent"))
	session := testkit.MustData(t, c.Run("session", "start", "--agent", "dev", "--request", "session"))
	sessionID := testkit.String(t, session, "session_id")
	ticketID := "11111111-1111-4111-8111-111111111111"
	db, err := sql.Open("sqlite", filepath.Join(c.Home, "conductor.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := "2026-09-12T00:00:00Z"
	if _, err = db.Exec("INSERT INTO tickets(id,project_id,display_key,title,body,revision,created_at,updated_at) VALUES(?,?,?,?,?,7,?,?)", ticketID, projectID, "APP-1", "ticket", "body", now, now); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()
	bodyPath := filepath.Join(t.TempDir(), "problem.json")
	if err = os.WriteFile(bodyPath, []byte(`{"summary":"Observed issue","expected":"Expected","actual":"Actual"}`), 0600); err != nil {
		t.Fatal(err)
	}
	byKey := testkit.MustData(t, c.Run("problem", "add", "--ticket", "APP-1", "--session", sessionID, "--body-file", bodyPath, "--request", "by-key"))
	if byKey["ticket_id"] != ticketID {
		t.Fatalf("display key was not resolved to UUID: %v", byKey)
	}
	byID := testkit.MustData(t, c.Run("problem", "add", "--ticket", ticketID, "--session", sessionID, "--body-file", bodyPath, "--request", "by-id"))
	if byID["ticket_id"] != ticketID {
		t.Fatalf("UUID selector changed: %v", byID)
	}
	db, err = sql.Open("sqlite", filepath.Join(c.Home, "conductor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var revision int
	if err = db.QueryRow("SELECT revision FROM tickets WHERE project_id=? AND id=?", projectID, ticketID).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 7 {
		t.Fatalf("problem reports changed ticket revision: %d", revision)
	}
}
