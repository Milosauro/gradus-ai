package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestReconcileCommand(t *testing.T) {
	tmp := t.TempDir()
	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	for _, d := range []string{teamRules, projRules} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: team
    source: { path: .gradus/levels/team }
    locked: true
  - id: project
    source: { path: .gradus/levels/project }
targets:
  - agents-md
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projRules, "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Project rule\nseverity: MUST\n---\nStay clean.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Resolve + compile clean baseline
	if code, _, _ := runCLI(t, []string{"resolve", "--dir", tmp, "--write"}); code != cli.ExitPass {
		t.Fatalf("resolve failed")
	}
	if code, _, _ := runCLI(t, []string{"compile", "--dir", tmp}); code != cli.ExitPass {
		t.Fatalf("compile failed")
	}

	// 1. Clean workspace -> reconcile returns exit 0
	if code, _, _ := runCLI(t, []string{"reconcile", "--dir", tmp, "--json"}); code != cli.ExitPass {
		t.Fatalf("expected exit 0 on clean reconcile, got %d", code)
	}

	// 2. Introduce drift -> reconcile returns exit 1
	_ = os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("manual tamper\n"), 0o644)
	if code, _, _ := runCLI(t, []string{"reconcile", "--dir", tmp, "--json"}); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on drifted reconcile, got %d", code)
	}

	// 3. --restore clears drift and verify passes
	if code, out, _ := runCLI(t, []string{"reconcile", "--dir", tmp, "--restore", "--json"}); code != cli.ExitPass {
		t.Fatalf("expected exit 0 on reconcile --restore, got %d (%s)", code, out)
	}
	if code, out, _ := runCLI(t, []string{"verify", "--dir", tmp, "--json"}); code != cli.ExitPass {
		t.Fatalf("expected verify exit 0 after reconcile --restore, got %d (%s)", code, out)
	}

	// 4. --move refuses locked team level (exit 2) and succeeds on unlocked project level (exit 0)
	newRule := filepath.Join(tmp, "PRJ-02.md")
	_ = os.WriteFile(newRule, []byte("---\nid: PRJ-02\ntitle: Second rule\nseverity: SHOULD\n---\nKeep tests small.\n"), 0o644)

	if code, _, _ := runCLI(t, []string{"reconcile", "--dir", tmp, "--move", "--rule-file", newRule, "--target-level", "team"}); code != cli.ExitError {
		t.Fatalf("expected exit 2 when moving into locked level, got %d", code)
	}
	if code, out, _ := runCLI(t, []string{"reconcile", "--dir", tmp, "--move", "--rule-file", newRule, "--target-level", "project", "--json"}); code != cli.ExitPass {
		t.Fatalf("expected exit 0 when moving into unlocked project level, got %d (%s)", code, out)
	}
	if _, err := os.Stat(filepath.Join(projRules, "PRJ-02.md")); err != nil {
		t.Fatalf("expected PRJ-02.md in project rules, err=%v", err)
	}
}
