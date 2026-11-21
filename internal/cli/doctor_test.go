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

func TestDoctorCommand(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets: [claude-code, antigravity, agents-md]
`), 0o644)
	_ = os.WriteFile(filepath.Join(projDir, "PRJ-01.md"), []byte(`---
id: PRJ-01
title: Basic rule
severity: SHOULD
---
Keep clean.
`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--write", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --write failed: %s\n%s", stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile failed: %s\n%s", stdout.String(), stderr.String())
	}

	// 1. Healthy repository -> exit 0
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"doctor", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("expected doctor pass on healthy repo, got %d: %s\n%s", code, stdout.String(), stderr.String())
	}
	var healthy struct {
		Verdict string `json:"verdict"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &healthy); err != nil || healthy.Verdict != "pass" {
		t.Fatalf("unexpected healthy doctor output: err=%v out=%s", err, stdout.String())
	}

	// 2. Corrupted .claude/settings.json with invalid hook type "commbnd" -> exit 1
	claudeSettingsPath := filepath.Join(tmp, ".claude", "settings.json")
	_ = os.WriteFile(claudeSettingsPath, []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"commbnd","commbnd":"gradusctl guard"}]}]}}`), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"doctor", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected doctor exit 1 on invalid claude hook type, got %d (%s)", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "commbnd") || !strings.Contains(stdout.String(), ".claude/settings.json") {
		t.Fatalf("expected doctor output to cite commbnd and .claude/settings.json, got: %s", stdout.String())
	}

	// Restore .claude/settings.json and corrupt .agents/hooks.json -> exit 1
	_ = os.Remove(claudeSettingsPath)
	_ = cli.ExecuteCode([]string{"compile", "--dir", tmp}, &stdout, &stderr)
	_ = os.WriteFile(filepath.Join(tmp, ".agents", "hooks.json"), []byte(`{bad json`), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"doctor", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitViolation {
		t.Fatalf("expected doctor exit 1 on malformed .agents/hooks.json, got %d (%s)", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), ".agents/hooks.json") {
		t.Fatalf("expected doctor output to cite .agents/hooks.json, got: %s", stdout.String())
	}
}

func TestPruneAndStatusUserRules(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	evDir := filepath.Join(tmp, ".gradus", "evidence")
	lessonsDir := filepath.Join(tmp, "lessons")
	userRulesDir := filepath.Join(tmp, "user", "rules")
	for _, d := range []string{projDir, evDir, lessonsDir, userRulesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\ntargets: [agents-md]\n"), 0o644)
	_ = os.WriteFile(filepath.Join(projDir, "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Basic\nseverity: SHOULD\n---\nKeep clean.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(userRulesDir, "USR-GIT-001.md"), []byte("---\nid: USR-GIT-001\ntitle: Confirm commit\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: git commit\n  decision: ask\n---\nConfirm before git commit.\n"), 0o644)

	_ = os.WriteFile(filepath.Join(evDir, "ev-old.json"), []byte(`{"id":"ev-old","timestamp":"2025-01-01T00:00:00Z"}`), 0o644)
	_ = os.WriteFile(filepath.Join(evDir, "ev-new.json"), []byte(`{"id":"ev-new","timestamp":"2026-11-20T00:00:00Z"}`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"evidence", "prune", "--before", "2026-01-01", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence prune --before failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var pruneOut struct {
		Removed   int `json:"removed"`
		Remaining int `json:"remaining"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &pruneOut); err != nil || pruneOut.Removed != 1 || pruneOut.Remaining != 1 {
		t.Fatalf("expected removed=1 remaining=1, got err=%v out=%s", err, stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"evidence", "prune", "--all", "--dir", tmp, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("evidence prune --all failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}

	// Lessons prune --dismissed
	_ = os.WriteFile(filepath.Join(lessonsDir, "les-open.json"), []byte(`{"schema":"gradus/v1","id":"les-open","status":"open","statement":"keep"}`), 0o644)
	_ = os.WriteFile(filepath.Join(lessonsDir, "les-dis.json"), []byte(`{"schema":"gradus/v1","id":"les-dis","status":"dismissed","statement":"drop"}`), 0o644)
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"lessons", "prune", "--dismissed", "--lessons-dir", lessonsDir, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("lessons prune --dismissed failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &pruneOut); err != nil || pruneOut.Removed != 1 || pruneOut.Remaining != 1 {
		t.Fatalf("expected lessons removed=1 remaining=1, got err=%v out=%s", err, stdout.String())
	}

	// Status --json exposes user_rules
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"status", "--dir", tmp, "--user-dir", filepath.Join(tmp, "user"), "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("status failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var st struct {
		UserRules []string `json:"user_rules"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &st); err != nil || len(st.UserRules) != 1 || st.UserRules[0] != "USR-GIT-001" {
		t.Fatalf("expected user_rules=[USR-GIT-001], got err=%v out=%s", err, stdout.String())
	}
}
