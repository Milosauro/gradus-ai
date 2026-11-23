package resolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestResolveObligationsAndBindings(t *testing.T) {
	setup := func(t *testing.T, teamLevelYAML, projectLevelYAML, gradusYAML string) resolve.ResolveResult {
		t.Helper()
		tmp := t.TempDir()
		teamDir := filepath.Join(tmp, "levels", "team", "rules")
		projDir := filepath.Join(tmp, "levels", "project", "rules")
		if err := os.MkdirAll(teamDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(projDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "levels", "team", "level.yaml"), []byte(teamLevelYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "levels", "project", "level.yaml"), []byte(projectLevelYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := resolve.Resolve(tmp, resolve.Options{})
		if err != nil {
			t.Fatalf("Resolve failed unexpectedly: %v", err)
		}
		return res
	}

	baseGradus := `schema: gradus/v1
levels:
  - id: team
    source: { path: levels/team }
  - id: project
    source: { path: levels/project }
`
	teamWithOb := `schema: gradus/v1
id: team
obligations:
  - id: OBL-SDD-01
    rule: "New behavior requires an approved specification before code."
    allowed_providers: [walden, spec-kit]
`

	// 1. Valid binding in project level -> satisfied, 0 conflicts, deterministic hash
	projValid := `schema: gradus/v1
id: project
bindings:
  - obligation: OBL-SDD-01
    provider: walden
    requires:
      cli: ">=1.4"
    scope: ["services/payments/**"]
`
	res1 := setup(t, teamWithOb, projValid, baseGradus)
	if len(res1.View.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got: %+v", res1.View.Conflicts)
	}
	if len(res1.View.Obligations) != 1 || !res1.View.Obligations[0].Satisfied || len(res1.View.Bindings) != 1 {
		t.Fatalf("expected 1 satisfied obligation and 1 binding, got: %+v %+v", res1.View.Obligations, res1.View.Bindings)
	}

	// 2. Disallowed provider -> obligation conflict
	projDisallowed := `schema: gradus/v1
id: project
bindings:
  - obligation: OBL-SDD-01
    provider: custom-tool
`
	res2 := setup(t, teamWithOb, projDisallowed, baseGradus)
	if len(res2.View.Conflicts) == 0 || res2.View.Conflicts[0].Kind != "obligation" || !strings.Contains(res2.View.Conflicts[0].Message, "disallowed provider") {
		t.Fatalf("expected disallowed provider conflict, got: %+v", res2.View.Conflicts)
	}

	// 3. Unknown obligation -> obligation conflict
	projUnknown := `schema: gradus/v1
id: project
bindings:
  - obligation: OBL-MISSING
    provider: walden
`
	res3 := setup(t, "schema: gradus/v1\nid: team\n", projUnknown, baseGradus)
	if len(res3.View.Conflicts) == 0 || res3.View.Conflicts[0].Kind != "obligation" || !strings.Contains(res3.View.Conflicts[0].Message, "unknown obligation") {
		t.Fatalf("expected unknown obligation conflict, got: %+v", res3.View.Conflicts)
	}

	// 4. Lower level widening allowed_providers -> obligation conflict
	projWiden := `schema: gradus/v1
id: project
obligations:
  - id: OBL-SDD-01
    rule: "Widened"
    allowed_providers: [walden, custom]
bindings:
  - obligation: OBL-SDD-01
    provider: walden
`
	res4 := setup(t, teamWithOb, projWiden, baseGradus)
	if len(res4.View.Conflicts) == 0 || res4.View.Conflicts[0].Kind != "obligation" || !strings.Contains(res4.View.Conflicts[0].Message, "widen allowed_providers") {
		t.Fatalf("expected widen allowed_providers conflict, got: %+v", res4.View.Conflicts)
	}

	// 5. Unbound obligation -> obligation conflict
	res5 := setup(t, teamWithOb, "schema: gradus/v1\nid: project\n", baseGradus)
	if len(res5.View.Conflicts) == 0 || res5.View.Conflicts[0].Kind != "obligation" || !strings.Contains(res5.View.Conflicts[0].Message, "no valid binding") {
		t.Fatalf("expected unbound obligation conflict, got: %+v", res5.View.Conflicts)
	}

	// 6. NFR2: Empty obligations/bindings omit fields cleanly
	res6 := setup(t, "schema: gradus/v1\nid: team\n", "schema: gradus/v1\nid: project\n", baseGradus)
	if res6.View.Obligations != nil || res6.View.Bindings != nil {
		t.Fatalf("expected nil obligations and bindings when none declared")
	}
}
