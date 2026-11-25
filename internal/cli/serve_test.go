package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestServeMCPCommandCLI(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Test rule\nseverity: MUST\nenforcement: prompt\n---\n\nAlways test.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	// 1. Omitting --mcp exits 2
	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"serve", "--dir", tmp}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 when --mcp is omitted, got %d", code)
	}

	// 2. serve --mcp processes stdio JSON-RPC requests and exits 0 on EOF
	cmd := cli.NewRootCmd()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"resolve_rules","arguments":{}}}` + "\n")
	stdout.Reset()
	stderr.Reset()
	cmd.SetIn(in)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"serve", "--mcp", "--dir", tmp})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("serve --mcp failed: %v (stderr=%s)", err, stderr.String())
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, `"protocolVersion":"2024-11-05"`) || !strings.Contains(outStr, "PRJ-01") {
		t.Fatalf("unexpected serve --mcp stdout:\n%s", outStr)
	}
}
