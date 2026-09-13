// Package skills embeds the supported skill bundle for user-level installation.
package skills

import "embed"

//go:embed conductor-*/SKILL.md conductor-work/references/*.md
var Files embed.FS
