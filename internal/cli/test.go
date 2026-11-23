package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/check"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func newTestCmd(jsonOutput *bool) *cobra.Command {
	var dir, ruleID string

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run declarative rule tests from Compliant and Non-compliant examples in rule files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			pyramid, err := schema.LoadLocalPyramid(dir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			rep, err := check.RunRuleTests(pyramid.Rules, ruleID)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				if rep.Passed == 0 && rep.Failed == 0 {
					fmt.Fprintf(out, "rule tests: %s (0 passed, 0 failed — no rule examples found)\n", rep.Verdict)
				} else {
					fmt.Fprintf(out, "rule tests: %s (%d passed, %d failed)\n", rep.Verdict, rep.Passed, rep.Failed)
				}
				for _, c := range rep.Cases {
					status := "PASS"
					if !c.Passed {
						status = "FAIL"
					}
					if c.Message != "" {
						fmt.Fprintf(out, "  [%s] %s (%s #%d): %s\n", status, c.RuleID, c.Kind, c.Index, c.Message)
					} else {
						fmt.Fprintf(out, "  [%s] %s (%s #%d)\n", status, c.RuleID, c.Kind, c.Index)
					}
				}
			}

			if rep.Failed > 0 {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{Code: ExitViolation, Err: fmt.Errorf("%d rule test case(s) failed", rep.Failed)}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&ruleID, "rule", "", "Run tests only for a specific rule ID")
	return cmd
}
