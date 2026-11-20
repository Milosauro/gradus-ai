package guard_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Milosauro/gradus-ai/internal/guard"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func TestParsePayload(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name       string
		raw        string
		wantAction string
		wantTarget string
		wantErr    bool
	}{
		{
			name:       "claude code Bash",
			raw:        `{"tool_name":"Bash","tool_input":{"command":"git commit -m 'test'"}}`,
			wantAction: guard.ActionCommand,
			wantTarget: "git commit -m 'test'",
		},
		{
			name:       "claude code Edit with relative dot-slash",
			raw:        `{"tool_name":"Edit","tool_input":{"file_path":"./gradus.yaml"}}`,
			wantAction: guard.ActionWrite,
			wantTarget: "gradus.yaml",
		},
		{
			name:       "claude code Write with absolute workspace path",
			raw:        `{"tool_name":"Write","tool_input":{"file_path":"` + filepath.ToSlash(filepath.Join(root, ".gradus", "levels", "project", "rules", "R1.md")) + `"}}`,
			wantAction: guard.ActionWrite,
			wantTarget: ".gradus/levels/project/rules/R1.md",
		},
		{
			name:       "claude code MultiEdit",
			raw:        `{"tool_name":"MultiEdit","tool_input":{"file_path":"internal/cli/root.go"}}`,
			wantAction: guard.ActionWrite,
			wantTarget: "internal/cli/root.go",
		},
		{
			name:       "antigravity run_command",
			raw:        `{"tool_name":"run_command","tool_input":{"CommandLine":"git push origin main"}}`,
			wantAction: guard.ActionCommand,
			wantTarget: "git push origin main",
		},
		{
			name:       "antigravity write_to_file with windows backslashes",
			raw:        `{"tool_name":"write_to_file","tool_input":{"TargetFile":".\\.walden\\constitution.md"}}`,
			wantAction: guard.ActionWrite,
			wantTarget: ".walden/constitution.md",
		},
		{
			name:       "antigravity replace_file_content",
			raw:        `{"tool_name":"replace_file_content","tool_input":{"TargetFile":".agents/hooks.json"}}`,
			wantAction: guard.ActionWrite,
			wantTarget: ".agents/hooks.json",
		},
		{
			name:       "antigravity multi_replace_file_content",
			raw:        `{"tool_name":"multi_replace_file_content","tool_input":{"TargetFile":".claude/settings.json"}}`,
			wantAction: guard.ActionWrite,
			wantTarget: ".claude/settings.json",
		},
		{
			name:    "empty stdin fails closed",
			raw:     "   ",
			wantErr: true,
		},
		{
			name:    "malformed json fails closed",
			raw:     `{"tool_name":`,
			wantErr: true,
		},
		{
			name:    "missing tool_name fails closed",
			raw:     `{"tool_input":{"command":"ls"}}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := guard.ParsePayload(root, []byte(tc.raw))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s, got %+v", tc.name, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
			if got.Action != tc.wantAction {
				t.Fatalf("action mismatch for %s: got %q, want %q", tc.name, got.Action, tc.wantAction)
			}
			if got.Target != tc.wantTarget {
				t.Fatalf("target mismatch for %s: got %q, want %q", tc.name, got.Target, tc.wantTarget)
			}
		})
	}
}

func TestAntiTamperProtection(t *testing.T) {
	protected := []string{
		"gradus.yaml",
		"gradus.lock",
		".gradus/levels/project/rules/R1.md",
		".walden/constitution.md",
		".agents/hooks.json",
		".claude/settings.json",
	}

	for _, p := range protected {
		res, err := guard.Evaluate(guard.NormalizedAction{
			ToolName: "Edit",
			Action:   guard.ActionWrite,
			Target:   p,
		}, nil)
		if err != nil {
			t.Fatalf("Evaluate failed for %s: %v", p, err)
		}
		if res.Decision != guard.DecisionAsk || !res.AntiTamper {
			t.Fatalf("expected ask with anti_tamper=true for %s, got decision=%q anti_tamper=%v", p, res.Decision, res.AntiTamper)
		}
	}

	// Non-protected file returns allow.
	resAllow, err := guard.Evaluate(guard.NormalizedAction{
		ToolName: "Write",
		Action:   guard.ActionWrite,
		Target:   "internal/app/handler.go",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resAllow.Decision != guard.DecisionAllow || resAllow.AntiTamper {
		t.Fatalf("expected allow without anti-tamper for normal file, got %+v", resAllow)
	}

	// Explicit deny rule overrides built-in anti-tamper ask floor (R2.AC2).
	denyRules := []resolve.ResolvedRule{
		{
			ID:          "ORG-SEC-LOCK",
			Title:       "Forbid lockfile edits",
			Level:       "org",
			Enforcement: "hook",
			Scope:       []string{"**/*"},
			Guard: &schema.GuardConfig{
				Action:   "write",
				Pattern:  `^gradus\.lock$`,
				Decision: "deny",
			},
		},
	}
	resDeny, err := guard.Evaluate(guard.NormalizedAction{
		ToolName: "Edit",
		Action:   guard.ActionWrite,
		Target:   "gradus.lock",
	}, denyRules)
	if err != nil {
		t.Fatal(err)
	}
	if resDeny.Decision != guard.DecisionDeny || !resDeny.AntiTamper || len(resDeny.MatchedRules) != 1 {
		t.Fatalf("expected deny overriding anti-tamper ask, got %+v", resDeny)
	}

	// Shell commands targeting governance files must also trigger anti-tamper ask (T2)
	for _, cmd := range []string{
		"rm gradus.yaml",
		"sed -i '' 's/MUST/MAY/' .gradus/levels/project/rules/R1.md",
		"echo '{}' > .claude/settings.json",
		"mv .agents/hooks.json /tmp/hooks.json",
	} {
		resShell, err := guard.Evaluate(guard.NormalizedAction{
			ToolName: "Bash",
			Action:   guard.ActionCommand,
			Target:   cmd,
		}, nil)
		if err != nil {
			t.Fatalf("Evaluate failed for %q: %v", cmd, err)
		}
		if resShell.Decision != guard.DecisionAsk || !resShell.AntiTamper {
			t.Fatalf("expected shell anti-tamper ask for %q, got %+v", cmd, resShell)
		}
	}

	// Read-only shell inspection of governance files remains allowed
	resRead, err := guard.Evaluate(guard.NormalizedAction{
		ToolName: "Bash",
		Action:   guard.ActionCommand,
		Target:   "cat gradus.yaml",
	}, nil)
	if err != nil || resRead.Decision != guard.DecisionAllow || resRead.AntiTamper {
		t.Fatalf("expected read-only shell command to be allowed, got %+v (err=%v)", resRead, err)
	}
}

func TestGuardEvaluatePrecedenceAndUserLevel(t *testing.T) {
	tmp := t.TempDir()
	projRulesDir := filepath.Join(tmp, ".gradus", "levels", "project", "rules")
	userDir := filepath.Join(tmp, "home-user")
	userRulesDir := filepath.Join(userDir, "rules")
	for _, d := range []string{projRulesDir, userRulesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	gradusYAML := `schema: gradus/v1
levels:
  - id: project
    source: { path: .gradus/levels/project }
`
	if err := os.WriteFile(filepath.Join(tmp, "gradus.yaml"), []byte(gradusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project hook rule: deny force push
	denyPush := `---
id: PRJ-HOOK-01
title: Forbid git push --force
severity: MUST
enforcement: hook
guard:
  action: command
  pattern: "git\\s+push.*--force"
  decision: deny
---
Never force push.
`
	if err := os.WriteFile(filepath.Join(projRulesDir, "PRJ-HOOK-01.md"), []byte(denyPush), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project non-hook rule (prompt only) should not trigger guard
	promptRule := `---
id: PRJ-PROMPT-01
title: Prompt only rule
severity: SHOULD
enforcement: prompt
---
Just guidance.
`
	if err := os.WriteFile(filepath.Join(projRulesDir, "PRJ-PROMPT-01.md"), []byte(promptRule), 0o644); err != nil {
		t.Fatal(err)
	}

	// Personal user-level hook rule (ADR 0003): ask before git commit or push (including -C/-c flags)
	userRule := `---
id: USR-HOOK-01
title: Confirm git commit or push
severity: MUST
enforcement: hook
guard:
  action: command
  pattern: '\bgit\b(?:\s+(?:-[Cc]\s+\S+|-\S+))*\s+(commit|push)\b'
  decision: ask
---
Ask before committing or pushing.
`
	if err := os.WriteFile(filepath.Join(userRulesDir, "USR-HOOK-01.md"), []byte(userRule), 0o644); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	rules, err := guard.LoadLocalGuardRules(tmp, userDir)
	if err != nil {
		t.Fatalf("LoadLocalGuardRules failed: %v", err)
	}

	// 1. User-level ask on git commit (including git -C . commit and git -c ... commit)
	for _, cmd := range []string{
		"git commit -m 'feat: test'",
		"git -C . commit -m 'feat: bypass attempt'",
		"git -c user.name=x commit -m 'feat: bypass attempt'",
	} {
		resCommit, err := guard.Evaluate(guard.NormalizedAction{
			ToolName: "Bash",
			Action:   guard.ActionCommand,
			Target:   cmd,
		}, rules)
		if err != nil {
			t.Fatal(err)
		}
		if resCommit.Decision != guard.DecisionAsk || len(resCommit.MatchedRules) != 1 || resCommit.MatchedRules[0].ID != "USR-HOOK-01" {
			t.Fatalf("expected user-level ask on %q, got %+v", cmd, resCommit)
		}
	}

	// 2. Precedence deny > ask when both PRJ-HOOK-01 (deny) and USR-HOOK-01 (ask) match
	resForcePush, err := guard.Evaluate(guard.NormalizedAction{
		ToolName: "run_command",
		Action:   guard.ActionCommand,
		Target:   "git push --force origin main",
	}, rules)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if resForcePush.Decision != guard.DecisionDeny || len(resForcePush.MatchedRules) != 2 {
		t.Fatalf("expected deny with 2 matched rules, got %+v", resForcePush)
	}
	if elapsed >= 100*time.Millisecond {
		t.Fatalf("expected local guard load+evaluate < 100ms, took %v", elapsed)
	}

	// 3. Waived rule must be skipped
	waivedRules := []resolve.ResolvedRule{
		{
			ID:          "PRJ-HOOK-WAIVED",
			Title:       "Waived guard",
			Enforcement: "hook",
			Waived:      true,
			Guard:       &schema.GuardConfig{Action: "command", Pattern: ".*", Decision: "deny"},
		},
	}
	resWaived, err := guard.Evaluate(guard.NormalizedAction{Action: guard.ActionCommand, Target: "ls"}, waivedRules)
	if err != nil {
		t.Fatal(err)
	}
	if resWaived.Decision != guard.DecisionAllow || len(resWaived.MatchedRules) != 0 {
		t.Fatalf("expected waived rule to be skipped, got %+v", resWaived)
	}

	// 4. Invalid regex fails closed
	badRules := []resolve.ResolvedRule{
		{
			ID:          "BAD-REGEX",
			Title:       "Broken regex",
			Enforcement: "hook",
			Guard:       &schema.GuardConfig{Action: "command", Pattern: "[unclosed", Decision: "deny"},
		},
	}
	if _, err := guard.Evaluate(guard.NormalizedAction{Action: guard.ActionCommand, Target: "ls"}, badRules); err == nil {
		t.Fatalf("expected invalid guard regex to return error")
	}
}
