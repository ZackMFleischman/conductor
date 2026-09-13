package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

func TestLegacyMigrationPreservesData(t *testing.T) {
	p := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(schema + `PRAGMA user_version=1; INSERT INTO projects(id,common_dir,prefix) VALUES('legacy','/repo/.git','OLD');`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	ro, err := OpenReadOnly(p)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	ro.DB.QueryRow("PRAGMA user_version").Scan(&version)
	ro.DB.Close()
	if version != 1 {
		t.Fatal("read-only discovery migrated the database")
	}
	s, err := Open(p, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	if err = s.DB.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != SchemaVersion {
		t.Fatalf("version %d: %v", version, err)
	}
	var prefix string
	if err = s.DB.QueryRow("SELECT prefix FROM projects WHERE id='legacy'").Scan(&prefix); err != nil || prefix != "OLD" {
		t.Fatalf("lost legacy project: %q %v", prefix, err)
	}
	backups, _ := filepath.Glob(p + fmt.Sprintf(".pre-v%d-*", SchemaVersion))
	if len(backups) != 1 {
		t.Fatalf("expected consistent pre-migration backup, got %v", backups)
	}
}
