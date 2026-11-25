package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/trailers"
)

func newTrailersCmd(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trailers",
		Short: "Generate or verify optional Gradus process-attribution git commit trailers (X-05, D-65)",
	}
	cmd.AddCommand(newTrailersGenerateCmd(jsonOutput))
	cmd.AddCommand(newTrailersVerifyCmd(jsonOutput))
	return cmd
}

func newTrailersGenerateCmd(jsonOutput *bool) *cobra.Command {
	var dir, agent, modelTier, reviewID string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Gradus-* git commit trailer lines for the current pyramid",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			ts, err := trailers.Generate(dir, trailers.GenerateOptions{
				Agent:     agent,
				ModelTier: modelTier,
				ReviewID:  reviewID,
			})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(ts)
			}
			for _, line := range ts.Lines {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&agent, "agent", "", "Optional agent identifier (e.g. claude-code, antigravity)")
	cmd.Flags().StringVar(&modelTier, "model-tier", "", "Optional model capability tier (tier-high|tier-standard|tier-light)")
	cmd.Flags().StringVar(&reviewID, "review-id", "", "Optional evidence/review ID")
	return cmd
}

func newTrailersVerifyCmd(jsonOutput *bool) *cobra.Command {
	var dir, ref string
	var requireLockHash bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify Gradus-* git commit trailers on a commit ref",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			rep, err := trailers.VerifyCommit(dir, ref, trailers.VerifyOptions{
				RequireLockHash: requireLockHash,
			})
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
				if rep.Verdict == "pass" {
					fmt.Fprintf(cmd.OutOrStdout(), "trailers verify passed for %s (%d trailer(s))\n", rep.Ref, len(rep.Trailers))
				} else {
					for _, issue := range rep.Issues {
						fmt.Fprintf(cmd.OutOrStdout(), "TRAILER ISSUE: %s\n", issue)
					}
				}
			}
			if rep.Verdict != "pass" {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("trailer verification failed: %d issue(s)", len(rep.Issues)),
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&ref, "ref", "HEAD", "Git commit reference to inspect")
	cmd.Flags().BoolVar(&requireLockHash, "require-lock-hash", false, "Require Gradus-Lock-Hash trailer matching current gradus.lock")
	return cmd
}
