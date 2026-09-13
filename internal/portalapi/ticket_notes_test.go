package portalapi

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestTicketNotesPaginationAndIsolation(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.project(t, "other", "O")
	f.ticket(t, "ticket", "p", "ready")
	f.ticket(t, "second", "p", "ready")
	f.ticket(t, "foreign", "other", "ready")
	boardExec(t, f.writer, `INSERT INTO agents(id,project_id,name) VALUES('agent','p','Reporter')`)
	boardExec(t, f.writer, `INSERT INTO sessions(id,project_id,agent_id,last_seen_at,worktree_root) VALUES('session','p','agent','2026-01-01','root')`)
	for i := 1; i <= 25; i++ {
		boardExec(t, f.writer, `INSERT INTO events(id,project_id,ticket_id,actor_id,kind,body,created_at) VALUES(?,'p','ticket','session','ticket.note',?,'2026-01-01')`, fmt.Sprint(i), fmt.Sprintf("Note %d", i))
	}
	boardExec(t, f.writer, `INSERT INTO events(id,project_id,ticket_id,actor_id,kind,body,created_at) VALUES('state','p','ticket','session','ticket.submit','Not a comment','2026-01-01'),('unrelated','p','second','session','ticket.note','Other ticket','2026-01-01'),('foreign','other','foreign','unknown','ticket.note','Other project','2026-01-01')`)
	first, err := f.reader.TicketNotes(context.Background(), "p", "ticket", "")
	if err != nil || first.Total != 25 || len(first.Notes) != 20 || first.Notes[0].Body != "Note 25" || first.Notes[0].Author != "Reporter" || first.NextCursor == "" {
		t.Fatalf("first: %#v %v", first, err)
	}
	// Appends between pages must not shift or duplicate the older-page boundary.
	boardExec(t, f.writer, `INSERT INTO events(id,project_id,ticket_id,actor_id,kind,body,created_at) VALUES('new','p','ticket','unknown','ticket.note','New note','2026-01-01')`)
	second, err := f.reader.TicketNotes(context.Background(), "p", "ticket", first.NextCursor)
	if err != nil || len(second.Notes) != 5 || second.Notes[0].Body != "Note 5" || second.Notes[4].Body != "Note 1" || second.NextCursor != "" {
		t.Fatalf("second: %#v %v", second, err)
	}
	newest, err := f.reader.TicketNotes(context.Background(), "p", "ticket", "")
	if err != nil || newest.Notes[0].Author != "Unknown author" {
		t.Fatalf("fallback: %#v %v", newest, err)
	}
	empty, err := f.reader.TicketNotes(context.Background(), "p", "second", "")
	if err != nil || empty.Total != 1 {
		t.Fatalf("isolated: %#v %v", empty, err)
	}
	for _, ids := range [][2]string{{"p", "foreign"}, {"other", "ticket"}, {"missing", "ticket"}} {
		if _, err := f.reader.TicketNotes(context.Background(), ids[0], ids[1], ""); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%v: %v", ids, err)
		}
	}
	if _, err := f.reader.TicketNotes(context.Background(), "p", "ticket", "invalid"); !errors.Is(err, ErrInvalidNotesCursor) {
		t.Fatal(err)
	}
}

func TestTicketNotesHTTP(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.ticket(t, "t", "p", "ready")
	handler := NewHandler(f.reader, Options{})
	for _, tc := range []struct {
		path   string
		status int
	}{{"/api/v1/projects/p/tickets/t/notes", 200}, {"/api/v1/projects/p/tickets/t/notes?before=bad", 400}, {"/api/v1/projects/p/tickets/absent/notes", 404}, {"/api/v1/projects/absent/tickets/t/notes", 404}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost"+tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
}
