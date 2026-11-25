package compile

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// PlanClaudeCode builds the in-memory file map for the claude-code target (§11.3).
func PlanClaudeCode(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)

	var globalBody strings.Builder
	globalBody.WriteString(RenderPrecedenceHeader(view))
	globalBody.WriteString("\n")

	var userBody strings.Builder
	userBody.WriteString("# Gradus User Rules\n\n")
	hasUserRules := false
	var scopedRuleIDs []string

	for _, r := range view.Rules {
		if r.Waived {
			continue
		}
		if r.Level == "user" {
			hasUserRules = true
			userBody.WriteString(RenderRuleMarkdown(r))
			continue
		}
		if isGlobalScope(r.Scope) {
			globalBody.WriteString(RenderRuleMarkdown(r))
		} else {
			scopedRuleIDs = append(scopedRuleIDs, r.ID)
			// Scoped project rule -> .claude/rules/gradus-<id>.md with paths frontmatter.
			var sb strings.Builder
			sb.WriteString("---\npaths:\n")
			for _, p := range r.Scope {
				fmt.Fprintf(&sb, "  - %q\n", p)
			}
			sb.WriteString("---\n\n")
			sb.WriteString(RenderRuleMarkdown(r))
			ruleRel := filepath.ToSlash(filepath.Join(".claude", "rules", fmt.Sprintf("gradus-%s.md", strings.ToLower(r.ID))))
			files[ruleRel] = []byte(sb.String())
		}
	}

	if len(scopedRuleIDs) > 0 {
		fmt.Fprintf(&globalBody, "> **Scoped rules:** %d path-scoped rule(s) (`%s`) are compiled into `.claude/rules/gradus-*.md` and loaded automatically when matching files are touched.\n",
			len(scopedRuleIDs), strings.Join(scopedRuleIDs, "`, `"))
	}

	// CLAUDE.md managed section.
	claudeMDPath := filepath.Join(rootDir, "CLAUDE.md")
	existingClaudeMD, err := readOptionalFile(claudeMDPath)
	if err != nil {
		return nil, err
	}
	block := RenderManagedBlock(view.ViewHash, globalBody.String())
	files["CLAUDE.md"] = []byte(ApplyManagedSection(existingClaudeMD, block))

	// CLAUDE.local.md for user-level rules (git-ignored).
	if hasUserRules {
		localPath := filepath.Join(rootDir, "CLAUDE.local.md")
		existingLocal, err := readOptionalFile(localPath)
		if err != nil {
			return nil, err
		}
		userBlock := RenderManagedBlock(view.ViewHash, userBody.String())
		files["CLAUDE.local.md"] = []byte(ApplyManagedSection(existingLocal, userBlock))
	}

	// .claude/settings.json static hook dispatchers + permissions.
	settingsPath := filepath.Join(rootDir, ".claude", "settings.json")
	settingsBytes, err := mergeClaudeSettings(settingsPath, view)
	if err != nil {
		return nil, err
	}
	files[filepath.ToSlash(filepath.Join(".claude", "settings.json"))] = settingsBytes

	// Embedded skills -> .claude/skills/gradus-*/SKILL.md.
	skillFiles, err := PlanEmbeddedSkills(skillsFS, filepath.ToSlash(filepath.Join(".claude", "skills")))
	if err != nil {
		return nil, err
	}
	for k, v := range skillFiles {
		if view.Models != nil && strings.HasSuffix(k, "/SKILL.md") {
			skillName := filepath.Base(filepath.Dir(k))
			if model := view.Models.ModelForSkill("claude-code", skillName); model != "" {
				v = injectSkillFrontmatterModel(v, model)
			}
		}
		files[k] = v
	}

	return files, nil
}

func injectSkillFrontmatterModel(raw []byte, model string) []byte {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return raw
	}
	rest := text[4:]
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return raw
	}
	fm := rest[:idx]
	tail := rest[idx:]
	var lines []string
	replaced := false
	for _, line := range strings.Split(fm, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "model:") {
			lines = append(lines, "model: "+model)
			replaced = true
		} else {
			lines = append(lines, line)
		}
	}
	if !replaced {
		lines = append(lines, "model: "+model)
	}
	return []byte("---\n" + strings.Join(lines, "\n") + tail)
}

func mergeClaudeSettings(settingsPath string, view resolve.ResolvedView) ([]byte, error) {
	settings := make(map[string]any)
	raw, err := os.ReadFile(settingsPath)
	if err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse existing %s: %w", settingsPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", settingsPath, err)
	}

	hooksMap := ensureStringMap(settings, "hooks")
	ensureHookDispatcher(hooksMap, "PreToolUse", "Bash|Edit|Write|MultiEdit", "gradusctl guard")
	ensureHookDispatcher(hooksMap, "SessionStart", "", "gradusctl status --brief --hook")

	// Merge permission rules.
	var denyList, askList []string
	for _, r := range view.Rules {
		if r.Waived || r.Level == "user" || r.Enforcement != "permission" || r.Guard == nil {
			continue
		}
		entry := formatClaudePermission(r.Guard.Action, r.Guard.Pattern)
		switch r.Guard.Decision {
		case "deny":
			denyList = append(denyList, entry)
		case "ask":
			askList = append(askList, entry)
		}
	}

	if len(denyList) > 0 || len(askList) > 0 {
		permMap := ensureStringMap(settings, "permissions")
		if len(denyList) > 0 {
			permMap["deny"] = mergeUniqueStrings(permMap["deny"], denyList)
		}
		if len(askList) > 0 {
			permMap["ask"] = mergeUniqueStrings(permMap["ask"], askList)
		}
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func ensureHookDispatcher(hooksMap map[string]any, event, matcher, command string) {
	var entries []any
	if existing, ok := hooksMap[event].([]any); ok {
		entries = existing
	}

	for _, e := range entries {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		innerHooks, ok := m["hooks"].([]any)
		if !ok {
			continue
		}
		for _, ih := range innerHooks {
			hm, ok := ih.(map[string]any)
			if !ok {
				continue
			}
			if hm["command"] == command {
				return
			}
			// Upgrade legacy SessionStart command in place instead of duplicating.
			if event == "SessionStart" && hm["command"] == "gradusctl status --brief" {
				hm["command"] = command
				return
			}
		}
	}

	entry := map[string]any{
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": command,
			},
		},
	}
	if matcher != "" {
		entry["matcher"] = matcher
	}
	hooksMap[event] = append(entries, entry)
}

func formatClaudePermission(action, pattern string) string {
	switch strings.ToLower(action) {
	case "command", "bash":
		return fmt.Sprintf("Bash(%s)", pattern)
	case "write", "edit":
		return fmt.Sprintf("Edit(%s)", pattern)
	default:
		return fmt.Sprintf("%s(%s)", action, pattern)
	}
}

func ensureStringMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok {
		return existing
	}
	m := make(map[string]any)
	parent[key] = m
	return m
}

func mergeUniqueStrings(existing any, additions []string) []string {
	var out []string
	seen := make(map[string]bool)
	if slice, ok := existing.([]any); ok {
		for _, item := range slice {
			if s, ok := item.(string); ok && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	} else if slice, ok := existing.([]string); ok {
		for _, s := range slice {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	for _, add := range additions {
		if !seen[add] {
			seen[add] = true
			out = append(out, add)
		}
	}
	return out
}

func isGlobalScope(scope []string) bool {
	if len(scope) == 0 {
		return true
	}
	return len(scope) == 1 && (scope[0] == "**/*" || scope[0] == "**")
}

func readOptionalFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}
	return string(data), nil
}
