package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Milosauro/gradus-ai/internal/cli"
	"github.com/Milosauro/gradus-ai/internal/guard"
)

func runGuardWithStdin(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	cmd := cli.NewRootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"guard"}, args...))

	err := cmd.Execute()
	if err == nil {
		return cli.ExitPass, stdout.String(), stderr.String()
	}
	if coded, ok := err.(*cli.ExitErrorWithCode); ok {
		return coded.Code, stdout.String(), stderr.String()
	}
	return cli.ExitError, stdout.String(), err.Error()
}

func TestGuardCommand(t *testing.T) {
	tmp := t.TempDir()
	origWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	projRulesDir := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	userDir := filepath.Join(tmp, "user-level")
	userRulesDir := filepath.Join(userDir, "rules")
	for _, d := range []string{projRulesDir, userRulesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte("schema: gradus/v1\nlevels:\n  - id: project\n    source: { path: .gradus/levels/project }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projRulesDir, "PRJ-DENY.md"), []byte("---\nid: PRJ-DENY\ntitle: No rm -rf\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: 'rm\\s+-rf'\n  decision: deny\n---\nNever run rm -rf.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userRulesDir, "USR-ASK.md"), []byte("---\nid: USR-ASK\ntitle: Confirm git commit\nseverity: MUST\nenforcement: hook\nguard:\n  action: command\n  pattern: 'git\\s+commit'\n  decision: ask\n---\nAsk before committing.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Allowed action -> exit 0
	code, out, _ := runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}`)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for allow, got %d (%s)", code, out)
	}
	var resAllow guard.EvaluationResult
	if err := json.Unmarshal([]byte(out), &resAllow); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if resAllow.Decision != guard.DecisionAllow || resAllow.HookSpecificOutput == nil || resAllow.HookSpecificOutput.PermissionDecision != guard.DecisionAllow {
		t.Fatalf("unexpected allow result: %+v", resAllow)
	}

	// 2. Anti-tamper ask -> exit 0 with anti_tamper=true
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"tool_name":"write_to_file","tool_input":{"TargetFile":".agents/hooks.json"}}`)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for anti-tamper ask, got %d (%s)", code, out)
	}
	var resTamper guard.EvaluationResult
	if err := json.Unmarshal([]byte(out), &resTamper); err != nil {
		t.Fatal(err)
	}
	if resTamper.Decision != guard.DecisionAsk || !resTamper.AntiTamper {
		t.Fatalf("expected ask with anti_tamper=true, got %+v", resTamper)
	}

	// 3. User-level hook ask -> exit 0
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"tool_name":"run_command","tool_input":{"CommandLine":"git commit -m 'hi'"}}`)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for user ask, got %d", code)
	}
	var resAsk guard.EvaluationResult
	if err := json.Unmarshal([]byte(out), &resAsk); err != nil {
		t.Fatal(err)
	}
	if resAsk.Decision != guard.DecisionAsk || len(resAsk.MatchedRules) != 1 || resAsk.MatchedRules[0].ID != "USR-ASK" {
		t.Fatalf("expected user-level ask result, got %+v", resAsk)
	}

	// 4. Deny rule -> exit 1
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"tool_name":"Bash","tool_input":{"command":"rm -rf /tmp/foo"}}`)
	if code != cli.ExitViolation {
		t.Fatalf("expected exit 1 for deny, got %d (%s)", code, out)
	}
	var resDeny guard.EvaluationResult
	if err := json.Unmarshal([]byte(out), &resDeny); err != nil {
		t.Fatal(err)
	}
	if resDeny.Decision != guard.DecisionDeny || resDeny.HookSpecificOutput.PermissionDecision != guard.DecisionDeny {
		t.Fatalf("expected deny result, got %+v", resDeny)
	}

	// 5. Native Antigravity protojson payload emits strict {decision: "force_ask", reason} without unknown fields
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"toolCall":{"id":"toolu_1","name":"run_command","args":{"CommandLine":"git commit -m hi"}},"cwd":"`+filepath.ToSlash(tmp)+`","hookEventName":"PreToolUse"}`)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for native Antigravity ask, got %d (%s)", code, out)
	}
	if strings.Contains(out, "matched_rules") || strings.Contains(out, "hookSpecificOutput") || !strings.Contains(out, `"decision":"force_ask"`) {
		t.Fatalf("expected strict protojson force_ask output for native Antigravity, got: %s", out)
	}

	// 6. Native Claude Code payload emits {} on allow
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"session_id":"sess_1","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"git status"}}`)
	if code != cli.ExitPass || strings.TrimSpace(out) != "{}" {
		t.Fatalf("expected exit 0 and {} for native Claude Code allow, got code=%d out=%s", code, out)
	}

	// 7. Native Claude Code payload emits strict hookSpecificOutput only on ask/anti-tamper (exit 0)
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"session_id":"sess_1","hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":"gradus.yaml"}}`)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for native Claude Code ask, got %d (%s)", code, out)
	}
	if strings.Contains(out, `"decision"`) || strings.Contains(out, "matched_rules") || strings.Contains(out, "anti_tamper") || !strings.Contains(out, `"permissionDecision":"ask"`) {
		t.Fatalf("expected strict hookSpecificOutput ask for native Claude Code, got: %s", out)
	}

	// 8. Native Claude Code payload emits strict hookSpecificOutput deny with exit 0
	code, out, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{"session_id":"sess_1","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rm -rf /tmp/foo"}}`)
	if code != cli.ExitPass {
		t.Fatalf("expected exit 0 for native Claude Code deny, got %d (%s)", code, out)
	}
	if strings.Contains(out, `"decision"`) || !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("expected strict hookSpecificOutput deny for native Claude Code, got: %s", out)
	}

	// 9. Malformed JSON -> fail closed exit 2
	code, _, _ = runGuardWithStdin(t, []string{"--user-dir", userDir}, `{invalid json`)
	if code != cli.ExitError {
		t.Fatalf("expected exit 2 on malformed JSON, got %d", code)
	}
}
