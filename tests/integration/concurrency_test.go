package integration

import (
	"fmt"
	"github.com/ZackMFleischman/conductor/internal/testkit"
	"reflect"
	"testing"
)

// Removing the transactional ownership/assignment check must fail these real process races.
func TestClaimRace(t *testing.T) {
	for _, assigned := range []bool{false, true} {
		t.Run(fmt.Sprint(assigned), func(t *testing.T) {
			c, a, b := fixture(t)
			ok(t, c, "agent", "register", "--name", "right", "--request", "right")
			ok(t, c, "agent", "register", "--name", "other", "--request", "other")
			sessions := []string{session(t, a, "right", "s1"), session(t, b, "other", "s2")}
			args := []string{"ticket", "create", "--title", "race", "--body-file", file(t, "claim exactly once"), "--request", "create"}
			if assigned {
				args = append(args, "--assigned-to", "right")
			}
			id := testkit.String(t, ok(t, c, args...), "id")
			type result struct {
				v       map[string]any
				code, i int
			}
			start := make(chan struct{})
			results := make(chan result, 2)
			for i, cli := range []testkit.CLI{a, b} {
				go func(i int, cli testkit.CLI) {
					<-start
					v, code := cli.Process("ticket", "claim", id, "--session", sessions[i], "--expect-revision", "1", "--request", fmt.Sprintf("claim%d", i))
					results <- result{v, code, i}
				}(i, cli)
			}
			close(start)
			wins, losses := 0, 0
			for range 2 {
				r := <-results
				switch r.code {
				case 0:
					wins++
					if assigned && r.i != 0 {
						t.Fatal("wrong assignee won", r.v)
					}
				case 3:
					losses++
					if r.v["ok"] != false || r.v["error"].(map[string]any)["code"] == "" {
						t.Fatal(r.v)
					}
				default:
					t.Fatal(r)
				}
			}
			if wins != 1 || losses != 1 {
				t.Fatalf("wins=%d conflicts=%d", wins, losses)
			}
			if n := count(t, c, "SELECT count(*) FROM claims WHERE ticket_id=? AND released_at IS NULL", id); n != 1 {
				t.Fatal(n)
			}
			if n := count(t, c, "SELECT count(*) FROM events WHERE ticket_id=? AND kind='ticket.claim'", id); n != 1 {
				t.Fatal(n)
			}
			if v := ok(t, c, "ticket", "show", id); testkit.Number(t, v, "revision") != 2 {
				t.Fatal(v)
			}
		})
	}
}

func TestConcurrentNotesCompose(t *testing.T) {
	c, a, _ := fixture(t)
	ok(t, c, "agent", "register", "--name", "worker", "--request", "agent")
	s := session(t, a, "worker", "session")
	body := file(t, "append-only progress")
	id := testkit.String(t, ok(t, c, "ticket", "create", "--title", "notes", "--body-file", body, "--request", "create"), "id")
	cl := testkit.String(t, ok(t, a, "ticket", "claim", id, "--session", s, "--expect-revision", "1", "--request", "claim"), "claim_id")
	start := make(chan struct{})
	done := make(chan int, 8)
	wantBodies := map[string]int{}
	for i := range 8 {
		text := fmt.Sprintf("progress from contender %d", i)
		wantBodies[text] = 1
		path := file(t, text)
		go func(i int, path string) {
			<-start
			_, code := a.Process("ticket", "note", id, "--session", s, "--claim", cl, "--body-file", path, "--request", fmt.Sprintf("note-%d", i))
			done <- code
		}(i, path)
	}
	close(start)
	for range 8 {
		if code := <-done; code != 0 {
			t.Fatalf("note exit %d", code)
		}
	}
	if n := count(t, c, "SELECT count(*) FROM events WHERE ticket_id=? AND kind='ticket.note'", id); n != 8 {
		t.Fatal(n)
	}
	v := ok(t, c, "ticket", "show", id)
	if testkit.Number(t, v, "revision") != 10 {
		t.Fatal(v)
	}
	gotBodies := map[string]int{}
	for _, raw := range v["events"].([]any) {
		event := raw.(map[string]any)
		if event["kind"] != "ticket.note" {
			continue
		}
		gotBodies[testkit.String(t, event, "body")]++
		if event["actor_id"] != s {
			t.Fatal("note attributed to wrong session", event)
		}
	}
	if !reflect.DeepEqual(gotBodies, wantBodies) {
		t.Fatal("persisted note bodies differ", gotBodies, wantBodies)
	}
}

