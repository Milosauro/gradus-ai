package adapter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/adapters"
	"github.com/Milosauro/gradus-ai/internal/adapter"
)

func TestAdapterRegistry(t *testing.T) {
	tmp := t.TempDir()

	// 1. Embedded official adapters (walden + spec-kit)
	reg, err := adapter.LoadRegistry(tmp, adapters.FS)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if w, ok := reg["walden"]; !ok || w.Source != "official" || w.TargetFile != ".walden/constitution.md" {
		t.Fatalf("unexpected walden adapter: %+v (ok=%v)", w, ok)
	}
	if s, ok := reg["spec-kit"]; !ok || s.Source != "official" || s.TargetFile != ".specify/memory/constitution.md" {
		t.Fatalf("unexpected spec-kit adapter: %+v (ok=%v)", s, ok)
	}

	// 2. Custom adapter in .gradus/adapters/internal-sdd.yaml
	customDir := filepath.Join(tmp, ".gradus", "adapters")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customYAML := `schema: gradus/v1
id: internal-sdd
name: Internal SDD
target_file: docs/sdd-constitution.md
`
	if err := os.WriteFile(filepath.Join(customDir, "internal-sdd.yaml"), []byte(customYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err = adapter.LoadRegistry(tmp, adapters.FS)
	if err != nil {
		t.Fatalf("LoadRegistry with custom adapter failed: %v", err)
	}
	if c, ok := reg["internal-sdd"]; !ok || c.Source != "custom" || c.TargetFile != "docs/sdd-constitution.md" {
		t.Fatalf("unexpected custom adapter: %+v (ok=%v)", c, ok)
	}

	// 3. Reject path traversal target_file (NFR2)
	badYAML := `schema: gradus/v1
id: evil-adapter
target_file: ../outside.md
`
	if err := os.WriteFile(filepath.Join(customDir, "evil.yaml"), []byte(badYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.LoadRegistry(tmp, adapters.FS); err == nil || !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("expected path traversal rejection, got: %v", err)
	}
}
