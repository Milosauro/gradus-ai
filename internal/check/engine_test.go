package check_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/check"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func runGitCheckTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Gradus Test",
		"GIT_AUTHOR_EMAIL=test@example.test",
		"GIT_COMMITTER_NAME=Gradus Test",
		"GIT_COMMITTER_EMAIL=test@example.test",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func TestCheckEngine(t *testing.T) {
	tmp := t.TempDir()
	runGitCheckTest(t, tmp, "init")

	if err := os.MkdirAll(filepath.Join(tmp, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCheckTest(t, tmp, "add", "README.md")
	runGitCheckTest(t, tmp, "commit", "-m", "initial")

	// Stage a fixture with a forbidden email
	if err := os.WriteFile(filepath.Join(tmp, "fixtures", "users.json"), []byte(`{"email": "mario@realcompany.com"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCheckTest(t, tmp, "add", "fixtures/users.json")

	// 1. Built-in regex checker detects staged violation
	view := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          "ORG-SEC-004",
				Title:       "No real personal data",
				Level:       "org",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"fixtures/**"},
				Checker:     &schema.CheckerConfig{Name: "regex", Pattern: `@realcompany\.com`},
				Provenance:  resolve.RuleProvenance{Source: "org@v1"},
			},
		},
	}

	res, err := check.RunChecks(tmp, view, check.Options{Staged: true})
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}
	if len(res.Findings) != 1 || res.Findings[0].RuleID != "ORG-SEC-004" || res.Findings[0].Line != 1 {
		t.Fatalf("expected 1 finding for ORG-SEC-004 at line 1, got: %+v", res.Findings)
	}

	// 2. Missing external checker binary must return an error (mapped to exit code 2)
	missingView := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          "ORG-EXT-001",
				Title:       "External tool",
				Level:       "org",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"**/*"},
				Checker:     &schema.CheckerConfig{Name: "gradus-nonexistent-checker-xyz"},
			},
		},
	}
	if _, err := check.RunChecks(tmp, missingView, check.Options{Staged: true}); err == nil || !strings.Contains(err.Error(), "not found on PATH") {
		t.Fatalf("expected missing checker error, got: %v", err)
	}

	// 3. External checker exiting 1 reports violation finding
	binDir := t.TempDir()
	var fakeChecker, fakeBody string
	if runtime.GOOS == "windows" {
		fakeChecker = filepath.Join(binDir, "fake-checker.cmd")
		fakeBody = "@echo off\r\necho found secret\r\nexit /b 1\r\n"
	} else {
		fakeChecker = filepath.Join(binDir, "fake-checker")
		fakeBody = "#!/bin/sh\necho 'found secret'\nexit 1\n"
	}
	if err := os.WriteFile(fakeChecker, []byte(fakeBody), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	extView := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          "ORG-EXT-002",
				Title:       "Fake checker",
				Level:       "org",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"**/*"},
				Checker:     &schema.CheckerConfig{Name: "fake-checker"},
				Provenance:  resolve.RuleProvenance{Source: "org@v1"},
			},
		},
	}
	extRes, err := check.RunChecks(tmp, extView, check.Options{Staged: true})
	if err != nil {
		t.Fatalf("expected exit 1 external checker to return finding rather than error, got: %v", err)
	}
	if len(extRes.Findings) != 1 || !strings.Contains(extRes.Findings[0].Message, "found secret") {
		t.Fatalf("expected external violation finding, got: %+v", extRes.Findings)
	}
}

func TestCheckEngineEdgeCasesV032(t *testing.T) {
	tmp := t.TempDir()
	_ = os.MkdirAll(filepath.Join(tmp, "services", "billing", "internal"), 0o755)
	_ = os.MkdirAll(filepath.Join(tmp, "docs"), 0o755)

	// 1. Binary file with NUL bytes and matching token must be skipped cleanly
	binData := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x00}, []byte("FORBIDDEN_TOKEN")...)
	if err := os.WriteFile(filepath.Join(tmp, "services", "billing", "logo.png"), binData, 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Large file with > 100KB single line containing match must scan without scanner error
	longLine := strings.Repeat("a", 120*1024) + "FORBIDDEN_TOKEN\n"
	if err := os.WriteFile(filepath.Join(tmp, "services", "billing", "internal", "huge.go"), []byte(longLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Out-of-scope file containing match must be excluded by complex glob
	if err := os.WriteFile(filepath.Join(tmp, "docs", "notes.md"), []byte("FORBIDDEN_TOKEN\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := []string{
		"services/billing/logo.png",
		"services/billing/internal/huge.go",
		"docs/notes.md",
	}

	view := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-CHK-GLOB",
				Title:       "No forbidden token in services",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"services/**/*.{go,ts}", "services/**/*.png"},
				Checker:     &schema.CheckerConfig{Name: "regex", Pattern: `FORBIDDEN_TOKEN`},
				Provenance:  resolve.RuleProvenance{Source: "project"},
			},
			{
				ID:          "PRJ-CHK-WAIVED",
				Title:       "Waived check",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "check",
				Waived:      true,
				Scope:       []string{"**/*"},
				Checker:     &schema.CheckerConfig{Name: "regex", Pattern: `FORBIDDEN_TOKEN`},
			},
		},
	}

	res, err := check.RunChecksOnFiles(tmp, view, files)
	if err != nil {
		t.Fatalf("RunChecksOnFiles failed on binary/large files: %v", err)
	}
	if len(res.Findings) != 1 || res.Findings[0].Path != "services/billing/internal/huge.go" || res.Findings[0].Line != 1 {
		t.Fatalf("expected exactly 1 finding on huge.go (skipping binary logo.png and out-of-scope docs/notes.md), got: %+v", res.Findings)
	}

	// 4. Invalid regex returns error
	badRegexView := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          "PRJ-BAD-RE",
				Title:       "Invalid regex",
				Level:       "project",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"**/*"},
				Checker:     &schema.CheckerConfig{Name: "regex", Pattern: `[unclosed`},
			},
		},
	}
	if _, err := check.RunChecksOnFiles(tmp, badRegexView, files); err == nil {
		t.Fatalf("expected error for invalid regex pattern")
	}

	// 5. External checker exiting with code 2 returns technical error (not finding)
	binDir := t.TempDir()
	var errChecker, errBody string
	if runtime.GOOS == "windows" {
		errChecker = filepath.Join(binDir, "err-checker.cmd")
		errBody = "@echo off\r\necho fatal crash 1>&2\r\nexit /b 2\r\n"
	} else {
		errChecker = filepath.Join(binDir, "err-checker")
		errBody = "#!/bin/sh\necho 'fatal crash' >&2\nexit 2\n"
	}
	if err := os.WriteFile(errChecker, []byte(errBody), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	errExtView := resolve.ResolvedView{
		Rules: []resolve.ResolvedRule{
			{
				ID:          "ORG-EXT-ERR",
				Title:       "Crashing checker",
				Level:       "org",
				Severity:    "MUST",
				Enforcement: "check",
				Scope:       []string{"**/*"},
				Checker:     &schema.CheckerConfig{Name: "err-checker"},
			},
		},
	}
	if _, err := check.RunChecksOnFiles(tmp, errExtView, []string{"docs/notes.md"}); err == nil || !strings.Contains(err.Error(), "fatal crash") {
		t.Fatalf("expected technical error from exit-2 external checker, got: %v", err)
	}
}
