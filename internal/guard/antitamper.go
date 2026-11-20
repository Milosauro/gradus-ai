package guard

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var protectedGovernanceGlobs = []string{
	"gradus.yaml",
	"gradus.lock",
	".gradus/**",
	".walden/**",
	".agents/hooks.json",
	".claude/settings.json",
}

// IsProtectedGovernancePath reports whether relPath targets a Gradus/Walden governance
// or hook configuration file protected by built-in anti-tamper (Threat Model T2).
func IsProtectedGovernancePath(relPath string) bool {
	cleaned := NormalizeTargetPath("", relPath)
	if cleaned == "" || cleaned == "." {
		return false
	}
	cleaned = strings.TrimPrefix(cleaned, "/")
	for _, pat := range protectedGovernanceGlobs {
		if matched, err := doublestar.Match(pat, cleaned); err == nil && matched {
			return true
		}
	}
	return false
}
