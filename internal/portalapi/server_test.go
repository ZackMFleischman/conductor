package portalapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type testReader struct {
	mu    sync.Mutex
	board Board
	err   error
}

func (r *testReader) Projects(context.Context) ([]Project, error) {
	return []Project{{ID: "p", Name: "CON"}}, r.err
}
func (r *testReader) Board(_ context.Context, id string) (Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id != "p" {
		return Board{}, ErrNotFound
	}
	return r.board, r.err
}
func testBoard() Board {
	return Board{Project: Project{ID: "p", Name: "CON"}, Tickets: []Ticket{}, Revision: "one"}
}

func TestReadRoutesAndLocalBoundary(t *testing.T) {
	h := NewHandler(&testReader{board: testBoard()}, Options{})
	for _, c := range []struct {
		method, path, host, origin string
		want                       int
	}{
		{"GET", "/api/v1/projects", "localhost:7331", "", 200},
		{"GET", "/api/v1/projects/p/board", "localhost:7331", "http://localhost:7331", 200},
		{"GET", "/api/v1/projects/missing/board", "localhost:7331", "", 404},
		{"POST", "/api/v1/projects/p/board", "localhost:7331", "", 405},
		{"GET", "/api/v1/projects/p/board", "evil.example", "", 403},
		{"GET", "/api/v1/projects/p/board", "localhost:7331", "https://evil.example", 403},
		{"GET", "/api/v1/missing", "localhost:7331", "", 404},
	} {
		t.Run(c.method+c.path+c.host+c.origin, func(t *testing.T) {
			q := httptest.NewRequest(c.method, c.path, nil)
			q.Host = c.host
			q.Header.Set("Origin", c.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, q)
			if w.Code != c.want {
				t.Fatalf("got %d body %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "<html") {
				t.Fatal("API returned HTML")
			}
		})
	}
	q := httptest.NewRequest("GET", "/api/v1/projects/p/board", nil)
	q.Host = "localhost"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, q)
	var b Board
	if e := json.Unmarshal(w.Body.Bytes(), &b); e != nil {
		t.Fatal(e)
	}
	if b.Revision != "one" || b.Tickets == nil || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("invalid snapshot: %s", w.Body.String())
	}
}

func TestMutationsWithBodiesRemainMethodNotAllowed(t *testing.T) {
	h := NewHandler(&testReader{board: testBoard()}, Options{})
	q := httptest.NewRequest("POST", "/api/v1/projects/p/board", strings.NewReader(`{"title":"change"}`))
	q.Host = "localhost"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, q)
	if w.Code != 405 || w.Header().Get("Connection") != "close" {
		t.Fatalf("expected405 close, got%d %v", w.Code, w.Header())
	}
}
