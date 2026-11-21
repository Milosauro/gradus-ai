package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/check"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func newSimulateCmd(jsonOutput *bool) *cobra.Command {
	var dir, since, ruleID string
	var commits int

	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "Simulate check rule impact across recent git commits without modifying the working tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			resolved, err := resolve.Resolve(dir, resolve.Options{})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			rep, err := check.RunSimulation(dir, resolved.View, check.SimulateOptions{
				Commits: commits,
				Since:   since,
				RuleID:  ruleID,
			})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rep)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Simulated %d commit(s): %d affected\n", rep.CommitsScanned, rep.AffectedCommits)
			for _, c := range rep.Commits {
				fmt.Fprintf(out, "  commit %s: %d finding(s)\n", c.SHA, len(c.Findings))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().IntVar(&commits, "commits", 10, "Number of recent commits to scan")
	cmd.Flags().StringVar(&since, "since", "", "Git ref to scan <since>..HEAD")
	cmd.Flags().StringVar(&ruleID, "rule", "", "Simulate only a specific rule ID")
	return cmd
}
