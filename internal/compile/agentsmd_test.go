package compile_test

import (
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestAgentsMDCompiler(t *testing.T) {
	view := resolve.ResolvedView{
		ViewHash: "sha256:abc",
		Levels: []resolve.LevelSummary{
			{ID: "team"},
			{ID: "project"},
			{ID: "user"},
		},
		Rules: []resolve.ResolvedRule{
			{
				ID:               "TEAM-01",
				Title:            "No secrets",
				Level:            "team",
				Severity:         "MUST",
				Enforcement:      "check",
				Scope:            []string{"**/*"},
				Text:             "Never commit real secrets.",
				Rationalizations: []string{"\"Only a test\" -> still forbidden."},
				Provenance:       resolve.RuleProvenance{Source: "team@v1.0.0"},
			},
			{
				ID:          "PRJ-01",
				Title:       "Use guard",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "hook",
				Scope:       []string{"**/*"},
				Text:        "Ask before force push.",
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "USR-01",
				Title:       "Personal habit",
				Level:       "user",
				Severity:    "MAY",
				Enforcement: "prompt",
				Scope:       []string{"**/*"},
				Text:        "User-only preference.",
				Provenance:  resolve.RuleProvenance{Source: "user"},
			},
		},
	}

	out := string(compile.PlanAgentsMD("# Existing Header\n", view))
	if !strings.Contains(out, "# Existing Header") {
		t.Fatalf("expected existing header preserved")
	}
	if !strings.Contains(out, "TEAM-01") || !strings.Contains(out, "PRJ-01") || !strings.Contains(out, "Only a test") {
		t.Fatalf("expected project and team rules (all enforcement tiers per ADR 0002) in AGENTS.md, got:\n%s", out)
	}
	if strings.Contains(out, "USR-01") {
		t.Fatalf("expected user-level rule USR-01 to be excluded from AGENTS.md")
	}
	if !strings.Contains(out, "team -> project -> user") {
		t.Fatalf("expected explicit precedence header (§6.3)")
	}
}
