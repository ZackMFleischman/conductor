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
			if !bytes.Equal(current, r.After) && !(r.Previous != nil && bytes.Equal(current, r.Previous)) && !(current == nil && r.Before == nil) {
				return false, fmt.Errorf("owned content conflict: %s", dest)
			}
			if !bytes.Equal(next, r.After) {
				// Finish an interrupted earlier upgrade before introducing another version.
				if current != nil && !bytes.Equal(current, r.After) {
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
