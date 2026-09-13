package portalapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestProblemReportsInBoardAndDetails(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "CON")
	f.project(t, "other", "OTHER")
	f.ticket(t, "t", "p", "ready")
	boardExec(t, f.writer, "INSERT INTO agents(id,project_id,name) VALUES('a','p','observer')")
	boardExec(t, f.writer, "INSERT INTO sessions(id,project_id,agent_id,declared_state,last_seen_at,worktree_root) VALUES('s','p','a','active','2026-01-01','root')")
	boardExec(t, f.writer, "INSERT INTO problems(id,project_id,display_key,ticket_id,session_id,summary,expected,actual,correction,evidence,created_at) VALUES('report','p','CON-P47','t','s','Workflow issue','Expected behavior','Observed behavior','Proposed correction','Evidence text','2026-01-01')")
	before := readBoard(t, f.reader, "p")
	if len(before.Problems) != 1 || before.Problems[0].Key != "CON-P47" || before.Problems[0].TicketKey != "KEY-t" {
		t.Fatalf("problem index: %+v", before.Problems)
	}
	boardExec(t, f.writer, "INSERT INTO events(id,project_id,ticket_id,problem_id,actor_id,kind,body,payload,created_at) VALUES('note','p','t','report','s','problem.append','Follow-up evidence','{}','2026-01-02')")
	after := readBoard(t, f.reader, "p")
	if before.Revision == after.Revision || after.Problems[0].NoteCount != 1 {
		t.Fatal("report update missing from board revision")
	}
	detail, err := f.reader.Problem(context.Background(), "p", "report")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Expected != "Expected behavior" || detail.Actual != "Observed behavior" || len(detail.Notes) != 1 || detail.Notes[0].Body != "Follow-up evidence" || detail.Reporter != "observer" {
		t.Fatalf("details: %+v", detail)
	}
	if _, err = f.reader.Problem(context.Background(), "other", "report"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project access: %v", err)
	}
	if got := readBoard(t, f.reader, "other"); len(got.Problems) != 0 {
		t.Fatal("cross-project problem in board")
	}
	h := NewHandler(f.reader, Options{})
	for _, check := range []struct {
		path string
		want int
	}{{"/api/v1/projects/p/problems/report", 200}, {"/api/v1/projects/other/problems/report", 404}, {"/api/v1/projects/p/problems/missing", 404}} {
		request := httptest.NewRequest("GET", check.path, nil)
		request.Host = "localhost"
		response := httptest.NewRecorder()
		h.ServeHTTP(response, request)
		if response.Code != check.want {
			t.Fatalf("%s = %d", check.path, response.Code)
		}
	}
}
