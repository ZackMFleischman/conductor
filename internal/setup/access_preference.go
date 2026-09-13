package setup

import (
	"bytes"
	"fmt"
	"runtime"
)

func validateCommandAccess(o Options) error {
	if o.CommandAccess == "" {
		return nil
	}
	if o.CommandAccess != "host-default" && o.CommandAccess != "require_escalated" {
		return fmt.Errorf("unsupported command access preference")
	}
	platform := o.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform != "windows" || len(o.Agents) != 1 || o.Agents[0] != "codex" {
		return fmt.Errorf("command access preference requires Windows Codex only")
	}
	if o.Remove {
		return fmt.Errorf("command access preference cannot be combined with removal")
	}
	return nil
}

// An exact owned block must be unique; another marker is never silently adopted.
func validateBootstrap(current []byte, r record) error {
	if digest(current) == digest(r.Before) {
		return nil
	}
	if bytes.Count(current, []byte("<!-- conductor:start -->")) != 1 || bytes.Count(current, []byte("<!-- conductor:end -->")) != 1 {
		return fmt.Errorf("owned bootstrap conflict: %s", r.Path)
	}
	if bytes.Count(current, r.After) == 1 || (len(r.Previous) > 0 && bytes.Count(current, r.Previous) == 1) {
		return nil
	}
	return fmt.Errorf("owned bootstrap conflict: %s", r.Path)
}

func upgradeBootstrap(m *manifest, o Options, host string) (bool, error) {
	next := m.CommandAccess
	if o.CommandAccess != "" {
		next = o.CommandAccess
	}
	if next != "" && next != "host-default" && next != "require_escalated" {
		return false, fmt.Errorf("invalid command access preference in manifest")
	}
	if next == "require_escalated" {
		scoped := o
		scoped.CommandAccess, scoped.Agents = next, []string{host}
		if err := validateCommandAccess(scoped); err != nil {
			return false, err
		}
	}
	changed := next != m.CommandAccess
	for i := range m.Records {
		r := &m.Records[i]
		if r.Kind != "append" {
			continue
		}
		current, err := read(r.Path)
		if err != nil {
			return false, err
		}
		if err = validateBootstrap(current, *r); err != nil {
			return false, err
		}
		desired := bootstrap(o, next)
		if !bytes.Equal(desired, r.After) {
			// Complete the recorded transition before introducing another desired block.
			if bytes.Count(current, r.After) != 1 {
				return false, fmt.Errorf("finish previous bootstrap upgrade before upgrading again: %s", r.Path)
			}
			r.Previous = append([]byte(nil), r.After...)
			r.After = desired
			r.Hash = digest(desired)
			changed = true
		}
	}
	m.CommandAccess = next
	return changed, nil
}
