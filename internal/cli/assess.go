package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/assess"
)

func newAssessCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "assess",
		Short: "Evaluate repository governance process maturity (never individuals, X-13 / P16)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			rep, err := assess.Evaluate(dir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rep)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Process Maturity: %s (%d/%d)\n", rep.MaturityLevel, rep.TotalScore, rep.MaxScore)
			for _, d := range rep.Dimensions {
				fmt.Fprintf(out, "  - %-26s %2d/%2d\n", d.Name, d.Score, d.Max)
			}
			if len(rep.Recommendations) > 0 {
				fmt.Fprintln(out, "Recommendations:")
				for _, r := range rep.Recommendations {
					fmt.Fprintf(out, "  * %s\n", r)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}
