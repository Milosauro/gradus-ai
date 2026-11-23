package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/toolchain"
)

func newToolchainCmd(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toolchain",
		Short: "Inspect and verify governed toolchain requirements",
	}
	cmd.AddCommand(newToolchainCheckCmd(jsonOutput))
	return cmd
}

func newToolchainCheckCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check required, recommended, and forbidden skills, tools, MCP servers, and plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			res, err := resolve.Resolve(dir, resolve.Options{OfflineOnly: true})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			rep := toolchain.Check(dir, res.View.Toolchain)

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				if rep.Verdict == "pass" {
					fmt.Fprintf(out, "toolchain check passed (%d item(s) checked)\n", rep.Checked)
				} else {
					for _, f := range rep.Findings {
						fmt.Fprintf(out, "VIOLATION [%s]: %s\n", f.Kind, f.Message)
					}
				}
				for _, w := range rep.Warnings {
					fmt.Fprintf(out, "WARNING: %s\n", w)
				}
			}

			if rep.Verdict != "pass" {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("toolchain check failed: %d finding(s)", len(rep.Findings)),
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}
