package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestExtraTargetsCLIAndMatrix(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	rule := `---
id: PRJ-GO-01
title: Static Go builds
severity: MUST
enforcement: check
scope: ["**/*.go"]
checker:
  name: pattern
  pattern: "cgo_placeholder"
---

Use CGO_ENABLED=0.
`
	if err := os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-GO-01.md"), []byte(rule), 0o644); err != nil {
		t.Fatal(err)
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets:
  - codex
  - copilot
  - opencode
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. First compile writes all target files
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("first compile failed: %d %s", code, stderr.String())
	}

	// 2. Second compile reports Changed: 0 (R4.AC1)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass || !strings.Contains(stdout.String(), "Changed: 0") {
		t.Fatalf("expected Changed: 0 on second compile, got code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}

	// 3. verify passes when clean
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("verify failed: %d out=%s err=%s", code, stdout.String(), stderr.String())
	}

	// 4. Modifying .github/copilot-instructions.md triggers drift in verify (R4.AC1 / R4.AC2)
	if err := os.WriteFile(filepath.Join(tmp, ".github", "copilot-instructions.md"), []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"verify", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation || !strings.Contains(stdout.String(), ".github/copilot-instructions.md") {
		t.Fatalf("expected exit 1 with copilot drift, got code=%d out=%s", code, stdout.String())
	}

	// 5. Unsupported target -> exit 2 (R4.AC3)
	badCfg := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets:
  - unknown-agent
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(badCfg), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for unknown target, got %d", code)
	}

	// 6. docs/compatibility-matrix.md covers all six official targets (R4.AC4)
	matrixBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "compatibility-matrix.md"))
	if err != nil {
		t.Fatalf("failed to read docs/compatibility-matrix.md: %v", err)
	}
	matrix := string(matrixBytes)
	for _, target := range []string{"claude-code", "antigravity", "agents-md", "codex", "copilot", "opencode"} {
		if !strings.Contains(matrix, "`"+target+"`") {
			t.Fatalf("compatibility-matrix.md missing target %q", target)
		}
	}
}
