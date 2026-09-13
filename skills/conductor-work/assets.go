// Package workskill packages the one source skill with the executable.
package workskill

import "embed"

// Files contains the instructions copied by setup. No repository is needed at runtime.
//
//go:embed SKILL.md references/commands.md references/startup-corrections.md
var Files embed.FS
