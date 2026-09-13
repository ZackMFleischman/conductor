package portalapi

import (
	"bufio"
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ZackMFleischman/conductor/internal/store"
)

func TestEventsObserveIndependentSQLiteWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conductor.db")
	writer, e := store.Open(path, true)
	if e != nil {
		t.Fatal(e)
	}
	defer writer.DB.Close()
	if _, e = writer.DB.Exec("INSERT INTO projects(id,common_dir,prefix) VALUES('p','fixture','APP')"); e != nil {
		t.Fatal(e)
	}
	reader, e := store.OpenReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	defer reader.DB.Close()
	source := &DBReader{DB: reader.DB}
	first, e := source.Board(context.Background(), "p")
	if e != nil {
		t.Fatal(e)
	}
	host := httptest.NewServer(NewHandler(source, Options{PollInterval: 5 * time.Millisecond, HeartbeatInterval: time.Hour}))
	defer host.Close()
	res, e := host.Client().Get(host.URL + "/api/v1/projects/p/events")
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("events status%d", res.StatusCode)
	}
	rd := bufio.NewReader(res.Body)
	frame(t, rd)
	if _, e = writer.DB.Exec("INSERT INTO tickets(id,project_id,display_key,title,body,created_at,updated_at) VALUES('t','p','APP-1','New task','From another connection','2026','2026')"); e != nil {
		t.Fatal(e)
	}
	f := frame(t, rd)
	if !strings.Contains(f, "board.changed") || strings.Contains(f, first.Revision) {
		t.Fatalf("writer change not notified: %s", f)
	}
	second, e := source.Board(context.Background(), "p")
	if e != nil {
		t.Fatal(e)
	}
	if len(second.Tickets) != 1 || second.Tickets[0].Title != "New task" {
		t.Fatalf("wrong board %+v", second)
	}
	// Progress events may not alter a currently displayed field, but still must refresh.
	if _, e = writer.DB.Exec("INSERT INTO events(id,project_id,ticket_id,actor_id,kind,body,created_at) VALUES('event','p','t','actor','ticket.note','Progress','2026')"); e != nil {
		t.Fatal(e)
	}
	f = frame(t, rd)
	if !strings.Contains(f, "board.changed") || strings.Contains(f, second.Revision) {
		t.Fatalf("event change not notified: %s", f)
	}
}
