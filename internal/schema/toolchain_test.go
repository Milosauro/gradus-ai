package schema_test

import (
	"testing"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestToolchainSchema(t *testing.T) {
	validLevel := `schema: gradus/v1
id: team
toolchain:
  required:
    - { type: skill, id: gradus-review }
  recommended:
    - { type: tool, id: golangci-lint }
  forbidden:
    - { type: mcp, id: "*-public-pastebin", reason: "Data exfiltration risk" }
`
	res, err := schema.ParseLevelConfig([]byte(validLevel))
	if err != nil {
		t.Fatalf("expected valid level toolchain, got: %v", err)
	}
	if res.Value.Toolchain == nil || len(res.Value.Toolchain.Required) != 1 || len(res.Value.Toolchain.Forbidden) != 1 {
		t.Fatalf("unexpected toolchain parsed: %+v", res.Value.Toolchain)
	}

	validPyramid := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
toolchain:
  required:
    - { type: plugin, id: security-scanner }
`
	if _, err := schema.ParsePyramidConfig([]byte(validPyramid)); err != nil {
		t.Fatalf("expected valid pyramid toolchain, got: %v", err)
	}

	invalidType := `schema: gradus/v1
id: team
toolchain:
  required:
    - { type: unknown, id: foo }
`
	if _, err := schema.ParseLevelConfig([]byte(invalidType)); err == nil {
		t.Fatalf("expected invalid toolchain type to fail")
	}

	emptyID := `schema: gradus/v1
id: team
toolchain:
  forbidden:
    - { type: mcp, id: "" }
`
	if _, err := schema.ParseLevelConfig([]byte(emptyID)); err == nil {
		t.Fatalf("expected empty toolchain id to fail")
	}
}
