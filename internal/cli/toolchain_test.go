package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/toolchain"
)

func TestToolchainCommand(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".claude", "skills", "gradus-review"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".claude", "skills", "gradus-review", "SKILL.md"), []byte("---\nname: gradus-review\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
toolchain:
  required:
    - { type: skill, id: gradus-review }
  recommended:
    - { type: tool, id: non-existent-rec-tool-xyz }
  forbidden:
    - { type: plugin, id: "evil-*" }
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Pass with recommended warning
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"toolchain", "check", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass || stderr.Len() != 0 {
		t.Fatalf("expected exit 0 with empty stderr, got code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}
	var rep toolchain.Report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("expected valid JSON report: %v", err)
	}
	if rep.Verdict != "pass" || len(rep.Warnings) != 1 {
		t.Fatalf("unexpected report: %+v", rep)
	}

	// 2. Forbidden plugin enabled in .claude/settings.json -> exit 1 with clean JSON
	if err := os.WriteFile(filepath.Join(tmp, ".claude", "settings.json"), []byte(`{"enabledPlugins":{"evil-helper":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"toolchain", "check", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation || stderr.Len() != 0 {
		t.Fatalf("expected exit 1 with empty stderr, got code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}
	var repFail toolchain.Report
	if err := json.Unmarshal(stdout.Bytes(), &repFail); err != nil {
		t.Fatalf("expected valid JSON fail report: %v", err)
	}
	if repFail.Verdict != "fail" || len(repFail.Findings) != 1 {
		t.Fatalf("unexpected fail report: %+v", repFail)
	}
}
