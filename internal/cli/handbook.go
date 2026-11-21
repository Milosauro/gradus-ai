package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func newHandbookCmd(jsonOutput *bool) *cobra.Command {
	var dir, outputPath string

	cmd := &cobra.Command{
		Use:   "handbook",
		Short: "Generate a human-readable Markdown onboarding handbook from the resolved pyramid",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			resolved, err := resolve.Resolve(dir, resolve.Options{})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			md := compile.RenderHandbook(resolved.View)
			if outputPath != "" {
				if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if err := os.WriteFile(outputPath, []byte(md), 0o644); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			}

			if *jsonOutput {
				levels := make([]string, 0, len(resolved.View.Levels))
				for _, l := range resolved.View.Levels {
					levels = append(levels, l.ID)
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"view_hash":   resolved.View.ViewHash,
					"levels":      levels,
					"rules_count": len(resolved.View.Rules),
					"output_path": outputPath,
					"markdown":    md,
				})
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), md)
			return err
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&outputPath, "output", "", "Optional output file path for the generated Markdown handbook")
	return cmd
}
