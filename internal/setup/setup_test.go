package setup

import (
	"bytes"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
	"path/filepath"
	"reflect"

	"strings"
	"testing"
)

func fixture(t *testing.T) Options {
	t.Helper()
	root := t.TempDir()
	return Options{Platform: "linux", Agents: []string{"codex", "claude"}, Executable: filepath.Join(root, "Program Files", "conductor.exe"), DataHome: filepath.Join(root, "shared data"), UserHome: root, CodexHome: filepath.Join(root, "custom codex"), ClaudeHome: filepath.Join(root, "custom claude"), SkillFiles: map[string][]byte{"SKILL.md": []byte("test skill"), "references/commands.md": []byte("commands")}}
}
func put(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func get(t *testing.T, p string) []byte {
	t.Helper()
	b, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestPreviewApplyRepeatRemove(t *testing.T) {
	o := fixture(t)
	cp := filepath.Join(o.CodexHome, "AGENTS.md")
	hp := filepath.Join(o.ClaudeHome, "CLAUDE.md")
	original := []byte("# Personal\r\nKeep these bytes.\r\n")
	put(t, cp, original)
	put(t, hp, original)
	config := filepath.Join(o.ClaudeHome, "settings.json")
	cb := []byte("{\n  \"model\": \"sonnet\",\n  \"permissions\": {\"additionalDirectories\": [\"/existing\"]}\n}\n")
	put(t, config, cb)
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if len(edits) == 0 {
		t.Fatal("empty preview")
	}
	if !bytes.Equal(get(t, cp), original) {
		t.Fatal("preview wrote")
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	b := get(t, cp)
	if strings.Count(string(b), "<!-- conductor:start -->") != 1 {
		t.Fatalf("bootstrap: %s", b)
	}
	if !bytes.Contains(b, []byte("REGISTRY_UNAVAILABLE")) || !bytes.Contains(b, []byte("context --if-registered --json")) {
		t.Fatal("routing missing")
	}
	again, e := Plan(o)
	if e != nil || len(again) != 0 {
		t.Fatalf("repeat: %v %+v", e, again)
	}
	// An unrelated instruction edit survives uninstall.
	put(t, cp, append(get(t, cp), []byte("\nNew personal note\n")...))
	o.Remove = true
	edits, e = Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(get(t, cp), append(original, []byte("\nNew personal note\n")...)) {
		t.Fatal("removed unrelated content")
	}
	if !bytes.Equal(get(t, config), cb) {
		t.Fatalf("settings not restored: %s", get(t, config))
	}
	again, e = Plan(o)
	if e != nil || len(again) != 0 {
		t.Fatalf("repeat remove: %v", e)
	}
}
func TestOverridePrecedence(t *testing.T) {
	for _, nonempty := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "active"}[nonempty], func(t *testing.T) {
			o := fixture(t)
			o.Agents = []string{"codex"}
			override := filepath.Join(o.CodexHome, "AGENTS.override.md")
			v := []byte{}
			if nonempty {
				v = []byte("Override instructions")
			}
			put(t, override, v)
			edits, e := Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits); e != nil {
				t.Fatal(e)
			}
			if nonempty {
				if !bytes.HasPrefix(get(t, override), v) {
					t.Fatal("lost override")
				}
			} else if len(get(t, override)) != 0 {
				t.Fatal("created precedence override")
			}
		})
	}
}
func TestChangedOwnedFileConflicts(t *testing.T) {
	o := fixture(t)
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	put(t, p, []byte("user edited skill"))
	if _, e = Plan(o); e == nil {
		t.Fatal("overwrote owned edit")
	}
	o.Remove = true
	if _, e = Plan(o); e == nil {
		t.Fatal("removed edited skill")
	}
}
func TestPreviewConflictAndPartialRepair(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if len(edits) < 3 {
		t.Fatal("expected multiple files")
	}
	if e = Apply(edits[:2]); e != nil {
		t.Fatal(e)
	}
	repair, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(repair); e != nil {
		t.Fatal(e)
	}
	again, e := Plan(o)
	if e != nil || len(again) != 0 {
		t.Fatal("repair not stable", e)
	}
	o = fixture(t)
	edits, e = Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	put(t, edits[1].Path, []byte("concurrent edit"))
	if e = Apply(edits); e == nil || !strings.Contains(e.Error(), "completed") {
		t.Fatalf("expected partial error, got %v", e)
	}
}
func TestUnsupportedConfig(t *testing.T) {
	for _, data := range []string{"{broken", `{"permissions":5}`, `{"permissions":{},"permissions":{}}`} {
		o := fixture(t)
		o.Agents = []string{"claude"}
		put(t, filepath.Join(o.ClaudeHome, "settings.json"), []byte(data))
		if _, e := Plan(o); e == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
func TestReadOnlyAndUnownedSkill(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	p := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md")
	put(t, p, []byte("existing personal skill"))
	if _, e := Plan(o); e == nil {
		t.Fatal("accepted unowned skill")
	}
	o = fixture(t)
	p = filepath.Join(o.CodexHome, "AGENTS.md")
	put(t, p, []byte("personal"))
	if e := os.Chmod(p, 0400); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { os.Chmod(p, 0600) })
	if _, e := Plan(o); e == nil {
		t.Fatal("accepted read-only instruction")
	}
}

func TestSetupAdoptsMatchingUnownedSkillReferenceAndPreservesItOnRemoval(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	matching := []byte("shared reference\n")
	o.SkillFiles = map[string][]byte{"SKILL.md": []byte("test skill"), "references/ticket-comments.md": matching}
	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "references", "ticket-comments.md")
	put(t, path, matching)

	edits, err := Plan(o)
	if err != nil {
		t.Fatalf("matching unowned reference should be adopted: %v", err)
	}
	if got := get(t, path); !bytes.Equal(got, matching) {
		t.Fatalf("preview changed matching unowned reference: %q", got)
	}
	var guard *Edit
	for i := range edits {
		if edits[i].Path == path {
			guard = &edits[i]
			break
		}
	}
	if guard == nil || guard.Action != "verify" {
		t.Fatalf("matching unowned reference should only be guarded: %#v", edits)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, path); !bytes.Equal(got, matching) {
		t.Fatalf("apply changed matching unowned reference: %q", got)
	}
	if edits, err = Plan(o); err != nil || len(edits) != 0 {
		t.Fatalf("matching reference setup not idempotent: %v %#v", err, edits)
	}

	o.Remove = true
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if got := get(t, path); !bytes.Equal(got, matching) {
		t.Fatalf("removal deleted pre-existing matching reference: %q", got)
	}
}

