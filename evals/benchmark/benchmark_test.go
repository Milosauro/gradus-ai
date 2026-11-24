package benchmark_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/guard"
)

type benchmarkFile struct {
	Scenarios []benchmarkScenario `yaml:"scenarios"`
}

type benchmarkScenario struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Category string   `yaml:"category"`
	Targets  []string `yaml:"targets"`
}

func runGradusCLI(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	cmd := cli.NewRootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if stdin != "" {
		cmd.SetIn(strings.NewReader(stdin))
	}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err == nil {
		return cli.ExitPass, stdout.String(), stderr.String()
	}
	if coded, ok := err.(*cli.ExitErrorWithCode); ok {
		return coded.Code, stdout.String(), stderr.String()
	}
	return cli.ExitError, stdout.String(), err.Error()
}

func TestComparativeBenchmark(t *testing.T) {
	raw, err := os.ReadFile("scenarios.yaml")
	if err != nil {
		t.Fatalf("failed to read scenarios.yaml: %v", err)
	}
	var bf benchmarkFile
	if err := yaml.Unmarshal(raw, &bf); err != nil {
		t.Fatalf("failed to parse scenarios.yaml: %v", err)
	}
	if len(bf.Scenarios) < 10 {
		t.Fatalf("E9 requires >= 10 benchmark scenarios, got %d", len(bf.Scenarios))
	}
	if _, err := os.Stat("RESULTS.md"); err != nil {
		t.Fatalf("missing published RESULTS.md: %v", err)
	}

	targets := []string{"claude-code", "antigravity"}

	for _, sc := range bf.Scenarios {
		for _, target := range targets {
			t.Run(sc.ID+"/"+target, func(t *testing.T) {
				baselineScore := 0.0
				if sc.Category == "advisory" {
					baselineScore = 1.0
				}
				gradusScore := executeBenchmarkScenario(t, sc.ID, target)
				if gradusScore < baselineScore {
					t.Fatalf("%s (%s): gradusScore %.2f < baselineScore %.2f", sc.ID, target, gradusScore, baselineScore)
				}
				if sc.Category == "deterministic" && gradusScore <= baselineScore {
					t.Fatalf("%s (%s): expected gradusScore > baselineScore for deterministic scenario, got %.2f <= %.2f", sc.ID, target, gradusScore, baselineScore)
				}
			})
		}
	}
}

