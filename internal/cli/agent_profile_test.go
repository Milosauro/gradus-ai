package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Milosauro/gradus-ai/internal/cli"
)

func TestGitHubActionSpec(t *testing.T) {
	actionPath := filepath.Join("..", "..", ".github", "actions", "gradus", "action.yml")
	raw, err := os.ReadFile(actionPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", actionPath, err)
	}

	var doc struct {
		Name   string         `yaml:"name"`
		Inputs map[string]any `yaml:"inputs"`
		Runs   struct {
			Using string `yaml:"using"`
			Steps []struct {
				Uses string `yaml:"uses"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("invalid action.yml YAML: %v", err)
	}
	if doc.Runs.Using != "composite" {
		t.Fatalf("expected runs.using == composite, got %q", doc.Runs.Using)
	}
	for _, reqInput := range []string{"version", "working-directory", "base-ref", "write-evidence", "upload-evidence"} {
		if _, ok := doc.Inputs[reqInput]; !ok {
			t.Fatalf("missing required input %q in action.yml", reqInput)
		}
	}

	text := string(raw)
	for _, needle := range []string{"gradusctl verify", "gradusctl check", "GITHUB_STEP_SUMMARY", "CGO_ENABLED=0"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected %q in action.yml", needle)
		}
	}

	shaPinRe := regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+@[0-9a-f]{40}$`)
	usesCount := 0
	for _, s := range doc.Runs.Steps {
		if s.Uses == "" {
			continue
		}
		usesCount++
		if !shaPinRe.MatchString(s.Uses) {
			t.Fatalf("uses: reference %q must be pinned to a 40-char commit SHA (PRJ-CI-001)", s.Uses)
		}
	}
	if usesCount == 0 {
		t.Fatalf("expected at least one uses: step in action.yml")
	}
}

func TestAgentProfileWorkflow(t *testing.T) {
	tmp := t.TempDir()
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	profileDir := filepath.Join(tmp, "ci-agent-profile")
	profileRules := filepath.Join(profileDir, "rules")
	for _, d := range []string{projRules, profileRules} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	_ = os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(`schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
targets: [agents-md]
`), 0o644)
	_ = os.WriteFile(filepath.Join(projRules, "PRJ-01.md"), []byte("---\nid: PRJ-01\ntitle: Keep clean\nseverity: SHOULD\n---\nKeep clean.\n"), 0o644)
	_ = os.WriteFile(filepath.Join(profileDir, "level.yaml"), []byte("schema: gradus/v1\nid: agent-profile\n"), 0o644)
	_ = os.WriteFile(filepath.Join(profileRules, "AGENT-PR-001.md"), []byte(`---
id: AGENT-PR-001
title: Never merge PRs directly
severity: MUST
enforcement: hook
scope: ["**/*"]
guard:
  action: command
  pattern: '\bgh\s+pr\s+merge\b'
  decision: deny
---
CI agents may open draft pull requests only, never merge.
`), 0o644)

	var stdout, stderr bytes.Buffer
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--agent-profile", profileDir, "--json"}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("resolve --agent-profile failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	var resolved struct {
		Rules []struct {
			ID    string `json:"id"`
			Level string `json:"level"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	var sawAgentRule bool
	for _, r := range resolved.Rules {
		if r.ID == "AGENT-PR-001" && r.Level == "agent-profile" {
			sawAgentRule = true
		}
	}
	if !sawAgentRule {
		t.Fatalf("expected AGENT-PR-001 at level agent-profile, got %+v", resolved.Rules)
	}

	// Compile includes agent-profile rule in AGENTS.md.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"compile", "--dir", tmp, "--agent-profile", profileDir}, &stdout, &stderr); code != cli.ExitPass {
		t.Fatalf("compile --agent-profile failed (%d): %s\n%s", code, stdout.String(), stderr.String())
	}
	agentsMD, err := os.ReadFile(filepath.Join(tmp, "AGENTS.md"))
	if err != nil || !strings.Contains(string(agentsMD), "AGENT-PR-001") {
		t.Fatalf("expected AGENT-PR-001 in AGENTS.md, err=%v", err)
	}

	// Guard enforces agent-profile hook rule.
	origWD, _ := os.Getwd()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(origWD) }()
	code, out, _ := runGuardWithStdin(t, []string{"--agent-profile", profileDir}, `{"tool_name":"Bash","tool_input":{"command":"gh pr merge 12"}}`)
	if code != cli.ExitViolation || !strings.Contains(out, `"decision":"deny"`) {
		t.Fatalf("expected guard deny for gh pr merge under agent-profile, got code=%d out=%s", code, out)
	}

	// Non-existent agent-profile fails closed with exit 2.
	stdout.Reset()
	stderr.Reset()
	if code := cli.ExecuteCode([]string{"resolve", "--dir", tmp, "--agent-profile", filepath.Join(tmp, "missing-profile")}, &stdout, &stderr); code != cli.ExitError {
		t.Fatalf("expected exit 2 for missing --agent-profile, got %d", code)
	}
}
