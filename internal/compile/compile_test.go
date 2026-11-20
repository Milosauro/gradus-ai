package compile_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestCompilePlanAndPrune(t *testing.T) {
	tmp := t.TempDir()

	// Seed a stale gradus rule file in .agents/rules/
	agRulesDir := filepath.Join(tmp, ".agents", "rules")
	if err := os.MkdirAll(agRulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	staleFile := filepath.Join(agRulesDir, "gradus-always-999.md")
	if err := os.WriteFile(staleFile, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	view := resolve.ResolvedView{
		ViewHash: "sha256:plan1",
		Levels:   []resolve.LevelSummary{{ID: "project"}},
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-01",
				Title:       "First rule",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "Do good.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
		},
	}

	targets := []string{"claude-code", "antigravity", "agents-md"}
	memFS := fstest.MapFS{
		"gradus-review/SKILL.md": {Data: []byte("---\nname: gradus-review\n---\n")},
	}

	// First run: writes files and prunes staleFile
	res1, err := compile.ApplyCompile(tmp, view, targets, memFS)
	if err != nil {
		t.Fatalf("first ApplyCompile failed: %v", err)
	}
	if len(res1.ChangedFiles) == 0 {
		t.Fatalf("expected changed files on first compile")
	}
	if len(res1.RemovedFiles) != 1 || res1.RemovedFiles[0] != ".agents/rules/gradus-always-999.md" {
		t.Fatalf("expected stale file pruned, got: %v", res1.RemovedFiles)
	}

	// Second run: strictly idempotent -> 0 changed files
	res2, err := compile.ApplyCompile(tmp, view, targets, memFS)
	if err != nil {
		t.Fatalf("second ApplyCompile failed: %v", err)
	}
	if len(res2.ChangedFiles) != 0 {
		t.Fatalf("expected 0 changed files on second compile (idempotency), got: %v", res2.ChangedFiles)
	}
	if len(res2.RemovedFiles) != 0 {
		t.Fatalf("expected 0 removed files on second compile, got: %v", res2.RemovedFiles)
	}
}
