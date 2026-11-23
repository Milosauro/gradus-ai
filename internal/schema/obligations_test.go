package schema_test

import (
	"testing"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestObligationsAndBindingsSchema(t *testing.T) {
	validLevel := `schema: gradus/v1
id: team
obligations:
  - id: OBL-SDD-01
    rule: "New behavior requires an approved specification before code."
    allowed_providers: [walden, spec-kit]
bindings:
  - obligation: OBL-SDD-01
    provider: walden
    requires:
      cli: ">=1.4"
    scope: ["services/payments/**"]
`
	lvlRes, err := schema.ParseLevelConfig([]byte(validLevel))
	if err != nil {
		t.Fatalf("expected valid level with obligations/bindings, got: %v", err)
	}
	if len(lvlRes.Value.Obligations) != 1 || lvlRes.Value.Obligations[0].ID != "OBL-SDD-01" {
		t.Fatalf("unexpected obligations: %+v", lvlRes.Value.Obligations)
	}
	if len(lvlRes.Value.Bindings) != 1 || lvlRes.Value.Bindings[0].Requires["cli"] != ">=1.4" {
		t.Fatalf("unexpected bindings: %+v", lvlRes.Value.Bindings)
	}

	validPyramid := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
bindings:
  - obligation: OBL-SDD-01
    provider: walden
`
	if _, err := schema.ParsePyramidConfig([]byte(validPyramid)); err != nil {
		t.Fatalf("expected valid pyramid with bindings, got: %v", err)
	}

	invalidCases := []struct {
		name  string
		level string
	}{
		{
			name: "missing obligation id",
			level: `schema: gradus/v1
id: team
obligations:
  - rule: "Missing id"
`,
		},
		{
			name: "missing obligation rule",
			level: `schema: gradus/v1
id: team
obligations:
  - id: OBL-1
`,
		},
		{
			name: "duplicate obligation id",
			level: `schema: gradus/v1
id: team
obligations:
  - id: OBL-1
    rule: "First"
  - id: OBL-1
    rule: "Duplicate"
`,
		},
		{
			name: "missing binding provider",
			level: `schema: gradus/v1
id: team
bindings:
  - obligation: OBL-1
`,
		},
	}

	for _, tc := range invalidCases {
		if _, err := schema.ParseLevelConfig([]byte(tc.level)); err == nil {
			t.Fatalf("expected %s to fail validation", tc.name)
		}
	}
}
