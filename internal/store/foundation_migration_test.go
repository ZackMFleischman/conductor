package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFoundationMigrationAndHistoricalReplay(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "old.db")
			db, e := sql.Open("sqlite", path)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = db.Exec(schema); e != nil {
				t.Fatal(e)
			}
			if version == 2 {
				for _, name := range orderedMigrations[0].files {
					b, e := migrations.ReadFile("migrations/" + name)
					if e != nil {
						t.Fatal(e)
					}
					if _, e = db.Exec(string(b)); e != nil {
						t.Fatal(e)
					}
				}
			}
			if _, e = db.Exec(fmt.Sprintf("PRAGMA user_version=%d", version)); e != nil {
				t.Fatal(e)
			}
			if _, e = db.Exec(`INSERT INTO projects(id,common_dir,prefix) VALUES('p','common','P'); INSERT INTO tickets(id,project_id,display_key,title,body,state,created_at,updated_at) VALUES('legacy','p','P-1','old','markdown links','review','now','now')`); e != nil {
				t.Fatal(e)
			}
			if version == 2 {
				if _, e = db.Exec(`INSERT INTO workflow_policies VALUES('p',4,'delegated','lightweight','human','[]','local-user','grant','now'); INSERT INTO tickets(id,project_id,display_key,title,body,state,summary,evidence,qa,created_at,updated_at) VALUES('managed','p','P-2','managed','unchanged','review','summary','proof','steps','now','now'); INSERT INTO workflow_ticket_specs(ticket_id,project_id,spec_revision,parent_id,kind,validation_mode,required_checks,policy_revision,execution_mode,plan_review,prepared_revision,authorized_revision,submitted_commit,submitted_spec_revision) VALUES('managed','p',7,'legacy','improvement','human','[]',4,'delegated','lightweight',7,7,'commit',7); INSERT INTO workflow_dependencies VALUES('p','managed','legacy'); INSERT INTO workflow_versions VALUES('version','p','managed',7,'local-user','{"title":"managed"}','now'); INSERT INTO workflow_validations VALUES('decision','p','managed',7,'human',NULL,NULL,'','commit','criteria','proof','{}','now')`); e != nil {
					t.Fatal(e)
				}
			}
			payload := map[string]any{"ProjectID": "p", "Title": "old", "Body": "body"}
			if version == 2 {
				payload["Commit"] = ""
				payload["Validation"] = nil
			}
			b, _ := json.Marshal(payload)
			st := &Store{DB: db}
			historical, e := st.Write(context.Background(), Request{ID: "replay", ProjectID: "p", ActorID: "local-user", Operation: "ticket.create", Payload: b}, func(*sql.Conn) (json.RawMessage, error) {
				return json.RawMessage(`{"id":"legacy","historical":true}`), nil
			})
			if e != nil {
				t.Fatal(e)
			}
			var oldHash string
			if e = db.QueryRow("SELECT payload_hash FROM requests WHERE request_id='replay'").Scan(&oldHash); e != nil {
				t.Fatal(e)
			}
			db.Close()
			ro, e := OpenReadOnly(path)
			if e != nil {
				t.Fatal(e)
			}
			var got int
			ro.DB.QueryRow("PRAGMA user_version").Scan(&got)
			ro.DB.Close()
			if got != version {
				t.Fatal("read-only changed old schema")
			}
			migrated, e := Open(path, false)
			if e != nil {
				t.Fatal(e)
			}
			defer migrated.DB.Close()
			if e = migrated.DB.QueryRow("PRAGMA user_version").Scan(&got); e != nil || got != 3 {
				t.Fatalf("version %d %v", got, e)
			}
			var legacy bool
			if e = migrated.DB.QueryRow("SELECT legacy_human FROM ticket_metadata WHERE ticket_id='legacy'").Scan(&legacy); e != nil || !legacy {
				t.Fatalf("legacy restriction %v %v", legacy, e)
			}
			if version == 2 {
				var commit, parent, kind string
				var rev, prepared int
				if e = migrated.DB.QueryRow(`SELECT m.spec_revision,m.parent_id,m.kind,m.submitted_commit,s.prepared_revision FROM ticket_metadata m JOIN workflow_ticket_specs s ON s.ticket_id=m.ticket_id WHERE m.ticket_id='managed'`).Scan(&rev, &parent, &kind, &commit, &prepared); e != nil || rev != 7 || parent != "legacy" || kind != "improvement" || commit != "commit" || prepared != 7 {
					t.Fatalf("metadata %d %s %s %s %d: %v", rev, parent, kind, commit, prepared, e)
				}
				for table, id := range map[string]string{"ticket_versions": "version", "ticket_decisions": "decision", "ticket_submissions": "migration-managed"} {
					var n int
					if e = migrated.DB.QueryRow("SELECT count(*) FROM "+table+" WHERE id=?", id).Scan(&n); e != nil || n != 1 {
						t.Fatalf("lost %s %s: %v", table, id, e)
					}
				}
			}
			delete(payload, "Commit")
			delete(payload, "Validation")
			b, _ = json.Marshal(payload)
			request := Request{ID: "replay", ProjectID: "p", ActorID: "local-user", Operation: "ticket.create", Payload: b}
			callback := func(*sql.Conn) (json.RawMessage, error) { t.Fatal("replayed callback ran"); return nil, nil }
			replay, e := migrated.Write(context.Background(), request, callback)
			if e != nil || string(replay) != string(historical) {
				t.Fatalf("historical replay %s %v", replay, e)
			}
			payload["Commit"] = "changed"
			b, _ = json.Marshal(payload)
			request.Payload = b
			if _, e = migrated.Write(context.Background(), request, callback); e != ErrRequestConflict {
				t.Fatalf("changed commit accepted: %v", e)
			}
			delete(payload, "Commit")
			payload["Validation"] = map[string]string{"evidence": "new"}
			b, _ = json.Marshal(payload)
			request.Payload = b
			if _, e = migrated.Write(context.Background(), request, callback); e != ErrRequestConflict {
				t.Fatalf("changed evidence accepted: %v", e)
			}
			delete(payload, "Validation")
			payload["Title"] = "changed"
			b, _ = json.Marshal(payload)
			request.Payload = b
			if _, e = migrated.Write(context.Background(), request, callback); e != ErrRequestConflict {
				t.Fatalf("changed title accepted: %v", e)
			}
			var afterHash string
			migrated.DB.QueryRow("SELECT payload_hash FROM requests WHERE request_id='replay'").Scan(&afterHash)
			if afterHash != oldHash {
				t.Fatal("journal rewritten")
			}
			backups, _ := filepath.Glob(path + ".pre-v3-*")
			if len(backups) != 1 {
				t.Fatal(backups)
			}
			backup, e := OpenReadOnly(backups[0])
			if e != nil {
				t.Fatal(e)
			}
			backup.DB.QueryRow("PRAGMA user_version").Scan(&got)
			backup.DB.Close()
			if got != version {
				t.Fatal("wrong backup version")
			}
			migrated.DB.Close()
			ro, e = OpenReadOnly(path)
			if e != nil {
				t.Fatal(e)
			}
			ro.DB.QueryRow("PRAGMA user_version").Scan(&got)
			ro.DB.Close()
			if got != 3 {
				t.Fatal("wrong read-only v3 version")
			}
			if _, e = os.Stat(path); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestHistoricalTicketHashKeepsNonemptyEvidence(t *testing.T) {
	st, e := Open(filepath.Join(t.TempDir(), "db"), true)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	if _, e = st.DB.Exec("INSERT INTO projects(id,common_dir,prefix) VALUES('p','repo','P')"); e != nil {
		t.Fatal(e)
	}
	for i, payload := range []map[string]any{
		{"Title": "same", "Commit": "source", "Validation": nil},
		{"Title": "same", "Commit": "", "Validation": map[string]string{"evidence": "proof"}},
	} {
		b, _ := json.Marshal(payload)
		request := Request{ID: fmt.Sprint(i), ProjectID: "p", ActorID: "local-user", Operation: "ticket.submit", Payload: b}
		if _, e = st.Write(context.Background(), request, func(*sql.Conn) (json.RawMessage, error) { return json.RawMessage(`{}`), nil }); e != nil {
			t.Fatal(e)
		}
		if payload["Commit"] == "" {
			delete(payload, "Commit")
		}
		if payload["Validation"] == nil {
			delete(payload, "Validation")
		}
		b, _ = json.Marshal(payload)
		request.Payload = b
		if _, e = st.Write(context.Background(), request, func(*sql.Conn) (json.RawMessage, error) { t.Fatal("replay callback"); return nil, nil }); e != nil {
			t.Fatal(e)
		}
		payload["Commit"] = "changed"
		b, _ = json.Marshal(payload)
		request.Payload = b
		if _, e = st.Write(context.Background(), request, func(*sql.Conn) (json.RawMessage, error) { return nil, nil }); e != ErrRequestConflict {
			t.Fatalf("nonempty commit not fenced: %v", e)
		}
	}
}

func TestFoundationMigrationRollsBackOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.db")
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(schema); e != nil {
		t.Fatal(e)
	}
	for _, name := range orderedMigrations[0].files {
		b, e := migrations.ReadFile("migrations/" + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(string(b)); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = db.Exec("PRAGMA user_version=2; CREATE TABLE ticket_metadata(sentinel TEXT); INSERT INTO ticket_metadata VALUES('preserve')"); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if st, e := Open(path, false); e == nil {
		st.DB.Close()
		t.Fatal("expected migration collision")
	}
	ro, e := OpenReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	defer ro.DB.Close()
	var version int
	ro.DB.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 2 {
		t.Fatal("partial migration committed")
	}
	var sentinel string
	if e = ro.DB.QueryRow("SELECT sentinel FROM ticket_metadata").Scan(&sentinel); e != nil || sentinel != "preserve" {
		t.Fatal("old data lost", e)
	}
	var n int
	if e = ro.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='workflow_dependencies'").Scan(&n); e != nil || n != 1 {
		t.Fatal("v2 schema changed", e)
	}
}

func TestFoundationMigrationSubmissionProvenance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "submissions.db")
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(schema); e != nil {
		t.Fatal(e)
	}
	for _, name := range orderedMigrations[0].files {
		b, e := migrations.ReadFile("migrations/" + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(string(b)); e != nil {
			t.Fatal(e)
		}
	}
	_, e = db.Exec(`PRAGMA user_version=2;
 INSERT INTO projects(id,common_dir,prefix) VALUES('p','repo','P');
 INSERT INTO agents(id,project_id,name) VALUES('a','p','A'),('b','p','B');
 INSERT INTO sessions(id,project_id,agent_id,last_seen_at,worktree_root) VALUES('sa','p','a','now','a'),('sb','p','b','now','b');
 INSERT INTO tickets(id,project_id,display_key,title,body,state,summary,evidence,qa,created_at,updated_at) VALUES('t','p','P-1','work','scope','in_progress','latest','latest-proof','steps','created','claim-B'),('unknown','p','P-2','unknown','scope','review','orphan snapshot','proof','steps','created','other-edit');
 INSERT INTO claims(id,project_id,ticket_id,session_id,created_at,released_at) VALUES('old','p','t','sa','claim-A','submit-A'),('new','p','t','sb','claim-B',NULL);
 INSERT INTO events(id,project_id,ticket_id,actor_id,kind,payload,created_at) VALUES('first','p','t','sa','ticket.submit','{"Summary":"earlier","Evidence":"earlier-proof","QA":"steps","Commit":"c1"}','submit-first'),('second','p','t','sa','ticket.submit','{"Summary":"latest","Evidence":"latest-proof","QA":"steps","Commit":"c2"}','submit-A'),('reject','p','t','local-user','ticket.reject','{}','reject-A');`)
	if e != nil {
		t.Fatal(e)
	}
	db.Close()
	st, e := Open(path, false)
	if e != nil {
		t.Fatal(e)
	}
	defer st.DB.Close()
	var session, at, commit string
	e = st.DB.QueryRow("SELECT session_id,created_at,commit_id FROM ticket_submissions WHERE ticket_id='t' AND summary='latest'").Scan(&session, &at, &commit)
	if e != nil || session != "sa" || at != "submit-A" || commit != "c2" {
		t.Fatalf("submission misattributed: session=%q time=%q commit=%q: %v", session, at, commit, e)
	}
	var n int
	st.DB.QueryRow("SELECT count(*) FROM ticket_submissions WHERE ticket_id='t'").Scan(&n)
	if n != 2 {
		t.Fatalf("lost submission history: %d", n)
	}
	var unknown sql.NullString
	e = st.DB.QueryRow("SELECT session_id,created_at FROM ticket_submissions WHERE ticket_id='unknown'").Scan(&unknown, &at)
	if e != nil || unknown.Valid || at != "" {
		t.Fatalf("invented provenance: %v %q %v", unknown, at, e)
	}
}
