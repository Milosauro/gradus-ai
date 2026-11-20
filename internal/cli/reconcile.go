package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/state"
	"github.com/Milosauro/gradus-ai/skills"
)

func newReconcileCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var restore bool
	var move bool
	var ruleFile string
	var targetLevel string

	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Inspect compiled-target drift and restore or move rules to editable levels",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if restore && move {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("specify either --restore or --move, not both")}
			}

			cfgBytes, err := os.ReadFile(filepath.Join(dir, "gradus.yaml"))
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("failed to read gradus.yaml: %w", err)}
			}
			cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if move {
				destRel, err := state.MoveRuleToLevel(dir, cfgRes.Value, ruleFile, targetLevel)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if *jsonOutput {
					return encodeJSON(cmd, map[string]any{
						"action":       "move",
						"target_level": targetLevel,
						"destination":  destRel,
					})
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "moved rule to %s (%s)\n", targetLevel, destRel)
				return err
			}

			resolved, err := resolve.Resolve(dir, resolve.Options{})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if len(resolved.View.Conflicts) > 0 {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("cannot reconcile while %d blocking conflict(s) exist", len(resolved.View.Conflicts))}
			}

			if restore {
				res, err := state.RestoreDrift(dir, cfgRes.Value.Targets, resolved.View, skills.FS)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if *jsonOutput {
					return encodeJSON(cmd, map[string]any{
						"action":        "restore",
						"changed_files": res.ChangedFiles,
						"removed_files": res.RemovedFiles,
					})
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "restored %d file(s)\n", len(res.ChangedFiles))
				return err
			}

			drift, err := state.InspectDrift(dir, cfgRes.Value.Targets, resolved.View, skills.FS)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				if err := encodeJSON(cmd, map[string]any{"drift": drift}); err != nil {
					return err
				}
			} else if len(drift) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No compiled-target drift detected.")
				return err
			} else {
				for _, d := range drift {
					fmt.Fprintf(cmd.OutOrStdout(), "DRIFT [%s] %s (options: %v)\n", d.Reason, d.Path, d.Options)
				}
			}

			if len(drift) > 0 {
				return &ExitErrorWithCode{Code: ExitViolation, Err: fmt.Errorf("found %d drifted file(s)", len(drift))}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().BoolVar(&restore, "restore", false, "Restore compiled targets deterministically")
	cmd.Flags().BoolVar(&move, "move", false, "Move a rule file into an unlocked target level")
	cmd.Flags().StringVar(&ruleFile, "rule-file", "", "Rule Markdown file to move")
	cmd.Flags().StringVar(&targetLevel, "target-level", "", "Destination unlocked level ID")
	return cmd
}
