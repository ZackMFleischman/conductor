package cli

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/store"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type Env struct {
	CWD, Home string
	Out, Err  io.Writer
}
type Handler func(context.Context, Env, []string) (any, error)

var handlers = map[string]Handler{}

func Register(name string, h Handler) {
	if _, ok := handlers[name]; ok {
		panic("duplicate command: " + name)
	}
	handlers[name] = h
}
func ResolveHome(explicit string) (string, error) {
	h := explicit
	if h == "" {
		h = os.Getenv("CONDUCTOR_HOME")
	}
	if h == "" {
		if runtime.GOOS == "windows" {
			h = os.Getenv("LOCALAPPDATA")
			if h == "" {
				return "", core.Fail("REGISTRY_UNAVAILABLE", "LOCALAPPDATA unavailable")
			}
			h = filepath.Join(h, "Conductor")
		} else {
			h = os.Getenv("XDG_DATA_HOME")
			if h == "" {
				u, e := os.UserHomeDir()
				if e != nil {
					return "", e
				}
				h = filepath.Join(u, ".local", "share")
			}
			h = filepath.Join(h, "conductor")
		}
	}
	return filepath.Abs(h)
}
func Run(ctx context.Context, env Env, args []string) int {
	if env.Out == nil {
		env.Out = os.Stdout
	}
	if env.Err == nil {
		env.Err = os.Stderr
	}
	if env.CWD == "" {
		env.CWD, _ = os.Getwd()
	}
	var result any
	var err error
	rest := []string{}
	project := ""
	seenGlobals := map[string]bool{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--json" {
			continue
		}
		if a == "--home" || strings.HasPrefix(a, "--home=") || a == "--project" || strings.HasPrefix(a, "--project=") {
			key := strings.SplitN(a, "=", 2)[0]
			if seenGlobals[key] {
				err = core.Fail("USAGE", "duplicate flag "+key)
				break
			}
			seenGlobals[key] = true
			v := ""
			if strings.Contains(a, "=") {
				v = strings.SplitN(a, "=", 2)[1]
			} else {
				i++
				if i < len(args) {
					v = args[i]
				}
			}
			if v == "" || strings.HasPrefix(v, "--") {
				err = core.Fail("USAGE", key+" requires a value")
				break
			}
			if key == "--home" {
				env.Home = v
			} else {
				project = v
			}
			continue
		}
		rest = append(rest, a)
	}
	if err == nil {
		env.Home, err = ResolveHome(env.Home)
	}
	if err == nil {
		if project != "" {
			rest = append(rest, "--project", project)
		}
		if len(rest) == 0 {
			err = core.Fail("USAGE", "command required")
		} else if len(rest) == 1 && (rest[0] == "--help" || rest[0] == "help") {
			names := []string{}
			for k := range handlers {
				names = append(names, k)
			}
			sort.Strings(names)
			result = map[string]any{"commands": names, "global_options": []string{"--home PATH", "--project ID", "--json"}}
		} else if len(rest) == 1 && rest[0] == "--version" {
			result = map[string]any{"version": "0.2.0-dev", "schema_version": store.SchemaVersion, "skill_contract": "layers-v1"}
		} else if h, ok := handlers[rest[0]]; ok {
			result, err = h(ctx, env, rest[1:])
		} else {
			err = core.Fail("USAGE", "unknown command: "+rest[0])
		}
	}
	if err == nil {
		json.NewEncoder(env.Out).Encode(map[string]any{"ok": true, "data": result})
		return 0
	}
	var f *core.Fault
	if errors.Is(err, store.ErrRequestConflict) {
		f = &core.Fault{Code: "REQUEST_CONFLICT", Message: "request ID was already used with another payload"}
	} else if !errors.As(err, &f) {
		f = &core.Fault{Code: "REGISTRY_UNAVAILABLE", Message: err.Error()}
	}
	if f.Details == nil {
		f.Details = map[string]any{}
	}
	json.NewEncoder(env.Out).Encode(map[string]any{"ok": false, "error": f})
	switch f.Code {
	case "USAGE", "INVALID_INPUT":
		return 2
	case "NOT_FOUND":
		return 4
	case "REGISTRY_UNAVAILABLE", "STORAGE_ERROR", "DISCOVERY_UNAVAILABLE":
		return 5
	default:
		return 3
	}
}
