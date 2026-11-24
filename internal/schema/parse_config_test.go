package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestConfigParseAndValidate(t *testing.T) {
	validGradus := `
schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
    approvers: [repo-codeowners]
targets: [claude-code, antigravity, agents-md]
hooks:
  manager: auto
evidence:
  sinks:
    - type: local
`
	res, err := schema.ParsePyramidConfig([]byte(validGradus))
	if err != nil {
		t.Fatalf("expected valid gradus.yaml to parse, got error: %v", err)
	}
	if len(res.Value.Levels) != 1 || res.Value.Levels[0].ID != "project" {
		t.Fatalf("unexpected parsed levels: %+v", res.Value.Levels)
	}
	if len(res.Notes) != 0 {
		t.Fatalf("expected zero notes for standard keys, got: %v", res.Notes)
	}

	// Invalid cases
	invalidCases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "unsupported schema",
			yaml: "schema: gradus/v99\nlevels:\n  - id: project\n    source: { path: . }\n",
			want: "unsupported schema",
		},
		{
			name: "invalid level ID",
			yaml: "schema: gradus/v1\nlevels:\n  - id: Project_1\n    source: { path: . }\n",
			want: "must match",
		},
		{
			name: "duplicate level ID",
			yaml: "schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: a }\n  - id: project\n    source: { path: b }\n",
			want: "duplicate level id",
		},
		{
			name: "malformed yaml",
			yaml: "schema: [\n",
			want: "invalid gradus.yaml YAML",
		},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := schema.ParsePyramidConfig([]byte(tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}

	// Level config
	validLevel := `
schema: gradus/v1
id: team
version: 1.2.0
owners: [platform-team]
approvers: [team-leads]
charter:
  purpose: "Team conventions"
  allowed_severities: [MUST, SHOULD]
exceptions:
  max_duration: 30d
  break_glass: true
`
	lvlRes, err := schema.ParseLevelConfig([]byte(validLevel))
	if err != nil {
		t.Fatalf("expected valid level.yaml to parse, got: %v", err)
	}
	if lvlRes.Value.ID != "team" || lvlRes.Value.Charter == nil || !lvlRes.Value.Exceptions.BreakGlass {
		t.Fatalf("unexpected parsed level.yaml: %+v", lvlRes.Value)
	}

	// Overlay config
	validOverlay := `
schema: gradus/v1
id: stack-python-fastapi
strength: team
activation: profile
`
	ovRes, err := schema.ParseOverlayConfig([]byte(validOverlay))
	if err != nil {
		t.Fatalf("expected valid overlay.yaml to parse, got: %v", err)
	}
	if ovRes.Value.Strength != "team" || ovRes.Value.Activation != "profile" {
		t.Fatalf("unexpected parsed overlay.yaml: %+v", ovRes.Value)
	}

	if _, err := schema.ParseOverlayConfig([]byte("schema: gradus/v1\nid: gdpr\nstrength: Bad_Strength\n")); err == nil {
		t.Fatalf("expected invalid overlay strength to fail")
	}
}

func TestProfileAndActivationRuleParsing(t *testing.T) {
	// 1. Extensible profile parsing + JSON round-trip + Lookup
	gradusYAML := `
schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
profile:
  data: personal
  exposure: public
  domain: [payments, billing]
  client: [acme]
  region: eu
  regulated: true
  stack: [go, postgres]
`
	pRes, err := schema.ParsePyramidConfig([]byte(gradusYAML))
	if err != nil {
		t.Fatalf("ParsePyramidConfig failed: %v", err)
	}
	if len(pRes.Notes) != 0 {
		t.Fatalf("expected zero unknown-key notes for extensible profile, got %v", pRes.Notes)
	}
	if pRes.Value.Profile == nil {
		t.Fatalf("expected non-nil Profile")
	}
	if vals, ok := pRes.Value.Profile.Lookup("region"); !ok || len(vals) != 1 || vals[0] != "eu" {
		t.Fatalf("expected custom profile lookup region=eu, got %v (ok=%v)", vals, ok)
	}
	if vals, ok := pRes.Value.Profile.Lookup("regulated"); !ok || len(vals) != 1 || vals[0] != "true" {
		t.Fatalf("expected custom profile lookup regulated=true, got %v (ok=%v)", vals, ok)
	}
	if vals, ok := pRes.Value.Profile.Lookup("stack"); !ok || len(vals) != 2 {
		t.Fatalf("expected custom profile lookup stack=[go postgres], got %v (ok=%v)", vals, ok)
	}

	rawJSON, err := json.Marshal(pRes.Value.Profile)
	if err != nil {
		t.Fatal(err)
	}
	var roundTripped schema.Profile
	if err := json.Unmarshal(rawJSON, &roundTripped); err != nil {
		t.Fatal(err)
	}
	if vals, ok := roundTripped.Lookup("region"); !ok || vals[0] != "eu" {
		t.Fatalf("expected JSON round-trip to preserve custom profile key region, got %+v", roundTripped)
	}

	// 2. Valid level.yaml activation_rules with zero notes
	levelYAML := `
schema: gradus/v1
id: org
activation_rules:
  - overlay: gdpr
    when:
      data: personal
      region: [eu, uk]
    required: true
`
	lRes, err := schema.ParseLevelConfig([]byte(levelYAML))
	if err != nil {
		t.Fatalf("ParseLevelConfig failed: %v", err)
	}
	if len(lRes.Notes) != 0 {
		t.Fatalf("expected zero notes for activation_rules, got %v", lRes.Notes)
	}
	if len(lRes.Value.ActivationRules) != 1 || lRes.Value.ActivationRules[0].Overlay != "gdpr" || !lRes.Value.ActivationRules[0].Required {
		t.Fatalf("unexpected parsed activation_rules: %+v", lRes.Value.ActivationRules)
	}

	// 3. Invalid activation_rules rejected (empty when)
	badLevel := "schema: gradus/v1\nid: org\nactivation_rules:\n  - overlay: gdpr\n    when: {}\n"
	if _, err := schema.ParseLevelConfig([]byte(badLevel)); err == nil {
		t.Fatalf("expected empty activation_rules[].when to fail validation")
	}

	// 4. Overlay when predicate parsed with zero notes
	ovYAML := `
schema: gradus/v1
id: pci-dss
strength: org
activation: profile
when:
  domain: payments
  exposure: [public, partner]
`
	ovRes, err := schema.ParseOverlayConfig([]byte(ovYAML))
	if err != nil {
		t.Fatalf("ParseOverlayConfig failed: %v", err)
	}
	if len(ovRes.Notes) != 0 {
		t.Fatalf("expected zero notes for overlay when, got %v", ovRes.Notes)
	}
	if len(ovRes.Value.When) != 2 {
		t.Fatalf("expected 2 predicates in overlay when, got %+v", ovRes.Value.When)
	}
}
