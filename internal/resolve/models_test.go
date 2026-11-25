package resolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestModelPolicySchemaAndResolution(t *testing.T) {
	// 1. Schema parsing with zero notes and invalid tier rejection
	validYAML := `
schema: gradus/v1
id: org
models:
  tiers:
    tier-high:
      claude-code: opus
    tier-standard:
      claude-code: sonnet
  operations:
    review: tier-high
    learn: tier-standard
`
	resCfg, err := schema.ParseLevelConfig([]byte(validYAML))
	if err != nil {
		t.Fatalf("ParseLevelConfig failed: %v", err)
	}
	if len(resCfg.Notes) != 0 {
		t.Fatalf("expected zero notes for models block, got %v", resCfg.Notes)
	}

	badYAML := "schema: gradus/v1\nid: org\nmodels:\n  operations:\n    review: tier-ultra\n"
	if _, err := schema.ParseLevelConfig([]byte(badYAML)); err == nil {
		t.Fatalf("expected invalid tier-ultra to fail validation")
	}

	// 2. Monotonic tightening succeeds (team sets learn: tier-standard -> project raises to tier-high)
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "org"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "level.yaml"), []byte(validYAML), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte(`schema: gradus/v1
id: project
models:
  operations:
    learn: tier-high
`), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: org
    source: { path: .gradus/levels/org }
  - id: project
    source: { path: .gradus/levels/project }
`), 0o644)

	resOK, err := resolve.Resolve(tmp, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(resOK.View.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts on monotonic tightening, got %+v", resOK.View.Conflicts)
	}
	if resOK.View.Models == nil || resOK.View.Models.Operations["learn"] != "tier-high" {
		t.Fatalf("expected tightened learn=tier-high, got %+v", resOK.View.Models)
	}
	if m := resOK.View.Models.ModelForSkill("claude-code", "gradus-review"); m != "opus" {
		t.Fatalf("expected ModelForSkill(claude-code, gradus-review)=opus, got %q", m)
	}

	// 3. Lower level relaxing review from tier-high to tier-light emits blocking model_policy conflict
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte(`schema: gradus/v1
id: project
models:
  operations:
    review: tier-light
`), 0o644)
	resConflict, err := resolve.Resolve(tmp, resolve.Options{OfflineOnly: true})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(resConflict.View.Conflicts) != 1 || resConflict.View.Conflicts[0].Kind != "model_policy" || !strings.Contains(resConflict.View.Conflicts[0].Message, "relax") {
		t.Fatalf("expected 1 model_policy relaxation conflict, got %+v", resConflict.View.Conflicts)
	}
	if resConflict.View.Models.Operations["review"] != "tier-high" {
		t.Fatalf("expected stricter tier-high to be retained, got %q", resConflict.View.Models.Operations["review"])
	}
}
