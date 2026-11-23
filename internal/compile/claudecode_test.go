package compile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestClaudeCodeCompiler(t *testing.T) {
	tmp := t.TempDir()

	// Seed existing CLAUDE.md prose and unrelated .claude/settings.json keys
	if err := os.WriteFile(filepath.Join(tmp, "CLAUDE.md"), []byte("# Custom Repo Guide\nDo not erase me.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".claude", "settings.json"), []byte(`{"theme": "dark", "hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "gradusctl status --brief"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	view := resolve.ResolvedView{
		ViewHash: "sha256:claude1",
		Levels:   []resolve.LevelSummary{{ID: "project"}, {ID: "user"}},
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-01",
				Title:       "Global project rule",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "Global rule text.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "PRJ-API-01",
				Title:       "API scoped rule",
				Level:       "project",
				Severity:    "SHOULD",
				Enforcement: "review",
				Scope:       []string{"services/api/**"},
				Text:        "API rule text.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "PRJ-PERM-01",
				Title:       "Forbid rm -rf",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "permission",
				Scope:       []string{"**/*"},
				Text:        "Never run rm -rf.",
				Guard:       &schema.GuardConfig{Action: "command", Pattern: "rm -rf:*", Decision: "deny"},
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "USR-01",
				Title:       "Personal habit",
				Level:       "user",
				Severity:    "MAY",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "User-only text.",
				Provenance:  resolve.RuleProvenance{Source: "user"},
			},
		},
	}

	memFS := fstest.MapFS{
		"gradus-review/SKILL.md": {Data: []byte("---\nname: gradus-review\n---\n")},
	}

	planned, err := compile.PlanClaudeCode(tmp, view, memFS)
	if err != nil {
		t.Fatalf("PlanClaudeCode failed: %v", err)
	}

	claudeMD := string(planned["CLAUDE.md"])
	if !strings.Contains(claudeMD, "Do not erase me.") || !strings.Contains(claudeMD, "PRJ-01") || strings.Contains(claudeMD, "USR-01") {
		t.Fatalf("unexpected CLAUDE.md:\n%s", claudeMD)
	}
	if !strings.Contains(claudeMD, ".claude/rules/gradus-*.md") || !strings.Contains(claudeMD, "PRJ-API-01") {
		t.Fatalf("expected scoped rules note referencing .claude/rules/gradus-*.md and PRJ-API-01 in CLAUDE.md:\n%s", claudeMD)
	}

	scopedRule := string(planned[".claude/rules/gradus-prj-api-01.md"])
	if !strings.Contains(scopedRule, "services/api/**") || !strings.Contains(scopedRule, "API rule text.") {
		t.Fatalf("unexpected scoped rule file:\n%s", scopedRule)
	}

	localMD := string(planned["CLAUDE.local.md"])
	if !strings.Contains(localMD, "USR-01") {
		t.Fatalf("expected USR-01 in CLAUDE.local.md, got:\n%s", localMD)
	}

	var settings map[string]any
	if err := json.Unmarshal(planned[".claude/settings.json"], &settings); err != nil {
		t.Fatal(err)
	}
	if settings["theme"] != "dark" {
		t.Fatalf("expected existing settings key 'theme' to be preserved, got: %v", settings)
	}
	rawSettings := string(planned[".claude/settings.json"])
	if !strings.Contains(rawSettings, "gradusctl guard") || !strings.Contains(rawSettings, "Bash(rm -rf:*)") || !strings.Contains(rawSettings, "gradusctl status --brief --hook") {
		t.Fatalf("expected static hook dispatcher, SessionStart --hook, and permission rule in .claude/settings.json, got:\n%s", rawSettings)
	}
	if strings.Count(rawSettings, "gradusctl status --brief") != 1 {
		t.Fatalf("expected legacy SessionStart command to be upgraded in place without duplication, got:\n%s", rawSettings)
	}

	if string(planned[".claude/skills/gradus-review/SKILL.md"]) == "" {
		t.Fatalf("expected embedded skill extracted to .claude/skills/")
	}
}
