package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/changelog"
	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestChangelogCommand(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote-team")
	if err := os.MkdirAll(filepath.Join(remote, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = remote
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}
	gitRun("init")
	gitRun("config", "user.name", "Test")
	gitRun("config", "user.email", "test@gradus.local")

	writeRemote := func(rel, content string) {
		t.Helper()
		p := filepath.Join(remote, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeRemote("level.yaml", "schema: gradus/v1\nid: team\nversion: 1.0.0\n")
	writeRemote("rules/TEAM-001.md", "---\nid: TEAM-001\ntitle: Rule one\nseverity: SHOULD\nenforcement: prompt\n---\n\nInitial text.\n")
	gitRun("add", ".")
	gitRun("commit", "-m", "v1.0.0")
	gitRun("tag", "v1.0.0")

	writeRemote("level.yaml", "schema: gradus/v1\nid: team\nversion: 1.1.0\n")
	writeRemote("rules/TEAM-001.md", "---\nid: TEAM-001\ntitle: Rule one\nseverity: MUST\nenforcement: prompt\n---\n\nUpdated text.\n")
	writeRemote("rules/TEAM-002.md", "---\nid: TEAM-002\ntitle: Rule two\nseverity: MUST\nenforcement: review\n---\n\nAdded rule.\n")
	gitRun("add", ".")
	gitRun("commit", "-m", "v1.1.0")
	gitRun("tag", "v1.1.0")

	consumer := filepath.Join(tmp, "consumer")
	if err := os.MkdirAll(consumer, 0o755); err != nil {
		t.Fatal(err)
	}
	remoteGit := filepath.ToSlash(remote)
	cfgV1 := fmt.Sprintf("schema: gradus/v1\nlevels:\n  - id: team\n    source: { git: %s, ref: v1.0.0 }\n    locked: true\n", remoteGit)
	if err := os.WriteFile(filepath.Join(consumer, "gradus.yaml"), []byte(cfgV1), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", consumer, "--write"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write failed: %d %s", code, stderr.String())
	}

	// Update gradus.yaml ref to v1.1.0 without re-locking yet, then run default changelog
	cfgV2 := fmt.Sprintf("schema: gradus/v1\nlevels:\n  - id: team\n    source: { git: %s, ref: v1.1.0 }\n    locked: true\n", remoteGit)
	if err := os.WriteFile(filepath.Join(consumer, "gradus.yaml"), []byte(cfgV2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Capture cached repo HEAD before changelog to ensure NFR3 (no cache checkout mutation)
	cacheRepo := filepath.Join(consumer, ".gradus", "cache", "team", "repo")
	headBeforeCmd := exec.Command("git", "rev-parse", "HEAD")
	headBeforeCmd.Dir = cacheRepo
	headBefore, err := headBeforeCmd.Output()
	if err != nil {
		t.Fatalf("failed to read cached HEAD before changelog: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"changelog", "team", "--dir", consumer, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("changelog --json failed: %d %s", code, stderr.String())
	}
	headAfterCmd := exec.Command("git", "rev-parse", "HEAD")
	headAfterCmd.Dir = cacheRepo
	headAfter, err := headAfterCmd.Output()
	if err != nil {
		t.Fatalf("failed to read cached HEAD after changelog: %v", err)
	}
	if strings.TrimSpace(string(headBefore)) != strings.TrimSpace(string(headAfter)) {
		t.Fatalf("expected changelog not to mutate cached HEAD: before=%s after=%s", strings.TrimSpace(string(headBefore)), strings.TrimSpace(string(headAfter)))
	}
	var rep changelog.Report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("expected valid JSON report, got: %v\n%s", err, stdout.String())
	}
	if len(rep.Added) != 1 || rep.Added[0].ID != "TEAM-002" || len(rep.Modified) != 1 || rep.Modified[0].ID != "TEAM-001" {
		t.Fatalf("unexpected changelog report: %+v", rep)
	}

	// Markdown output with explicit refs
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"changelog", "team", "--dir", consumer, "--from", "v1.0.0", "--to", "v1.1.0"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("changelog markdown failed: %d %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "TEAM-001") || !strings.Contains(stdout.String(), "TEAM-002") {
		t.Fatalf("unexpected markdown output:\n%s", stdout.String())
	}

	// Invalid level or '-' ref -> exit 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"changelog", "nonexistent", "--dir", consumer}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for unknown level, got %d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"changelog", "team", "--dir", consumer, "--from", "-evil"}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for leading '-' ref, got %d", code)
	}
}
