package schema_test

import (
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
