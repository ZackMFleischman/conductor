package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
	o.SkillFiles["conductor-onboard/SKILL.md"] = []byte("onboarding skill")
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
	if !bytes.Equal(get(t, filepath.Join(o.UserHome, ".agents", "skills", "conductor-onboard", "SKILL.md")), o.SkillFiles["conductor-onboard/SKILL.md"]) {
		t.Fatal("onboarding skill not installed after interrupted upgrade")
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
	if _, err = os.Stat(filepath.Join(o.UserHome, ".agents", "skills", "conductor-onboard", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("onboarding skill not removed")
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

func TestUpgradePreservesCanonicalLocalAddition(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.SkillFiles["SKILL.md"] = []byte("base\n")
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	local := []byte("base\ncomment guidance\n")
	put(t, path, local)
	o.SkillFiles["SKILL.md"] = []byte("base\ncomment guidance\nnew canonical guidance\n")
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, path), local) {
		t.Fatal("preview changed local skill")
	}
	var skill Edit
	for _, e := range edits {
		if e.Path == path {
			skill = e
		}
	}
	if skill.Description != "Safely merge preserved Conductor skill guidance" {
		t.Fatalf("missing merge preview: %#v", skill)
	}
	if skill.BeforeHash != digest(local) {
		t.Fatalf("merge preview hash = %q, want %q", skill.BeforeHash, digest(local))
	}
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
	if got := get(t, path); !bytes.Equal(got, o.SkillFiles["SKILL.md"]) {
		t.Fatalf("canonical merge lost content: %q", got)
	}
	if edits, err = Plan(o); err != nil || len(edits) != 0 {
		t.Fatalf("upgrade not idempotent: %v %#v", err, edits)
	}
}

func TestUpgradePreservesCanonicalLocalAdditionAcrossLineEndings(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.SkillFiles["SKILL.md"] = []byte("alpha\nomega\n")
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	local := []byte("alpha\nreviewed comment discipline one\r\nreviewed comment discipline two\r\nomega\n")
	next := []byte("alpha\nreviewed comment discipline one\nreviewed comment discipline two\nomega\nnew canonical guidance\n")
	put(t, path, local)
	o.SkillFiles["SKILL.md"] = next

	edits, err := Plan(o)
	if err != nil {
		t.Fatalf("mixed-EOL preserved canonical additions should permit preview: %v", err)
	}
	if got := get(t, path); !bytes.Equal(got, local) {
		t.Fatalf("preview changed installed bytes: %q", got)
	}
	var skill *Edit
	for i := range edits {
		if edits[i].Path == path {
			skill = &edits[i]
			break
		}
	}
	if skill == nil || skill.Description != "Safely merge preserved Conductor skill guidance" {
		t.Fatalf("missing exact preserved-guidance preview: %#v", edits)
	}
	if skill.BeforeHash != digest(local) {
		t.Fatalf("preview hash = %q, want %q", skill.BeforeHash, digest(local))
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, path); !bytes.Equal(got, next) {
		t.Fatalf("apply did not write reviewed canonical bytes: %q", got)
	}
	if edits, err = Plan(o); err != nil || len(edits) != 0 {
		t.Fatalf("mixed-EOL upgrade not idempotent: %v %#v", err, edits)
	}
}

func TestUpgradePreservesLocalGuidanceWhileCanonicalParagraphsChange(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	base := []byte("heading\nold model and concurrency paragraph\nold workflow fixer paragraph\nclosing\n")
	o.SkillFiles["SKILL.md"] = base
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	local := []byte("heading\nold model and concurrency paragraph\nold workflow fixer paragraph\nclosing\n\r\nApply ticket comment discipline.\r\n")
	next := []byte("heading\nchoose a supported model and effort for each task\nthe dedicated workflow fixer takes one ready remedy at a time\nclosing\n\nApply ticket comment discipline.\n")
	put(t, path, local)
	o.SkillFiles["SKILL.md"] = next

	edits, err := Plan(o)
	if err != nil {
		t.Fatalf("preserved mixed-EOL guidance should survive canonical paragraph rewrites: %v", err)
	}
	if got := get(t, path); !bytes.Equal(got, local) {
		t.Fatalf("preview changed installed bytes: %q", got)
	}
	var skill *Edit
	for i := range edits {
		if edits[i].Path == path {
			skill = &edits[i]
			break
		}
	}
	if skill == nil || skill.Description != "Safely merge preserved Conductor skill guidance" {
		t.Fatalf("missing preserved-guidance preview: %#v", edits)
	}
	if skill.BeforeHash != digest(local) {
		t.Fatalf("preview hash = %q, want %q", skill.BeforeHash, digest(local))
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, path); !bytes.Equal(got, next) {
		t.Fatalf("apply did not write reviewed canonical bytes: %q", got)
	}
	if edits, err = Plan(o); err != nil || len(edits) != 0 {
		t.Fatalf("paragraph-rewrite upgrade not idempotent: %v %#v", err, edits)
	}
}

func TestUpgradeAdoptsOwnershipWhenCurrentAlreadyEqualsCanonical(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.SkillFiles["SKILL.md"] = []byte("base\n")
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	next := []byte("base\nreviewed guidance\n")
	put(t, path, next)
	o.SkillFiles["SKILL.md"] = next
	edits, err := Plan(o)
	if err != nil {
		t.Fatalf("current canonical bytes should permit ownership upgrade: %v", err)
	}
	for _, e := range edits {
		if e.Path == path {
			t.Fatalf("ownership upgrade should not rewrite unchanged canonical skill: %#v", e)
		}
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, path); !bytes.Equal(got, next) {
		t.Fatalf("ownership upgrade changed canonical skill: %q", got)
	}
	if edits, err = Plan(o); err != nil || len(edits) != 0 {
		t.Fatalf("ownership upgrade not idempotent: %v %#v", err, edits)
	}
}

func TestUpgradeAdoptsMatchingUnownedReferenceAndGuardsStaleApply(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.SkillFiles = map[string][]byte{"SKILL.md": []byte("base\n")}
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}

	matching := []byte("shared reference\n")
	reference := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "references", "ticket-comments.md")
	put(t, reference, matching)
	o.SkillFiles["references/ticket-comments.md"] = matching
	edits, err := Plan(o)
	if err != nil {
		t.Fatalf("matching unowned reference should be adopted during upgrade: %v", err)
	}
	var guard *Edit
	for i := range edits {
		if edits[i].Path == reference {
			guard = &edits[i]
			break
		}
	}
	if guard == nil || guard.Action != "verify" {
		t.Fatalf("missing stale-apply guard for matching reference: %#v", edits)
	}
	if got := get(t, reference); !bytes.Equal(got, matching) {
		t.Fatalf("preview changed matching reference: %q", got)
	}
	put(t, reference, []byte("concurrent edit\n"))
	if err = Apply(edits); err == nil || !strings.Contains(err.Error(), "preview conflict") {
		t.Fatalf("accepted stale matching reference: %v", err)
	}
	if got := get(t, reference); !bytes.Equal(got, []byte("concurrent edit\n")) {
		t.Fatalf("stale apply rewrote matching reference: %q", got)
	}
}

