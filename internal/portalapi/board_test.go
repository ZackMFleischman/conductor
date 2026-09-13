package portalapi

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ZackMFleischman/conductor/internal/store"
	"modernc.org/sqlite"
)

type boardFixture struct {
	writer *sql.DB
	reader DBReader
	path   string
}

func newBoardFixture(t *testing.T) boardFixture {
	t.Helper()
	path := filepath.Join(t.TempDir(), "registry.db")
	w, err := store.Open(path, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.DB.Close() })
	r, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.DB.Close() })
	return boardFixture{w.DB, DBReader{DB: r.DB}, path}
}

func boardExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func (f boardFixture) project(t *testing.T, id, prefix string) {
	t.Helper()
	boardExec(t, f.writer, "INSERT INTO projects(id,common_dir,prefix) VALUES(?,?,?)", id, "private/path/"+id, prefix)
}

func (f boardFixture) ticket(t *testing.T, id, project, state string) {
	t.Helper()
	boardExec(t, f.writer, `INSERT INTO tickets(id,project_id,display_key,title,body,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, id, project, "KEY-"+id, "Title "+id, "Description <b>"+id+"</b>", state, "2026-01-01", "2026-01-01")
}

func readBoard(t *testing.T, reader DBReader, project string) Board {
	t.Helper()
	b, err := reader.Board(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBoardIncludesStoredDetails(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.ticket(t, "details", "p", "review")
	boardExec(t, f.writer, `UPDATE ticket_metadata SET kind='bug' WHERE ticket_id='details'`)
	boardExec(t, f.writer, `UPDATE tickets SET summary='Summary',evidence='Evidence',qa='QA' WHERE id='details'`)
	b := readBoard(t, f.reader, "p")
	data, err := json.Marshal(b.Tickets[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"kind": "bug", "summary": "Summary", "evidence": "Evidence", "qa": "QA", "createdAt": "2026-01-01", "updatedAt": "2026-01-01"} {
		if fields[key] != want {
			t.Errorf("%s = %v, want %s", key, fields[key], want)
		}
	}
}

func TestCompletionTimestampIgnoresLaterNotes(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.ticket(t, "t", "p", "done")
	boardExec(t, f.writer, "INSERT INTO events(id,project_id,ticket_id,actor_id,kind,payload,created_at) VALUES('accept','p','t','human','ticket.accept','{}','2026-02-01')")
	boardExec(t, f.writer, "INSERT INTO events(id,project_id,ticket_id,actor_id,kind,payload,created_at) VALUES('note','p','t','human','ticket.note','{}','2026-03-01')")
	boardExec(t, f.writer, "UPDATE tickets SET updated_at='2026-03-01' WHERE id='t'")
	if got := readBoard(t, f.reader, "p").Tickets[0].CompletedAt; got != "2026-02-01" {
		t.Fatalf("completion = %s", got)
	}
}

func TestDBReaderEmptyProjectsAndParameterizedLookup(t *testing.T) {
	f := newBoardFixture(t)
	projects, err := f.reader.Projects(context.Background())
	if err != nil || projects == nil || len(projects) != 0 {
		t.Fatalf("empty projects: %#v %v", projects, err)
	}
	f.project(t, "b", "ZZZ")
	f.project(t, "a", "AAA")
	projects, err = f.reader.Projects(context.Background())
	want := []Project{{ID: "a", Name: "AAA", Description: ""}, {ID: "b", Name: "ZZZ", Description: ""}}
	if err != nil || !reflect.DeepEqual(projects, want) {
		t.Fatalf("projects: %#v %v", projects, err)
	}
	b := readBoard(t, f.reader, "a")
	if b.Project != want[0] || b.Tickets == nil || len(b.Tickets) != 0 || b.Revision == "" {
		t.Fatalf("empty board: %#v", b)
	}
	if again := readBoard(t, f.reader, "a"); again.Revision != b.Revision {
		t.Fatal("read changed revision")
	}
	for _, id := range []string{"missing", "AAA", "' OR 1=1 --"} {
		if _, err := f.reader.Board(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%q: %v", id, err)
		}
	}
}

func TestDBReaderLifecycleAndOptionalPolicy(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	for _, state := range []string{"ready", "draft", "in_progress", "blocked", "review", "done"} {
		f.ticket(t, state, "p", state)
	}
	boardExec(t, f.writer, `INSERT INTO agents(id,project_id,name) VALUES('agent','p','idle-stopped-worker')`)
	boardExec(t, f.writer, `UPDATE tickets SET assigned_agent_id='agent' WHERE id='in_progress'`)
	boardExec(t, f.writer, `UPDATE ticket_metadata SET blocked_reason='Waiting for user input' WHERE ticket_id='blocked'`)
	b := readBoard(t, f.reader, "p")
	wantStates := map[string]string{"ready": "ready", "draft": "blocked", "in_progress": "in_progress", "blocked": "blocked", "review": "review", "done": "done"}
	for _, ticket := range b.Tickets {
		if ticket.Status != wantStates[ticket.ID] || ticket.Ancestors == nil || ticket.Blockers == nil {
			t.Fatalf("ticket: %#v", ticket)
		}
		if ticket.ID == "in_progress" {
			if !reflect.DeepEqual(ticket.Assignee, &Assignee{ID: "agent", Name: "idle-stopped-worker"}) {
				t.Fatal(ticket.Assignee)
			}
		} else if ticket.Assignee != nil {
			t.Fatal("invented assignment")
		}
		if ticket.Description != "Description <b>"+ticket.ID+"</b>" {
			t.Fatal("altered text")
		}
		if (ticket.ID == "blocked" || ticket.ID == "draft") && len(ticket.Blockers) == 0 {
			t.Fatal("missing lifecycle reason")
		}
	}
	// A completed dependency and released deferral do not block an ordinary ticket.
	boardExec(t, f.writer, `INSERT INTO ticket_dependencies VALUES('p','ready','done')`)
	boardExec(t, f.writer, `INSERT INTO retrospective_deferrals VALUES('p','ready','m','Already released','','2026-01-01')`)
	if b := readBoard(t, f.reader, "p"); findBoardTicket(t, b, "ready").Status != "ready" {
		t.Fatal(b)
	}
	// All active causes remain visible, without replacing review/in_progress/done state.
	for _, id := range []string{"ready", "in_progress", "review", "done"} {
		boardExec(t, f.writer, `INSERT INTO workflow_ticket_specs(ticket_id,project_id,validation_mode,policy_revision,execution_mode,plan_review,prepared_revision,authorized_revision,paused) VALUES(?,'p','human',1,'delegated','lightweight',0,0,1)`, id)
		boardExec(t, f.writer, `INSERT INTO ticket_dependencies VALUES('p',?,'draft')`, id)
	}
	boardExec(t, f.writer, `UPDATE retrospective_deferrals SET released_at=NULL,condition_text='After milestone ships' WHERE ticket_id='ready'`)
	b = readBoard(t, f.reader, "p")
	for _, id := range []string{"ready", "in_progress", "review", "done"} {
		ticket := findBoardTicket(t, b, id)
		want := id
		if id == "ready" {
			want = "blocked"
		}
		if ticket.Status != want {
			t.Fatalf("%s status %s", id, ticket.Status)
		}
		if id == "done" {
			if len(ticket.Blockers) != 0 {
				t.Fatal("historical blockers on done", ticket)
			}
			continue
		}
		for _, text := range []string{"paused", "preparation", "authorization", "Waiting for KEY-draft: Title draft"} {
			assertBlockerContains(t, ticket, text)
		}
		if id == "ready" {
			assertBlockerContains(t, ticket, "After milestone ships")
		}
	}
	// Current preparation and authorization restore a policy ticket's ready state.
	boardExec(t, f.writer, `UPDATE workflow_ticket_specs SET prepared_revision=1,authorized_revision=1,paused=0 WHERE ticket_id='ready'`)
	boardExec(t, f.writer, `DELETE FROM ticket_dependencies WHERE ticket_id='ready' AND depends_on='draft'`)
	boardExec(t, f.writer, `UPDATE retrospective_deferrals SET released_at='now' WHERE ticket_id='ready'`)
	if ticket := findBoardTicket(t, readBoard(t, f.reader, "p"), "ready"); ticket.Status != "ready" || len(ticket.Blockers) != 0 {
		t.Fatal(ticket)
	}
	boardExec(t, f.writer, `UPDATE ticket_metadata SET spec_revision=2 WHERE ticket_id='ready'`)
	if ticket := findBoardTicket(t, readBoard(t, f.reader, "p"), "ready"); ticket.Status != "blocked" {
		t.Fatal("stale policy accepted", ticket)
	}
	boardExec(t, f.writer, `UPDATE ticket_metadata SET blocked_reason='' WHERE ticket_id='blocked'`)
	assertBlockerContains(t, findBoardTicket(t, readBoard(t, f.reader, "p"), "blocked"), "details unavailable")
}

func TestDBReaderStableBlockersDespiteDatabaseRowOrder(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	for _, id := range []string{"child", "a", "z"} {
		f.ticket(t, id, "p", "ready")
	}
	boardExec(t, f.writer, `INSERT INTO ticket_dependencies VALUES('p','child','z'),('p','child','a')`)
	before := readBoard(t, f.reader, "p")
	want := []Blocker{{Reason: "Waiting for KEY-a: Title a", TicketKey: "KEY-a"}, {Reason: "Waiting for KEY-z: Title z", TicketKey: "KEY-z"}}
	if got := findBoardTicket(t, before, "child").Blockers; !reflect.DeepEqual(got, want) {
		t.Fatalf("blockers: %#v", got)
	}
	boardExec(t, f.reader.DB, `PRAGMA reverse_unordered_selects=ON`)
	if after := readBoard(t, f.reader, "p"); !reflect.DeepEqual(before, after) {
		t.Fatal("query order changed snapshot or revision")
	}
}

func findBoardTicket(t *testing.T, b Board, id string) Ticket {
	t.Helper()
	for _, ticket := range b.Tickets {
		if ticket.ID == id {
			return ticket
		}
	}
	t.Fatalf("missing ticket %s", id)
	return Ticket{}
}

func assertBlockerContains(t *testing.T, ticket Ticket, text string) {
	t.Helper()
	for _, blocker := range ticket.Blockers {
		if strings.Contains(strings.ToLower(blocker.Reason), strings.ToLower(text)) {
			return
		}
	}
	t.Fatalf("missing blocker %q: %#v", text, ticket.Blockers)
}

func TestDBReaderCompleteAncestorsAndStableOrder(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.project(t, "other", "OTHER")
	for _, id := range []string{"root", "middle", "child"} {
		f.ticket(t, id, "p", "ready")
	}
	f.ticket(t, "foreign", "other", "ready")
	boardExec(t, f.writer, `UPDATE ticket_metadata SET parent_id='root' WHERE ticket_id='middle'`)
	boardExec(t, f.writer, `UPDATE ticket_metadata SET parent_id='middle' WHERE ticket_id='child'`)
	boardExec(t, f.writer, `UPDATE tickets SET created_at='2025' WHERE id='root'`)
	b := readBoard(t, f.reader, "p")
	if len(b.Tickets) != 3 || b.Tickets[0].ID != "root" || b.Tickets[1].ID != "child" || b.Tickets[2].ID != "middle" {
		t.Fatalf("order/isolation: %#v", b.Tickets)
	}
	want := []Reference{{ID: "middle", Key: "KEY-middle", Title: "Title middle"}, {ID: "root", Key: "KEY-root", Title: "Title root"}}
	if got := findBoardTicket(t, b, "child"); !reflect.DeepEqual(got.Ancestors, want) || got.Status != "ready" {
		t.Fatal(got)
	}
	encoded, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private/path") || strings.Contains(string(encoded), "foreign") || !strings.Contains(string(encoded), `"assignee":null`) || !strings.Contains(string(encoded), `"ancestors":[]`) {
		t.Fatal(string(encoded))
	}
}

func TestDBReaderRejectsInvalidReferences(t *testing.T) {
	cases := map[string]string{
		"missing metadata":         `DELETE FROM ticket_metadata WHERE ticket_id='child'`,
		"wrong metadata project":   `UPDATE ticket_metadata SET project_id='other' WHERE ticket_id='child'`,
		"missing parent":           `UPDATE ticket_metadata SET parent_id='missing' WHERE ticket_id='child'`,
		"foreign parent":           `UPDATE ticket_metadata SET parent_id='foreign' WHERE ticket_id='child'`,
		"self cycle":               `UPDATE ticket_metadata SET parent_id='child' WHERE ticket_id='child'`,
		"nested cycle":             `UPDATE ticket_metadata SET parent_id=CASE ticket_id WHEN 'child' THEN 'root' ELSE 'child' END WHERE project_id='p'`,
		"missing dependency":       `INSERT INTO ticket_dependencies VALUES('p','child','missing')`,
		"foreign dependency":       `INSERT INTO ticket_dependencies VALUES('p','child','foreign')`,
		"wrong dependency project": `INSERT INTO ticket_dependencies VALUES('other','child','root')`,
		"missing dependency owner": `INSERT INTO ticket_dependencies VALUES('p','missing','root')`,
		"missing assignee":         `UPDATE tickets SET assigned_agent_id='missing' WHERE id='child'`,
		"foreign assignee":         `UPDATE tickets SET assigned_agent_id='foreign-agent' WHERE id='child'`,
		"wrong policy project":     `INSERT INTO workflow_ticket_specs(ticket_id,project_id,validation_mode,policy_revision,execution_mode,plan_review) VALUES('child','other','human',1,'delegated','lightweight')`,
		"foreign deferral":         `INSERT INTO retrospective_deferrals VALUES('other','child','m','condition','',NULL)`,
		"orphan deferral":          `INSERT INTO retrospective_deferrals VALUES('p','missing','m','condition','',NULL)`,
		"unknown state":            `PRAGMA ignore_check_constraints=ON; UPDATE tickets SET state='unexpected' WHERE id='child'`,
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			f := newBoardFixture(t)
			f.project(t, "p", "P")
			f.project(t, "other", "OTHER")
			f.ticket(t, "child", "p", "ready")
			f.ticket(t, "root", "p", "done")
			f.ticket(t, "foreign", "other", "ready")
			boardExec(t, f.writer, `INSERT INTO agents(id,project_id,name) VALUES('foreign-agent','other','external')`)
			boardExec(t, f.writer, `PRAGMA foreign_keys=OFF`)
			boardExec(t, f.writer, query)
			if b, err := f.reader.Board(context.Background(), "p"); !errors.Is(err, ErrInvalidData) || !reflect.DeepEqual(b, Board{}) {
				t.Fatalf("invalid snapshot escaped: %#v %v", b, err)
			}
		})
	}
}

func TestDBReaderRevisionTracksExternalContentAndProjectEvents(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.project(t, "other", "OTHER")
	f.ticket(t, "child", "p", "ready")
	previous := readBoard(t, f.reader, "p").Revision
	boardExec(t, f.writer, `UPDATE tickets SET title='Changed externally' WHERE id='child'`)
	next := readBoard(t, f.reader, "p")
	if next.Revision == previous || next.Tickets[0].Title != "Changed externally" {
		t.Fatal("external content not observed")
	}
	previous = next.Revision
	boardExec(t, f.writer, `INSERT INTO events(id,project_id,actor_id,kind,created_at) VALUES('note','p','agent','ticket.note','now')`)
	if next := readBoard(t, f.reader, "p"); next.Revision == previous {
		t.Fatal("progress event not observed")
	} else {
		previous = next.Revision
	}
	boardExec(t, f.writer, `INSERT INTO events(id,project_id,actor_id,kind,created_at) VALUES('unrelated','other','agent','ticket.note','now')`)
	if next := readBoard(t, f.reader, "p"); next.Revision != previous {
		t.Fatal("other project invalidated board")
	}
}

func TestDBReaderRequiresCurrentSchemaWithoutMutation(t *testing.T) {
	for _, version := range []int{1, 2, 3, store.SchemaVersion, store.SchemaVersion + 1} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			f := newBoardFixture(t)
			f.project(t, "p", "P")
			f.ticket(t, "child", "p", "ready")
			boardExec(t, f.writer, fmt.Sprintf("PRAGMA user_version=%d", version))
			var before int64
			if err := f.writer.QueryRow("SELECT total_changes()").Scan(&before); err != nil {
				t.Fatal(err)
			}
			_, projectsErr := f.reader.Projects(context.Background())
			_, boardErr := f.reader.Board(context.Background(), "p")
			if version != store.SchemaVersion {
				if !errors.Is(projectsErr, ErrUnsupportedSchema) || !errors.Is(boardErr, ErrUnsupportedSchema) {
					t.Fatalf("schema %d: %v / %v", version, projectsErr, boardErr)
				}
			} else if projectsErr != nil || boardErr != nil {
				t.Fatal(projectsErr, boardErr)
			}
			var after int64
			var actualVersion int
			if err := f.writer.QueryRow("SELECT total_changes()").Scan(&after); err != nil {
				t.Fatal(err)
			}
			if err := f.writer.QueryRow("PRAGMA user_version").Scan(&actualVersion); err != nil {
				t.Fatal(err)
			}
			if before != after || actualVersion != version {
				t.Fatal("read mutated writer/schema")
			}
			var readerChanges int
			if err := f.reader.DB.QueryRow("SELECT total_changes()").Scan(&readerChanges); err != nil || readerChanges != 0 {
				t.Fatal("reader mutations", readerChanges, err)
			}
		})
	}
}

// This driver delegates every operation to real SQLite. Its single query-boundary
// callback commits an independent writer while Board's read transaction is open.
// It deliberately adds no hook to the production reader.
type coordinatedDriver struct{ beforeDependencies func() }

func (d coordinatedDriver) Open(name string) (driver.Conn, error) {
	c, err := (&sqlite.Driver{}).Open(name)
	if err != nil {
		return nil, err
	}
	return &coordinatedConn{Conn: c, beforeDependencies: d.beforeDependencies}, nil
}

type coordinatedConn struct {
	driver.Conn
	beforeDependencies func()
}

func (c *coordinatedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
}
func (c *coordinatedConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, "FROM ticket_dependencies") {
		c.beforeDependencies()
	}
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func TestDBReaderSnapshotSurvivesWriterBetweenQueries(t *testing.T) {
	f := newBoardFixture(t)
	f.project(t, "p", "P")
	f.ticket(t, "child", "p", "ready")
	f.ticket(t, "root", "p", "ready")
	boardExec(t, f.writer, `INSERT INTO ticket_dependencies VALUES('p','child','root')`)
	before := readBoard(t, f.reader, "p")
	fired := false
	driverName := "board-coordinated-" + t.TempDir()
	sql.Register(driverName, coordinatedDriver{beforeDependencies: func() {
		if fired {
			return
		}
		fired = true
		boardExec(t, f.writer, `BEGIN; UPDATE tickets SET title='New child' WHERE id='child'; UPDATE tickets SET state='done' WHERE id='root'; INSERT INTO events(id,project_id,actor_id,kind,created_at) VALUES('change','p','agent','ticket.note','now'); COMMIT;`)
	}})
	p := filepath.ToSlash(f.path)
	if filepath.VolumeName(f.path) != "" {
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro"}
	db, err := sql.Open(driverName, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	during := readBoard(t, DBReader{DB: db}, "p")
	if !fired {
		t.Fatal("writer was not coordinated between queries")
	}
	if !reflect.DeepEqual(before, during) {
		t.Fatalf("mixed snapshot:\nbefore %#v\nduring %#v", before, during)
	}
	after := readBoard(t, f.reader, "p")
	if after.Revision == before.Revision || findBoardTicket(t, after, "child").Title != "New child" || findBoardTicket(t, after, "child").Status != "ready" {
		t.Fatalf("next snapshot missed write: %#v", after)
	}
}