func TestSetupRejectsDifferingUnownedSkillReference(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.SkillFiles = map[string][]byte{"SKILL.md": []byte("test skill"), "references/ticket-comments.md": []byte("canonical reference\n")}
	path := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "references", "ticket-comments.md")
	put(t, path, []byte("personal reference\n"))
	if _, err := Plan(o); err == nil {
		t.Fatal("accepted differing unowned reference")
	}
	if got := get(t, path); !bytes.Equal(got, []byte("personal reference\n")) {
		t.Fatalf("preview changed differing reference: %q", got)
	}
}

func TestSetupMatchingUnownedReferenceRetainsStaleApplyProtection(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	matching := []byte("shared reference\n")
	o.SkillFiles = map[string][]byte{"SKILL.md": []byte("test skill"), "references/ticket-comments.md": matching}
	reference := filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "references", "ticket-comments.md")
	put(t, reference, matching)
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
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
	put(t, reference, []byte("concurrent edit"))
	if err = Apply(edits); err == nil || !strings.Contains(err.Error(), "preview conflict") {
		t.Fatalf("accepted stale preview: %v", err)
	}
	if got := get(t, reference); !bytes.Equal(got, []byte("concurrent edit")) {
		t.Fatalf("stale apply rewrote matching reference: %q", got)
	}
}
func TestSkillSourceRequired(t *testing.T) {
	o := fixture(t)
	o.SkillFiles = nil
	if _, e := Plan(o); e == nil {
		t.Fatal("accepted missing skill")
	}
}
func TestConfigPreservesUnrelatedBytesAndLaterEdits(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"claude"}
	p := filepath.Join(o.ClaudeHome, "settings.json")
	before := []byte("{\r\n \"model\" : \"sonnet\", \"permissions\": { \"additionalDirectories\": [\"/existing\"] }, \"env\" : {\"ABC\":\"x\"}\r\n}\r\n")
	put(t, p, before)
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	after := get(t, p)
	if !bytes.Contains(after, []byte("\"model\" : \"sonnet\"")) || !bytes.Contains(after, []byte("\"env\" : {\"ABC\":\"x\"}")) {
		t.Fatal("rewrote unrelated JSON bytes")
	}
	put(t, p, bytes.Replace(after, []byte("sonnet"), []byte("opus"), 1))
	o.Remove = true
	edits, e = Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(get(t, p), bytes.Replace(before, []byte("sonnet"), []byte("opus"), 1)) {
		t.Fatal("uninstall discarded unrelated setting")
	}
}
func TestTOMLMergeAndInvalid(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	p := filepath.Join(o.CodexHome, "config.toml")
	before := []byte("# keep\nmodel = \"x\"\n[sandbox_workspace_write]\nwritable_roots = [\n  'existing', # root\n]\n[other]\nvalue = 1\n")
	put(t, p, before)
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	after := get(t, p)
	if !bytes.Contains(after, []byte("[other]\nvalue = 1\n")) {
		t.Fatal("changed unrelated TOML")
	}
	o.Remove = true
	edits, e = Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(get(t, p), before) {
		t.Fatal("TOML round trip changed bytes")
	}
	for _, bad := range []string{"broken ???", "[sandbox_workspace_write]\nwritable_roots = 5", "sandbox_workspace_write = {writable_roots = []}"} {
		o = fixture(t)
		o.Agents = []string{"codex"}
		put(t, filepath.Join(o.CodexHome, "config.toml"), []byte(bad))
		if _, e = Plan(o); e == nil {
			t.Fatal("accepted bad/unsupported TOML", bad)
		}
	}
}
func TestTOMLHeaderWithoutNewlineAndQuotedKey(t *testing.T) {
	for _, b := range []string{"[sandbox_workspace_write]", "[sandbox_workspace_write]\n\"writable_roots\" = []\n"} {
		o := fixture(t)
		o.Agents = []string{"codex"}
		p := filepath.Join(o.CodexHome, "config.toml")
		put(t, p, []byte(b))
		edits, e := Plan(o)
		if e != nil {
			continue
		}
		if e = Apply(edits); e != nil {
			t.Fatal(e)
		}
		if _, e = Plan(o); e != nil {
			t.Fatalf("installer produced invalid config: %v", e)
		}
		var parsed map[string]any
		if e = toml.Unmarshal(get(t, p), &parsed); e != nil {
			t.Fatalf("invalid resulting TOML: %v", e)
		}
	}
}
func TestExistingManifestRejectsMalformedConfig(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"claude"}
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(o.ClaudeHome, "settings.json")
	put(t, p, append(get(t, p), []byte("broken")...))
	o.Remove = true
	if _, e = Plan(o); e == nil {
		t.Fatal("accepted malformed user config")
	}
}
func TestUnselectedHostDoesNotRequireRoot(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.ClaudeHome = ""
	if _, e := Plan(o); e != nil {
		t.Fatal(e)
	}
}
func TestTOMLMultilineStringCannotMasqueradeAsSetting(t *testing.T) {
	for _, body := range []string{"[sandbox_workspace_write]\ncomment = '''\nwritable_roots = []\n'''\nwritable_roots = ['existing']\n", "comment = '''\n[sandbox_workspace_write]\nwritable_roots = []\n'''\n[sandbox_workspace_write]\nwritable_roots = ['existing']\n"} {
		o := fixture(t)
		o.Agents = []string{"codex"}
		p := filepath.Join(o.CodexHome, "config.toml")
		put(t, p, []byte(body))
		edits, e := Plan(o)
		if e != nil {
			if !bytes.Equal(get(t, p), []byte(body)) {
				t.Fatal("rejected preview wrote config")
			}
			continue
		}
		if e = Apply(edits); e != nil {
			t.Fatal(e)
		}
		var before, after map[string]any
		toml.Unmarshal([]byte(body), &before)
		toml.Unmarshal(get(t, p), &after)
		old := before["sandbox_workspace_write"].(map[string]any)
		old["writable_roots"] = []any{"existing", o.DataHome}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("changed unrelated TOML or missed roots: %#v", after)
		}
	}
}
func TestInterruptedRemovalRetainsPersonalEdits(t *testing.T) {
	for _, cut := range []int{1, 2, 3, 4, 5} {
		t.Run(fmt.Sprint(cut), func(t *testing.T) {
			o := fixture(t)
			o.Agents = []string{"codex"}
			instruction := filepath.Join(o.CodexHome, "AGENTS.md")
			config := filepath.Join(o.CodexHome, "config.toml")
			put(t, instruction, []byte("personal\n"))
			put(t, config, []byte("model = 'old'\n"))
			edits, e := Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits); e != nil {
				t.Fatal(e)
			}
			put(t, instruction, append(get(t, instruction), []byte("new personal note\n")...))
			put(t, config, bytes.Replace(get(t, config), []byte("'old'"), []byte("'new'"), 1))
			o.Remove = true
			edits, e = Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			n := cut
			if n > len(edits) {
				n = len(edits)
			}
			if e = Apply(edits[:n]); e != nil {
				t.Fatal(e)
			}
			repair, e := Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(repair); e != nil {
				t.Fatal(e)
			}
			if string(get(t, instruction)) != "personal\nnew personal note\n" || string(get(t, config)) != "model = 'new'\n" {
				t.Fatal("lost personal edits")
			}
			if _, e = os.Stat(filepath.Join(o.CodexHome, "conductor-setup.json")); !os.IsNotExist(e) {
				t.Fatal("removal journal remains")
			}
		})
	}
}