func TestUpgradeAdoptsMatchingUnownedReferenceAndPreservesItOnRemoval(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.SkillFiles = map[string][]byte{"SKILL.md": []byte("base\n")}
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}

	matching := []byte("shared reference\n")
	reference := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "references", "ticket-comments.md")
	put(t, reference, matching)
	o.SkillFiles["references/ticket-comments.md"] = matching
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, reference); !bytes.Equal(got, matching) {
		t.Fatalf("upgrade changed matching reference: %q", got)
	}
	if edits, err = Plan(o); err != nil || len(edits) != 0 {
		t.Fatalf("matching-reference upgrade not idempotent: %v %#v", err, edits)
	}

	o.Remove = true
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, reference); !bytes.Equal(got, matching) {
		t.Fatalf("removal deleted pre-existing matching reference: %q", got)
	}
}

func TestUpgradePreservesCanonicalAdditionsAcrossSkillFiles(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	keys := []string{"SKILL.md", "conductor-plan/SKILL.md", "conductor-worker/SKILL.md", "conductor-orchestrator/SKILL.md", "conductor-retrospective/SKILL.md", "conductor-workflow-improver/SKILL.md"}
	for _, key := range keys {
		o.SkillFiles[key] = []byte("base\n")
	}
	initial, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(initial); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		path, err := skillDestination(key, o, "codex")
		if err != nil {
			t.Fatal(err)
		}
		put(t, path, []byte("base\ncomment guidance\n"))
		o.SkillFiles[key] = []byte("base\ncomment guidance\nnew canonical guidance\n")
	}
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	merges := 0
	for _, e := range edits {
		if e.Description == "Safely merge preserved Conductor skill guidance" {
			merges++
		}
	}
	if merges != len(keys) {
		t.Fatalf("merge previews = %d, want %d", merges, len(keys))
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		path, err := skillDestination(key, o, "codex")
		if err != nil {
			t.Fatal(err)
		}
		if got := get(t, path); !bytes.Equal(got, o.SkillFiles[key]) {
			t.Fatalf("%s did not retain canonical addition: %q", key, got)
		}
	}
}

