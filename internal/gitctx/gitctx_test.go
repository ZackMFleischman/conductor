package gitctx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLinkedWorktreeAndCloneDiscovery(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	git := func(dir string, args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %s %v", args, b, e)
		}
	}
	git(root, "init", "-q", repo)
	git(repo, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", "initial")
	linked := filepath.Join(root, "linked")
	git(repo, "worktree", "add", "-qb", "linked", linked)
	clone := filepath.Join(root, "clone")
	git(root, "clone", "--local", "-q", filepath.ToSlash(repo), clone)
	a, e := Resolve(repo)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Resolve(linked)
	if e != nil {
		t.Fatal(e)
	}
	c, e := Resolve(clone)
	if e != nil {
		t.Fatal(e)
	}
	if a.CommonDir != b.CommonDir || a.CommonDir == c.CommonDir || !a.Primary || b.Primary {
		t.Fatal(a, b, c)
	}
	if _, e = os.Stat(a.CommonDir); e != nil {
		t.Fatal(e)
	}
}
