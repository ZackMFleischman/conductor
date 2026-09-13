package gitctx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotRepository identifies Git's positive determination that cwd is outside a repository.
var ErrNotRepository = errors.New("not a Git repository")

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
	p, e = canonicalPath(p)
	if e != nil {
		return "", e
	}
	return filepath.Clean(p), nil
}
func Resolve(cwd string) (Context, error) {
	run := func(args ...string) (string, error) {
		c := exec.Command("git", args...)
		c.Dir = cwd
		// Keep Git's non-repository diagnostic recognizable regardless of host locale.
		c.Env = append(os.Environ(), "LC_ALL=C")
		b, e := c.Output()
		if e != nil {
			var exit *exec.ExitError
			if errors.As(e, &exit) {
				diagnostic := strings.TrimSpace(string(exit.Stderr))
				if strings.HasPrefix(diagnostic, "fatal: not a git repository (or any of the parent directories): .git") ||
					strings.HasPrefix(diagnostic, "fatal: not a git repository (or any parent up to mount point ") {
					return "", fmt.Errorf("%w: %s", ErrNotRepository, diagnostic)
				}
				return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), e, diagnostic)
			}
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), e)
		}
		return strings.TrimSpace(string(b)), e
	}
	var g Context
	root, e := run("rev-parse", "--show-toplevel")
	if e != nil {
		return g, e
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
