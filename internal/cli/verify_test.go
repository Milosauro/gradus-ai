package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
