package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestNestedGitBoundaryAndInvalidConfigV042(t *testing.T) {
	// Parent directory governed by Gradus
	parent := t.TempDir()
	_ = os.MkdirAll(filepath.Join(parent, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(parent, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parent, "gradus.yaml"), []byte("schema: gradus/v1\nanti_tamper:\n  mode: deny\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	// Simulate running gradusctl from inside the Gradus parent repository (v0.4.3 regression check).
	t.Chdir(parent)

	// R2.AC1: direct child (and its subdirectory) with .git and no gradus.yaml must NOT inherit parent's gradus.yaml
	nestedGit := filepath.Join(parent, "nested-repo")
	nestedSub := filepath.Join(nestedGit, "src")
	_ = os.MkdirAll(filepath.Join(nestedGit, ".git"), 0o755)
	_ = os.MkdirAll(nestedSub, 0o755)

	for _, testCwd := range []string{nestedGit, nestedSub} {
		nestedPayload := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"git commit --no-verify -m test"},"cwd":` + `"` + filepath.ToSlash(testCwd) + `"` + `}`
		var stdout, stderr bytes.Buffer
		cmd := cli.NewRootCmd()
		cmd.SetIn(strings.NewReader(nestedPayload))
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"guard"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("guard inside nested .git repo (%s) failed: %v (stderr=%s)", testCwd, err, stderr.String())
		}
		if strings.TrimSpace(stdout.String()) != "{}" {
			t.Fatalf("expected {} allow in nested standalone .git repo (%s), got: %s", testCwd, stdout.String())
		}
	}
	var stdout, stderr bytes.Buffer

	// R2.AC2: regular subdirectory without .git (e.g. .agents/) must still inherit parent's gradus.yaml
	agentsSub := filepath.Join(parent, ".agents")
	_ = os.MkdirAll(agentsSub, 0o755)
	subPayload := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"git commit --no-verify -m test"},"cwd":` + `"` + filepath.ToSlash(agentsSub) + `"` + `}`
	stdout.Reset()
	stderr.Reset()
	cmdSub := cli.NewRootCmd()
	cmdSub.SetIn(strings.NewReader(subPayload))
	cmdSub.SetOut(&stdout)
	cmdSub.SetErr(&stderr)
	cmdSub.SetArgs([]string{"guard"})
	if err := cmdSub.Execute(); err != nil {
		t.Fatalf("guard inside .agents/ subdirectory failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("expected parent anti-tamper deny inside .agents/ subdirectory, got: %s", stdout.String())
	}

	// R3.AC1 & R3.AC2: malformed gradus.yaml -> status --hook / --antigravity-hook exit 0 with lock=invalid;
	// non-hook status and guard exit 2.
	broken := t.TempDir()
	_ = os.WriteFile(filepath.Join(broken, "gradus.yaml"), []byte("schema: gradus/v1\nlevels: [invalid\n"), 0o644)

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", broken, "--brief", "--hook"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected exit 0 for status --hook on malformed gradus.yaml, got %d (stderr=%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "lock=invalid") {
		t.Fatalf("expected lock=invalid diagnostic in status --hook output, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", broken, "--antigravity-hook"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected exit 0 for status --antigravity-hook on malformed gradus.yaml, got %d (stderr=%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "injectSteps") || !strings.Contains(stdout.String(), "lock=invalid") {
		t.Fatalf("expected injectSteps lock=invalid JSON in status --antigravity-hook, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", broken}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for non-hook status on malformed gradus.yaml, got %d", code)
	}

	stdout.Reset()
	stderr.Reset()
	brokenPayload := `{"tool_name":"Bash","tool_input":{"command":"ls"},"cwd":` + `"` + filepath.ToSlash(broken) + `"` + `}`
	cmdBroken := cli.NewRootCmd()
	cmdBroken.SetIn(strings.NewReader(brokenPayload))
	cmdBroken.SetOut(&stdout)
	cmdBroken.SetErr(&stderr)
	cmdBroken.SetArgs([]string{"guard"})
	err := cmdBroken.Execute()
	if err == nil {
		t.Fatalf("expected guard to fail closed on malformed gradus.yaml")
	}
	var exitErr *cli.ExitErrorWithCode
	if !errors.As(err, &exitErr) || exitErr.Code != cli.ExitError {
		t.Fatalf("expected exit code 2 from guard on malformed gradus.yaml, got %v", err)
	}
}

func TestDocsAndDesignClosureV042(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	readme, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "repaired manually") {
		t.Fatalf("expected README.md to document manual repair for malformed gradus.yaml")
	}

	design, err := os.ReadFile(filepath.Join(repoRoot, "docs", "design", "gradus-design.md"))
	if err != nil {
		t.Fatal(err)
	}
	designText := string(design)
	if !strings.Contains(designText, "24.3 Phase 0 exit criteria **[Validated on `wallet-tracker`]**") {
		t.Fatalf("expected §24.3 in gradus-design.md to be marked Validated on wallet-tracker")
	}
	if !strings.Contains(designText, "| O-1 | Phase 0 exit criteria (§24.3): **Closed**") {
		t.Fatalf("expected O-1 in gradus-design.md to be marked Closed")
	}

	changelog, err := os.ReadFile(filepath.Join(repoRoot, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changelog), "## [0.4.2]") {
		t.Fatalf("expected CHANGELOG.md to contain ## [0.4.2]")
	}
}
