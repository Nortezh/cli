// Package skill embeds the ntzh Claude Code skill so the CLI can install it
// to ~/.claude/skills/ntzh/SKILL.md on demand.
package skill

import _ "embed"

//go:embed SKILL.md
var Content string

const FileName = "SKILL.md"
