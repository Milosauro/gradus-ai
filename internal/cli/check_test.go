package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/evidence"
)

func runGitCLI(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Gradus Test",
		"GIT_AUTHOR_EMAIL=test@example.test",
		"GIT_COMMITTER_NAME=Gradus Test",
		"GIT_COMMITTER_EMAIL=test@example.test",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func TestCheckCommand(t *testing.T) {
	tmp := t.TempDir()
	runGitCLI(t, tmp, "init")

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"init", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("init failed: %d %s", code, stderr.String())
	}

	checkRule := `---
id: PRJ-SEC-001
title: Forbid realcompany email
severity: MUST
enforcement: check
scope: ["fixtures/**"]
checker:
  name: regex
  pattern: '@realcompany\.example'
---
Do not commit real emails.
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-SEC-001.md"), []byte(checkRule), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "fixtures", "bad.json"), []byte(`{"email":"a@realcompany.example"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCLI(t, tmp, "add", ".")

	stdout.Reset()
	stderr.Reset()
	code := cli.ExecuteCode([]string{"check", "--dir", tmp, "--staged", "--write-evidence", "--json"}, &stdout, &stderr)
	if code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on check violation, got %d (stderr: %s)", code, stderr.String())
	}

	var rep struct {
		Verdict         string             `json:"verdict"`
		Findings        []evidence.Finding `json:"findings"`
		EvidenceWritten []string           `json:"evidence_written"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("invalid check JSON: %v", err)
	}
	if rep.Verdict != "fail" || len(rep.Findings) != 1 || len(rep.EvidenceWritten) != 1 {
		t.Fatalf("unexpected check report: %+v", rep)
	}

	// Missing external checker -> exit 2
	badRule := "---\nid: PRJ-BAD-01\ntitle: Missing\nseverity: MUST\nenforcement: check\nchecker:\n  name: no-such-binary-xyz\n---\nCheck.\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-BAD-01.md"), []byte(badRule), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"check", "--dir", tmp, "--staged"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 on missing checker, got %d", code)
	}
}
