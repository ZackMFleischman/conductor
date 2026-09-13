package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
)

// JSON keeps the baseline compilable before the option exists.
func preference(t *testing.T, o Options, value string) Options {
	t.Helper()
	b, _ := json.Marshal(o)
	var fields map[string]any
	json.Unmarshal(b, &fields)
	fields["CommandAccess"] = value
	b, _ = json.Marshal(fields)
	json.Unmarshal(b, &o)
	return o
}
func TestCommandAccessBootstrapLifecycle(t *testing.T) {
	o := fixture(t)
	o.Agents = []string{"codex"}
	o.Platform = "windows"
	p := filepath.Join(o.CodexHome, "AGENTS.md")
	personal := []byte("personal before\n")
	put(t, p, personal)
	apply := func() {
		t.Helper()
		edits, e := Plan(o)
		if e != nil {
			t.Fatal(e)
		}
		if e = Apply(edits); e != nil {
			t.Fatal(e)
		}
	}
	apply() // upgrade an existing default installation
	o = preference(t, o, "require_escalated")
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if len(edits) < 2 {
		t.Fatal("missing preference journal and bootstrap upgrade")
	}
	if e = Apply(edits[:1]); e != nil {
		t.Fatal(e)
	} // crash after journal
	o = preference(t, o, "")
	apply() // omitted resumes and preserves
	b := get(t, p)
	rule := bytes.Index(b, []byte("sandbox_permissions: require_escalated"))
	discovery := bytes.Index(b, []byte("context --if-registered"))
	if rule < 0 || rule >= discovery {
		t.Fatalf("access preference absent before discovery: %s", b)
	}
	if !bytes.Contains(b, []byte(o.Executable)) || !bytes.Contains(b, []byte(o.DataHome)) {
		t.Fatal("scope missing")
	}
	put(t, p, append(b, []byte("personal after\n")...))
	apply()
	edits, e = Plan(o)
	if e != nil || len(edits) != 0 {
		t.Fatalf("not idempotent: %v %d", e, len(edits))
	}
	o = preference(t, o, "host-default")
	apply()
	if bytes.Contains(get(t, p), []byte("require_escalated")) {
		t.Fatal("explicit reset ignored")
	}
	o = preference(t, o, "")
	o.Remove = true
	apply()
	if !bytes.Equal(get(t, p), append(personal, []byte("personal after\n")...)) {
		t.Fatal("personal text lost")
	}
}
func TestCommandAccessUnsupported(t *testing.T) {
	for _, host := range []string{"codex", "claude"} {
		o := fixture(t)
		o.Agents = []string{host}
		o = preference(t, o, "require_escalated")
		if _, e := Plan(o); e == nil {
			t.Fatalf("accepted unsupported %s on linux", host)
		}
	}
}

func TestCommandAccessUpgradeSafety(t *testing.T) {
	for _, stop := range []int{1, 2} {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			o := fixture(t)
			o.Platform = "windows"
			o.Agents = []string{"codex"}
			p := filepath.Join(o.CodexHome, "AGENTS.md")
			original := []byte("user\n")
			put(t, p, original)
			edits, e := Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits); e != nil {
				t.Fatal(e)
			}
			o.CommandAccess = "require_escalated"
			edits, e = Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits[:stop]); e != nil {
				t.Fatal(e)
			}
			if stop == 1 {
				o.CommandAccess = "host-default"
				if _, e = Plan(o); e == nil {
					t.Fatal("superseded interrupted upgrade")
				}
			}
			o.CommandAccess = ""
			o.Remove = true
			edits, e = Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits[:1]); e != nil {
				t.Fatal(e)
			}
			edits, e = Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits); e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(get(t, p), original) {
				t.Fatal("removal lost personal bytes")
			}
		})
	}
	for _, change := range []string{"edit", "duplicate", "preview"} {
		t.Run(change, func(t *testing.T) {
			o := fixture(t)
			o.Platform = "windows"
			o.Agents = []string{"codex"}
			edits, e := Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if e = Apply(edits); e != nil {
				t.Fatal(e)
			}
			p := filepath.Join(o.CodexHome, "AGENTS.md")
			b := get(t, p)
			o.CommandAccess = "require_escalated"
			edits, e = Plan(o)
			if e != nil {
				t.Fatal(e)
			}
			if change == "duplicate" {
				b = append(b, b...)
			} else {
				b = bytes.Replace(b, []byte("Before repository work"), []byte("Personal owned edit"), 1)
			}
			put(t, p, b)
			if change == "preview" {
				if e = Apply(edits); e == nil {
					t.Fatal("accepted stale preview")
				}
			} else {
				if _, e = Plan(o); e == nil {
					t.Fatal("accepted changed owned block")
				}
			}
			if !bytes.Equal(get(t, p), b) {
				t.Fatal("mutated personal edit")
			}
		})
	}
}

func TestCommandAccessFreshInstallation(t *testing.T) {
	o := fixture(t)
	o.Platform = "windows"
	o.Agents = []string{"codex"}
	o.CommandAccess = "require_escalated"
	config := filepath.Join(o.CodexHome, "config.toml")
	original := []byte("[windows]\nsandbox = \"unelevated\"\n")
	put(t, config, original)
	edits, e := Plan(o)
	if e != nil {
		t.Fatal(e)
	}
	if e = Apply(edits); e != nil {
		t.Fatal(e)
	}
	b := get(t, filepath.Join(o.CodexHome, "AGENTS.md"))
	if bytes.Index(b, []byte("sandbox_permissions: require_escalated")) < 0 || bytes.Index(b, []byte("sandbox_permissions: require_escalated")) > bytes.Index(b, []byte("context --if-registered")) {
		t.Fatal("fresh preference not before first discovery")
	}
	if !bytes.Equal(get(t, config), original) {
		t.Fatal("changed security settings")
	}
	var m manifest
	if e = json.Unmarshal(get(t, filepath.Join(o.CodexHome, "conductor-setup.json")), &m); e != nil || m.CommandAccess != "require_escalated" {
		t.Fatal("preference not persisted", e)
	}
	o.CommandAccess = ""
	o.Platform = "linux"
	if _, e = Plan(o); e == nil {
		t.Fatal("silently applied retained Windows preference to another host")
	}
}
