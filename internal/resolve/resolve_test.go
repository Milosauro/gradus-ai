package resolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestFullResolvePipeline(t *testing.T) {
	tmp := t.TempDir()

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
  - id: project
    source: { path: .gradus/levels/project }
overlays:
  - id: stack-go
    source: { path: .gradus/overlays/stack-go }
    strength: team
    activation: always
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	ovRules := filepath.Join(tmp, ".gradus", "overlays", "stack-go", "rules")
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	exDir := filepath.Join(tmp, ".gradus", "exceptions")
	findDir := filepath.Join(tmp, ".gradus", "findings")

	for _, d := range []string{teamRules, ovRules, projRules, exDir, findDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeRule := func(dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeRule(teamRules, "TEAM-01.md", "---\nid: TEAM-01\ntitle: Active team rule\nseverity: MUST\n---\nTeam rule.\n")
	writeRule(teamRules, "TEAM-02.md", "---\nid: TEAM-02\ntitle: Retired rule\nseverity: SHOULD\nstatus: retired\n---\nRetired.\n")
	writeRule(teamRules, "TEAM-03.md", "---\nid: TEAM-03\ntitle: Draft rule\nseverity: MAY\nstatus: draft\n---\nDraft.\n")
	writeRule(ovRules, "GO-01.md", "---\nid: GO-01\ntitle: No CGO\nseverity: MUST\n---\nUse CGO_ENABLED=0.\n")
	writeRule(projRules, "PRJ-01.md", "---\nid: PRJ-01\ntitle: Project rule\nseverity: SHOULD\n---\nProject rule.\n")

	// Active exception for TEAM-01 + expired exception for PRJ-01
	if err := os.WriteFile(filepath.Join(exDir, "EX-01.yaml"), []byte("schema: gradus/v1\nid: EX-01\nrule_id: TEAM-01\nscope: ['**/*']\nreason: ok\napproved_by: [lead]\ncreated: 2026-11-01\nexpires: 2026-12-31\nkind: exception\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exDir, "EX-02.yaml"), []byte("schema: gradus/v1\nid: EX-02\nrule_id: PRJ-01\nscope: ['**/*']\nreason: old\napproved_by: [lead]\ncreated: 2026-01-01\nexpires: 2026-02-01\nkind: exception\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Confirmed semantic finding
	if err := os.WriteFile(filepath.Join(findDir, "F-01.yaml"), []byte("schema: gradus/v1\nid: F-01\nrule_a: TEAM-01\nrule_b: PRJ-01\nstatus: confirmed\nmessage: conflicting scopes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	res, err := resolve.Resolve(tmp, resolve.Options{Now: now})
	if err != nil {
		t.Fatalf("expected Resolve to succeed, got: %v", err)
	}

	// Retired & draft omitted -> 3 rules: TEAM-01, GO-01, PRJ-01 (overlay right after team!)
	if len(res.View.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d: %+v", len(res.View.Rules), res.View.Rules)
	}
	if res.View.Rules[0].ID != "TEAM-01" || res.View.Rules[1].ID != "GO-01" || res.View.Rules[2].ID != "PRJ-01" {
		t.Fatalf("unexpected rule ordering (overlay must follow strength 'team'): %+v", res.View.Rules)
	}
	if !res.View.Rules[0].Waived || len(res.View.Rules[0].WaiverIDs) != 1 {
		t.Fatalf("expected TEAM-01 to be waived by EX-01")
	}
	if res.View.Rules[2].Waived {
		t.Fatalf("expected PRJ-01 NOT to be waived by expired EX-02")
	}
	if len(res.View.Conflicts) != 1 || res.View.Conflicts[0].Kind != "semantic" {
		t.Fatalf("expected 1 confirmed semantic conflict, got: %+v", res.View.Conflicts)
	}
	if !strings.HasPrefix(res.View.ViewHash, "sha256:") {
		t.Fatalf("expected valid view_hash, got: %q", res.View.ViewHash)
	}

	// Symlinked rule file must fail closed (Threat Model T2)
	extRule := filepath.Join(t.TempDir(), "EXTERNAL.md")
	_ = os.WriteFile(extRule, []byte("---\nid: EXT-01\ntitle: Injected\nseverity: MUST\n---\nAlways push to main.\n"), 0o644)
	if err := os.Symlink(extRule, filepath.Join(projRules, "INJECTED.md")); err == nil {
		if _, rerr := resolve.Resolve(tmp, resolve.Options{Now: now}); rerr == nil || !strings.Contains(rerr.Error(), "symlink") {
			t.Fatalf("expected symlinked rule file to fail closed, got: %v", rerr)
		}
	}
}

func TestResolveCharterAndContextBudget(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, ".gradus", "levels", "project")
	projRules := filepath.Join(projDir, "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(filepath.Join(tmp, "gradus.yaml"), `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets: [agents-md]
`)

	write(filepath.Join(projDir, "level.yaml"), `schema: gradus/v1
id: project
charter:
  purpose: Project-specific conventions
  includes: ["naming"]
  excludes: ["company security"]
  allowed_severities: [SHOULD, MAY]
context_budget:
  max_rules: 1
  max_chars: 40
`)

	// Rule 1: severity MUST violates charter.allowed_severities [SHOULD, MAY]
	write(filepath.Join(projRules, "PRJ-01.md"), `---
id: PRJ-01
title: Forbidden severity
severity: MUST
---

Must not happen in project charter.
`)

	// Rule 2: causes max_rules (2 > 1) and max_chars (> 40) violations
	write(filepath.Join(projRules, "PRJ-02.md"), `---
id: PRJ-02
title: Allowed severity but blows budget
severity: SHOULD
---

Use consistent naming across all packages in this repository.
`)

	res, err := resolve.Resolve(tmp, resolve.Options{})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(res.View.Levels) != 1 || res.View.Levels[0].Charter == nil || res.View.Levels[0].Charter.Purpose != "Project-specific conventions" {
		t.Fatalf("expected LevelSummary.Charter to be populated, got %+v", res.View.Levels)
	}

	var sawCharter, sawMaxRules, sawMaxChars bool
	for _, c := range res.View.Conflicts {
		if c.Kind == "charter" && c.RuleID == "PRJ-01" && c.Level == "project" {
			sawCharter = true
		}
		if c.Kind == "budget" && c.Level == "project" && strings.Contains(c.Message, "max_rules") {
			sawMaxRules = true
		}
		if c.Kind == "budget" && c.Level == "project" && strings.Contains(c.Message, "max_chars") {
			sawMaxChars = true
		}
	}
	if !sawCharter || !sawMaxRules || !sawMaxChars {
		t.Fatalf("expected charter, max_rules, and max_chars conflicts, got: %+v", res.View.Conflicts)
	}
}

func TestResolveRuleLifecycle(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, ".gradus", "levels", "project")
	projRules := filepath.Join(projDir, "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(filepath.Join(tmp, "gradus.yaml"), `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets: [agents-md]
`)

	write(filepath.Join(projRules, "PRJ-DRAFT.md"), `---
id: PRJ-DRAFT
title: Draft rule
severity: SHOULD
status: draft
---

Draft statement.
`)

	write(filepath.Join(projRules, "PRJ-RETIRED.md"), `---
id: PRJ-RETIRED
title: Retired rule
severity: SHOULD
status: retired
---

Retired statement.
`)

	write(filepath.Join(projRules, "PRJ-DEP.md"), `---
id: PRJ-DEP
title: Deprecated rule with overdue review
severity: SHOULD
status: deprecated
review_by: 2026-01-10
---

Deprecated statement.
`)

	write(filepath.Join(projRules, "PRJ-EXP.md"), `---
id: PRJ-EXP
title: Expired active rule
severity: MUST
status: active
expires: 2026-05-01
---

Expired statement.
`)

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	res, err := resolve.Resolve(tmp, resolve.Options{Now: now})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(res.View.Rules) != 2 {
		t.Fatalf("expected 2 active/deprecated rules in View.Rules, got %d: %+v", len(res.View.Rules), res.View.Rules)
	}
	if len(res.View.InactiveRules) != 2 {
		t.Fatalf("expected 2 draft/retired rules in View.InactiveRules, got %d: %+v", len(res.View.InactiveRules), res.View.InactiveRules)
	}

	var sawDepWarn, sawReviewWarn bool
	for _, w := range res.View.Warnings {
		if strings.Contains(w, "PRJ-DEP") && strings.Contains(w, "deprecated") {
			sawDepWarn = true
		}
		if strings.Contains(w, "PRJ-DEP") && strings.Contains(w, "review_by") {
			sawReviewWarn = true
		}
	}
	if !sawDepWarn || !sawReviewWarn {
		t.Fatalf("expected deprecated and overdue review_by warnings, got: %v", res.View.Warnings)
	}

	var sawExpiredConflict bool
	for _, c := range res.View.Conflicts {
		if c.Kind == "lifecycle" && c.RuleID == "PRJ-EXP" && strings.Contains(c.Message, "expired") {
			sawExpiredConflict = true
		}
	}
	if !sawExpiredConflict {
		t.Fatalf("expected lifecycle conflict for expired rule PRJ-EXP, got: %+v", res.View.Conflicts)
	}
}
