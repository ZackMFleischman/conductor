package gitctx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// This also exercises a checkout whose parents may permit traversal but not
// directory enumeration, as in the Windows agent sandbox.
func TestCanonicalWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Canonical(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) || got != strings.ToLower(got) {
		t.Fatalf("noncanonical working directory: %q", got)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("canonical working directory is not accessible: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("canonical working directory is not a directory: %q", got)
	}
}

func TestCanonicalWindowsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "symlink")
	if err := os.Symlink(target, link); err != nil {
		if errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD) {
			t.Skipf("Windows symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	got, err := Canonical(link)
	if err != nil || got != strings.ToLower(target) {
		t.Fatalf("Canonical(symlink) = %q, %v; want %q", got, err, strings.ToLower(target))
	}
}

func TestCanonicalWindowsLongPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), strings.Repeat("a", 120), strings.Repeat("b", 120))
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	got, err := Canonical(`\\?\` + path)
	if err != nil || got != strings.ToLower(path) {
		t.Fatalf("Canonical(long path) = %q, %v; want %q", got, err, strings.ToLower(path))
	}
	if again, err := Canonical(got); err != nil || again != got {
		t.Fatalf("canonical path is not reusable: %q, %v", again, err)
	}
}

func TestNormalizeWindowsFinalPath(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{`\\?\C:\Mixed\Repo`, `c:\mixed\repo`},
		{`\\?\UNC\Server\Share\Repo`, `\\server\share\repo`},
		{`\\Server\Share\Repo`, `\\server\share\repo`},
		{`\\?\C:\`, `c:\`},
	} {
		if got := normalizeWindowsFinalPath(tc.path); got != tc.want {
			t.Errorf("normalizeWindowsFinalPath(%q) = %q; want %q", tc.path, got, tc.want)
		}
	}
}

func TestCanonicalWindowsAliases(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "MixedCaseTarget")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	want := strings.ToLower(target)
	junction := filepath.Join(root, "junction")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Fatalf("create junction: %v: %s", err, output)
	}
	for _, path := range []string{target, strings.ToUpper(target), junction, `\\?\` + target} {
		got, err := Canonical(path)
		if err != nil || got != want {
			t.Errorf("Canonical(%q) = %q, %v; want %q", path, got, err, want)
		}
	}
	file := filepath.Join(target, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Canonical(filepath.Join(junction, "file.txt"))
	if err != nil || got != strings.ToLower(file) {
		t.Fatalf("file through junction = %q, %v", got, err)
	}
	if got, err := Canonical(filepath.Join(junction, "missing")); err == nil || got != "" || !os.IsNotExist(err) {
		t.Fatalf("missing target = %q, %v; want empty and not-exist error", got, err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if got, err := Canonical(junction); err == nil || got != "" {
		t.Fatalf("dangling junction = %q, %v; want error", got, err)
	}
}
