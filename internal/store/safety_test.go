package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadonlyDoesNotChangeDatabase(t *testing.T) {
	p := filepath.Join(t.TempDir(), "db")
	s, e := Open(p, true)
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Close()
	before, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	s, e = OpenReadOnly(p)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec("INSERT INTO projects(id,common_dir,prefix) VALUES('a','b','C')"); e == nil {
		t.Fatal("readonly write succeeded")
	}
	s.DB.Close()
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Fatal("discovery modified database")
	}
}
func TestPanicRollsBack(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer s.DB.Close()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("missing panic")
			}
		}()
		s.Write(context.Background(), Request{ID: "a", Operation: "op", ProjectID: "p", ActorID: "a", Payload: json.RawMessage(`{}`)}, func(c *sql.Conn) (json.RawMessage, error) {
			c.ExecContext(context.Background(), "INSERT INTO projects(id,common_dir,prefix) VALUES('a','b','C')")
			panic("injected")
		})
	}()
	var n int
	if e = s.DB.QueryRow("SELECT count(*) FROM projects").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
