package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/skills"
)

func newCompileCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile the resolved pyramid into target agent instructions, skills, and hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}

			cfgBytes, err := os.ReadFile(filepath.Join(dir, "gradus.yaml"))
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("failed to read gradus.yaml: %w", err)}
			}
			cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			resolved, err := resolve.Resolve(dir, resolve.Options{})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if len(resolved.View.Conflicts) > 0 {
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("compilation aborted: %d blocking conflict(s) detected", len(resolved.View.Conflicts)),
				}
			}

			res, err := compile.ApplyCompile(dir, resolved.View, cfgRes.Value.Targets, skills.FS)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Compiled targets %v (view %s)\n", res.Targets, res.ViewHash)
			fmt.Fprintf(out, "Changed: %d, Unchanged: %d, Removed: %d\n",
				len(res.ChangedFiles), len(res.UnchangedFiles), len(res.RemovedFiles))
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}
