package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/state"
)

func runCLI(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cli.ExecuteCode(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestConflictsCommand(t *testing.T) {
	tmp := t.TempDir()
	teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	for _, d := range []string{teamRules, projRules} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: team\n    source: { path: .gradus/levels/team }\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(teamRules, "TEAM-LOG.md"), []byte("---\nid: TEAM-LOG\ntitle: Structured logging\nseverity: MUST\n---\nEmit structured JSON logging.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projRules, "PRJ-LOG.md"), []byte("---\nid: PRJ-LOG\ntitle: Plain logging\nseverity: SHOULD\n---\nEmit plain text logging.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. --candidates finds TEAM-LOG <-> PRJ-LOG
	code, out, _ := runCLI(t, []string{"conflicts", "--dir", tmp, "--candidates", "--json"})
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for --candidates, got %d (%s)", code, out)
	}
	var candResp struct {
		Candidates []state.CandidatePair `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(out), &candResp); err != nil {
		t.Fatal(err)
	}
	if len(candResp.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %+v", candResp.Candidates)
	}

	// 2. --dismiss suppresses unchanged pair
	code, out, _ = runCLI(t, []string{"conflicts", "--dir", tmp, "--dismiss", "--rule-a", "TEAM-LOG", "--rule-b", "PRJ-LOG", "--message", "Compatible scopes"})
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for --dismiss, got %d (%s)", code, out)
	}
	code, out, _ = runCLI(t, []string{"conflicts", "--dir", tmp, "--candidates", "--json"})
	if code != cli.ExitPass {
		t.Fatal(code)
	}
	_ = json.Unmarshal([]byte(out), &candResp)
	if len(candResp.Candidates) != 0 {
		t.Fatalf("expected 0 candidates after dismiss, got %+v", candResp.Candidates)
	}

	// 3. --confirm makes resolve fail with semantic conflict (exit 1)
	code, _, _ = runCLI(t, []string{"conflicts", "--dir", tmp, "--confirm", "--rule-a", "PRJ-LOG", "--rule-b", "TEAM-LOG", "--message", "Conflicting log formats"})
	if code != cli.ExitPass {
		t.Fatal(code)
	}
	code, _, _ = runCLI(t, []string{"resolve", "--dir", tmp, "--json"})
	if code != cli.ExitViolation {
		t.Fatalf("expected resolve exit 1 after confirmed semantic conflict, got %d", code)
	}

	// 4. Default list returns exit 0 with recorded finding
	code, out, _ = runCLI(t, []string{"conflicts", "--dir", tmp, "--json"})
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for conflicts list, got %d", code)
	}
	var listResp struct {
		Findings []state.FindingRecord `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &listResp); err != nil || len(listResp.Findings) != 1 {
		t.Fatalf("expected 1 finding listed, got %+v (err=%v)", listResp, err)
	}

	// 5. Unknown rule ID fails with exit 2
	code, _, _ = runCLI(t, []string{"conflicts", "--dir", tmp, "--confirm", "--rule-a", "NOPE", "--rule-b", "TEAM-LOG"})
	if code != cli.ExitError {
		t.Fatalf("expected exit 2 for unknown rule ID, got %d", code)
	}
}