func TestWindowsUnelevatedPreservesConfigAndInstallsSkill(t *testing.T) {
	o := fixture(t)
	o.Platform = "windows"
	o.Agents = []string{"codex"}
	p := filepath.Join(o.CodexHome, "config.toml")
	before := []byte("# personal\r\n[windows]\r\nsandbox = 'unelevated'\r\n[sandbox_workspace_write]\r\nwritable_roots = ['existing']\r\n")
	put(t, p, before)
	instruction := filepath.Join(o.CodexHome, "AGENTS.md")
	put(t, instruction, []byte("personal\n"))
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	for _, edit := range edits {
		if edit.Path == p {
			t.Fatal("incompatible Windows setup must not change global writable roots")
		}
	}
	if err = Apply(edits[:1]); err != nil {
		t.Fatal(err)
	}
	repair, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(repair); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p), before) {
		t.Fatal("changed personal config bytes")
	}
	if !bytes.Contains(get(t, instruction), []byte("<!-- conductor:start -->")) {
		t.Fatal("missing bootstrap")
	}
	if string(get(t, filepath.Join(o.UserHome, ".agents", "skills", "conductor-work", "SKILL.md"))) != "test skill" {
		t.Fatal("missing skill")
	}
	again, err := Plan(o)
	if err != nil || len(again) != 0 {
		t.Fatalf("not idempotent: %v %+v", err, again)
	}
	o.Remove = true
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p), before) || string(get(t, instruction)) != "personal\n" {
		t.Fatal("remove lost personal content")
	}
}

func TestCodexAccessPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, platform, config string
		add                    bool
	}{
		{"linux", "linux", "[windows]\nsandbox = 'unelevated'\n", true},
		{"windows elevated", "windows", "[windows]\nsandbox = 'elevated'\n", true},
		{"windows missing", "windows", "", false},
		{"windows unknown", "windows", "windows = {sandbox = 'future'}\n", false},
		{"windows dotted", "windows", "windows.sandbox = 'unelevated'\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := fixture(t)
			o.Platform = tc.platform
			o.Agents = []string{"codex"}
			p := filepath.Join(o.CodexHome, "config.toml")
			put(t, p, []byte(tc.config))
			edits, err := Plan(o)
			if err != nil {
				t.Fatal(err)
			}
			access, err := PlannedAccess(o, edits)
			if err != nil {
				t.Fatal(err)
			}
			if access["codex"].RootConfigured != tc.add || access["codex"].Verified {
				t.Fatalf("misleading access: %+v", access)
			}
			if err = Apply(edits); err != nil {
				t.Fatal(err)
			}
			var parsed map[string]any
			if err = toml.Unmarshal(get(t, p), &parsed); err != nil {
				t.Fatal(err)
			}
			workspace, _ := parsed["sandbox_workspace_write"].(map[string]any)
			roots, _ := workspace["writable_roots"].([]any)
			if tc.add {
				if len(roots) != 1 || roots[0] != o.DataHome {
					t.Fatalf("missing narrow root: %+v", roots)
				}
			} else if !bytes.Equal(get(t, p), []byte(tc.config)) {
				t.Fatal("changed incompatible config")
			}
		})
	}
}

