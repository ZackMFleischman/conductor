package cli_test

import (
	"database/sql"
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"path/filepath"
	"testing"
)

func TestAgentListPreservesJournalMode(t *testing.T) {
	c := testkit.New(t)
	testkit.MustData(t, c.Run("init", "--prefix", "APP", "--request", "i"))
	path := filepath.Join(c.Home, "conductor.db")
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	var mode string
	if e = db.QueryRow("PRAGMA journal_mode=DELETE").Scan(&mode); e != nil {
		t.Fatal(e)
	}
	db.Close()
	testkit.MustData(t, c.Run("agent", "list"))
	db, e = sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.QueryRow("PRAGMA journal_mode").Scan(&mode); e != nil {
		t.Fatal(e)
	}
	if mode != "delete" {
		t.Fatalf("agent list changed journal mode to %s", mode)
	}
}
