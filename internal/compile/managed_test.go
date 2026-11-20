package compile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Milosauro/gradus-ai/internal/compile"
)

func TestManagedSectionsAndIdempotency(t *testing.T) {
	// 1. Preserve human prose before and after managed markers
	existing := "# My Project Notes\n\nKeep this intro.\n\n<!-- gradus:begin view_hash=old -->\nold rules\n<!-- gradus:end -->\n\nKeep this footer.\n"
	block := compile.RenderManagedBlock("sha256:new", "## New Rules\nRule body.")
	updated := compile.ApplyManagedSection(existing, block)

	if !strings.Contains(updated, "Keep this intro.") || !strings.Contains(updated, "Keep this footer.") {
		t.Fatalf("expected human prose to be preserved, got:\n%s", updated)
	}
	if strings.Contains(updated, "old rules") || !strings.Contains(updated, "New Rules") {
		t.Fatalf("expected managed block to be replaced, got:\n%s", updated)
	}

	// 2. WriteIfChanged skips write when bytes are identical
	tmp := t.TempDir()
	p := filepath.Join(tmp, "AGENTS.md")
	changed, err := compile.WriteIfChanged(p, []byte(updated))
	if err != nil || !changed {
		t.Fatalf("expected first write to report changed=true, got changed=%v err=%v", changed, err)
	}
	changed, err = compile.WriteIfChanged(p, []byte(updated))
	if err != nil || changed {
		t.Fatalf("expected second identical write to report changed=false, got changed=%v err=%v", changed, err)
	}

	// 3. PruneStaleGradusFiles only removes unkept gradus-*.md files
	rulesDir := filepath.Join(tmp, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(rulesDir, "gradus-project-001.md"), []byte("keep"), 0o644)
	_ = os.WriteFile(filepath.Join(rulesDir, "gradus-project-002.md"), []byte("stale"), 0o644)
	_ = os.WriteFile(filepath.Join(rulesDir, "custom-user.md"), []byte("user"), 0o644)

	removed, err := compile.PruneStaleGradusFiles(rulesDir, "gradus-", map[string]bool{"gradus-project-001.md": true})
	if err != nil || len(removed) != 1 || removed[0] != "gradus-project-002.md" {
		t.Fatalf("unexpected pruned list: %v err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(rulesDir, "custom-user.md")); err != nil {
		t.Fatalf("expected non-gradus user file to remain untouched")
	}

	// 4. PlanEmbeddedSkills skips gradus-placeholder
	memFS := fstest.MapFS{
		"gradus-placeholder/SKILL.md": {Data: []byte("placeholder")},
		"gradus-review/SKILL.md":      {Data: []byte("review skill")},
	}
	planned, err := compile.PlanEmbeddedSkills(memFS, ".claude/skills")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := planned[".claude/skills/gradus-placeholder/SKILL.md"]; ok {
		t.Fatalf("expected gradus-placeholder to be skipped")
	}
	if string(planned[".claude/skills/gradus-review/SKILL.md"]) != "review skill" {
		t.Fatalf("expected gradus-review skill to be planned, got: %v", planned)
	}
}
