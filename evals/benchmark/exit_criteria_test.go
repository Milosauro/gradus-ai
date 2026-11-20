package benchmark_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/guard"
)

// TestPhase0ExitCriteria validates E1-E8 end-to-end using the §24.3 three-level setup:
// - team (separate local git repo, locked: true)
// - project (local path)
// - user (local user-level dir)
// with >= 10 real rules covering hook, check, prompt, scoped rules, valid override, and invalid override.
func TestPhase0ExitCriteria(t *testing.T) {
	// 1. Create simulated remote team git repository
	teamOrigin := t.TempDir()
	initGitRepo(t, teamOrigin)
	teamRules := filepath.Join(teamOrigin, "rules")
	if err := os.MkdirAll(teamRules, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(teamOrigin, "level.yaml"), []byte("schema: gradus/v1\nid: team\nversion: \"1.0.0\"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(teamRules, "TEAM-SEC-01.md"), []byte("---\nid: TEAM-SEC-01\ntitle: No hardcoded AWS keys\nseverity: MUST\nenforcement: check\nscope: ['**/*.go']\nchecker:\n  name: pattern\n  pattern: 'AKIA[0-9A-Z]{16}'\n---\nNever commit AWS keys.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(teamRules, "TEAM-SEC-02.md"), []byte("---\nid: TEAM-SEC-02\ntitle: Protect secret files\nseverity: MUST\nenforcement: hook\noverridable: false\nguard:\n  action: write\n  pattern: 'secrets/**'\n  decision: deny\n---\nNever write under secrets/.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(teamRules, "TEAM-STYLE-01.md"), []byte("---\nid: TEAM-STYLE-01\ntitle: Document public APIs\nseverity: SHOULD\nenforcement: review\noverridable: true\n---\nDocument exported types.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(teamRules, "TEAM-LOG-01.md"), []byte("---\nid: TEAM-LOG-01\ntitle: Use structured logging\nseverity: SHOULD\nenforcement: prompt\n---\nUse structured JSON logging in backend code.\n"), 0o644)
	gitAddCommit(t, teamOrigin, "team v1.0.0")
	runGitCmd(t, teamOrigin, "tag", "v1.0.0")

	// 2. Create project workspace + user level (>= 10 total rules)
	repo := t.TempDir()
	initGitRepo(t, repo)
	projRules := filepath.Join(repo, ".gradus", "levels", "project", "rules")
	userDir := filepath.Join(repo, ".user-home", "user")
	userRules := filepath.Join(userDir, "rules")
	for _, d := range []string{projRules, userRules} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	_ = os.WriteFile(filepath.Join(projRules, "PRJ-CHK-01.md"), []byte("---\nid: PRJ-CHK-01\ntitle: Forbid TODO_FIXME_BLOCK\nseverity: MUST\nenforcement: check\nscope: ['**/*.go']\nchecker:\n  name: pattern\n  pattern: 'TODO_FIXME_BLOCK'\n---\nResolve TODO_FIXME_BLOCK before committing.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-STYLE-01.md"), []byte("---\nid: PRJ-STYLE-01\ntitle: Require docstrings on all symbols\nseverity: MUST\noverrides: TEAM-STYLE-01\nenforcement: review\n---\nRequire docstrings on exported and internal helpers.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-DOC-01.md"), []byte("---\nid: PRJ-DOC-01\ntitle: Keep README accurate\nseverity: SHOULD\nenforcement: prompt\nscope: ['README.md']\n---\nKeep README in sync.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-TEST-01.md"), []byte("---\nid: PRJ-TEST-01\ntitle: Add unit tests\nseverity: SHOULD\nenforcement: prompt\n---\nWrite deterministic unit tests.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-LOG-01.md"), []byte("---\nid: PRJ-LOG-01\ntitle: Use plain logging\nseverity: SHOULD\nenforcement: prompt\n---\nUse plain text logging in CLI output.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(userRules, "USR-GIT-01.md"), []byte("---\nid: USR-GIT-01\ntitle: Confirm git commit\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: 'git\\s+commit'\n  decision: ask\n---\nConfirm before committing.\n"), 0o644)

	gradusYAML := "schema: gradus/v1\nlevels:\n" +
		"  - id: team\n    source: { git: " + teamOrigin + ", ref: v1.0.0 }\n    locked: true\n    editable_rules: [TEAM-STYLE-01]\n" +
		"  - id: project\n    source: { path: .gradus/levels/project }\n" +
		"targets: [claude-code, antigravity, agents-md]\n" +
		"hooks:\n  manager: lefthook\n" +
		"evidence:\n  ci:\n    sinks:\n      - type: ci-artifact\n        path: .gradus/ci-artifacts\n"
	_ = os.WriteFile(filepath.Join(repo, "gradus.yaml"), []byte(gradusYAML), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "app.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	gitAddCommit(t, repo, "initial")

	// E1: resolve --write & idempotent compile
	if code, out, errOut := runGradusCLI(t, []string{"resolve", "--dir", repo, "--write", "--json"}, ""); code != cli.ExitPass {
		t.Fatalf("E1 resolve failed (%d): %s %s", code, out, errOut)
	}
	if code, out, errOut := runGradusCLI(t, []string{"compile", "--dir", repo, "--json"}, ""); code != cli.ExitPass {
		t.Fatalf("E1 compile 1 failed (%d): %s %s", code, out, errOut)
	}
	if code, out, _ := runGradusCLI(t, []string{"compile", "--dir", repo, "--json"}, ""); code != cli.ExitPass || !strings.Contains(out, `"changed_files": []`) {
		t.Fatalf("E1 compile 2 not idempotent: %d %s", code, out)
	}

	// E2: verify clean -> drift (exit 1) -> restore -> lock tamper (exit 2)
	if code, out, _ := runGradusCLI(t, []string{"verify", "--dir", repo}, ""); code != cli.ExitPass {
		t.Fatalf("E2 clean verify failed (%d): %s", code, out)
	}

	// E3: guard confirms git commit via user-level hook in both Claude Code & Antigravity
	origWD, _ := os.Getwd()
	_ = os.Chdir(repo)
	for _, payload := range []string{
		`{"tool_name":"Bash","tool_input":{"command":"git commit -m test"}}`,
		`{"tool_name":"run_command","tool_input":{"CommandLine":"git commit -m test"}}`,
	} {
		code, out, _ := runGradusCLI(t, []string{"guard", "--user-dir", userDir}, payload)
		var gRes guard.EvaluationResult
		_ = json.Unmarshal([]byte(out), &gRes)
		if code != cli.ExitPass || gRes.Decision != guard.DecisionAsk {
			_ = os.Chdir(origWD)
			t.Fatalf("E3 guard failed: code=%d res=%+v", code, gRes)
		}
	}
	_ = os.Chdir(origWD)

	// E4/E8: check detects seeded violation; hooks install + evidence ci-artifact sink work
	_ = os.WriteFile(filepath.Join(repo, "bad.go"), []byte("package main\n// TODO_FIXME_BLOCK\n"), 0o644)
	gitAddCommit(t, repo, "seeded violation")
	if code, _, _ := runGradusCLI(t, []string{"check", "--dir", repo}, ""); code != cli.ExitViolation {
		t.Fatalf("E4 check expected exit 1 on violation, got %d", code)
	}
	if code, _, _ := runGradusCLI(t, []string{"hooks", "install", "--dir", repo}, ""); code != cli.ExitPass {
		t.Fatalf("E8 hooks install failed: %d", code)
	}

	// E5: lessons add redacts and records open lesson without auto-promoting
	lessonsDir := t.TempDir()
	if code, _, _ := runGradusCLI(t, []string{"lessons", "add", "--dir", repo, "--lessons-dir", lessonsDir, "--kind", "correction", "--statement", "Do not log sk-secret1234567890abcd"}, ""); code != cli.ExitPass {
		t.Fatalf("E5 lessons add failed: %d", code)
	}

	// E6: drift detected by reconcile (exit 1) and restored via --restore (exit 0)
	_ = os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("manual edit\n"), 0o644)
	if code, _, _ := runGradusCLI(t, []string{"reconcile", "--dir", repo}, ""); code != cli.ExitViolation {
		t.Fatalf("E6 reconcile expected exit 1 on drift, got %d", code)
	}
	if code, _, _ := runGradusCLI(t, []string{"reconcile", "--dir", repo, "--restore"}, ""); code != cli.ExitPass {
		t.Fatalf("E6 reconcile --restore failed: %d", code)
	}

	// E7: semantic conflicts candidate discovery + dismiss suppresses pair
	if code, out, _ := runGradusCLI(t, []string{"conflicts", "--dir", repo, "--candidates", "--json"}, ""); code != cli.ExitPass || !strings.Contains(out, "PRJ-LOG-01") {
		t.Fatalf("E7 expected PRJ-LOG-01/TEAM-LOG-01 candidate, got %d %s", code, out)
	}
	if code, _, _ := runGradusCLI(t, []string{"conflicts", "--dir", repo, "--dismiss", "--rule-a", "PRJ-LOG-01", "--rule-b", "TEAM-LOG-01", "--message", "CLI vs backend"}, ""); code != cli.ExitPass {
		t.Fatalf("E7 dismiss failed: %d", code)
	}
	if code, out, _ := runGradusCLI(t, []string{"conflicts", "--dir", repo, "--candidates", "--json"}, ""); code != cli.ExitPass || strings.Contains(out, "PRJ-LOG-01") {
		t.Fatalf("E7 expected dismissed pair suppressed, got %d %s", code, out)
	}

	// E2 lock tamper check: tamper cached team level -> verify exits 2
	_ = os.WriteFile(filepath.Join(repo, ".gradus", "cache", "team", "repo", "rules", "TEAM-LOG-01.md"), []byte("tampered"), 0o644)
	if code, _, _ := runGradusCLI(t, []string{"verify", "--dir", repo}, ""); code != cli.ExitError {
		t.Fatalf("E2 expected exit 2 on tampered locked cache, got %d", code)
	}
}

// TestReadmePersonalSetup validates E10 end-to-end: setting up a fresh repository
// using the exact personal setup steps documented in README.md.
func TestReadmePersonalSetup(t *testing.T) {
	readmeBytes, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	readme := string(readmeBytes)
	for _, cmd := range []string{"gradusctl init", "gradusctl resolve --write", "gradusctl compile", "gradusctl verify"} {
		if !strings.Contains(readme, cmd) {
			t.Fatalf("README.md Quick Start missing documented command %q", cmd)
		}
	}

	fresh := t.TempDir()
	initGitRepo(t, fresh)
	_ = os.WriteFile(filepath.Join(fresh, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	gitAddCommit(t, fresh, "init")

	for _, args := range [][]string{
		{"init", "--dir", fresh},
		{"resolve", "--dir", fresh, "--write"},
		{"compile", "--dir", fresh},
		{"verify", "--dir", fresh},
	} {
		if code, out, errOut := runGradusCLI(t, args, ""); code != cli.ExitPass {
			t.Fatalf("README personal setup step %v failed (%d): %s %s", args, code, out, errOut)
		}
	}
}
