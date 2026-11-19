package schema_test

import (
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestOverridableAndUnknownKeys(t *testing.T) {
	// Test OverridableSpec as boolean
	var boolTarget struct {
		Overridable *schema.OverridableSpec `yaml:"overridable"`
	}
	if err := yaml.Unmarshal([]byte("overridable: true\n"), &boolTarget); err != nil {
		t.Fatalf("failed to unmarshal boolean overridable: %v", err)
	}
	if boolTarget.Overridable == nil || !boolTarget.Overridable.Allowed {
		t.Fatalf("expected Overridable.Allowed=true, got %+v", boolTarget.Overridable)
	}

	// Test OverridableSpec as object with constraints
	var objTarget struct {
		Overridable *schema.OverridableSpec `yaml:"overridable"`
	}
	if err := yaml.Unmarshal([]byte("overridable:\n  max_scope_depth: 2\n"), &objTarget); err != nil {
		t.Fatalf("failed to unmarshal object overridable: %v", err)
	}
	if objTarget.Overridable == nil || !objTarget.Overridable.Allowed || objTarget.Overridable.Constraints["max_scope_depth"] != 2 {
		t.Fatalf("expected Overridable object constraints, got %+v", objTarget.Overridable)
	}

	// Test CollectUnknownKeys for unknown and reserved keys
	yamlText := `
schema: gradus/v1
future_flag: true
obligations:
  - id: OBL-1
levels:
  - id: project
    extra_level_key: 123
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(yamlText), &doc); err != nil {
		t.Fatalf("failed to parse YAML node: %v", err)
	}

	spec := schema.FieldSpec{
		Allowed: map[string]schema.FieldSpec{
			"schema": {},
			"levels": {
				IsSeq: true,
				Allowed: map[string]schema.FieldSpec{
					"id": {},
				},
			},
		},
	}

	notes := schema.CollectUnknownKeys(&doc, spec, "", true)
	if !slices.ContainsFunc(notes, func(s string) bool { return strings.Contains(s, "future_flag") }) {
		t.Fatalf("expected note for unknown key 'future_flag', got %v", notes)
	}
	if !slices.ContainsFunc(notes, func(s string) bool { return strings.Contains(s, "reserved key \"obligations\"") }) {
		t.Fatalf("expected note for reserved key 'obligations', got %v", notes)
	}
	if !slices.ContainsFunc(notes, func(s string) bool { return strings.Contains(s, "levels[0].extra_level_key") }) {
		t.Fatalf("expected note for nested unknown key 'levels[0].extra_level_key', got %v", notes)
	}
}
