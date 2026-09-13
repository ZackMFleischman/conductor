// Package setup plans reversible user-level host integration. It never opens the registry.
package setup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Options struct {
	CommandAccess        string // Empty preserves the installed preference; otherwise host-default or require_escalated.
	Agents               []string
	Executable, DataHome string
	Remove               bool
	// Explicit roots make host installation testable without touching the real home.
	UserHome, CodexHome, ClaudeHome string
	SkillFiles                      map[string][]byte
	// Platform defaults to the running OS; explicit values support host-policy tests.
	Platform string
}

// Access describes configured intent, never proof that a host command can run.
type Access struct {
	Mode           string `json:"mode"`
	RootConfigured bool   `json:"root_configured"`
	Verified       bool   `json:"verified"`
	Warning        string `json:"warning,omitempty"`
}

func codexAccess(b []byte, o Options) (Access, error) {
	var config map[string]any
	if err := toml.Unmarshal(b, &config); err != nil {
		return Access{}, fmt.Errorf("invalid Codex config TOML")
	}
	a := Access{Mode: "writable_root"}
	if workspace, ok := config["sandbox_workspace_write"].(map[string]any); ok {
		if roots, ok := workspace["writable_roots"].([]any); ok {
			for _, root := range roots {
				if root == o.DataHome {
					a.RootConfigured = true
				}
			}
		}
	}
	platform := o.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform == "windows" {
		windows, _ := config["windows"].(map[string]any)
		// Only explicit elevated mode opts into split writable roots. Missing or
		// unknown modes have no established compatibility and remain unchanged.
		if windows["sandbox"] != "elevated" {
			a.Mode = "command_approval"
			a.Warning = "Windows Codex split writable roots are incompatible with unelevated sandboxing, and compatibility is unknown for unspecified modes. Setup does not add a global writable root in this mode. Use normal per-command approval for Conductor access; existing roots and security settings are preserved. Access remains unverified."
		}
	}
	return a, nil
}

// PlannedAccess reports the effective config after the preview, without exposing
// personal config contents. Removal has no installed-access claim.
func PlannedAccess(o Options, edits []Edit) (map[string]Access, error) {
	result := map[string]Access{}
	if o.Remove {
		return result, nil
	}
	for _, host := range o.Agents {
		if host != "codex" {
			result[host] = Access{Mode: "additional_directory", RootConfigured: true}
			continue
		}
		path := filepath.Join(o.CodexHome, "config.toml")
		b, err := read(path)
		if err != nil {
			return nil, err
		}
		for _, edit := range edits {
			if edit.Path == path {
				b = edit.After
			}
		}
		a, err := codexAccess(b, o)
		if err != nil {
			return nil, err
		}
		result[host] = a
	}
	return result, nil
}

