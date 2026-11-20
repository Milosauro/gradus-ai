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

			rules, err := guard.LoadLocalGuardRules(rootDir, userDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			res, err := guard.Evaluate(action, rules)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			enc := json.NewEncoder(cmd.OutOrStdout())
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
