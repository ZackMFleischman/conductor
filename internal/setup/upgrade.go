package setup

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func knownSkill(name string) bool {
	switch name {
	case "conductor-work", "conductor-onboard", "conductor-plan", "conductor-worker", "conductor-orchestrator", "conductor-retrospective", "conductor-workflow-improver":
		return true
	}
	return false
}

func skillDestination(key string, o Options, host string) (string, error) {
	if strings.ContainsAny(key, "\\:\x00\r\n") || strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("invalid skill path %q", key)
	}
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid skill path %q", key)
		}
	}
	if !knownSkill(parts[0]) {
		key = "conductor-work/" + key
	}
	root := filepath.Join(o.UserHome, ".agents", "skills")
	if host == "claude" {
		root = filepath.Join(o.ClaudeHome, "skills")
	}
	return filepath.Join(root, filepath.FromSlash(key)), nil
}

// preservedAddition accepts only line additions that the next canonical file
// already contains in the same order. Replacements and deletions remain owned
// content conflicts, so a preview cannot silently overwrite a local edit.
func preservedAddition(base, current, next []byte) bool {
	if bytes.Equal(base, current) || bytes.Equal(current, next) {
		return false
	}
	return lineSubsequence(bytes.SplitAfter(base, []byte("\n")), bytes.SplitAfter(current, []byte("\n"))) &&
		lineSubsequence(bytes.SplitAfter(current, []byte("\n")), bytes.SplitAfter(next, []byte("\n")))
}

func lineSubsequence(need, have [][]byte) bool {
	index := 0
	for _, line := range have {
		if index < len(need) && bytes.Equal(need[index], line) {
			index++
		}
	}
	return index == len(need)
}

// Persist both known old and desired bytes before changing any installed file.
// The original Before remains intact so uninstall never restores an old skill.
func upgradeRecords(m *manifest, o Options, host string) (bool, error) {
	index := map[string]int{}
	for i, r := range m.Records {
		if r.Hash != digest(r.After) || !allowed(r.Path, o, host) {
			return false, fmt.Errorf("invalid owned record %s", r.Path)
		}
		if r.Kind == "file" {
			index[r.Path] = i
		}
	}
	keys := make([]string, 0, len(o.SkillFiles))
	for k := range o.SkillFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	changed, err := upgradeBootstrap(m, o, host)
	if err != nil {
		return false, err
	}
	for _, key := range keys {
		dest, err := skillDestination(key, o, host)
		if err != nil {
			return false, err
		}
		current, err := read(dest)
		if err != nil {
			return false, err
		}
		next := o.SkillFiles[key]
		if i, ok := index[dest]; ok {
			r := &m.Records[i]
			preserved := preservedAddition(r.After, current, next) || (r.Previous != nil && preservedAddition(r.Previous, current, r.After))
			if !bytes.Equal(current, r.After) && !(r.Previous != nil && bytes.Equal(current, r.Previous)) && !(current == nil && r.Before == nil) && !bytes.Equal(current, next) && !preserved {
				return false, fmt.Errorf("owned content conflict: %s", dest)
			}
			if !bytes.Equal(next, r.After) {
				// Finish an interrupted earlier upgrade before introducing another version.
				if current != nil && !bytes.Equal(current, r.After) && !bytes.Equal(current, next) && !preservedAddition(r.After, current, next) {
					return false, fmt.Errorf("finish previous skill upgrade before upgrading again: %s", dest)
				}
				r.Previous = append([]byte(nil), r.After...)
				r.After = next
				r.Hash = digest(next)
				changed = true
			}
		} else {
			if current != nil {
				return false, fmt.Errorf("unowned skill exists: %s", dest)
			}
			m.Records = append(m.Records, record{Path: dest, Kind: "file", After: next, Hash: digest(next)})
			index[dest] = len(m.Records) - 1
			changed = true
		}
	}
	before := len(m.Records)
	if err := addPermissionRecord(m, o, host); err != nil {
		return false, err
	}
	return changed || len(m.Records) != before, nil
}
