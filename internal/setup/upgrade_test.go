package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSkillBundleUpgradeAndInterruptedRepair(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}
	originalBootstrap := get(t, filepath.Join(o.CodexHome, "AGENTS.md"))
	o.SkillFiles["SKILL.md"] = []byte("updated work skill")
	o.SkillFiles["conductor-plan/SKILL.md"] = []byte("planning skill")
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) < 3 {
		t.Fatalf("expected journal and two skill updates, got %d", len(edits))
	}
	// The journal is written first; a crash must recognize old and new owned bytes.
	if err = Apply(edits[:1]); err != nil {
		t.Fatal(err)
	}
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")), o.SkillFiles["SKILL.md"]) {
		t.Fatal("work skill not upgraded")
	}
	if !bytes.Equal(get(t, filepath.Join(o.UserHome, ".agents", "skills", "conductor-plan", "SKILL.md")), o.SkillFiles["conductor-plan/SKILL.md"]) {
		t.Fatal("additional skill not installed")
	}
	if !bytes.Equal(get(t, filepath.Join(o.CodexHome, "AGENTS.md")), originalBootstrap) {
		t.Fatal("upgrade changed bootstrap")
	}
	edits, err = Plan(o)
	if err != nil || len(edits) != 0 {
		t.Fatalf("not idempotent: %v %d", err, len(edits))
	}
	o.Remove = true
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(o.UserHome, ".agents", "skills", "conductor-plan", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("additional skill not removed")
	}
}

func TestUpgradeDoesNotOverwriteEditedSkill(t *testing.T) {
	o := fixture(t)
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	put(t, path, []byte("personal edits"))
	o.SkillFiles["SKILL.md"] = []byte("new bundle")
	if _, err = Plan(o); err == nil {
		t.Fatal("overwrote personal edit")
	}
	if string(get(t, path)) != "personal edits" {
		t.Fatal("preview mutated file")
	}
}
