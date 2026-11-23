package compile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestExtraCompileTargets(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".github", "instructions"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed human-written prose in AGENTS.md, .github/copilot-instructions.md, and custom instruction file
	if err := os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# Human Agents Header\nKeep me.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".github", "copilot-instructions.md"), []byte("# Team Copilot Guide\nPreserve prose.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".github", "instructions", "team.instructions.md"), []byte("custom human instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	view := resolve.ResolvedView{
		ViewHash: "sha256:extra1",
		Levels:   []resolve.LevelSummary{{ID: "project"}, {ID: "user"}},
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-01",
				Title:       "Global rule",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "Global guidance.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "PRJ-GO-01",
				Title:       "Go scoped rule",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"**/*.go"},
				Text:        "Go guidance.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "USR-01",
				Title:       "Personal habit",
				Level:       "user",
				Severity:    "MAY",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "User-only guidance.",
				Provenance:  resolve.RuleProvenance{Source: "user"},
			},
		},
	}

	memFS := fstest.MapFS{
		"gradus-review/SKILL.md": {Data: []byte("---\nname: gradus-review\n---\n")},
	}

	targets := []string{"codex", "copilot", "opencode"}
	res, err := compile.ApplyCompile(tmp, view, targets, memFS)
	if err != nil {
		t.Fatalf("ApplyCompile failed: %v", err)
	}
	if len(res.ChangedFiles) == 0 {
		t.Fatalf("expected changed files on first compile")
	}

	// Verify codex + opencode AGENTS.md & skill extraction
	agentsMD, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agentsMD), "Keep me.") || !strings.Contains(string(agentsMD), "PRJ-01") || strings.Contains(string(agentsMD), "USR-01") {
		t.Fatalf("unexpected AGENTS.md:\n%s", string(agentsMD))
	}
	for _, skillPath := range []string{
		filepath.Join(tmp, ".agents", "skills", "gradus-review", "SKILL.md"),
		filepath.Join(tmp, ".github", "skills", "gradus-review", "SKILL.md"),
		filepath.Join(tmp, ".opencode", "skills", "gradus-review", "SKILL.md"),
	} {
		if _, err := os.Stat(skillPath); err != nil {
			t.Fatalf("expected skill at %s: %v", skillPath, err)
		}
	}

	// Verify copilot instructions + scoped applyTo
	copilotMD, err := os.ReadFile(filepath.Join(tmp, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(copilotMD), "Preserve prose.") || !strings.Contains(string(copilotMD), "PRJ-01") || !strings.Contains(string(copilotMD), "PRJ-GO-01") {
		t.Fatalf("unexpected .github/copilot-instructions.md:\n%s", string(copilotMD))
	}
	scopedInstr, err := os.ReadFile(filepath.Join(tmp, ".github", "instructions", "gradus-prj-go-01.instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(scopedInstr), `applyTo: "**/*.go"`) || !strings.Contains(string(scopedInstr), "Go guidance.") {
		t.Fatalf("unexpected scoped copilot instruction:\n%s", string(scopedInstr))
	}

	// Second run is zero-diff idempotent (NFR1)
	res2, err := compile.ApplyCompile(tmp, view, targets, memFS)
	if err != nil {
		t.Fatalf("second ApplyCompile failed: %v", err)
	}
	if len(res2.ChangedFiles) != 0 {
		t.Fatalf("expected zero changed files on second compile, got: %v", res2.ChangedFiles)
	}

	// Removing scoped rule prunes gradus-prj-go-01.instructions.md while preserving team.instructions.md
	view.Rules = view.Rules[:1]
	res3, err := compile.ApplyCompile(tmp, view, targets, memFS)
	if err != nil {
		t.Fatalf("third ApplyCompile failed: %v", err)
	}
	if len(res3.RemovedFiles) != 1 || res3.RemovedFiles[0] != ".github/instructions/gradus-prj-go-01.instructions.md" {
		t.Fatalf("expected pruned copilot instruction, got: %v", res3.RemovedFiles)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".github", "instructions", "team.instructions.md")); err != nil {
		t.Fatalf("expected human instruction file preserved: %v", err)
	}

	// Unsupported target returns error
	if _, err := compile.PlanCompile(tmp, view, []string{"unknown-agent"}, memFS); err == nil {
		t.Fatalf("expected unsupported target error")
	}
}
