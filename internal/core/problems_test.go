package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/gitctx"
	"github.com/ZackMFleischman/conductor/internal/store"
)

func problemFixture(t *testing.T) (context.Context, *store.Store, Service, Project, Session) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "conductor.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.DB.Close() })
	svc := Service{Store: st}
	g := gitctx.Context{CommonDir: "common", Root: "root"}
	raw, err := svc.Init(ctx, g, "APP", "init")
	if err != nil {
		t.Fatal(err)
	}
	var project Project
	if err = json.Unmarshal(raw, &project); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.RegisterAgent(ctx, project.ID, "dev", "", "", "agent"); err != nil {
		t.Fatal(err)
	}
	raw, err = svc.StartSession(ctx, project.ID, "dev", "session", g)
	if err != nil {
		t.Fatal(err)
	}
	var session Session
	if err = json.Unmarshal(raw, &session); err != nil {
		t.Fatal(err)
	}
	return ctx, st, svc, project, session
}

func decodeProblem(t *testing.T, raw json.RawMessage) ProblemReport {
	t.Helper()
	var report ProblemReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func TestProblemAddAppendReplayAndImmutability(t *testing.T) {
	ctx, st, svc, project, session := problemFixture(t)
	input := AddProblemInput{SessionID: session.ID, Summary: "Wrong worktree", Expected: "assigned", Actual: "primary", Correction: "moved"}
	raw, err := svc.AddProblem(ctx, project.ID, "problem-1", input)
	if err != nil {
		t.Fatal(err)
	}
	first := decodeProblem(t, raw)
	raw, err = svc.AddProblem(ctx, project.ID, "problem-1", input)
	if err != nil || decodeProblem(t, raw).ID != first.ID {
		t.Fatalf("replay changed report: %s %v", raw, err)
	}
	if _, err = svc.AppendProblem(ctx, project.ID, "note-1", AppendProblemInput{SessionID: session.ID, ProblemID: first.ID, Body: "correction note"}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AppendProblem(ctx, project.ID, "note-1", AppendProblemInput{SessionID: session.ID, ProblemID: first.ID, Body: "correction note"}); err != nil {
		t.Fatal(err)
	}
	page, err := svc.ListProblems(ctx, project.ID, ListProblemsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Actual != "primary" || len(page.Items[0].Notes) != 1 || page.Items[0].Notes[0].Body != "correction note" {
		t.Fatalf("unexpected report: %+v", page)
	}
	var initial, appendEvents int
	if err = st.DB.QueryRowContext(ctx, "SELECT count(*) FROM events WHERE problem_id=? AND kind='problem.add'", first.ID).Scan(&initial); err != nil {
		t.Fatal(err)
	}
	if err = st.DB.QueryRowContext(ctx, "SELECT count(*) FROM events WHERE problem_id=? AND kind='problem.append'", first.ID).Scan(&appendEvents); err != nil {
		t.Fatal(err)
	}
	if initial != 1 || appendEvents != 1 {
		t.Fatalf("replay duplicated events: add=%d append=%d", initial, appendEvents)
	}
}

func TestProblemRejectsStoppedAndCrossProjectReferences(t *testing.T) {
	ctx, st, svc, project, session := problemFixture(t)
	seed, err := svc.AddProblem(ctx, project.ID, "seed", AddProblemInput{SessionID: session.ID, Summary: "seed", Expected: "e", Actual: "a"})
	if err != nil {
		t.Fatal(err)
	}
	seedReport := decodeProblem(t, seed)
	raw, err := svc.StartSession(ctx, project.ID, "dev", "active-session", gitctx.Context{CommonDir: "common", Root: "active-root"})
	if err != nil {
		t.Fatal(err)
	}
	var active Session
	if err = json.Unmarshal(raw, &active); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SessionAction(ctx, project.ID, session.ID, "stop", "stop"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddProblem(ctx, project.ID, "stopped", AddProblemInput{SessionID: session.ID, Summary: "s", Expected: "e", Actual: "a"})
	assertProblemFault(t, err, "SESSION_STOPPED")
	_, err = svc.AppendProblem(ctx, project.ID, "stopped-append", AppendProblemInput{SessionID: session.ID, ProblemID: seedReport.ID, Body: "late note"})
	assertProblemFault(t, err, "SESSION_STOPPED")

	var otherProject, otherSession, otherTicket, otherProblem string
	_, err = st.Write(ctx, store.Request{ID: "other", Operation: "fixture", ProjectID: "fixture-scope", ActorID: "local-user", Payload: JSON(map[string]any{})}, func(c *sql.Conn) (json.RawMessage, error) {
		otherProject, otherSession, otherTicket, otherProblem = UUID(), UUID(), UUID(), UUID()
		now := Now()
		if _, e := c.ExecContext(ctx, "INSERT INTO projects(id,common_dir,prefix) VALUES(?,?,?)", otherProject, "other", "OTH"); e != nil {
			return nil, e
		}
		agent := UUID()
		if _, e := c.ExecContext(ctx, "INSERT INTO agents(id,project_id,name) VALUES(?,?,?)", agent, otherProject, "other"); e != nil {
			return nil, e
		}
		if _, e := c.ExecContext(ctx, "INSERT INTO sessions(id,project_id,agent_id,last_seen_at,worktree_root) VALUES(?,?,?,?,?)", otherSession, otherProject, agent, now, "other"); e != nil {
			return nil, e
		}
		if _, e := c.ExecContext(ctx, "INSERT INTO tickets(id,project_id,display_key,title,body,created_at,updated_at) VALUES(?,?,?,?,?,?,?)", otherTicket, otherProject, "OTH-1", "t", "b", now, now); e != nil {
			return nil, e
		}
		if _, e := c.ExecContext(ctx, "INSERT INTO problems(id,project_id,display_key,session_id,summary,expected,actual,created_at) VALUES(?,?,?,?,?,?,?,?)", otherProblem, otherProject, "OTH-P1", otherSession, "s", "e", "a", now); e != nil {
			return nil, e
		}
		return JSON(map[string]any{}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddProblem(ctx, project.ID, "wrong-session", AddProblemInput{SessionID: otherSession, Summary: "s", Expected: "e", Actual: "a"})
	assertProblemFault(t, err, "NOT_FOUND")
	_, err = svc.AddProblem(ctx, project.ID, "wrong-ticket", AddProblemInput{SessionID: active.ID, TicketID: otherTicket, Summary: "s", Expected: "e", Actual: "a"})
	assertProblemFault(t, err, "NOT_FOUND")
	_, err = svc.AppendProblem(ctx, project.ID, "wrong-problem", AppendProblemInput{SessionID: active.ID, ProblemID: otherProblem, Body: "cross-project note"})
	assertProblemFault(t, err, "NOT_FOUND")
}

func TestProblemAllowsIdleAndClaimedSessionsWithoutTicketOwnership(t *testing.T) {
	ctx, st, svc, project, session := problemFixture(t)
	if _, err := svc.SessionAction(ctx, project.ID, session.ID, "idle", "idle"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddProblem(ctx, project.ID, "idle-report", AddProblemInput{SessionID: session.ID, Summary: "idle", Expected: "e", Actual: "a"}); err != nil {
		t.Fatalf("idle report rejected: %v", err)
	}
	raw, err := svc.StartSession(ctx, project.ID, "dev", "claimed-session", gitctx.Context{CommonDir: "common", Root: "claimed-root"})
	if err != nil {
		t.Fatal(err)
	}
	var claimed Session
	if err = json.Unmarshal(raw, &claimed); err != nil {
		t.Fatal(err)
	}
	_, err = st.Write(ctx, store.Request{ID: "claim-fixture", Operation: "fixture", ProjectID: project.ID, ActorID: "local-user", Payload: JSON(map[string]any{})}, func(c *sql.Conn) (json.RawMessage, error) {
		now := Now()
		if _, e := c.ExecContext(ctx, "INSERT INTO tickets(id,project_id,display_key,title,body,state,created_at,updated_at) VALUES('owned-ticket',?,'APP-1','owned','body','in_progress',?,?)", project.ID, now, now); e != nil {
			return nil, e
		}
		if _, e := c.ExecContext(ctx, "INSERT INTO claims(id,project_id,ticket_id,session_id,created_at) VALUES('owned-claim',?,'owned-ticket',?,?)", project.ID, claimed.ID, now); e != nil {
			return nil, e
		}
		return JSON(map[string]any{}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddProblem(ctx, project.ID, "claimed-report", AddProblemInput{SessionID: claimed.ID, Summary: "standalone", Expected: "e", Actual: "a"}); err != nil {
		t.Fatalf("claimed session standalone report rejected: %v", err)
	}
}

func TestProblemAddRollsBackAndSameRequestCanRetry(t *testing.T) {
	ctx, st, svc, project, session := problemFixture(t)
	if _, err := st.DB.ExecContext(ctx, "CREATE TEMP TRIGGER fail_problem_event BEFORE INSERT ON events WHEN NEW.kind='problem.add' BEGIN SELECT RAISE(ABORT, 'injected event failure'); END"); err != nil {
		t.Fatal(err)
	}
	input := AddProblemInput{SessionID: session.ID, Summary: "retry", Expected: "e", Actual: "a"}
	if _, err := svc.AddProblem(ctx, project.ID, "retry", input); err == nil {
		t.Fatal("injected failure succeeded")
	}
	var problems, requests int
	var next int
	if err := st.DB.QueryRowContext(ctx, "SELECT count(*) FROM problems WHERE project_id=?", project.ID).Scan(&problems); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRowContext(ctx, "SELECT count(*) FROM requests WHERE project_id=? AND request_id='retry'", project.ID).Scan(&requests); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRowContext(ctx, "SELECT next_problem FROM projects WHERE id=?", project.ID).Scan(&next); err != nil {
		t.Fatal(err)
	}
	if problems != 0 || requests != 0 || next != 1 {
		t.Fatalf("failed mutation survived: problems=%d requests=%d next=%d", problems, requests, next)
	}
	if _, err := st.DB.ExecContext(ctx, "DROP TRIGGER fail_problem_event"); err != nil {
		t.Fatal(err)
	}
	raw, err := svc.AddProblem(ctx, project.ID, "retry", input)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeProblem(t, raw); got.DisplayKey != "APP-P1" {
		t.Fatalf("rollback consumed display key: %+v", got)
	}
}

func TestProblemValidationAndBoundedCursor(t *testing.T) {
	ctx, _, svc, project, session := problemFixture(t)
	cases := []AddProblemInput{
		{SessionID: session.ID, Summary: "", Expected: "e", Actual: "a"},
		{SessionID: session.ID, Summary: strings.Repeat("s", 4097), Expected: "e", Actual: "a"},
		{SessionID: session.ID, Summary: "s", Expected: strings.Repeat("e", 65537), Actual: "a"},
	}
	for i, input := range cases {
		_, err := svc.AddProblem(ctx, project.ID, "invalid-"+string(rune('a'+i)), input)
		assertProblemFault(t, err, "INVALID_INPUT")
	}
	for i := 0; i < 3; i++ {
		_, err := svc.AddProblem(ctx, project.ID, "add-"+string(rune('a'+i)), AddProblemInput{SessionID: session.ID, Summary: string(rune('a' + i)), Expected: "e", Actual: "a"})
		if err != nil {
			t.Fatal(err)
		}
	}
	first, err := svc.ListProblems(ctx, project.ID, ListProblemsInput{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.NextCursor == "" || first.Items[0].Summary != "a" || first.Items[1].Summary != "b" {
		t.Fatalf("first page: %+v", first)
	}
	second, err := svc.ListProblems(ctx, project.ID, ListProblemsInput{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.NextCursor != "" || second.Items[0].Summary != "c" {
		t.Fatalf("second page: %+v", second)
	}
	if _, err = svc.ListProblems(ctx, project.ID, ListProblemsInput{Limit: 101}); err == nil {
		t.Fatal("accepted oversized limit")
	}
	if _, err = svc.ListProblems(ctx, project.ID, ListProblemsInput{Cursor: "bad"}); err == nil {
		t.Fatal("accepted malformed cursor")
	}
}

func TestProblemConcurrentAppendsSurviveAndOrder(t *testing.T) {
	ctx, st, svc, project, firstSession := problemFixture(t)
	raw, err := svc.StartSession(ctx, project.ID, "dev", "session-2", gitctx.Context{CommonDir: "common", Root: "root-2"})
	if err != nil {
		t.Fatal(err)
	}
	var secondSession Session
	if err = json.Unmarshal(raw, &secondSession); err != nil {
		t.Fatal(err)
	}
	raw, err = svc.AddProblem(ctx, project.ID, "add", AddProblemInput{SessionID: firstSession.ID, Summary: "s", Expected: "e", Actual: "a"})
	if err != nil {
		t.Fatal(err)
	}
	report := decodeProblem(t, raw)
	// Derive the shared path from SQLite itself, then open an independent store connection.
	var file string
	if err = st.DB.QueryRowContext(ctx, "SELECT file FROM pragma_database_list WHERE name='main'").Scan(&file); err != nil {
		t.Fatal(err)
	}
	other, err := store.Open(file, false)
	if err != nil {
		t.Fatal(err)
	}
	defer other.DB.Close()
	services := []Service{svc, {Store: other}}
	sessions := []string{firstSession.ID, secondSession.ID}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := services[i].AppendProblem(ctx, project.ID, "append-"+string(rune('a'+i)), AppendProblemInput{SessionID: sessions[i], ProblemID: report.ID, Body: "note-" + string(rune('a'+i))})
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	page, err := svc.ListProblems(ctx, project.ID, ListProblemsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || len(page.Items[0].Notes) != 2 {
		t.Fatalf("lost append: %+v", page)
	}
	if page.Items[0].Notes[0].Seq >= page.Items[0].Notes[1].Seq {
		t.Fatalf("notes out of event order: %+v", page.Items[0].Notes)
	}
	_, err = svc.AppendProblem(ctx, project.ID, "append-a", AppendProblemInput{SessionID: firstSession.ID, ProblemID: report.ID, Body: "note-a"})
	if err != nil {
		t.Fatal(err)
	}
	page, err = svc.ListProblems(ctx, project.ID, ListProblemsInput{})
	if err != nil || len(page.Items[0].Notes) != 2 {
		t.Fatalf("replay duplicated append: %+v %v", page, err)
	}
}

func TestProblemListBoundsNotesAndReturnsNewestInEventOrder(t *testing.T) {
	ctx, st, svc, project, session := problemFixture(t)
	raw, err := svc.AddProblem(ctx, project.ID, "add", AddProblemInput{SessionID: session.ID, Summary: "s", Expected: "e", Actual: "a"})
	if err != nil {
		t.Fatal(err)
	}
	report := decodeProblem(t, raw)
	_, err = st.Write(ctx, store.Request{ID: "notes", Operation: "fixture", ProjectID: project.ID, ActorID: "local-user", Payload: JSON(map[string]any{})}, func(c *sql.Conn) (json.RawMessage, error) {
		for i := 0; i < 102; i++ {
			if _, e := c.ExecContext(ctx, "INSERT INTO events(id,project_id,problem_id,actor_id,kind,body,created_at) VALUES(?,?,?,?,?,?,?)", UUID(), project.ID, report.ID, session.ID, "problem.append", "note-"+strconv.Itoa(i), Now()); e != nil {
				return nil, e
			}
		}
		return JSON(map[string]any{}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := svc.ListProblems(ctx, project.ID, ListProblemsInput{})
	if err != nil {
		t.Fatal(err)
	}
	got := page.Items[0]
	if len(got.Notes) != 100 || !got.NotesTruncated {
		t.Fatalf("notes not bounded with truncation: count=%d report=%+v", len(got.Notes), got)
	}
	if got.Notes[0].Body != "note-2" || got.Notes[99].Body != "note-101" || got.Notes[0].Seq >= got.Notes[99].Seq {
		t.Fatalf("wrong bounded event order: first=%+v last=%+v", got.Notes[0], got.Notes[99])
	}
}

func assertProblemFault(t *testing.T, err error, code string) {
	t.Helper()
	var fault *Fault
	if !errors.As(err, &fault) || fault.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}