func TestWindowsPreservesExistingRootOwnership(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(fmt.Sprint(owned), func(t *testing.T) {
			o := fixture(t)
			o.Agents = []string{"codex"}
			p := filepath.Join(o.CodexHome, "config.toml")
			before := []byte("[windows]\nsandbox = 'unelevated'\n")
			if !owned {
				var err error
				before, err = mergeTOML(before, o.DataHome)
				if err != nil {
					t.Fatal(err)
				}
				o.Platform = "windows"
			}
			put(t, p, before)
			edits, err := Plan(o)
			if err != nil {
				t.Fatal(err)
			}
			if err = Apply(edits); err != nil {
				t.Fatal(err)
			}
			o.Platform = "windows"
			manifestPath := filepath.Join(o.CodexHome, "conductor-setup.json")
			journal := get(t, manifestPath)
			edits, err = Plan(o)
			if err != nil || len(edits) != 0 {
				t.Fatalf("repeat: %v %+v", err, edits)
			}
			access, err := PlannedAccess(o, edits)
			if err != nil {
				t.Fatal(err)
			}
			if !access["codex"].RootConfigured || access["codex"].Mode != "command_approval" {
				t.Fatalf("lost existing root status: %+v", access)
			}
			if !bytes.Equal(journal, get(t, manifestPath)) {
				t.Fatal("lost ownership journal")
			}
			o.Remove = true
			edits, err = Plan(o)
			if err != nil {
				t.Fatal(err)
			}
			if err = Apply(edits); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, get(t, p)) {
				t.Fatal("lost caller roots or retained owned root")
			}
		})
	}
}

func TestWindowsDoesNotRepairPendingIncompatibleRoot(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	p := filepath.Join(o.CodexHome, "config.toml")
	before := []byte("[windows]\nsandbox = 'unelevated'\n")
	put(t, p, before)
	edits, err := Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits[:1]); err != nil {
		t.Fatal(err)
	}
	o.Platform = "windows"
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p), before) {
		t.Fatal("repair added incompatible root")
	}
	o.Remove = true
	edits, err = Plan(o)
	if err != nil {
		t.Fatal(err)
	}
	if err = Apply(edits); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p), before) {
		t.Fatal("remove changed original config")
	}
}
