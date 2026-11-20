package guard

import (
	"regexp"
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

var protectedShellTargetsRe = regexp.MustCompile(`(?i)(?:^|[\s"'=/\\])(?:gradus\.yaml|gradus\.lock|\.gradus(?:[/\\]|$)|\.walden(?:[/\\]|$)|\.agents[/\\]hooks\.json|\.claude[/\\]settings\.json)`)
var mutatingShellVerbRe = regexp.MustCompile(`(?i)(?:\b(?:rm|mv|cp|sed|awk|perl|python3?|ruby|tee|truncate|chmod|chown|touch|dd|del|erase|ren|rename|move|copy|remove-item|set-content|add-content|out-file|clear-content)\b|git\s+(?:rm|mv|checkout\s+--|restore)|>{1,2})`)

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

// IsProtectedShellCommand reports whether a shell command appears to modify, overwrite,
// or delete a protected governance or hook configuration path (Threat Model T2).
func IsProtectedShellCommand(command string) bool {
	if strings.TrimSpace(command) == "" {
		return false
	}
	return protectedShellTargetsRe.MatchString(command) && mutatingShellVerbRe.MatchString(command)
}
