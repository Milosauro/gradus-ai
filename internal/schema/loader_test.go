package schema_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestLoadLocalPyramid(t *testing.T) {
	tmp := t.TempDir()

	// Write gradus.yaml with project level and ~/ user level
	gradusYAML := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
  - id: user
    source: { path: ~/.gradus/levels/user }
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	projDir := filepath.Join(tmp, ".gradus", "levels", "project")
	rulesDir := filepath.Join(projDir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ruleMD := `---
id: PRJ-GEN-001
title: Keep functions small
severity: SHOULD
---
Keep functions small and focused.
`
	if err := os.WriteFile(filepath.Join(rulesDir, "PRJ-GEN-001.md"), []byte(ruleMD), 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate CI=true so ~/ level is skipped with a note
	t.Setenv("CI", "true")

	pyramid, err := schema.LoadLocalPyramid(tmp)
	if err != nil {
		t.Fatalf("expected LoadLocalPyramid to succeed, got: %v", err)
	}
	if len(pyramid.Rules) != 1 || pyramid.Rules[0].ID != "PRJ-GEN-001" || pyramid.Rules[0].Level != "project" {
		t.Fatalf("unexpected loaded rules: %+v", pyramid.Rules)
	}
	if !slices.ContainsFunc(pyramid.Notes, func(s string) bool { return strings.Contains(s, "in CI") }) {
		t.Fatalf("expected note about skipping ~/ level in CI, got: %v", pyramid.Notes)
	}
}
