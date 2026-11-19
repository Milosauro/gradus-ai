package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestExplainCommand(t *testing.T) {
	tmp := t.TempDir()

	// Setup a two-level pyramid (team + project) with an override
	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
  - id: project
    source: { path: .gradus/levels/project }
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	if err := os.MkdirAll(teamRules, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}

	teamRule := `---
id: TEAM-SEC-001
title: Avoid hardcoded secrets
severity: MUST
enforcement: check
checker:
  name: gitleaks
source: "Team Security Policy v1"
---
Never commit hardcoded secrets.

## Why
Secrets in git history cannot be easily purged.

## Rationalizations
- "It is a test key" -> test keys must still be mocked.
`
	projRule := `---
id: PRJ-SEC-001
title: Strict secret scanning on fixtures
severity: MUST
enforcement: check
overrides: TEAM-SEC-001
checker:
  name: gitleaks
---
Scan all test fixtures as well.
`
	if err := os.WriteFile(filepath.Join(teamRules, "TEAM-SEC-001.md"), []byte(teamRule), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projRules, "PRJ-SEC-001.md"), []byte(projRule), 0o644); err != nil {
		t.Fatal(err)
	}

	// Text output
	var stdout, stderr bytes.Buffer
	code := cli.ExecuteCode([]string{"explain", "TEAM-SEC-001", "--dir", tmp}, &stdout, &stderr)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	text := stdout.String()
	if !strings.Contains(text, "TEAM-SEC-001: Avoid hardcoded secrets") ||
		!strings.Contains(text, "Overridden by: PRJ-SEC-001") ||
		!strings.Contains(text, "Team Security Policy v1") {
		t.Fatalf("unexpected explain text output:\n%s", text)
	}

	// JSON output
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"explain", "TEAM-SEC-001", "--dir", tmp, "--json"}, &stdout, &stderr)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	var res struct {
		Rule         schema.Rule `json:"rule"`
		OverriddenBy []string    `json:"overridden_by"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v", err)
	}
	if res.Rule.ID != "TEAM-SEC-001" || res.Rule.Level != "team" || len(res.OverriddenBy) != 1 || res.OverriddenBy[0] != "PRJ-SEC-001" {
		t.Fatalf("unexpected explain JSON result: %+v", res)
	}

	// Unknown rule ID -> ExitError (2)
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"explain", "UNKNOWN-001", "--dir", tmp}, &stdout, &stderr)
	if code != cli.ExitError {
		t.Fatalf("expected exit %d for unknown rule, got %d", cli.ExitError, code)
	}

	// Missing gradus.yaml -> ExitError (2)
	emptyDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"explain", "TEAM-SEC-001", "--dir", emptyDir}, &stdout, &stderr)
	if code != cli.ExitError {
		t.Fatalf("expected exit %d for missing gradus.yaml, got %d", cli.ExitError, code)
	}
}
