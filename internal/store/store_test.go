package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestWriteRollbackReplayAndConflict(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "conductor.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	r := Request{ID: "r", Operation: "test", ProjectID: "scope", ActorID: "local-user", Payload: json.RawMessage(`{"a":1}`)}
	calls := 0
	fn := func(c *sql.Conn) (json.RawMessage, error) {
		calls++
		_, e := c.ExecContext(ctx, "INSERT INTO projects(id,common_dir,prefix) VALUES('p','dir','P')")
		if e != nil {
			return nil, e
		}
		return nil, errors.New("injected")
	}
	if _, err = s.Write(ctx, r, fn); err == nil {
		t.Fatal("wanted rollback")
	}
	var n int
	s.DB.QueryRow("SELECT count(*) FROM projects").Scan(&n)
	if n != 0 {
		t.Fatal("not rolled back")
	}
	fn = func(c *sql.Conn) (json.RawMessage, error) { calls++; return json.RawMessage(`{"saved":true}`), nil }
	a, err := s.Write(ctx, r, fn)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Write(ctx, r, fn)
	if err != nil || string(a) != string(b) || calls != 2 {
		t.Fatalf("replay %s %s %v %d", a, b, err, calls)
	}
	r.Payload = json.RawMessage(`{"a":2}`)
	if _, err = s.Write(ctx, r, fn); err == nil {
		t.Fatal("expected conflict")
	}
}
