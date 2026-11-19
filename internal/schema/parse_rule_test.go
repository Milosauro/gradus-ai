package schema_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestRuleParseAndLevelDerivation(t *testing.T) {
	validRule := `---
id: ORG-SEC-004
title: No real personal data in tests
severity: MUST
enforcement: check
scope: ["tests/**"]
source: "Internal data policy v3, section 4.2"
checker:
  name: gitleaks
  config: .gitleaks.toml
tags: [data, testing]
version: 3
future_rule_meta: hello
---
Test fixtures, logs and examples must never contain real personal data.

## Why
Repositories are copied and cached far more widely than production databases.

## Rationalizations
- "It is only a local test" -> the rule applies to local tests.
- "I will anonymize it later" -> real data must never enter the repository.

## Examples
### Non-compliant
` + "```json\n{ \"email\": \"mario.rossi@realcompany.example\" }\n```" + `
### Compliant
` + "```json\n{ \"email\": \"user-001@example.test\" }\n```" + `
`

	// ADR 0004: Rule ID is ORG-SEC-004, but loaded inside "team" level -> Level must be "team".
	res, err := schema.ParseRuleBytes([]byte(validRule), "rules/ORG-SEC-004.md", "team", "team@v1.2.0")
	if err != nil {
		t.Fatalf("expected valid rule to parse, got error: %v", err)
	}

	r := res.Value
	if r.Level != "team" {
		t.Fatalf("expected derived level 'team' (ADR 0004), got %q", r.Level)
	}
	if r.ID != "ORG-SEC-004" || r.Severity != "MUST" || r.Enforcement != "check" || r.Status != "active" {
		t.Fatalf("unexpected rule metadata: %+v", r)
	}
	if !strings.Contains(r.Statement, "never contain real personal data") {
		t.Fatalf("unexpected statement: %q", r.Statement)
	}
	if !strings.Contains(r.Why, "production databases") {
		t.Fatalf("unexpected why: %q", r.Why)
	}
	if len(r.Rationalizations) != 2 {
		t.Fatalf("expected 2 rationalizations, got %d", len(r.Rationalizations))
	}
	if len(r.Examples.NonCompliant) != 1 || len(r.Examples.Compliant) != 1 {
		t.Fatalf("expected 1 non-compliant and 1 compliant example, got %+v", r.Examples)
	}
	if !slices.ContainsFunc(res.Notes, func(s string) bool { return strings.Contains(s, "future_rule_meta") }) {
		t.Fatalf("expected unknown frontmatter key note for future_rule_meta, got %v", res.Notes)
	}

	// Default enforcement/scope/status
	minimalRule := `---
id: PRJ-NAM-001
title: Use camelCase
severity: SHOULD
---
Use camelCase for local variables.
`
	minRes, err := schema.ParseRuleBytes([]byte(minimalRule), "rules/PRJ-NAM-001.md", "project", "local")
	if err != nil {
		t.Fatalf("expected minimal rule to parse, got error: %v", err)
	}
	if minRes.Value.Enforcement != "prompt" || minRes.Value.Status != "active" || len(minRes.Value.Scope) != 1 || minRes.Value.Scope[0] != "**/*" {
		t.Fatalf("unexpected defaults: %+v", minRes.Value)
	}

	// Invalid rule cases
	invalidCases := []struct {
		name string
		md   string
		want string
	}{
		{
			name: "missing frontmatter",
			md:   "Just markdown without frontmatter.",
			want: "missing YAML frontmatter",
		},
		{
			name: "missing statement",
			md:   "---\nid: PRJ-01\ntitle: T\nseverity: MUST\n---\n\n## Why\nOnly why.\n",
			want: "non-empty statement",
		},
		{
			name: "check without checker",
			md:   "---\nid: PRJ-01\ntitle: T\nseverity: MUST\nenforcement: check\n---\nCheck something.\n",
			want: "requires checker.name",
		},
		{
			name: "hook without guard",
			md:   "---\nid: PRJ-01\ntitle: T\nseverity: MUST\nenforcement: hook\n---\nBlock something.\n",
			want: "requires guard.action and guard.pattern",
		},
		{
			name: "hook with invalid decision",
			md:   "---\nid: PRJ-01\ntitle: T\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: '^git'\n  decision: allow\n---\nBlock.\n",
			want: "must be 'deny' or 'ask'",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := schema.ParseRuleBytes([]byte(tc.md), "bad.md", "project", "local")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}
