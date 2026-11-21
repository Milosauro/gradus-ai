package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestRuleTestCommand(t *testing.T) {
	tmp := t.TempDir()
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\ntargets: [agents-md]\n"), 0o644)

	checkRulePath := filepath.Join(projRules, "PRJ-CHK-01.md")
	_ = os.WriteFile(checkRulePath, []byte(`---
id: PRJ-CHK-01
title: Forbid debug print
severity: MUST
enforcement: check
scope: ["backend/**/*.py"]
checker:
  name: pattern
  pattern: '\bprint\('
---

Do not leave print statements in backend code.

## Examples

### Compliant
`+"```python"+`
logger.info("ok")
`+"```"+`

### Non-compliant
`+"```python"+`
print("debug")
`+"```"+`
`), 0o644)

	_ = os.WriteFile(filepath.Join(projRules, "PRJ-HK-01.md"), []byte(`---
id: PRJ-HK-01
title: Confirm before commit
severity: MUST
enforcement: hook
scope: ["**/*"]
guard:
  action: command
  pattern: '\bgit\s+commit\b'
  decision: ask
---

Ask before running git commit.

## Examples

### Compliant
`+"```bash"+`
git status
`+"```"+`

### Non-compliant
`+"```bash"+`
git commit -m "wip"
`+"```"+`
`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"test", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected gradusctl test to pass, got %d: %s\n%s", code, stdout.String(), stderr.String())
	}
	var rep struct {
		Verdict string `json:"verdict"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if rep.Verdict != "pass" || rep.Passed != 4 || rep.Failed != 0 {
		t.Fatalf("expected 4 passed cases and 0 failed, got %+v", rep)
	}

	// --rule filter
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"test", "--rule", "PRJ-CHK-01", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected --rule PRJ-CHK-01 pass, got %d", code)
	}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil || rep.Passed != 2 {
		t.Fatalf("expected 2 cases for single rule, got err=%v rep=%+v", err, rep)
	}

	// Unknown rule -> exit 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"test", "--rule", "MISSING-99", "--dir", tmp}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for unknown --rule, got %d", code)
	}

	// Failing Compliant example -> exit 1
	_ = os.WriteFile(checkRulePath, []byte(`---
id: PRJ-CHK-01
title: Forbid debug print
severity: MUST
enforcement: check
scope: ["backend/**/*.py"]
checker:
  name: pattern
  pattern: '\bprint\('
---

Do not leave print statements in backend code.

## Examples

### Compliant
`+"```python"+`
print("oops")
`+"```"+`

### Non-compliant
`+"```python"+`
print("debug")
`+"```"+`
`), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"test", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected exit 1 on failing example, got %d (%s)", code, stdout.String())
	}
}

func TestSimulateCommand(t *testing.T) {
	tmp := t.TempDir()
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\ntargets: [agents-md]\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-CHK-01.md"), []byte(`---
id: PRJ-CHK-01
title: Forbid debug print
severity: MUST
enforcement: check
scope: ["**/*.py"]
checker:
  name: pattern
  pattern: '\bprint\('
---
Do not use print.
`), 0o644)

	runGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
		return string(out)
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	_ = os.WriteFile(filepath.Join(tmp, "app.py"), []byte("logger.info('ok')\n"), 0o644)
	runGit("add", ".")
	runGit("commit", "-m", "commit 1 clean")

	_ = os.WriteFile(filepath.Join(tmp, "app.py"), []byte("print('debug')\n"), 0o644)
	runGit("add", "app.py")
	runGit("commit", "-m", "commit 2 violation")

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"simulate", "--commits", "2", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("simulate --commits 2 failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var rep struct {
		CommitsScanned  int            `json:"commits_scanned"`
		AffectedCommits int            `json:"affected_commits"`
		FindingsByRule  map[string]int `json:"findings_by_rule"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("invalid simulate json: %v\n%s", err, stdout.String())
	}
	if rep.CommitsScanned != 2 || rep.AffectedCommits != 1 || rep.FindingsByRule["PRJ-CHK-01"] != 1 {
		t.Fatalf("unexpected simulate report: %+v", rep)
	}

	// --since HEAD~1 -> 1 commit scanned
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"simulate", "--since", "HEAD~1", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("simulate --since HEAD~1 failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	rep = struct {
		CommitsScanned  int            `json:"commits_scanned"`
		AffectedCommits int            `json:"affected_commits"`
		FindingsByRule  map[string]int `json:"findings_by_rule"`
	}{}
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil || rep.CommitsScanned != 1 || rep.AffectedCommits != 1 {
		t.Fatalf("expected 1 commit scanned and 1 affected, got err=%v rep=%+v", err, rep)
	}

	// Ref starting with '-' -> exit 2
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"simulate", "--since", "--exec=evil", "--dir", tmp}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 on flag-like --since ref, got %d", code)
	}

	// Working tree must remain completely clean.
	if st := strings.TrimSpace(runGit("status", "--short")); st != "" {
		t.Fatalf("expected clean git working tree after simulate, got: %s", st)
	}
}
