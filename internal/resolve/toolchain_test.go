package resolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestResolveToolchain(t *testing.T) {
	setup := func(t *testing.T, teamLevelYAML, projectLevelYAML, gradusYAML string) resolve.ResolveResult {
		t.Helper()
		tmp := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tmp, "levels", "team", "rules"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tmp, "levels", "project", "rules"), 0o755); err != nil {
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
			t.Fatalf("Resolve failed: %v", err)
		}
		return res
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: levels/team }
  - id: project
    source: { path: levels/project }
`
	teamTC := `schema: gradus/v1
id: team
toolchain:
  required:
    - { type: skill, id: gradus-review }
  forbidden:
    - { type: mcp, id: "*-public-pastebin" }
`
	projClean := `schema: gradus/v1
id: project
toolchain:
  recommended:
    - { type: tool, id: golangci-lint }
`

	// 1. Clean union across levels
	res1 := setup(t, teamTC, projClean, gradusYAML)
	if len(res1.View.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got: %+v", res1.View.Conflicts)
	}
	if res1.View.Toolchain == nil || len(res1.View.Toolchain.Required) != 1 || len(res1.View.Toolchain.Recommended) != 1 || len(res1.View.Toolchain.Forbidden) != 1 {
		t.Fatalf("unexpected resolved toolchain: %+v", res1.View.Toolchain)
	}

	// 2. Lower level enabling forbidden MCP pattern -> toolchain conflict
	projConflict := `schema: gradus/v1
id: project
toolchain:
  required:
    - { type: mcp, id: acme-public-pastebin }
`
	res2 := setup(t, teamTC, projConflict, gradusYAML)
	if len(res2.View.Conflicts) == 0 || res2.View.Conflicts[0].Kind != "toolchain" || !strings.Contains(res2.View.Conflicts[0].Message, "*-public-pastebin") {
		t.Fatalf("expected toolchain conflict, got: %+v", res2.View.Conflicts)
	}

	// 3. NFR1: Absent toolchain keeps ResolvedView.Toolchain nil
	res3 := setup(t, "schema: gradus/v1\nid: team\n", "schema: gradus/v1\nid: project\n", gradusYAML)
	if res3.View.Toolchain != nil {
		t.Fatalf("expected nil Toolchain when none declared, got: %+v", res3.View.Toolchain)
	}
}
