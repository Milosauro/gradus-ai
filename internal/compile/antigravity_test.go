package compile_test

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestAntigravityCompilerAndSplit(t *testing.T) {
	tmp := t.TempDir()

	// Create 30 global rules totaling > 25,000 chars to exercise < 12,000 char splitting
	var rules []resolve.ResolvedRule
	longText := strings.Repeat("Always validate input boundaries and emit structured errors. ", 16) // ~960 chars
	for i := 1; i <= 30; i++ {
		rules = append(rules, resolve.ResolvedRule{
			ID:          fmt.Sprintf("PRJ-GEN-%03d", i),
			Title:       fmt.Sprintf("Generated Rule %d", i),
			Level:       "project",
			Severity:    "MUST",
			Enforcement: "prompt",
			Scope:       []string{"**/*"},
			Text:        longText,
			Provenance:  resolve.RuleProvenance{Source: "project"},
		})
	}
	// Add 1 scoped rule and 1 user rule
	rules = append(rules,
		resolve.ResolvedRule{
			ID:          "PRJ-API-001",
			Title:       "API Rule",
			Level:       "project",
			Severity:    "SHOULD",
			Enforcement: "review",
			Scope:       []string{"api/**"},
			Text:        "Use versioned API paths.",
			Provenance:  resolve.RuleProvenance{Source: "project"},
		},
		resolve.ResolvedRule{
			ID:          "USR-001",
			Title:       "User Rule",
			Level:       "user",
			Severity:    "MAY",
			Enforcement: "prompt",
			Scope:       []string{"**/*"},
			Text:        "User personal preference.",
			Provenance:  resolve.RuleProvenance{Source: "user"},
		},
	)

	view := resolve.ResolvedView{
		ViewHash: "sha256:ag1",
		Levels:   []resolve.LevelSummary{{ID: "project"}, {ID: "user"}},
		Rules:    rules,
	}

	memFS := fstest.MapFS{
		"gradus-review/SKILL.md": {Data: []byte("---\nname: gradus-review\n---\n")},
	}

	planned, err := compile.PlanAntigravity(tmp, view, memFS)
	if err != nil {
		t.Fatalf("PlanAntigravity failed: %v", err)
	}

	// Verify multiple always_on chunks and every .agents/rules/gradus-*.md file strictly < 12,000 chars
	alwaysChunks := 0
	for path, content := range planned {
		if strings.HasPrefix(path, ".agents/rules/gradus-") && strings.HasSuffix(path, ".md") {
			if len(content) >= 12000 {
				t.Fatalf("Antigravity rule file %s exceeded 12,000 chars: %d", path, len(content))
			}
			if strings.HasPrefix(path, ".agents/rules/gradus-always-") {
				alwaysChunks++
				if !strings.Contains(string(content), "trigger: always_on") {
					t.Fatalf("expected trigger: always_on in %s", path)
				}
			}
		}
	}
	if alwaysChunks < 2 {
		t.Fatalf("expected > 25,000 chars of global rules to split into >= 2 chunks, got %d", alwaysChunks)
	}

	// Verify scoped rule uses string globs (Antigravity requires scalar string, not YAML sequence)
	scopedContent := string(planned[".agents/rules/gradus-scoped-g1-001.md"])
	if !strings.Contains(scopedContent, "trigger: glob\nglobs: \"api/**\"\n") {
		t.Fatalf("expected scalar string globs in scoped rule, got:\n%s", scopedContent)
	}

	// Verify user rule went strictly to gradus-user-001.md
	userContent := string(planned[".agents/rules/gradus-user-001.md"])
	if !strings.Contains(userContent, "USR-001") {
		t.Fatalf("expected USR-001 in gradus-user-001.md, got:\n%s", userContent)
	}
	if strings.Contains(string(planned["AGENTS.md"]), "USR-001") {
		t.Fatalf("expected USR-001 excluded from AGENTS.md")
	}

	// Verify static .agents/hooks.json dispatcher uses Antigravity top-level named hook schema (ADR 0003)
	hooksJSON := string(planned[".agents/hooks.json"])
	if !strings.Contains(hooksJSON, `"gradus"`) || !strings.Contains(hooksJSON, "write_file|edit_file") || !strings.Contains(hooksJSON, "gradusctl guard") || !strings.Contains(hooksJSON, "gradusctl status --brief --antigravity-hook") {
		t.Fatalf("expected Antigravity named hook dispatchers in .agents/hooks.json, got:\n%s", hooksJSON)
	}

	// Verify skill extracted
	if string(planned[".agents/skills/gradus-review/SKILL.md"]) == "" {
		t.Fatalf("expected skill extracted to .agents/skills/")
	}
}