func TestUpgradeRejectsUnrepresentedOrOverlappingLocalSkillEdits(t *testing.T) {
	for name, tc := range map[string]struct {
		base  []byte
		local []byte
		next  []byte
	}{
		"unrepresented":     {base: []byte("base\n"), local: []byte("base\nlocal-only\n"), next: []byte("base\nnew canonical guidance\n")},
		"overlapping":       {base: []byte("base\nold guidance\n"), local: []byte("base\nlocal rewrite\n"), next: []byte("base\ncanonical rewrite\n")},
		"deletion":          {base: []byte("base\nold guidance\n"), local: []byte("base\n"), next: []byte("base\nold guidance\nnew canonical guidance\n")},
		"reordered":         {base: []byte("alpha\nomega\n"), local: []byte("alpha\nsecond addition\nfirst addition\nomega\n"), next: []byte("new alpha\nfirst addition\nsecond addition\nnew omega\n")},
		"borrowed baseline": {base: []byte("alpha\nguidance\nomega\n"), local: []byte("alpha\nguidance\nguidance\nomega\n"), next: []byte("new alpha\nguidance\nnew omega\n")},
		"missing newline":   {base: []byte("base\n"), local: []byte("base"), next: []byte("base\nnew canonical guidance\n")},
	} {
		t.Run(name, func(t *testing.T) {
			o := fixture(t)
			o.Agents = []string{"codex"}
			o.SkillFiles["SKILL.md"] = tc.base
			initial, err := Plan(o)
			if err != nil {
				t.Fatal(err)
			}
			if err = Apply(initial); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
			put(t, path, tc.local)
			o.SkillFiles["SKILL.md"] = tc.next
			if _, err = Plan(o); err == nil {
				t.Fatal("expected conflict")
			}
			if got := get(t, path); !bytes.Equal(got, tc.local) {
				t.Fatalf("preview overwrote local skill: %q", got)
			}
		})
	}
}

func TestUpgradeAddsRootAfterUserEnablesElevatedSandbox(t *testing.T) {
	o := fixture(t)
	o.Platform = "windows"
	o.Agents = []string{"codex"}
	p := filepath.Join(o.CodexHome, "config.toml")
	put(t, p, []byte("[windows]\nsandbox = \"unelevated\"\n"))
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	put(t, p, bytes.ReplaceAll(get(t, p), []byte("unelevated"), []byte("elevated")))
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	access, err := PlannedAccess(o, edits)
	if err != nil {
		t.Fatal(err)
	}
	if !access["codex"].RootConfigured {
		t.Fatal("existing installation did not add supported root after user mode change")
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	o.Remove = true
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if string(get(t, p)) != "[windows]\nsandbox = \"elevated\"\n" {
		t.Fatal("uninstall undid user's mode change")
	}
}
