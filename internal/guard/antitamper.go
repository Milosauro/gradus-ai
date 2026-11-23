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
	".claude/settings.local.json",
	".git/hooks/**",
	".githooks/**",
	".pre-commit-config.yaml",
	"lefthook.yml",
	".lefthook.yml",
	".husky/**",
	".github/workflows/**",
	"opencode.json",
	".codex/config.toml",
}

var protectedShellTargetsRe = regexp.MustCompile(`(?i)(?:^|[\s"'=/\\><|;&])(?:gradus(?:\.yaml|\.lock|\.[a-z0-9*?\[\]]+)|\.gradus(?:[/\\*?\s"']|$)|\.walden(?:[/\\*?\s"']|$)|\.agents[/\\]hooks\.json|\.claude[/\\]settings(?:\.local)?\.json|\.git[/\\]hooks(?:[/\\\s"']|$)|\.githooks(?:[/\\\s"']|$)|\.pre-commit-config\.yaml|\.?lefthook\.yml|\.husky(?:[/\\\s"']|$)|\.github[/\\]workflows(?:[/\\\s"']|$)|opencode\.json|\.codex[/\\]config\.toml)`)
var mutatingShellVerbRe = regexp.MustCompile(`(?i)(?:\b(?:rm|mv|cp|sed|awk|perl|python3?|ruby|tee|truncate|chmod|chown|touch|dd|ln|unlink|install|del|erase|ren|rename|move|copy|remove-item|set-content|add-content|out-file|clear-content)\b|\bfind\b[^\n]*-delete\b|\bgit\s+(?:rm|mv|checkout|restore|reset)\b|>{1,2})`)
var gitHooksPathConfigRe = regexp.MustCompile(`(?i)\bgit\b[^\n]*\bconfig\b[^\n]*\bcore\.hooksPath\b`)
var gitCommitNoVerifyRe = regexp.MustCompile(`(?i)\bgit\b[^\n]*\bcommit\b[^\n]*(?:--no-verify\b|(?:^|\s)-n(?:\s|$))`)
var humanOnlyGradusCmdRe = regexp.MustCompile(`(?i)\bgradusctl\b[^\n]*(?:\bexception\s+(?:approve|break-glass)\b|\bconflicts\s+dismiss\b|\blessons\s+update\b|\bhooks\s+install\b)`)

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
// or delete a protected governance or hook configuration path, bypass git hooks, or invoke
// a human-only governance command (Threat Model T2).
func IsProtectedShellCommand(command string) bool {
	matched, _ := AntiTamperShellReason(command)
	return matched
}

// AntiTamperShellReason returns true and a specific reason when a shell command triggers anti-tamper.
func AntiTamperShellReason(command string) (bool, string) {
	if strings.TrimSpace(command) == "" {
		return false, ""
	}
	if humanOnlyGradusCmdRe.MatchString(command) {
		return true, "anti-tamper: human-only governance command requires explicit human confirmation"
	}
	if gitCommitNoVerifyRe.MatchString(command) {
		return true, "anti-tamper: git commit --no-verify/-n bypasses git hooks and requires explicit human confirmation"
	}
	if gitHooksPathConfigRe.MatchString(command) || (protectedShellTargetsRe.MatchString(command) && mutatingShellVerbRe.MatchString(command)) {
		return true, "anti-tamper: shell command targets protected governance path and requires explicit human confirmation"
	}
	return false, ""
}