func executeBenchmarkScenario(t *testing.T, id, target string) float64 {
	t.Helper()
	tmp := t.TempDir()
	initGitRepo(t, tmp)
	projRules := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	if err := os.MkdirAll(projRules, 0o755); err != nil {
		t.Fatal(err)
	}

	writeGradusConfig := func(extraLevels string) {
		cfg := fmt.Sprintf("schema: gradus/v1\nlevels:\n%s  - id: project\n    source: { path: .gradus/levels/project }\ntargets: [%s]\n", extraLevels, target)
		if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeGradusConfig("")

	writeRule := func(name, content string) {
		if err := os.WriteFile(filepath.Join(projRules, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeRule("PRJ-GIT.md", "---\nid: PRJ-GIT\ntitle: Confirm git commit\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: 'git\\s+commit'\n  decision: ask\n---\nAsk before committing.\n")
	writeRule("PRJ-RM.md", "---\nid: PRJ-RM\ntitle: Forbid rm -rf\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: 'rm\\s+-rf'\n  decision: deny\n---\nNever run rm -rf.\n")
	writeRule("PRJ-PAT.md", "---\nid: PRJ-PAT\ntitle: Forbid TODO_SECRET\nseverity: MUST\nenforcement: check\nscope: ['**/*.go']\nchecker:\n  name: pattern\n  pattern: 'TODO_SECRET'\n---\nNo TODO_SECRET.\n")
	writeRule("PRJ-ADV.md", "---\nid: PRJ-ADV\ntitle: Prefer clear names\nseverity: SHOULD\nenforcement: prompt\n---\nUse clear names.\n## Why\nReadability.\n## Rationalizations\n- \"Short is fine.\"\n")

	gitAddCommit(t, tmp, "seed")

	switch id {
	case "BENCH-01":
		payload := payloadForCommand(target, "git commit -m 'feat: x'")
		return runGuardCase(t, tmp, payload, guard.DecisionAsk, false, cli.ExitPass)
	case "BENCH-02":
		govPath := ".claude/settings.json"
		if target == "antigravity" {
			govPath = ".agents/hooks.json"
		}
		payload := payloadForWrite(target, govPath)
		return runGuardCase(t, tmp, payload, guard.DecisionAsk, true, cli.ExitPass)
	case "BENCH-03":
		payload := payloadForCommand(target, "rm -rf /tmp/data")
		return runGuardCase(t, tmp, payload, guard.DecisionDeny, false, cli.ExitViolation)
	case "BENCH-04":
		_ = os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main\n// TODO_SECRET\n"), 0o644)
		gitAddCommit(t, tmp, "violation")
		if code, _, _ := runGradusCLI(t, []string{"check", "--dir", tmp}, ""); code == cli.ExitViolation {
			return 1.0
		}
	case "BENCH-05":
		writeRule("PRJ-EXT.md", "---\nid: PRJ-EXT\ntitle: External missing\nseverity: MUST\nenforcement: check\nscope: ['**/*.go']\nchecker:\n  name: nonexistent-checker-bin\n---\nRun external.\n")
		_ = os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main\n"), 0o644)
		gitAddCommit(t, tmp, "ext")
		if code, _, _ := runGradusCLI(t, []string{"check", "--dir", tmp}, ""); code == cli.ExitError {
			return 1.0
		}
	case "BENCH-06":
		teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
		_ = os.MkdirAll(teamRules, 0o755)
		_ = os.WriteFile(filepath.Join(teamRules, "TEAM-01.md"), []byte("---\nid: TEAM-01\ntitle: Team rule\nseverity: MUST\n---\nKeep clean.\n"), 0o644)
		writeGradusConfig("  - id: team\n    source: { path: .gradus/levels/team }\n    locked: true\n")
		_, _, _ = runGradusCLI(t, []string{"resolve", "--dir", tmp, "--write"}, "")
		_, _, _ = runGradusCLI(t, []string{"compile", "--dir", tmp}, "")
		_ = os.WriteFile(filepath.Join(teamRules, "TEAM-01.md"), []byte("---\nid: TEAM-01\ntitle: Tampered\nseverity: MUST\n---\nTampered.\n"), 0o644)
		if code, _, _ := runGradusCLI(t, []string{"verify", "--dir", tmp}, ""); code == cli.ExitError {
			return 1.0
		}
	case "BENCH-07":
		_, _, _ = runGradusCLI(t, []string{"resolve", "--dir", tmp, "--write"}, "")
		_, _, _ = runGradusCLI(t, []string{"compile", "--dir", tmp}, "")
		targetFile := "CLAUDE.md"
		if target == "antigravity" {
			targetFile = "AGENTS.md"
		}
		_ = os.WriteFile(filepath.Join(tmp, targetFile), []byte("manual edit\n"), 0o644)
		if code, _, _ := runGradusCLI(t, []string{"verify", "--dir", tmp}, ""); code == cli.ExitViolation {
			return 1.0
		}
	case "BENCH-08":
		teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
		_ = os.MkdirAll(teamRules, 0o755)
		_ = os.WriteFile(filepath.Join(teamRules, "TEAM-01.md"), []byte("---\nid: TEAM-01\ntitle: Strict\nseverity: MUST\noverridable: false\n---\nStrict.\n"), 0o644)
		writeGradusConfig("  - id: team\n    source: { path: .gradus/levels/team }\n")
		writeRule("PRJ-LOOSEN.md", "---\nid: PRJ-LOOSEN\ntitle: Loosen\nseverity: SHOULD\noverrides: TEAM-01\n---\nLoosen.\n")
		if code, _, _ := runGradusCLI(t, []string{"resolve", "--dir", tmp}, ""); code == cli.ExitViolation {
			return 1.0
		}
	case "BENCH-09":
		longText := strings.Repeat("Validate boundaries deterministically. ", 25)
		for i := 1; i <= 15; i++ {
			writeRule(fmt.Sprintf("PRJ-BIG-%02d.md", i), fmt.Sprintf("---\nid: PRJ-BIG-%02d\ntitle: Big Rule %d\nseverity: MUST\n---\n%s\n", i, i, longText))
		}
		_, _, _ = runGradusCLI(t, []string{"resolve", "--dir", tmp, "--write"}, "")
		if code, _, _ := runGradusCLI(t, []string{"compile", "--dir", tmp}, ""); code != cli.ExitPass {
			return 0.0
		}
		if target == "antigravity" {
			entries, _ := os.ReadDir(filepath.Join(tmp, ".agents", "rules"))
			if len(entries) < 2 {
				return 0.0
			}
			for _, e := range entries {
				b, _ := os.ReadFile(filepath.Join(tmp, ".agents", "rules", e.Name()))
				if len(b) >= 12000 {
					return 0.0
				}
			}
		}
		if code, out, _ := runGradusCLI(t, []string{"compile", "--dir", tmp, "--json"}, ""); code == cli.ExitPass && strings.Contains(out, `"changed_files": []`) {
			return 1.0
		}
	case "BENCH-10":
		lessonsDir := t.TempDir()
		code, out, _ := runGradusCLI(t, []string{"lessons", "add", "--dir", tmp, "--lessons-dir", lessonsDir, "--kind", "correction", "--statement", "Token ghp_1234567890abcdef123456 and user@example.com"}, "")
		if code == cli.ExitPass && !strings.Contains(out, "ghp_") {
			files, _ := os.ReadDir(lessonsDir)
			if len(files) == 1 {
				raw, _ := os.ReadFile(filepath.Join(lessonsDir, files[0].Name()))
				if !strings.Contains(string(raw), "ghp_") && !strings.Contains(string(raw), "user@example.com") && strings.Contains(string(raw), "[REDACTED]") {
					return 1.0
				}
			}
		}
	case "BENCH-11":
		teamRules := filepath.Join(tmp, ".gradus", "levels", "team", "rules")
		_ = os.MkdirAll(teamRules, 0o755)
		writeGradusConfig("  - id: team\n    source: { path: .gradus/levels/team }\n    locked: true\n")
		draft := filepath.Join(tmp, "DRAFT.md")
		_ = os.WriteFile(draft, []byte("---\nid: DRAFT-01\ntitle: Draft\nseverity: MUST\n---\nDraft.\n"), 0o644)
		if code, _, _ := runGradusCLI(t, []string{"reconcile", "--dir", tmp, "--move", "--rule-file", draft, "--target-level", "team"}, ""); code == cli.ExitError {
			return 1.0
		}
	case "BENCH-12":
		if code, _, _ := runGradusCLI(t, []string{"compile", "--dir", tmp}, ""); code == cli.ExitPass {
			doc := "CLAUDE.md"
			if target == "antigravity" {
				doc = "AGENTS.md"
			}
			raw, _ := os.ReadFile(filepath.Join(tmp, doc))
			if strings.Contains(string(raw), "PRJ-ADV") && strings.Contains(string(raw), "Rejected rationalizations:") {
				return 1.0
			}
		}
	}
	return 0.0
}

func payloadForCommand(target, cmd string) string {
	if target == "antigravity" {
		return fmt.Sprintf(`{"tool_name":"run_command","tool_input":{"CommandLine":%q}}`, cmd)
	}
	return fmt.Sprintf(`{"tool_name":"Bash","tool_input":{"command":%q}}`, cmd)
}

func payloadForWrite(target, filePath string) string {
	if target == "antigravity" {
		return fmt.Sprintf(`{"tool_name":"write_to_file","tool_input":{"TargetFile":%q}}`, filePath)
	}
	return fmt.Sprintf(`{"tool_name":"Edit","tool_input":{"file_path":%q}}`, filePath)
}

func runGuardCase(t *testing.T, dir, payload, wantDecision string, wantTamper bool, wantExit int) float64 {
	t.Helper()
	origWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runGradusCLI(t, []string{"guard"}, payload)
	if code != wantExit {
		return 0.0
	}
	var res guard.EvaluationResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return 0.0
	}
	if res.Decision == wantDecision && res.AntiTamper == wantTamper {
		return 1.0
	}
	return 0.0
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")
}

func gitAddCommit(t *testing.T, dir, msg string) {
	t.Helper()
	runGitCmd(t, dir, "add", "-A")
	runGitCmd(t, dir, "commit", "-m", msg)
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
