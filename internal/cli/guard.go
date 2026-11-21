package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/guard"
)

func newGuardCmd(_ *bool) *cobra.Command {
	var userDir string

	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Evaluate a PreToolUse hook payload against local rules and anti-tamper protection",
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir, err := os.Getwd()
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			rawStdin, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			action, err := guard.ParsePayload(rootDir, rawStdin)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if action.WorkspaceDir != "" {
				rootDir = action.WorkspaceDir
			}

			rules, err := guard.LoadLocalGuardRules(rootDir, userDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			res, err := guard.Evaluate(action, rules)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			enc := json.NewEncoder(cmd.OutOrStdout())
			if action.IsAntigravityNative {
				// Antigravity parses PreToolUse stdout with strict protojson (rejects unknown fields)
				// and requires exit 0 to honor structured allow/force_ask/deny decisions.
				// Map ask -> force_ask so cached "Always Allow" tool approvals do not bypass Gradus guards.
				agDecision := res.Decision
				if agDecision == guard.DecisionAsk {
					agDecision = "force_ask"
				}
				agResp := struct {
					Decision string `json:"decision"`
					Reason   string `json:"reason,omitempty"`
				}{
					Decision: agDecision,
					Reason:   res.Reason,
				}
				if err := enc.Encode(agResp); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				return nil
			}
			if action.IsClaudeCodeNative {
				// Claude Code validates PreToolUse stdout strictly and rejects top-level "decision": "ask"
				// or non-zero exit codes when structured JSON is returned. Return {} on allow so normal
				// Claude Code permissions remain in effect, and hookSpecificOutput only on ask/deny with exit 0.
				if res.Decision == guard.DecisionAllow {
					if err := enc.Encode(struct{}{}); err != nil {
						return &ExitErrorWithCode{Code: ExitError, Err: err}
					}
					return nil
				}
				ccResp := struct {
					HookSpecificOutput *guard.HookSpecificOutput `json:"hookSpecificOutput"`
				}{
					HookSpecificOutput: res.HookSpecificOutput,
				}
				if err := enc.Encode(ccResp); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				return nil
			}
			if err := enc.Encode(res); err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if res.Decision == guard.DecisionDeny {
				return &ExitErrorWithCode{Code: ExitViolation}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userDir, "user-dir", "", "Override user-level rules directory (defaults to ~/.gradus/levels/user)")
	return cmd
}
