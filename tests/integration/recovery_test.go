package integration

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ZackMFleischman/conductor/internal/store"
	"github.com/ZackMFleischman/conductor/internal/testkit"
)

func TestLostResponses(t *testing.T) {
	c, a, _ := fixture(t)
	ok(t, c, "agent", "register", "--name", "worker", "--request", "agent")
	s := session(t, a, "worker", "session")
	body := file(t, "recover response")
	create := []string{"ticket", "create", "--title", "replay", "--body-file", body, "--request", "create"}
	// Deliberately discard successful mutation responses; inspect durable state independently.
	ok(t, c, create...)
	first := ok(t, c, "ticket", "show", "IT-1")
	retry := ok(t, c, create...)
	if retry["id"] != first["id"] {
		t.Fatal(retry, first)
	}
	id := testkit.String(t, retry, "id")
	claim := []string{"ticket", "claim", id, "--session", s, "--expect-revision", "1", "--request", "claim"}
	ok(t, a, claim...)
	resumed := ok(t, a, "session", "resume", "--session", s, "--request", "resume")
	cl := resumed["claim"].(map[string]any)["claim_id"].(string)
	r := ok(t, a, claim...)
	if r["claim_id"] != cl || r["active"] != true {
		t.Fatal(r)
	}
	submit := []string{"ticket", "submit", id, "--session", s, "--claim", cl, "--expect-revision", "2", "--summary-file", body, "--evidence-file", body, "--qa-file", body, "--request", "submit"}
	ok(t, a, submit...)
	shown := ok(t, c, "ticket", "show", id)
	replay := ok(t, a, submit...)
	for _, key := range []string{"id", "revision", "state", "summary", "evidence", "qa"} {
		if !reflect.DeepEqual(shown[key], replay[key]) {
			t.Fatal(key, shown, replay)
		}
	}
	r = ok(t, a, claim...)
	if r["claim_id"] != cl || r["active"] != false {
		t.Fatal(r)
	}
	if n := count(t, c, "SELECT count(*) FROM events WHERE ticket_id=?", id); n != 3 {
		t.Fatal(n)
	}
	if n := count(t, c, "SELECT count(*) FROM tickets"); n != 1 {
		t.Fatal(n)
	}
	conflict(t, c, "ticket", "create", "--title", "different", "--body-file", body, "--request", "create")
}

// A child holds a real Store.Write transaction. No production debug/failpoint command exists.
func TestCrashHelper(t *testing.T) {
	mode := os.Getenv("CONDUCTOR_CRASH_HELPER")
	if mode == "" {
		return
	}
	s, e := store.Open(os.Getenv("CONDUCTOR_CRASH_DB"), false)
	if e != nil {
		t.Fatal(e)
	}
	defer s.DB.Close()
	if mode == "check" {
		for _, q := range []string{"SELECT count(*) FROM tickets WHERE id='crash-ticket'", "SELECT count(*) FROM events WHERE id='crash-event'", "SELECT count(*) FROM requests WHERE request_id='crash-request'"} {
			var n int
			if e := s.DB.QueryRow(q).Scan(&n); e != nil || n != 0 {
				t.Fatalf("partial commit: %s %d %v", q, n, e)
			}
		}
		return
	}
	var p string
	if e = s.DB.QueryRow("SELECT id FROM projects LIMIT 1").Scan(&p); e != nil {
		t.Fatal(e)
	}
	_, e = s.Write(context.Background(), store.Request{ID: "crash-request", ProjectID: p, Operation: "crash", ActorID: "test", Payload: json.RawMessage(`{}`)}, func(c *sql.Conn) (json.RawMessage, error) {
		for _, q := range []string{
			`INSERT INTO tickets(id,project_id,display_key,title,body,created_at,updated_at) VALUES('crash-ticket',?,'CRASH-1','partial','partial','now','now')`,
			`INSERT INTO events(id,project_id,ticket_id,actor_id,kind,created_at) VALUES('crash-event',?,'crash-ticket','test','crash','now')`,
			`INSERT INTO requests(project_id,request_id,operation,actor_id,payload_hash,result) VALUES(?,'crash-request','crash','test','pending','{}')`,
		} {
			if _, e := c.ExecContext(context.Background(), q, p); e != nil {
				return nil, e
			}
		}
		fmt.Println("TRANSACTION_READY")
		for {
			time.Sleep(time.Second)
		}
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestKilledTransactionRollsBack(t *testing.T) {
	c, _, _ := fixture(t)
	child := func(mode string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCrashHelper$", "-test.v")
		cmd.Env = append(os.Environ(), "CONDUCTOR_CRASH_HELPER="+mode, "CONDUCTOR_CRASH_DB="+filepath.Join(c.Home, "conductor.db"))
		return cmd
	}
	cmd := child("hold")
	stdout, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stderr = os.Stderr
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	ready := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if scanner.Text() == "TRANSACTION_READY" {
				ready <- true
				return
			}
		}
		ready <- false
	}()
	select {
	case found := <-ready:
		if !found {
			t.Fatal("child exited before holding transaction")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("transaction readiness timeout")
	}
	if e = cmd.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	if e = cmd.Wait(); e == nil {
		t.Fatal("killed child succeeded")
	}
	if out, e := child("check").CombinedOutput(); e != nil {
		t.Fatalf("fresh process rollback: %s %v", out, e)
	}
	ok(t, c, "ticket", "create", "--title", "after recovery", "--body-file", file(t, "store remains writable"), "--request", "after-crash")
}
