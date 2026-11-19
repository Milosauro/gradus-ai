package skills

import "embed"

// FS embeds the canonical Agent Skills for Gradus so that gradusctl
// can install them locally without requiring external package downloads.
//
//go:embed gradus-*/*.md
var FS embed.FS
