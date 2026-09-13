package portalapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTicketAttachmentsAndServing(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".git"), 0700)
	os.WriteFile(filepath.Join(root, "preview.png"), []byte("\x89PNG\r\n\x1a\nimage"), 0600)
	os.WriteFile(filepath.Join(root, "notes with spaces.txt"), []byte("notes"), 0600)
	os.WriteFile(filepath.Join(root, ".git", "config"), []byte("private"), 0600)
	ticket := Ticket{ID: "t", Description: "Screenshot: " + filepath.ToSlash(filepath.Join(root, "preview.png")) + ".\n[notes](<notes with spaces.txt>)\n![duplicate](preview.png)\n[private](.git/config) [missing](missing.png) [escape](../outside.txt)", Ancestors: []Reference{}, Blockers: []Blocker{}}
	b := testBoard()
	b.Project.root = root
	b.Tickets = []Ticket{ticket}
	b.Tickets[0].Attachments = localAttachments(root, "p", ticket)
	if len(b.Tickets[0].Attachments) != 2 {
		t.Fatalf("attachments: %+v", b.Tickets[0].Attachments)
	}
	h := NewHandler(&testReader{board: b}, Options{})
	for _, a := range b.Tickets[0].Attachments {
		q := httptest.NewRequest("GET", a.URL, nil)
		q.Host = "localhost"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, q)
		if w.Code != 200 || w.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("serve: %d %v", w.Code, w.Header())
		}
		if a.MediaType == "image/png" && !strings.HasPrefix(w.Body.String(), "\x89PNG") {
			t.Fatal("wrong image")
		}
		if a.MediaType != "image/png" && !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") {
			t.Fatal("file must download")
		}
	}
	a := b.Tickets[0].Attachments[0]
	for _, path := range []string{strings.Replace(a.URL, "/tickets/t/", "/tickets/other/", 1), a.URL + "bad"} {
		q := httptest.NewRequest("GET", path, nil)
		q.Host = "localhost"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, q)
		if w.Code != 404 {
			t.Fatalf("unexpected access: %s %d", path, w.Code)
		}
	}
	os.Remove(filepath.Join(root, "preview.png"))
	q := httptest.NewRequest("GET", a.URL, nil)
	q.Host = "localhost"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, q)
	if w.Code != 404 {
		t.Fatalf("missing image: %d", w.Code)
	}
}

func TestAttachmentFileBoundary(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "outside.png"), []byte("private"), 0600)
	os.Mkdir(filepath.Join(root, ".git"), 0700)
	os.WriteFile(filepath.Join(root, ".git", "secret.png"), []byte("private"), 0600)
	for _, path := range []string{filepath.Join(outside, "outside.png"), ".git/secret.png", "../outside.png", root} {
		if f, err := openAttachment(root, path); err == nil {
			f.Close()
			t.Fatalf("accepted %q", path)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err == nil {
		if f, err := openAttachment(root, "escape/outside.png"); err == nil {
			f.Close()
			t.Fatal("accepted escaping symlink")
		}
	}
}

func TestLocalReferenceStyleAttachments(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "preview.png"), []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "test notes.txt"), []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}
	ticket := Ticket{ID: "t", Description: "![Preview][shot]\n\n[shot]: preview.png\n[Notes][notes]\n\n[notes]: <test notes.txt>"}
	if got := localAttachments(root, "p", ticket); len(got) != 2 {
		t.Fatalf("reference attachments: %+v", got)
	}
}

func TestBoardAttachmentMetadataAndRevision(t *testing.T) {
	f := newBoardFixture(t)
	root := t.TempDir()
	f.project(t, "p", "P")
	f.ticket(t, "t", "p", "ready")
	boardExec(t, f.writer, "UPDATE projects SET common_dir=? WHERE id='p'", filepath.Join(root, ".git"))
	boardExec(t, f.writer, "UPDATE tickets SET body='[Notes](notes.txt)' WHERE id='t'")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	before := readBoard(t, f.reader, "p")
	if len(before.Tickets[0].Attachments) != 1 {
		t.Fatal("missing attachment metadata")
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("changed size"), 0600); err != nil {
		t.Fatal(err)
	}
	after := readBoard(t, f.reader, "p")
	if before.Revision == after.Revision {
		t.Fatal("file change must refresh live board")
	}
	if err := os.Remove(filepath.Join(root, "notes.txt")); err != nil {
		t.Fatal(err)
	}
	if got := readBoard(t, f.reader, "p"); len(got.Tickets[0].Attachments) != 0 {
		t.Fatal("missing file retained")
	}
}