type Edit struct {
	Path        string `json:"path"`
	BeforeHash  string `json:"before_hash"`
	Before      []byte `json:"-"`
	After       []byte `json:"-"`
	Ownership   string `json:"ownership"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type record struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Before   []byte `json:"before,omitempty"`
	After    []byte `json:"after"`
	Hash     string `json:"hash"`
	Original []byte `json:"original,omitempty"`
	Desired  []byte `json:"desired,omitempty"`
	Previous []byte `json:"previous,omitempty"`
}
type manifest struct {
	CommandAccess string   `json:"command_access,omitempty"`
	Version       int      `json:"version"`
	Executable    string   `json:"executable"`
	DataHome      string   `json:"data_home"`
	Records       []record `json:"records"`
	Removing      bool     `json:"removing,omitempty"`
	Removal       []record `json:"removal,omitempty"`
}

func digest(b []byte) string {
	if b == nil {
		return "absent"
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func read(path string) ([]byte, error) {
	// Validate every existing ancestor even when the leaf (or intermediate
	// directories) does not exist yet. Both Plan and Apply use this check.
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		fi, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil && (fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir()) {
			return nil, fmt.Errorf("unsupported linked or non-directory parent: %s", p)
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	info, e := os.Lstat(path)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("unsupported nonregular path: %s", path)
	}
	if info.Mode().Perm()&0200 == 0 {
		return nil, fmt.Errorf("read-only path: %s", path)
	}
	return os.ReadFile(path)
}
func edit(path string, b, a []byte, ownership string) Edit {
	action := "replace"
	if b == nil {
		action = "create"
	}
	if a == nil {
		action = "remove"
	}
	description := "Update Conductor-owned work skill"
	switch filepath.Base(path) {
	case "conductor-setup.json":
		description = "Update private Conductor ownership journal"
	case "AGENTS.md", "AGENTS.override.md", "CLAUDE.md":
		description = "Update only the managed Conductor bootstrap block"
	case "config.toml":
		description = "Update only Conductor's data-home entry in sandbox_workspace_write.writable_roots"
	case "settings.json":
		description = "Update only Conductor's data-home entry in permissions.additionalDirectories"
	}
	return Edit{path, digest(b), b, a, ownership, action, description}
}

// Plan is read-only. A per-host journal is installed first so interrupted writes
// can be repaired from the exact before/after fragments on a later invocation.
func Plan(o Options) ([]Edit, error) {
	if err := validateCommandAccess(o); err != nil {
		return nil, err
	}
	if len(o.Agents) == 0 {
		return nil, fmt.Errorf("select codex and/or claude")
	}
	roots := []string{o.Executable, o.DataHome, o.UserHome}
	for _, host := range o.Agents {
		if host == "codex" {
			roots = append(roots, o.CodexHome)
		}
		if host == "claude" {
			roots = append(roots, o.ClaudeHome)
		}
	}
	for _, p := range roots {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("setup paths must be absolute: %q", p)
		}
		if strings.ContainsAny(p, "\r\n\x00") {
			return nil, fmt.Errorf("invalid setup path")
		}
	}
	if !o.Remove && len(o.SkillFiles["SKILL.md"]) == 0 {
		return nil, fmt.Errorf("packaged work skill missing")
	}
	var edits []Edit
	seen := map[string]bool{}
	for _, host := range o.Agents {
		if seen[host] {
			return nil, fmt.Errorf("duplicate host %q", host)
		}
		seen[host] = true
		root := o.CodexHome
		if host == "claude" {
			root = o.ClaudeHome
		} else if host != "codex" {
			return nil, fmt.Errorf("unsupported host %q", host)
		}
		mp := filepath.Join(root, "conductor-setup.json")
		mb, e := read(mp)
		if e != nil {
			return nil, e
		}
		var m manifest
		if mb != nil {
			if e = json.Unmarshal(mb, &m); e != nil || m.Version != 1 {
				return nil, fmt.Errorf("invalid ownership manifest %s", mp)
			}
			if !o.Remove && (m.Executable != o.Executable || m.DataHome != o.DataHome) {
				return nil, fmt.Errorf("setup paths changed; remove the previous integration before reinstalling")
			}
			if m.Removing {
				if !o.Remove {
					return nil, fmt.Errorf("removal in progress; finish setup --remove before installing")
				}
				for _, r := range m.Removal {
					if !allowed(r.Path, o, host) || r.Hash != digest(r.After) {
						return nil, fmt.Errorf("invalid removal journal")
					}
					b, err := read(r.Path)
					if err != nil {
						return nil, err
					}
					if digest(b) == digest(r.After) {
						continue
					}
					if digest(b) != digest(r.Before) {
						return nil, fmt.Errorf("removal conflict: %s changed after preview", r.Path)
					}
					edits = append(edits, edit(r.Path, b, r.After, r.Hash))
				}
				edits = append(edits, edit(mp, mb, nil, "conductor manifest v1"))
				continue
			}
		} else {
			if o.Remove {
				continue
			}
			m = manifest{Version: 1, Executable: o.Executable, DataHome: o.DataHome, CommandAccess: o.CommandAccess}
			if e = buildRecords(&m, o, host); e != nil {
				return nil, e
			}
			after, _ := json.MarshalIndent(m, "", "  ")
			after = append(after, '\n')
			edits = append(edits, edit(mp, nil, after, "conductor manifest v1"))
		}
		if mb != nil && !o.Remove {
			changed, err := upgradeRecords(&m, o, host)
			if err != nil {
				return nil, err
			}
			if changed {
				after, err := json.MarshalIndent(m, "", "  ")
				if err != nil {
					return nil, err
				}
				edits = append(edits, edit(mp, mb, append(after, '\n'), "conductor upgrade journal v1"))
			}
		}
		hostStart := len(edits)
		for _, r := range m.Records {
			if r.Hash != digest(r.After) || !allowed(r.Path, o, host) {
				return nil, fmt.Errorf("invalid owned record %s", r.Path)
			}
			b, e := read(r.Path)
			if e != nil {
				return nil, e
			}
			var a []byte
			preserved := false
			switch r.Kind {
			case "file":
				if !bytes.Equal(b, r.After) && !(r.Previous != nil && bytes.Equal(b, r.Previous)) && !(bytes.Equal(b, r.Before) && (b == nil) == (r.Before == nil)) {
					if !o.Remove && r.Previous != nil && preservedAddition(r.Previous, b, r.After) {
						preserved = true
					} else {
						return nil, fmt.Errorf("owned content conflict: %s", r.Path)
					}
				}
				if o.Remove {
					a = r.Before
				} else {
					a = r.After
				}
			case "fragment":
				if b != nil {
					if host == "codex" {
						var v map[string]any
						if err := toml.Unmarshal(b, &v); err != nil {
							return nil, fmt.Errorf("invalid current config %s: %w", r.Path, err)
						}
					} else {
						d := json.NewDecoder(bytes.NewReader(b))
						if err := uniqueJSON(d); err != nil {
							return nil, fmt.Errorf("invalid current config %s: %w", r.Path, err)
						}
						if _, err := d.Token(); err != io.EOF {
							return nil, fmt.Errorf("invalid trailing config data: %s", r.Path)
						}
					}
				}
				if bytes.Count(b, r.After) == 1 {
					if o.Remove {
						a = bytes.Replace(b, r.After, r.Before, 1)
					} else {
						a = b
					}
				} else if digest(b) == digest(r.Original) { // first installation or interrupted removal
					if o.Remove {
						a = b
					} else {
						a = r.Desired
					}
				} else {
					return nil, fmt.Errorf("owned fragment conflict: %s", r.Path)
				}
			case "append":
				if err := validateBootstrap(b, r); err != nil {
					return nil, err
				}
				if bytes.Count(b, r.After) == 1 {
					if o.Remove {
						a = bytes.Replace(b, r.After, nil, 1)
					} else {
						a = b
					}
				} else if len(r.Previous) > 0 && bytes.Count(b, r.Previous) == 1 {
					if o.Remove {
						a = bytes.Replace(b, r.Previous, nil, 1)
					} else {
						a = bytes.Replace(b, r.Previous, r.After, 1)
					}
				} else if digest(b) == digest(r.Before) {
					if o.Remove {
						a = b
					} else {
						a = append(append([]byte{}, b...), r.After...)
					}
				} else {
					return nil, fmt.Errorf("owned bootstrap conflict: %s", r.Path)
				}
			default:
				return nil, fmt.Errorf("unsupported ownership record")
			}
			// Restore absent files only if their owned content was the entire file.
			if !o.Remove && host == "codex" && r.Path == filepath.Join(root, "config.toml") {
				access, err := codexAccess(b, o)
				if err != nil {
					return nil, err
				}
				if access.Mode == "command_approval" {
					// Keep the ownership journal for removal, but never repair a
					// missing root into an incompatible Windows configuration.
					a = b
				}
			}
			if o.Remove && r.Before == nil && len(a) == 0 {
				a = nil
			}
			if digest(b) != digest(a) {
				next := edit(r.Path, b, a, r.Hash)
				if preserved {
					next.Description = "Safely merge preserved Conductor skill guidance"
				}
				edits = append(edits, next)
			}
		}
		if o.Remove {
			// Journal the exact removal snapshots (including unrelated personal
			// edits) before touching files, so a restart recognizes completed steps.
			hostEdits := append([]Edit(nil), edits[hostStart:]...)
			m.Removing = true
			for _, e := range hostEdits {
				m.Removal = append(m.Removal, record{Path: e.Path, Before: e.Before, After: e.After, Hash: digest(e.After)})
			}
			journal, _ := json.MarshalIndent(m, "", "  ")
			journal = append(journal, '\n')
			edits = append(edits[:hostStart], edit(mp, mb, journal, "conductor removal journal v1"))
			edits = append(edits, hostEdits...)
			edits = append(edits, edit(mp, journal, nil, "conductor manifest v1"))
		}
	}
	return edits, nil
}
func allowed(path string, o Options, host string) bool {
	root := o.CodexHome
	skill := filepath.Join(o.UserHome, ".agents", "skills")
	names := []string{"AGENTS.md", "AGENTS.override.md", "config.toml"}
	if host == "claude" {
		root = o.ClaudeHome
		skill = filepath.Join(root, "skills")
		names = []string{"CLAUDE.md", "settings.json"}
	}
	for _, n := range names {
		if path == filepath.Join(root, n) {
			return true
		}
	}
	rel, e := filepath.Rel(skill, path)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	return e == nil && len(parts) > 1 && knownSkill(parts[0]) && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func buildRecords(m *manifest, o Options, host string) error {
	root := o.CodexHome
	instruction := filepath.Join(root, "AGENTS.md")
	if host == "codex" {
		override := filepath.Join(root, "AGENTS.override.md")
		b, e := read(override)
		if e != nil {
			return e
		}
		if len(b) > 0 {
			instruction = override
		}
	} else {
		root = o.ClaudeHome
		instruction = filepath.Join(root, "CLAUDE.md")
	}
	b, e := read(instruction)
	if e != nil {
		return e
	}
	if bytes.Contains(b, []byte("<!-- conductor:")) {
		return fmt.Errorf("unowned Conductor marker: %s", instruction)
	}
	block := bootstrap(o, m.CommandAccess)
	m.Records = append(m.Records, record{Path: instruction, Kind: "append", Before: b, After: block, Hash: digest(block)})
	keys := make([]string, 0, len(o.SkillFiles))
	for k := range o.SkillFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p, err := skillDestination(k, o, host)
		if err != nil || !allowed(p, o, host) {
			return fmt.Errorf("invalid skill path %q", k)
		}
		old, e := read(p)
		if e != nil {
			return e
		}
		if old != nil {
			return fmt.Errorf("unowned skill exists: %s", p)
		}
		a := o.SkillFiles[k]
		m.Records = append(m.Records, record{Path: p, Kind: "file", After: a, Hash: digest(a)})
	}
	return addPermissionRecord(m, o, host)
}

func addPermissionRecord(m *manifest, o Options, host string) error {
	root := o.CodexHome
	if host == "claude" {
		root = o.ClaudeHome
	}
	config := filepath.Join(root, "config.toml")
	if host == "claude" {
		config = filepath.Join(root, "settings.json")
	}
	for _, r := range m.Records {
		if r.Path == config {
			return nil
		}
	}
	b, e := read(config)
	if e != nil {
		return e
	}
	var a []byte
	if host == "codex" {
		access, err := codexAccess(b, o)
		if err != nil {
			return err
		}
		if access.Mode == "command_approval" {
			return nil
		}
		a, e = mergeTOML(b, o.DataHome)
	} else {
		a, e = mergeJSON(b, o.DataHome)
	}
	if e != nil {
		return fmt.Errorf("unsupported config %s: %w", config, e)
	}
	if !bytes.Equal(a, b) {
		start, end := 0, 0
		for start < len(b) && start < len(a) && b[start] == a[start] {
			start++
		}
		for end < len(b)-start && end < len(a)-start && b[len(b)-1-end] == a[len(a)-1-end] {
			end++
		}
		old, owned := b[start:len(b)-end], a[start:len(a)-end]
		m.Records = append(m.Records, record{Path: config, Kind: "fragment", Before: old, After: owned, Hash: digest(owned), Original: b, Desired: a})
	}
	return nil
}
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// Apply checks each preview hash immediately before replacing that file through
// a temporary sibling. Errors identify partial completion; rerun Plan to repair.
func Apply(edits []Edit) error {
	for i, e := range edits {
		if err := applyOne(e); err != nil {
			return fmt.Errorf("setup completed %d of %d edits; %s: %w; rerun preview to inspect/repair", i, len(edits), e.Path, err)
		}
	}
	return nil
}
func applyOne(e Edit) error {
	b, err := read(e.Path)
	if err != nil {
		return err
	}
	if digest(b) != e.BeforeHash {
		return fmt.Errorf("preview conflict: file changed")
	}
	if e.After == nil {
		if b == nil {
			return nil
		}
		return os.Remove(e.Path)
	}
	if err = os.MkdirAll(filepath.Dir(e.Path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(e.Path), ".conductor-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	mode := os.FileMode(0600)
	if info, err := os.Stat(e.Path); err == nil {
		mode = info.Mode().Perm()
	}
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(e.After)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	// Recheck after writing the sibling too, narrowing concurrent-edit exposure.
	b, err = read(e.Path)
	if err != nil {
		return err
	}
	if digest(b) != e.BeforeHash {
		return fmt.Errorf("preview conflict: file changed")
	}
	return os.Rename(tmp, e.Path)
}

func mergeJSON(b []byte, home string) ([]byte, error) {
	if b == nil {
		b = []byte("{}\n")
	}
	// Token walk rejects duplicate object keys before ordinary decoding.
	d := json.NewDecoder(bytes.NewReader(b))
	if err := uniqueJSON(d); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON data")
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil || v == nil {
		return nil, fmt.Errorf("settings must be an object")
	}
	p := map[string]any{}
	if x, ok := v["permissions"]; ok {
		var valid bool
		p, valid = x.(map[string]any)
		if !valid {
			return nil, fmt.Errorf("permissions must be an object")
		}
	}
	dirs := []any{}
	if x, ok := p["additionalDirectories"]; ok {
		var valid bool
		dirs, valid = x.([]any)
		if !valid {
			return nil, fmt.Errorf("additionalDirectories must be an array")
		}
	}
	for _, x := range dirs {
		s, ok := x.(string)
		if !ok {
			return nil, fmt.Errorf("additionalDirectories must contain strings")
		}
		if s == home {
			return b, nil
		}
	}
	encoded, _ := json.Marshal(append(dirs, home))
	root, closeAt, err := jsonMembers(b)
	if err != nil {
		return nil, err
	}
	ps, ok := root["permissions"]
	if !ok {
		return addJSONMember(b, closeAt, len(root) > 0, "permissions", []byte(`{"additionalDirectories":`+string(encoded)+`}`)), nil
	}
	pm, pc, err := jsonMembers(b[ps[0]:ps[1]])
	if err != nil {
		return nil, err
	}
	if ds, ok := pm["additionalDirectories"]; ok {
		start, end := ps[0]+ds[0], ps[0]+ds[1]
		a := append([]byte{}, b[:start]...)
		a = append(a, encoded...)
		return append(a, b[end:]...), nil
	}
	return addJSONMember(b, ps[0]+pc, len(pm) > 0, "additionalDirectories", encoded), nil
}

// Locate values using the standard JSON decoder, retaining all other bytes.
func jsonMembers(b []byte) (map[string][2]int, int, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	if _, e := d.Token(); e != nil {
		return nil, 0, e
	}
	m := map[string][2]int{}
	for d.More() {
		k, e := d.Token()
		if e != nil {
			return nil, 0, e
		}
		start := int(d.InputOffset())
		for start < len(b) && (b[start] == ':' || b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
			start++
		}
		var raw json.RawMessage
		if e = d.Decode(&raw); e != nil {
			return nil, 0, e
		}
		m[k.(string)] = [2]int{start, int(d.InputOffset())}
	}
	end := int(d.InputOffset())
	for end < len(b) && b[end] != '}' {
		end++
	}
	return m, end, nil
}
func addJSONMember(b []byte, at int, comma bool, key string, value []byte) []byte {
	part := []byte(`"` + key + `": `)
	if comma {
		part = append([]byte(","), part...)
	}
	part = append(part, value...)
	a := append([]byte{}, b[:at]...)
	a = append(a, part...)
	return append(a, b[at:]...)
}
func uniqueJSON(d *json.Decoder) error {
	t, e := d.Token()
	if e != nil {
		return e
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return e
			}
			s, ok := k.(string)
			if !ok || seen[s] {
				return fmt.Errorf("duplicate or invalid JSON key")
			}
			seen[s] = true
			if e = uniqueJSON(d); e != nil {
				return e
			}
		}
	case '[':
		for d.More() {
			if e = uniqueJSON(d); e != nil {
				return e
			}
		}
	default:
		return fmt.Errorf("invalid JSON")
	}
	_, e = d.Token()
	return e
}
func mergeTOML(b []byte, home string) ([]byte, error) {
	// The narrow textual editor must produce exactly the expected semantic
	// change. In particular, lines inside multiline strings are never settings.
	a, err := mergeTOMLCandidate(b, home)
	if err != nil {
		return nil, err
	}
	var expected, actual map[string]any
	if err = toml.Unmarshal(b, &expected); err != nil {
		return nil, err
	}
	if expected == nil {
		expected = map[string]any{}
	}
	section, _ := expected["sandbox_workspace_write"].(map[string]any)
	if section == nil {
		section = map[string]any{}
		expected["sandbox_workspace_write"] = section
	}
	roots, _ := section["writable_roots"].([]any)
	found := false
	for _, r := range roots {
		if r == home {
			found = true
		}
	}
	if !found {
		section["writable_roots"] = append(roots, home)
	}
	if err = toml.Unmarshal(a, &actual); err != nil {
		return nil, fmt.Errorf("ambiguous TOML edit: resulting configuration is invalid")
	}
	if !reflect.DeepEqual(expected, actual) {
		return nil, fmt.Errorf("ambiguous TOML layout: cannot isolate writable_roots without changing other values")
	}
	return a, nil
}
func mergeTOMLCandidate(b []byte, home string) ([]byte, error) {
	var v map[string]any
	if e := toml.Unmarshal(b, &v); e != nil {
		return nil, e
	}
	section, exists := v["sandbox_workspace_write"]
	var dirs []any
	hasRoots := false
	if exists {
		m, ok := section.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("sandbox_workspace_write must be a table")
		}
		if x, ok := m["writable_roots"]; ok {
			hasRoots = true
			var valid bool
			dirs, valid = x.([]any)
			if !valid {
				return nil, fmt.Errorf("writable_roots must be an array")
			}
		}
	}
	for _, x := range dirs {
		s, ok := x.(string)
		if !ok {
			return nil, fmt.Errorf("writable_roots must contain strings")
		}
		if s == home {
			return b, nil
		}
	}
	raw, _ := json.Marshal(append(dirs, home))
	line := []byte("writable_roots = " + string(raw) + "\n")
	if !exists {
		return append(append(append([]byte{}, b...), []byte("\n[sandbox_workspace_write]\n")...), line...), nil
	}
	lines := bytes.SplitAfter(b, []byte("\n"))
	offset := 0
	start := -1
	end := len(b)
	for _, l := range lines {
		s := strings.TrimSpace(strings.SplitN(string(l), "#", 2)[0])
		if s == "[sandbox_workspace_write]" {
			if start != -1 {
				return nil, fmt.Errorf("ambiguous table")
			}
			start = offset + len(l)
		} else if start >= 0 && strings.HasPrefix(s, "[") {
			end = offset
			break
		}
		offset += len(l)
	}
	if start < 0 {
		return nil, fmt.Errorf("use a standalone [sandbox_workspace_write] table; inline/dotted table editing unsupported")
	}
	sectionBytes := b[start:end]
	off := start
	for _, l := range bytes.SplitAfter(sectionBytes, []byte("\n")) {
		s := strings.TrimSpace(string(l))
		if strings.HasPrefix(s, "writable_roots") {
			eq := bytes.IndexByte(l, '=')
			if eq < 0 || strings.TrimSpace(string(l[:eq])) != "writable_roots" {
				return nil, fmt.Errorf("unsupported writable_roots syntax")
			}
			valueStart := off + eq + 1
			for valueStart < len(b) && (b[valueStart] == ' ' || b[valueStart] == '\t') {
				valueStart++
			}
			valueEnd, e := arrayEnd(b, valueStart)
			if e != nil {
				return nil, e
			}
			a := append([]byte{}, b[:valueStart]...)
			a = append(a, raw...)
			return append(a, b[valueEnd:]...), nil
		}
		off += len(l)
	}
	if hasRoots {
		return nil, fmt.Errorf("unsupported writable_roots key syntax")
	}
	a := append([]byte{}, b[:start]...)
	if len(a) > 0 && a[len(a)-1] != '\n' {
		a = append(a, '\n')
	}
	a = append(a, line...)
	return append(a, b[start:]...), nil
}
func arrayEnd(b []byte, start int) (int, error) {
	if start >= len(b) || b[start] != '[' {
		return 0, fmt.Errorf("unsupported array syntax")
	}
	var quote byte
	escaped, comment := false, false
	for i := start + 1; i < len(b); i++ {
		c := b[i]
		if comment {
			if c == '\n' {
				comment = false
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if quote == '"' && c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '#' {
			comment = true
			continue
		}
		if c == '"' || c == '\'' {
			if i+2 < len(b) && b[i+1] == c && b[i+2] == c {
				return 0, fmt.Errorf("multiline array strings unsupported")
			}
			quote = c
			continue
		}
		if c == ']' {
			return i + 1, nil
		}
		if c == '[' {
			return 0, fmt.Errorf("nested array unsupported")
		}
	}
	return 0, fmt.Errorf("unterminated array")
}

func bootstrap(o Options, access string) []byte {
	block := []byte("\n<!-- conductor:start -->\nBefore repository work, run the installed executable with `context --if-registered --json` and the configured `--home`.\nPowerShell: `& " + psQuote(o.Executable) + " --home " + psQuote(o.DataHome) + " context --if-registered --json`\nBash: `" + shQuote(o.Executable) + " --home " + shQuote(o.DataHome) + " context --if-registered --json`\nIf registered, use the conductor-work skill to track authorized work and report workflow problems. If registered=false, continue normally. REGISTRY_UNAVAILABLE means registration is unknown: surface the error; never initialize a fallback or claim updates were saved.\n<!-- conductor:end -->\n")
	if access == "require_escalated" {
		rule := "Windows Codex installation preference: every exec_command invocation of " + psQuote(o.Executable) + " with --home " + psQuote(o.DataHome) + " uses sandbox_permissions: require_escalated from the outset, including discovery and version checks. Do not first retry ordinary access probes. This preference applies only to this executable/home in Windows Codex; other commands and hosts retain their own policy. It changes no host security settings and is not proof of access.\n"
		block = bytes.Replace(block, []byte("Before repository work"), []byte(rule+"Before repository work"), 1)
	}
	return block
}
