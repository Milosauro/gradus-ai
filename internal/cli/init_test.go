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

func TestInitCommand(t *testing.T) {
	tmp := t.TempDir()

	// Seed an existing .gitignore
	existingGitignore := "dist/\n*.log\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte(existingGitignore), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteCode([]string{"init", "--dir", tmp, "--json"}, &stdout, &stderr)
	if code != cli.ExitPass {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}

	var res struct {
		CreatedFiles []string `json:"created_files"`
		UpdatedFiles []string `json:"updated_files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v (raw: %q)", err, stdout.String())
	}
	if len(res.CreatedFiles) < 3 {
		t.Fatalf("expected at least 3 created files, got %v", res.CreatedFiles)
	}
	if len(res.UpdatedFiles) != 1 || res.UpdatedFiles[0] != ".gitignore" {
		t.Fatalf("expected .gitignore in updated_files, got %v", res.UpdatedFiles)
	}

	// Verify existing .gitignore entries preserved and Gradus entries appended
	giBytes, err := os.ReadFile(filepath.Join(tmp, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	gi := string(giBytes)
	if !strings.Contains(gi, "dist/") || !strings.Contains(gi, ".gradus/cache/") || !strings.Contains(gi, "CLAUDE.local.md") {
		t.Fatalf("unexpected .gitignore contents:\n%s", gi)
	}

	// Second run without --force must exit with ExitError (2)
	stdout.Reset()
	stderr.Reset()
	code = cli.ExecuteCode([]string{"init", "--dir", tmp}, &stdout, &stderr)
	if code != cli.ExitError {
		t.Fatalf("expected duplicate init without --force to exit %d, got %d", cli.ExitError, code)
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("expected 'already exists' on stderr, got %q", stderr.String())
	}
}