func TestProblemAppendsPreserveReportAndTicket(t *testing.T) {
	c, a, b := fixture(t)
	ok(t, c, "agent", "register", "--name", "worker", "--request", "agent")
	s1, s2 := session(t, a, "worker", "s1"), session(t, b, "worker", "s2")
	id := testkit.String(t, ok(t, c, "ticket", "create", "--title", "problem association", "--body-file", file(t, "immutable revision"), "--request", "ticket"), "id")
	problem := ok(t, a, "problem", "add", "--ticket", id, "--session", s1, "--body-file", file(t, `{"summary":"workflow failure","expected":"single owner","actual":"stale contact","correction":"explicit recovery","evidence":"observed stale claim"}`), "--request", "problem")
	pid := testkit.String(t, problem, "id")
	type response struct {
		v    map[string]any
		code int
	}
	start := make(chan struct{})
	done := make(chan response, 2)
	notes := []string{file(t, "first session correction"), file(t, "second session evidence")}
	for i, cli := range []testkit.CLI{a, b} {
		go func(i int, cli testkit.CLI) {
			<-start
			v, code := cli.Process("problem", "append", pid, "--session", []string{s1, s2}[i], "--body-file", notes[i], "--request", fmt.Sprintf("append%d", i))
			done <- response{v, code}
		}(i, cli)
	}
	close(start)
	for range 2 {
		if r := <-done; r.code != 0 {
			t.Fatal(r)
		}
	}
	before := ok(t, c, "agent", "list")
	list := ok(t, c, "problem", "list")
	after := ok(t, c, "agent", "list")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("problem list changed session contact", before, after)
	}
	ok(t, a, "problem", "append", pid, "--session", s1, "--body-file", notes[0], "--request", "append0")
	if !reflect.DeepEqual(list, ok(t, c, "problem", "list")) {
		t.Fatal("append replay duplicated or changed report")
	}
	items := list["items"].([]any)
	if len(items) != 1 {
		t.Fatal(list)
	}
	report := items[0].(map[string]any)
	if report["summary"] != "workflow failure" || report["expected"] != "single owner" || report["actual"] != "stale contact" || len(report["notes"].([]any)) != 2 {
		t.Fatal(report)
	}
	for _, key := range []string{"id", "project_id", "display_key", "ticket_id", "session_id", "summary", "expected", "actual", "correction", "evidence", "created_at"} {
		if !reflect.DeepEqual(report[key], problem[key]) {
			t.Fatalf("initial %s changed: %v -> %v", key, problem[key], report[key])
		}
	}
	if report["correction"] != "explicit recovery" || report["evidence"] != "observed stale claim" {
		t.Fatal(report)
	}
	wantAuthors := map[string]string{"first session correction": s1, "second session evidence": s2}
	gotBodies := map[string]int{}
	for _, raw := range report["notes"].([]any) {
		note := raw.(map[string]any)
		body := testkit.String(t, note, "body")
		gotBodies[body]++
		if author, found := wantAuthors[body]; !found || note["session_id"] != author {
			t.Fatal("unexpected problem body or author", note)
		}
	}
	if !reflect.DeepEqual(gotBodies, map[string]int{"first session correction": 1, "second session evidence": 1}) {
		t.Fatal("problem body multiset differs", gotBodies)
	}
	if v := ok(t, c, "ticket", "show", id); testkit.Number(t, v, "revision") != 1 {
		t.Fatal("problem changed ticket revision", v)
	}
}
