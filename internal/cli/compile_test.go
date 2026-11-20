package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/compile"
)

func TestCompileCommand(t *testing.T) {
	tmp := t.TempDir()

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"init", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("init failed: %d %s", code, stderr.String())
	}

	// 1. First compile --json -> exit 0 with changed_files
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("first compile failed: %d %s", code, stderr.String())
	}
	var res1 compile.CompileResult
	if err := json.Unmarshal(stdout.Bytes(), &res1); err != nil {
		t.Fatalf("expected valid compile JSON, got: %v", err)
	}
	if len(res1.ChangedFiles) == 0 {
		t.Fatalf("expected changed files on first compile")
	}

	// 2. Second compile --json -> exit 0 with 0 changed_files (idempotent, E1)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("second compile failed: %d %s", code, stderr.String())
	}
	var res2 compile.CompileResult
	if err := json.Unmarshal(stdout.Bytes(), &res2); err != nil {
		t.Fatalf("expected valid compile JSON, got: %v", err)
	}
	if len(res2.ChangedFiles) != 0 {
		t.Fatalf("expected 0 changed files on second compile, got: %v", res2.ChangedFiles)
	}

	// 3. Introduce duplicate conflict -> compile exits 1 and leaves AGENTS.md untouched
	beforeAgents, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	dupPath := filepath.Join(tmp, ".gradus", "levels", "project", "rules", "DUP.md")
	if err := os.WriteFile(dupPath, []byte("---\nid: PRJ-GEN-001\ntitle: Dup\nseverity: MUST\n---\nDup.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on conflict, got %d", code)
	}
	afterAgents, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeAgents, afterAgents) {
		t.Fatalf("expected AGENTS.md untouched on conflict")
	}

	// 4. Missing gradus.yaml -> exit 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", t.TempDir()}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 on missing gradus.yaml, got %d", code)
	}
}
