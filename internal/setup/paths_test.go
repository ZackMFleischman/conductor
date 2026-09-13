package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMissingSkillThroughLinkedParentRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "conductor-plan")
	if runtime.GOOS == "windows" {
		if b, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
			t.Fatalf("junction fixture: %v: %s", err, b)
		}
	} else if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(link, "references", "commands.md")
	if _, err := read(target); err == nil {
		t.Fatal("missing leaf beneath linked parent accepted")
	}
	if err := Apply([]Edit{edit(target, nil, []byte("unexpected"), "fixture")}); err == nil {
		t.Fatal("apply followed linked parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "references", "commands.md")); !os.IsNotExist(err) {
		t.Fatalf("outside destination touched: %v", err)
	}
}
