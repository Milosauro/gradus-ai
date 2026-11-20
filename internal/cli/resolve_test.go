package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func TestResolveCommand(t *testing.T) {
	tmp := t.TempDir()

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
    locked: true
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
	teamRulePath := filepath.Join(teamRules, "TEAM-01.md")
	if err := os.WriteFile(teamRulePath, []byte("---\nid: TEAM-01\ntitle: T\nseverity: MUST\n---\nTeam rule.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projRules, "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: P\nseverity: SHOULD\n---\nProject rule.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Clean resolve --write --json -> exit 0 and creates gradus.lock
	var stdout, stderr bytes.Buffer
	code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write", "--json"}, &stdout, &stderr)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	var view resolve.ResolvedView
	if err := json.Unmarshal(stdout.Bytes(), &view); err != nil {
		t.Fatalf("expected valid JSON resolved view, got: %v", err)
	}
	if len(view.Rules) != 2 || len(view.Conflicts) != 0 {
		t.Fatalf("unexpected view: %+v", view)
	}
	if _, err := os.Stat(filepath.Join(tmp, "gradus.lock")); err != nil {
		t.Fatalf("expected gradus.lock to be written: %v", err)
	}

	// 2. Tamper locked level -> exit 2
	if err := os.WriteFile(teamRulePath, []byte("---\nid: TEAM-01\ntitle: Tampered\nseverity: MUST\n---\nTampered.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--json"}, &stdout, &stderr)
	if code != cli.ExitError {
		t.Fatalf("expected exit 2 on tampered locked level, got %d", code)
	}

	// 3. Remove lockfile and re-lock, then add a duplicate rule conflict -> exit 1 + JSON still emitted
	if err := os.Remove(filepath.Join(tmp, "gradus.lock")); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr)
	if code != cli.ExitPass {
		t.Fatalf("expected re-lock --write to succeed, got %d (stderr: %s)", code, stderr.String())
	}
	if err := os.WriteFile(filepath.Join(projRules, "DUP.md"), []byte("---\nid: TEAM-01\ntitle: Dup\nseverity: MUST\n---\nDuplicate.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--json"}, &stdout, &stderr)
	if code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on duplicate conflict, got %d", code)
	}
	var conflictView resolve.ResolvedView
	if err := json.Unmarshal(stdout.Bytes(), &conflictView); err != nil {
		t.Fatalf("expected JSON resolved view even on conflict, got: %v", err)
	}
	if len(conflictView.Conflicts) == 0 {
		t.Fatalf("expected non-empty conflicts array")
	}
}
