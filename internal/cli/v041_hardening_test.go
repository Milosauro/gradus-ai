package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestGuardAndStatusOutsideGradusV041(t *testing.T) {
	// 1. Non-Gradus git repo: Claude Code PreToolUse payload must exit 0 with "{}"
	nonGradusRepo := t.TempDir()
	cmdInit := exec.Command("git", "init")
	cmdInit.Dir = nonGradusRepo
	if out, err := cmdInit.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(out))
	}

	payload := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls -la"},"cwd":` + `"` + filepath.ToSlash(nonGradusRepo) + `"` + `}`
	cmd := cli.NewRootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(payload))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"guard"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("guard inside non-Gradus repo failed: %v (stderr=%s)", err, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "{}" {
		t.Fatalf("expected empty JSON {} allow response outside Gradus, got %q", stdout.String())
	}

	// 2. Non-git directory: status --brief --hook must exit 0 silently
	plainDir := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", plainDir, "--brief", "--hook"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected exit 0 for status --hook outside Gradus, got %d (stderr=%s)", code, stderr.String())
	}

	// 3. Subdirectory inside a Gradus repo: guard walks upward to gradus.yaml and enforces rules
	gradusRepo := t.TempDir()
	subDir := filepath.Join(gradusRepo, "services", "billing")
	_ = os.MkdirAll(subDir, 0o755)
	_ = os.MkdirAll(filepath.Join(gradusRepo, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(gradusRepo, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(gradusRepo, "gradus.yaml"), []byte("schema: gradus/v1\nanti_tamper:\n  mode: deny\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	subPayload := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"git commit --no-verify -m skip"},"cwd":` + `"` + filepath.ToSlash(subDir) + `"` + `}`
	cmdSub := cli.NewRootCmd()
	stdout.Reset()
	stderr.Reset()
	cmdSub.SetIn(strings.NewReader(subPayload))
	cmdSub.SetOut(&stdout)
	cmdSub.SetErr(&stderr)
	cmdSub.SetArgs([]string{"guard"})
	if err := cmdSub.Execute(); err != nil {
		t.Fatalf("guard in Gradus subdirectory failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("expected anti-tamper deny when guard invoked from Gradus subdirectory, got:\n%s", stdout.String())
	}
}

func TestTrailersHistoricalAndResolveGuardV041(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "org", "rules"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, ".gradus", "levels", "project", "rules"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "level.yaml"), []byte("schema: gradus/v1\nid: org\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "rules", "ORG-01.md"), []byte("---\nid: ORG-01\ntitle: V1\nseverity: MUST\nenforcement: prompt\n---\n\nV1.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "level.yaml"), []byte("schema: gradus/v1\nid: project\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: org\n    source: { path: .gradus/levels/org }\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644)

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}
	runGit("init")
	runGit("config", "user.name", "Gradus Test")
	runGit("config", "user.email", "test@gradus.local")

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write failed: %d %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "generate", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("trailers generate failed: %d %s", code, stderr.String())
	}
	v1Trailers := stdout.String()
	runGit("add", ".")
	runGit("commit", "-m", "feat: v1\n\n"+v1Trailers)

	// Update rule + gradus.lock in a second commit with a forged all-zero Gradus-Lock-Hash
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "org", "rules", "ORG-01.md"), []byte("---\nid: ORG-01\ntitle: V2\nseverity: MUST\nenforcement: prompt\n---\n\nV2.\n"), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write v2 failed: %d %s", code, stderr.String())
	}
	runGit("add", ".")
	runGit("commit", "-m", "feat: forged\n\nGradus-Lock-Hash: sha256:0000000000000000000000000000000000000000000000000000000000000000")

	// 1. Historical v1 commit (HEAD~1) MUST still pass against its own gradus.lock!
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "verify", "--dir", tmp, "--ref", "HEAD~1", "--require-lock-hash", "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected historical commit HEAD~1 to pass against its own gradus.lock, got %d\n%s\n%s", code, stdout.String(), stderr.String())
	}

	// 2. Forged all-zero hash on HEAD MUST fail even WITHOUT --require-lock-hash!
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"trailers", "verify", "--dir", tmp, "--ref", "HEAD", "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected forged Gradus-Lock-Hash to fail (exit 1) even without --require-lock-hash, got %d\n%s", code, stdout.String())
	}

	// 3. resolve --write with blocking conflict MUST NOT overwrite gradus.lock
	beforeLock, _ := os.ReadFile(filepath.Join(tmp, "gradus.lock"))
	_ = os.WriteFile(filepath.Join(tmp, ".gradus", "levels", "project", "rules", "PRJ-REL.md"), []byte("---\nid: PRJ-REL\ntitle: Relax\nseverity: MAY\noverrides: ORG-01\n---\n\nRelax.\n"), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--write"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on blocking invariant conflict, got %d", code)
	}
	afterLock, _ := os.ReadFile(filepath.Join(tmp, "gradus.lock"))
	if string(beforeLock) != string(afterLock) {
		t.Fatalf("expected gradus.lock to remain untouched when resolve --write has blocking conflicts")
	}
}

func TestDesign12SyncV041(t *testing.T) {
	designBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "design", "gradus-design.md"))
	if err != nil {
		t.Fatal(err)
	}
	design := string(designBytes)
	for _, want := range []string{"| Document version | 1.2 |", "D-84", "D-89", "## 20. Other agents, chat agents and MCP **[Decided; Specified]**", "## 21. Model policy **[Decided; Specified]**"} {
		if !strings.Contains(design, want) {
			t.Fatalf("Design 1.2 missing expected marker %q", want)
		}
	}

	specBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "v1", "SPEC.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(specBytes), "Design 1.2") || !strings.Contains(string(specBytes), "`models` are active") {
		t.Fatalf("spec/v1/SPEC.md not synchronized with Design 1.2 / active models")
	}

	cmBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "compatibility-matrix.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cmBytes), "gradusctl serve --mcp") || !strings.Contains(string(cmBytes), "--managed-settings") {
		t.Fatalf("docs/compatibility-matrix.md missing Phase 3 enterprise/MCP entries")
	}

	readmeBytes, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readmeBytes), "v0.4.1") || !strings.Contains(string(readmeBytes), "--model-tier") {
		t.Fatalf("README.md missing v0.4.1 or --model-tier security boundary note")
	}
}
