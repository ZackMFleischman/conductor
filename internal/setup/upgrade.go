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
// already contains in the same order. CRLF and LF terminators are equivalent;
// replacements and deletions remain owned content conflicts, while unrelated
// canonical changes to baseline lines do not hide a preserved local addition.
func preservedAddition(base, current, next []byte) bool {
	if bytes.Equal(base, current) || bytes.Equal(current, next) {
		return false
	}
	baseLines := bytes.SplitAfter(base, []byte("\n"))
	additions, ok := addedLines(baseLines, bytes.SplitAfter(current, []byte("\n")))
	return ok && lineSubsequence(additions, unmatchedLines(baseLines, bytes.SplitAfter(next, []byte("\n"))))
}

func addedLines(base, current [][]byte) ([][]byte, bool) {
	index := 0
	var additions [][]byte
	for _, line := range current {
		if index < len(base) && sameLine(base[index], line) {
			index++
			continue
		}
		additions = append(additions, line)
	}
	return additions, index == len(base)
}

func unmatchedLines(base, next [][]byte) [][]byte {
	common := make([][]int, len(base)+1)
	for i := range common {
		common[i] = make([]int, len(next)+1)
	}
	for i := len(base) - 1; i >= 0; i-- {
		for j := len(next) - 1; j >= 0; j-- {
			if sameLine(base[i], next[j]) {
				common[i][j] = common[i+1][j+1] + 1
			} else if common[i+1][j] >= common[i][j+1] {
				common[i][j] = common[i+1][j]
			} else {
				common[i][j] = common[i][j+1]
			}
		}
	}
	var unmatched [][]byte
	for i, j := 0, 0; j < len(next); {
		if i < len(base) && sameLine(base[i], next[j]) {
			i++
			j++
		} else if i < len(base) && common[i+1][j] >= common[i][j+1] {
			i++
		} else {
			unmatched = append(unmatched, next[j])
			j++
		}
	}
	return unmatched
}

func lineSubsequence(need, have [][]byte) bool {
	index := 0
	for _, line := range have {
		if index < len(need) && sameLine(need[index], line) {
			index++
		}
	}
	return index == len(need)
}

func sameLine(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	if len(a) == 0 || len(b) == 0 || a[len(a)-1] != '\n' || b[len(b)-1] != '\n' {
		return false
	}
	a = a[:len(a)-1]
	b = b[:len(b)-1]
	if len(a) > 0 && a[len(a)-1] == '\r' {
		a = a[:len(a)-1]
	}
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}
	return bytes.Equal(a, b)
}

// Persist both known old and desired bytes before changing any installed file.
// The original Before remains intact so uninstall never restores an old skill.
func upgradeRecords(m *manifest, o Options, host string) (bool, map[string]bool, error) {
	adopted := map[string]bool{}
	index := map[string]int{}
	for i, r := range m.Records {
		if r.Hash != digest(r.After) || !allowed(r.Path, o, host) {
			return false, nil, fmt.Errorf("invalid owned record %s", r.Path)
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
		return false, nil, err
	}
	for _, key := range keys {
		dest, err := skillDestination(key, o, host)
		if err != nil {
			return false, nil, err
		}
		current, err := read(dest)
		if err != nil {
			return false, nil, err
		}
		next := o.SkillFiles[key]
		if i, ok := index[dest]; ok {
			r := &m.Records[i]
			preserved := preservedAddition(r.After, current, next) || (r.Previous != nil && preservedAddition(r.Previous, current, r.After))
			if !bytes.Equal(current, r.After) && !(r.Previous != nil && bytes.Equal(current, r.Previous)) && !(current == nil && r.Before == nil) && !bytes.Equal(current, next) && !preserved {
				return false, nil, fmt.Errorf("owned content conflict: %s", dest)
			}
			if !bytes.Equal(next, r.After) {
				// Finish an interrupted earlier upgrade before introducing another version.
				if current != nil && !bytes.Equal(current, r.After) && !bytes.Equal(current, next) && !preservedAddition(r.After, current, next) {
					return false, nil, fmt.Errorf("finish previous skill upgrade before upgrading again: %s", dest)
				}
				r.Previous = append([]byte(nil), r.After...)
				r.After = next
				r.Hash = digest(next)
				changed = true
			}
		} else {
			if current != nil && !bytes.Equal(current, next) {
				return false, nil, fmt.Errorf("unowned skill exists: %s", dest)
			}
			m.Records = append(m.Records, record{Path: dest, Kind: "file", Before: current, After: next, Hash: digest(next)})
			index[dest] = len(m.Records) - 1
			if current != nil {
				adopted[dest] = true
			}
			changed = true
		}
	}
	before := len(m.Records)
	if err := addPermissionRecord(m, o, host); err != nil {
		return false, nil, err
	}
	return changed || len(m.Records) != before, adopted, nil
}
