package source_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/source"
)

func TestResolveLocalSource(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, ".gradus", "levels", "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := source.ResolveLocalSource(tmp, "level", "project", ".gradus/levels/project", false)
	if err != nil {
		t.Fatalf("expected local source to resolve, got: %v", err)
	}
	if res.Skipped || res.Dir != projDir || !strings.HasPrefix(res.Hash, "sha256:") {
		t.Fatalf("unexpected resolved local source: %+v", res)
	}

	// Home-directory source must be skipped with a note in CI
	ciRes, err := source.ResolveLocalSource(tmp, "level", "user", "~/.gradus/levels/user", true)
	if err != nil {
		t.Fatalf("expected CI home skip to succeed, got: %v", err)
	}
	if !ciRes.Skipped || len(ciRes.Notes) == 0 {
		t.Fatalf("expected home source to be skipped with note in CI, got: %+v", ciRes)
	}
}
