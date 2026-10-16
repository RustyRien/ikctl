// Package skills embeds the LLM agent skill shipped with ikctl.
package skills

import "embed"

// Name is the skill directory name.
const Name = "ikctl"

// FS contains the skill files under "ikctl/" (SKILL.md and references/).
//
//go:embed ikctl
var FS embed.FS
