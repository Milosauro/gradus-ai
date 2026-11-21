package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestVerifyCommand(t *testing.T) {
	tmp := t.TempDir()

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
    locked: true
  - id: project
    source: { path: .gradus/levels/project }
targets: [claude-code, antigravity, agents-md]
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	_ = os.MkdirAll(teamRules, 0o755)
	_ = os.MkdirAll(projRules, 0o755)
	teamRulePath := filepath.Join(teamRules, "TEAM-01.md")
	_ = os.WriteFile(teamRulePath, []byte("---\nid: TEAM-01\ntitle: Team Rule\nseverity: MUST\n---\nTeam rule.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Project Rule\nseverity: SHOULD\n---\nProject rule.\n"), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write failed: %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed: %d %s", code, stderr.String())
	}

	// 1. Clean verify -> exit 0
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected clean verify exit 0, got %d (stderr: %s)", code, stderr.String())
	}

	// 2. Hand-edit inside generated AGENTS.md managed block -> drift detected, exit 1 (E2)
	agentsPath := filepath.Join(tmp, "AGENTS.md")
	origAgents, _ := os.ReadFile(agentsPath)
	editedAgents := bytes.Replace(origAgents, []byte("Team rule."), []byte("Team rule (edited by hand)."), 1)
	_ = os.WriteFile(agentsPath, editedAgents, 0o644)

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on drift, got %d", code)
	}
	var rep struct {
		Drift []cli.DriftItem `json:"drift"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Drift) == 0 || rep.Drift[0].Path != "AGENTS.md" {
		t.Fatalf("expected AGENTS.md drift reported, got: %+v", rep.Drift)
	}

	// 3. Tamper locked level -> exit 2 (E2)
	_ = os.WriteFile(teamRulePath, []byte("---\nid: TEAM-01\ntitle: Tampered\nseverity: MUST\n---\nTampered.\n"), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 on tampered locked level, got %d", code)
	}
}

func TestCharterLifecycleAndBudgetCLI(t *testing.T) {
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
  purpose: Project rules only
  includes: ["go"]
  excludes: ["org security"]
  allowed_severities: [SHOULD, MAY]
context_budget:
  max_rules: 2
`)
	write(filepath.Join(projRules, "PRJ-DEP.md"), `---
id: PRJ-DEP
title: Deprecated rule
severity: SHOULD
status: deprecated
review_by: 2020-01-01
---

Deprecated guidance.
`)
	write(filepath.Join(projRules, "PRJ-DRAFT.md"), `---
id: PRJ-DRAFT
title: Draft hook rule
severity: SHOULD
enforcement: hook
status: draft
guard:
  action: bash
  pattern: "git status"
  decision: deny
---

Draft hook must not be enforced.
`)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}

	agentsData, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agentsData), "PRJ-DEP") || strings.Contains(string(agentsData), "PRJ-DRAFT") {
		t.Fatalf("expected PRJ-DEP compiled and PRJ-DRAFT excluded, got:\n%s", string(agentsData))
	}

	// Verify passes with warnings for deprecated and overdue review_by.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected verify pass with warnings, got %d: %s\n%s", code, stdout.String(), stderr.String())
	}
	var vRep struct {
		Verdict  string   `json:"verdict"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &vRep); err != nil {
		t.Fatalf("invalid verify json: %v\n%s", err, stdout.String())
	}
	if len(vRep.Warnings) < 2 {
		t.Fatalf("expected >=2 lifecycle warnings in verify --json, got %v", vRep.Warnings)
	}

	// Explain exposes level charter.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"explain", "PRJ-DEP", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("explain failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var eRep struct {
		Charter *struct {
			Purpose string `json:"purpose"`
		} `json:"charter"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &eRep); err != nil {
		t.Fatalf("invalid explain json: %v\n%s", err, stdout.String())
	}
	if eRep.Charter == nil || eRep.Charter.Purpose != "Project rules only" {
		t.Fatalf("expected charter in explain --json, got %+v", eRep)
	}

	// Adding an expired MUST rule triggers charter, lifecycle, and budget conflicts (exit 1).
	write(filepath.Join(projRules, "PRJ-BAD1.md"), `---
id: PRJ-BAD1
title: Expired MUST rule
severity: MUST
expires: 2020-01-01
---

Expired and forbidden severity.
`)
	write(filepath.Join(projRules, "PRJ-BAD2.md"), `---
id: PRJ-BAD2
title: Third active rule
severity: SHOULD
---

Exceeds max_rules=2.
`)
	_ = cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on charter/lifecycle/budget violations, got %d (%s)", code, stdout.String())
	}
}
