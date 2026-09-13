package cli

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZackMFleischman/conductor/internal/core"
	"github.com/ZackMFleischman/conductor/internal/setup"
	skillbundle "github.com/ZackMFleischman/conductor/skills"
)

func init() { Register("setup", Setup) }

// Setup changes only explicit user-level integration; it never opens a store.
func Setup(ctx context.Context, env Env, args []string) (any, error) {
	f, err := Parse(args, []string{"agents", "command-access"}, []string{"apply", "remove"})
	if err != nil {
		return nil, err
	}
	if err = f.NoPositionals(); err != nil {
		return nil, err
	}
	if err = f.Require("agents"); err != nil {
		return nil, err
	}
	if f.Values["project"] != "" {
		return nil, core.Fail("USAGE", "setup is user-level; --project is not supported")
	}
	agents := strings.Split(f.Values["agents"], ",")
	seen := map[string]bool{}
	for _, a := range agents {
		if (a != "codex" && a != "claude") || seen[a] {
			return nil, core.Fail("USAGE", "--agents must select codex, claude, or codex,claude once")
		}
		seen[a] = true
	}
	if value, ok := f.Values["command-access"]; ok && value != "host-default" && value != "require_escalated" {
		return nil, core.Fail("USAGE", "--command-access must be host-default or require_escalated")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	userHome, err = filepath.Abs(userHome)
	if err != nil {
		return nil, err
	}
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(userHome, ".codex")
	}
	claudeHome := os.Getenv("CLAUDE_CONFIG_DIR")
	if claudeHome == "" {
		claudeHome = filepath.Join(userHome, ".claude")
	}
	// Refuse relative host roots: never route persistent integration by CWD.
	if (seen["codex"] && !filepath.IsAbs(codexHome)) || (seen["claude"] && !filepath.IsAbs(claudeHome)) {
		return nil, core.Fail("USAGE", "CODEX_HOME and CLAUDE_CONFIG_DIR must be absolute")
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	home, err := ResolveHome(env.Home)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	err = fs.WalkDir(skillbundle.Files, ".", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := skillbundle.Files.ReadFile(p)
		if e != nil {
			return e
		}
		files[strings.TrimPrefix(p, "conductor-work/")] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	options := setup.Options{CommandAccess: f.Values["command-access"], Agents: agents, Executable: exe, DataHome: home, Remove: f.Bools["remove"], UserHome: userHome, CodexHome: codexHome, ClaudeHome: claudeHome, SkillFiles: files}
	edits, err := setup.Plan(options)
	if err != nil {
		return nil, setupError(err)
	}
	access, err := setup.PlannedAccess(options, edits)
	if err != nil {
		return nil, setupError(err)
	}
	if f.Bools["apply"] {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if err = setup.Apply(edits); err != nil {
			return nil, setupError(err)
		}
	}
	if edits == nil {
		edits = []setup.Edit{}
	}
	return map[string]any{"applied": f.Bools["apply"], "remove": f.Bools["remove"], "edits": edits, "configured": map[string]any{"executable": exe, "data_home": home, "codex_home": codexHome, "codex_skill_home": filepath.Join(userHome, ".agents", "skills"), "claude_home": claudeHome}, "access": access, "access_verified": false, "next_steps": "Start fresh selected host sessions. Run doctor --probe-write inside each host, then verify context and conductor-work discovery. Configured paths do not prove effective sandbox access."}, nil
}
func setupError(err error) error {
	code := "STORAGE_ERROR"
	if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "unowned") {
		code = "SETUP_CONFLICT"
	}
	return core.Fail(code, err.Error())
}
