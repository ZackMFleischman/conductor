package portalapi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func frame(t *testing.T, rd *bufio.Reader) string {
	t.Helper()
	result := make(chan string, 1)
	go func() {
		var b strings.Builder
		for {
			s, e := rd.ReadString('\n')
			if e != nil {
				result <- "ERROR: " + e.Error()
				return
			}
			b.WriteString(s)
			if s == "\n" {
				result <- b.String()
				return
			}
		}
	}()
	select {
	case s := <-result:
		return s
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE frame")
		return ""
	}
}
func TestEventsInitialChangeReconnectAndCancellation(t *testing.T) {
	r := &testReader{board: testBoard()}
	host := httptest.NewServer(NewHandler(r, Options{PollInterval: 5 * time.Millisecond, HeartbeatInterval: time.Hour, MaxStreams: 1}))
	defer host.Close()
	connect := func(last string) *http.Response {
		t.Helper()
		q, _ := http.NewRequest("GET", host.URL+"/api/v1/projects/p/events", nil)
		q.Header.Set("Last-Event-ID", last)
		res, e := host.Client().Do(q)
		if e != nil {
			t.Fatal(e)
		}
		return res
	}
	res := connect("")
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("events status %d", res.StatusCode)
	}
	rd := bufio.NewReader(res.Body)
	if f := frame(t, rd); !strings.Contains(f, "event: board.changed") || !strings.Contains(f, `"revision":"one"`) {
		t.Fatal(f)
	}
	excess := connect("")
	excess.Body.Close()
	if excess.StatusCode != 503 {
		t.Fatalf("stream limit status%d", excess.StatusCode)
	}
	r.mu.Lock()
	r.board.Revision = "two"
	r.mu.Unlock()
	if f := frame(t, rd); !strings.Contains(f, `"revision":"two"`) {
		t.Fatal(f)
	}
	res.Body.Close()
	var again *http.Response
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		again = connect("two")
		if again.StatusCode == 200 {
			break
		}
		again.Body.Close()
		time.Sleep(time.Millisecond)
	}
	if again == nil || again.StatusCode != 200 {
		t.Fatal("cancellation did not release stream capacity")
	}
	defer again.Body.Close()
	if f := frame(t, bufio.NewReader(again.Body)); !strings.Contains(f, `"revision":"two"`) {
		t.Fatalf("reconnect must invalidate even matching ID: %s", f)
	}
}
func TestEventsHeartbeatAndSafeFailure(t *testing.T) {
	r := &testReader{board: testBoard()}
	host := httptest.NewServer(NewHandler(r, Options{PollInterval: 5 * time.Millisecond, HeartbeatInterval: 10 * time.Millisecond}))
	defer host.Close()
	res, e := host.Client().Get(host.URL + "/api/v1/projects/p/events")
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	rd := bufio.NewReader(res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("status%d", res.StatusCode)
	}
	frame(t, rd)
	if f := frame(t, rd); !strings.HasPrefix(f, ":") {
		t.Fatalf("expected heartbeat %q", f)
	}
	r.mu.Lock()
	r.err = errors.New("secret database path and SQL")
	r.mu.Unlock()
	for i := 0; i < 10; i++ {
		f := frame(t, rd)
		if strings.Contains(f, "secret") {
			t.Fatal("private error leaked")
		}
		if strings.Contains(f, "board.error") {
			return
		}
	}
	t.Fatal("no failure event")
}

func TestReadErrorsDoNotLeakDetails(t *testing.T) {
	h := NewHandler(&testReader{err: fmt.Errorf("private path: %w", ErrInvalidData)}, Options{})
	q := httptest.NewRequest("GET", "/api/v1/projects/p/board", nil)
	q.Host = "localhost"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, q)
	if w.Code != 503 || strings.Contains(w.Body.String(), "private") {
		t.Fatalf("unsafe failure %s", w.Body.String())
	}
}

// A canceled query must not keep a stream alive after its browser disconnects.
type blockingReader struct{}

func (blockingReader) Projects(context.Context) ([]Project, error) { return nil, nil }
func (blockingReader) Board(ctx context.Context, _ string) (Board, error) {
	<-ctx.Done()
	return Board{}, ctx.Err()
}
func TestSnapshotReadTimeout(t *testing.T) {
	host := httptest.NewServer(NewHandler(blockingReader{}, Options{ReadTimeout: 10 * time.Millisecond}))
	defer host.Close()
	res, e := host.Client().Get(host.URL + "/api/v1/projects/p/events")
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	if res.StatusCode != 503 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestUnfinishedGETBodyCannotOccupyStream(t *testing.T) {
	host := httptest.NewServer(NewHandler(&testReader{board: testBoard()}, Options{MaxStreams: 1}))
	defer host.Close()
	conn, e := net.Dial("tcp", strings.TrimPrefix(host.URL, "http://"))
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Second))
	fmt.Fprintf(conn, "GET /api/v1/projects/p/events HTTP/1.1\r\nHost: %s\r\nContent-Length: 1\r\n\r\n", strings.TrimPrefix(host.URL, "http://"))
	res, e := http.ReadResponse(bufio.NewReader(conn), nil)
	if e != nil {
		t.Fatalf("unfinished body blocked response: %v", e)
	}
	res.Body.Close()
	if res.StatusCode != 400 || !res.Close {
		t.Fatalf("must reject and close body-bearing GET: %+v", res)
	}
	next, e := host.Client().Get(host.URL + "/api/v1/projects/p/events")
	if e != nil {
		t.Fatal(e)
	}
	defer next.Body.Close()
	if next.StatusCode != 200 {
		t.Fatalf("stream slot leaked: %d", next.StatusCode)
	}
}
