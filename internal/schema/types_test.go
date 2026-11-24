package schema_test

import (
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

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
models:
  default: sonnet
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
	if !slices.ContainsFunc(notes, func(s string) bool { return strings.Contains(s, "reserved key \"models\"") }) {
		t.Fatalf("expected note for reserved key 'models', got %v", notes)
	}
	if !slices.ContainsFunc(notes, func(s string) bool { return strings.Contains(s, "levels[0].extra_level_key") }) {
		t.Fatalf("expected note for nested unknown key 'levels[0].extra_level_key', got %v", notes)
	}
}

func TestCharterAndLifecycleSchemaValidation(t *testing.T) {
	// Valid level with charter and context_budget.
	levelYAML := `
schema: gradus/v1
id: team
charter:
  purpose: Team conventions
  allowed_severities: [SHOULD, MAY]
context_budget:
  max_rules: 10
  max_chars: 4000
`
	lvlRes, err := schema.ParseLevelConfig([]byte(levelYAML))
	if err != nil {
		t.Fatalf("expected valid level config, got %v", err)
	}
	if len(lvlRes.Notes) != 0 {
		t.Fatalf("expected zero unknown-key notes for context_budget, got %v", lvlRes.Notes)
	}
	if lvlRes.Value.ContextBudget == nil || *lvlRes.Value.ContextBudget.MaxRules != 10 || *lvlRes.Value.ContextBudget.MaxChars != 4000 {
		t.Fatalf("unexpected context_budget: %+v", lvlRes.Value.ContextBudget)
	}

	// Negative context_budget must fail.
	badBudgetYAML := `
schema: gradus/v1
id: team
context_budget:
  max_rules: -1
`
	if _, err := schema.ParseLevelConfig([]byte(badBudgetYAML)); err == nil {
		t.Fatalf("expected error for negative context_budget.max_rules")
	}

	// Valid rule with review_by and expires.
	ruleMD := `---
id: TEAM-TEST-001
title: Valid lifecycle rule
severity: SHOULD
status: deprecated
review_by: 2026-12-01
expires: 2027-01-15
---

Use canonical formatting.
`
	ruleRes, err := schema.ParseRuleBytes([]byte(ruleMD), "TEAM-TEST-001.md", "team", "team@v1")
	if err != nil {
		t.Fatalf("expected valid rule, got %v", err)
	}
	if len(ruleRes.Notes) != 0 {
		t.Fatalf("expected zero unknown-key notes for expires, got %v", ruleRes.Notes)
	}
	if ruleRes.Value.ReviewBy != "2026-12-01" || ruleRes.Value.Expires != "2027-01-15" {
		t.Fatalf("unexpected lifecycle dates: review_by=%q expires=%q", ruleRes.Value.ReviewBy, ruleRes.Value.Expires)
	}

	// Invalid date format must fail.
	badDateMD := `---
id: TEAM-TEST-002
title: Bad date rule
severity: SHOULD
review_by: not-a-date
---

Use canonical formatting.
`
	if _, err := schema.ParseRuleBytes([]byte(badDateMD), "TEAM-TEST-002.md", "team", "team@v1"); err == nil {
		t.Fatalf("expected error for invalid review_by date")
	}

	badExpiresMD := `---
id: TEAM-TEST-003
title: Bad expires rule
severity: SHOULD
expires: 2026/12/01
---

Use canonical formatting.
`
	if _, err := schema.ParseRuleBytes([]byte(badExpiresMD), "TEAM-TEST-003.md", "team", "team@v1"); err == nil {
		t.Fatalf("expected error for invalid expires date")
	}
}
