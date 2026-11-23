package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestStatusCommand(t *testing.T) {
	tmp := t.TempDir()

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"init", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("init failed: %d %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write failed: %d %s", code, stderr.String())
	}

	// Status --brief --json must be fast (< 100ms) and zero-network
	stdout.Reset()
	stderr.Reset()
	start := time.Now()
	code := cli.ExecuteCode([]string{"status", "--dir", tmp, "--brief", "--json"}, &stdout, &stderr)
	elapsed := time.Since(start)
	if code != cli.ExitPass {
		t.Fatalf("expected status exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("expected status --brief to finish under 100ms, took %s", elapsed)
	}

	var rep struct {
		LockState string `json:"lock_state"`
		Conflicts int    `json:"conflicts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("expected valid status JSON, got: %v", err)
	}
	if rep.LockState != "ok" || rep.Conflicts != 0 {
		t.Fatalf("unexpected status report: %+v", rep)
	}

	// Brief text mode
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"status", "--dir", tmp, "--brief"}, &stdout, &stderr)
	if code != cli.ExitPass || !strings.Contains(stdout.String(), "gradus: lock=ok") {
		t.Fatalf("unexpected brief text output: code=%d out=%q", code, stdout.String())
	}

	// Confirmed finding -> exit 1
	findDir := filepath.Join(tmp, ".gradus", "findings")
	if err := os.MkdirAll(findDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(findDir, "F-1.yaml"), []byte("status: confirmed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", tmp, "--brief"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 with confirmed finding, got %d", code)
	}

	// Drift detection after compile -> status reports drift=1 and exit 1
	_ = os.Remove(filepath.Join(findDir, "F-1.yaml"))
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed: %d %s", code, stderr.String())
	}
	_ = os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("hand edited\n"), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", tmp, "--brief"}, &stdout, &stderr); code != cli.ExitViolation || !strings.Contains(stdout.String(), "drift=1") {
		t.Fatalf("expected exit 1 with drift=1 after AGENTS.md edit, got code=%d out=%q", code, stdout.String())
	}

	// SessionStart hook mode (--brief --hook) must exit 0 with clean stderr even when drift > 0
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", tmp, "--brief", "--hook"}, &stdout, &stderr); code != cli.ExitPass || !strings.Contains(stdout.String(), "drift=1") || stderr.Len() != 0 {
		t.Fatalf("expected exit 0 with drift=1 and empty stderr in --hook mode, got code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
}
