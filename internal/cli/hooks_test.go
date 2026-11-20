package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestHooksCommand(t *testing.T) {
	tmp := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmp, "lefthook.yml"), []byte(""), 0o644)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteCode([]string{"hooks", "print", "--dir", tmp, "--json"}, &stdout, &stderr)
	if code != cli.ExitPass {
		t.Fatalf("hooks print failed: %d %s", code, stderr.String())
	}
	var out struct {
		Manager string `json:"manager"`
		Snippet string `json:"snippet"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Manager != "lefthook" || !strings.Contains(out.Snippet, "gradusctl verify") {
		t.Fatalf("unexpected hooks print output: %+v", out)
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"hooks", "install", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("hooks install failed: %d %s", code, stderr.String())
	}
	lhBytes, _ := os.ReadFile(filepath.Join(tmp, "lefthook.yml"))
	if !strings.Contains(string(lhBytes), "gradusctl check --staged") {
		t.Fatalf("expected lefthook.yml updated, got:\n%s", string(lhBytes))
	}
}
