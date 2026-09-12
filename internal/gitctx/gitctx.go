package gitctx

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Context struct {
	CommonDir string  `json:"common_dir"`
	Root      string  `json:"worktree_root"`
	Branch    *string `json:"branch"`
	Head      string  `json:"head"`
	Primary   bool    `json:"primary"`
}

func Canonical(path string) (string, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	p, e = filepath.EvalSymlinks(p)
	if e != nil {
		return "", e
	}
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p, nil
}
func Resolve(cwd string) (Context, error) {
	run := func(args ...string) (string, error) {
		c := exec.Command("git", args...)
		c.Dir = cwd
		b, e := c.Output()
		return strings.TrimSpace(string(b)), e
	}
	var g Context
	root, e := run("rev-parse", "--show-toplevel")
	if e != nil {
		return g, fmt.Errorf("not a Git worktree: %w", e)
	}
	common, e := run("rev-parse", "--path-format=absolute", "--git-common-dir")
	if e != nil {
		return g, e
	}
	g.Root, e = Canonical(root)
	if e != nil {
		return g, e
	}
	g.CommonDir, e = Canonical(common)
	if e != nil {
		return g, e
	}
	if b, e := run("symbolic-ref", "--quiet", "--short", "HEAD"); e == nil {
		g.Branch = &b
	}
	g.Head, _ = run("rev-parse", "--verify", "HEAD")
	g.Primary = filepath.Clean(filepath.Dir(g.CommonDir)) == g.Root
	return g, nil
}
