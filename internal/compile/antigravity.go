package compile

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

const (
	antigravityChunkLimit = 11500
	antigravityHardLimit  = 12000
)

// PlanAntigravity builds the in-memory file map for the antigravity target (§11.3).
func PlanAntigravity(rootDir string, view resolve.ResolvedView, skillsFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)

	// 1. AGENTS.md managed section.
	agentsMDPath := filepath.Join(rootDir, "AGENTS.md")
	existingAgentsMD, err := readOptionalFile(agentsMDPath)
	if err != nil {
		return nil, err
	}
	files["AGENTS.md"] = PlanAgentsMD(existingAgentsMD, view)

	// 2. Partition rules into project global, project scoped, and user rules.
	var projectGlobal, userGlobal []resolve.ResolvedRule
	projectScoped := make(map[string][]resolve.ResolvedRule)
	userScoped := make(map[string][]resolve.ResolvedRule)

	for _, r := range view.Rules {
		if r.Waived {
			continue
		}
		if r.Level == "user" {
			if isGlobalScope(r.Scope) {
				userGlobal = append(userGlobal, r)
			} else {
				key := scopeKey(r.Scope)
				userScoped[key] = append(userScoped[key], r)
			}
			continue
		}
		if isGlobalScope(r.Scope) {
			projectGlobal = append(projectGlobal, r)
		} else {
			key := scopeKey(r.Scope)
			projectScoped[key] = append(projectScoped[key], r)
		}
	}

	// Emit project global chunks -> .agents/rules/gradus-always-NNN.md
	if len(projectGlobal) > 0 {
		header := "---\ntrigger: always_on\n---\n\n" + RenderPrecedenceHeader(view) + "\n"
		if err := chunkAntigravityRules(files, ".agents/rules", "gradus-always", header, projectGlobal); err != nil {
			return nil, err
		}
	}

	// Emit project scoped chunks -> .agents/rules/gradus-scoped-NNN.md
	if len(projectScoped) > 0 {
		if err := chunkScopedGroups(files, ".agents/rules", "gradus-scoped", projectScoped); err != nil {
			return nil, err
		}
	}

	// Emit user rules -> .agents/rules/gradus-user-NNN.md (git-ignored)
	if len(userGlobal) > 0 {
		header := "---\ntrigger: always_on\n---\n\n# Gradus User Rules\n\n"
		if err := chunkAntigravityRules(files, ".agents/rules", "gradus-user", header, userGlobal); err != nil {
			return nil, err
		}
	}
	if len(userScoped) > 0 {
		if err := chunkScopedGroups(files, ".agents/rules", "gradus-user-scoped", userScoped); err != nil {
			return nil, err
		}
	}

	// 3. .agents/hooks.json static dispatchers (ADR 0003).
	hooksPath := filepath.Join(rootDir, ".agents", "hooks.json")
	hooksBytes, err := mergeAntigravityHooks(hooksPath)
	if err != nil {
		return nil, err
	}
	files[filepath.ToSlash(filepath.Join(".agents", "hooks.json"))] = hooksBytes

	// 4. Embedded skills -> .agents/skills/gradus-*/SKILL.md.
	skillFiles, err := PlanEmbeddedSkills(skillsFS, filepath.ToSlash(filepath.Join(".agents", "skills")))
	if err != nil {
		return nil, err
	}
	for k, v := range skillFiles {
		files[k] = v
	}

	return files, nil
}

func chunkScopedGroups(files map[string][]byte, baseDir, filePrefix string, groups map[string][]resolve.ResolvedRule) error {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	idx := 1
	for _, k := range keys {
		rules := groups[k]
		var fm strings.Builder
		fm.WriteString("---\ntrigger: glob\n")
		fmt.Fprintf(&fm, "globs: %q\n", strings.Join(rules[0].Scope, ", "))
		fm.WriteString("---\n\n")
		header := fm.String()

		chunkPrefix := fmt.Sprintf("%s-g%d", filePrefix, idx)
		if err := chunkAntigravityRules(files, baseDir, chunkPrefix, header, rules); err != nil {
			return err
		}
		idx++
	}
	return nil
}

func chunkAntigravityRules(files map[string][]byte, baseDir, filePrefix, header string, rules []resolve.ResolvedRule) error {
	chunkNum := 1
	var current strings.Builder
	current.WriteString(header)

	flush := func() {
		rel := filepath.ToSlash(filepath.Join(baseDir, fmt.Sprintf("%s-%03d.md", filePrefix, chunkNum)))
		files[rel] = []byte(current.String())
		chunkNum++
	}

	for _, r := range rules {
		rendered := RenderRuleMarkdown(r)
		if len(header)+len(rendered) > antigravityHardLimit {
			return fmt.Errorf("rule %s alone exceeds Antigravity %d-character rule limit (%d chars)",
				r.ID, antigravityHardLimit, len(header)+len(rendered))
		}
		if current.Len() > len(header) && current.Len()+len(rendered) > antigravityChunkLimit {
			flush()
			current.Reset()
			current.WriteString(header)
		}
		current.WriteString(rendered)
	}

	if current.Len() > len(header) {
		flush()
	}
	return nil
}

func mergeAntigravityHooks(hooksPath string) ([]byte, error) {
	doc := make(map[string]any)
	raw, err := os.ReadFile(hooksPath)
	if err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("failed to parse existing %s: %w", hooksPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", hooksPath, err)
	}

	// Clean up legacy Claude-style top-level "hooks" wrapper emitted by v0.1.0/v0.1.1.
	if legacy, ok := doc["hooks"].(map[string]any); ok {
		if _, hasPre := legacy["PreToolUse"]; hasPre {
			delete(doc, "hooks")
		}
	}

	// Antigravity hooks.json uses top-level named hook objects; PreToolUse is grouped,
	// while SessionStart is a flat array of command handlers.
	doc["gradus"] = map[string]any{
		"PreToolUse": []any{
			map[string]any{
				"matcher": "run_command|write_file|edit_file|write_to_file|replace_file_content|multi_replace_file_content",
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "gradusctl guard",
					},
				},
			},
		},
		"SessionStart": []any{
			map[string]any{
				"type":    "command",
				"command": "gradusctl status --brief --antigravity-hook",
			},
		},
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func scopeKey(scope []string) string {
	cp := append([]string(nil), scope...)
	sort.Strings(cp)
	return strings.Join(cp, "|")
}
