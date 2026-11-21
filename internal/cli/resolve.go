package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func newResolveCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var writeLock bool
	var includeDraft bool

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve pyramid levels, overlays, exceptions, and conflicts into a resolved view",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			res, err := resolve.Resolve(dir, resolve.Options{
				IncludeDraft: includeDraft,
			})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if writeLock {
				lockPath := filepath.Join(dir, "gradus.lock")
				if err := lock.WriteLockfile(lockPath, res.Lockfile); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if lockBytes, readErr := os.ReadFile(lockPath); readErr == nil {
					res.View.LockHash = lock.HashBytes(lockBytes)
				}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(res.View); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "Resolved %d rule(s) across %d level(s), %d overlay(s)\n",
					len(res.View.Rules), len(res.View.Levels), len(res.View.Overlays))
				fmt.Fprintf(out, "View hash: %s\n", res.View.ViewHash)
				if writeLock {
					fmt.Fprintf(out, "Updated gradus.lock (run 'gradusctl compile' to update compiled agent files)\n")
				}
				for _, c := range res.View.Conflicts {
					fmt.Fprintf(out, "CONFLICT [%s]: %s\n", c.Kind, c.Message)
				}
			}

			if len(res.View.Conflicts) > 0 {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("%d blocking conflict(s) detected", len(res.View.Conflicts)),
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().BoolVar(&writeLock, "write", false, "Write resolved entries and view_hash to gradus.lock")
	cmd.Flags().BoolVar(&includeDraft, "include-draft", false, "Include rules with status: draft")
	return cmd
}
