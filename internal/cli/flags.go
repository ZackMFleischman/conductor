package cli

import (
	"github.com/ZackMFleischman/conductor/internal/core"
	"strings"
)

type Flags struct {
	Values      map[string]string
	Bools       map[string]bool
	Positionals []string
}

func Parse(args []string, valueFlags, boolFlags []string) (Flags, error) {
	f := Flags{Values: map[string]string{}, Bools: map[string]bool{}}
	vs := map[string]bool{"project": true}
	bs := map[string]bool{}
	for _, s := range valueFlags {
		vs[s] = true
	}
	for _, s := range boolFlags {
		bs[s] = true
	}
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			f.Positionals = append(f.Positionals, a)
			continue
		}
		kv := strings.SplitN(strings.TrimPrefix(a, "--"), "=", 2)
		k := kv[0]
		if seen[k] {
			return f, core.Fail("USAGE", "duplicate flag --"+k)
		}
		seen[k] = true
		if bs[k] {
			if len(kv) != 1 {
				return f, core.Fail("USAGE", "boolean flag has a value")
			}
			f.Bools[k] = true
			continue
		}
		if !vs[k] {
			return f, core.Fail("USAGE", "unknown flag --"+k)
		}
		var v string
		if len(kv) == 2 {
			v = kv[1]
		} else {
			i++
			if i < len(args) {
				v = args[i]
			}
		}
		if v == "" || strings.HasPrefix(v, "--") {
			return f, core.Fail("USAGE", "missing value for --"+k)
		}
		f.Values[k] = v
	}
	return f, nil
}
func (f Flags) Require(keys ...string) error {
	for _, k := range keys {
		if f.Values[k] == "" {
			return core.Fail("USAGE", "--"+k+" is required")
		}
	}
	return nil
}
func (f Flags) NoPositionals() error {
	if len(f.Positionals) > 0 {
		return core.Fail("USAGE", "unexpected positional argument")
	}
	return nil
}
