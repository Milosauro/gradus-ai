package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/state"
)

func newConflictsCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var candidates bool
	var confirm bool
	var dismiss bool
	var ruleA string
	var ruleB string
	var message string

	cmd := &cobra.Command{
		Use:   "conflicts",
		Short: "Discover semantic conflict candidates and manage conflict findings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if confirm && dismiss {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("specify either --confirm or --dismiss, not both")}
			}

			if confirm || dismiss {
				if ruleA == "" || ruleB == "" {
					return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("--rule-a and --rule-b are required with --confirm or --dismiss")}
				}
				res, err := resolve.Resolve(dir, resolve.Options{})
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				hashA, okA := findRuleHash(res.View.Rules, ruleA)
				hashB, okB := findRuleHash(res.View.Rules, ruleB)
				if !okA {
					return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("rule %q not found in resolved view", ruleA)}
				}
				if !okB {
					return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("rule %q not found in resolved view", ruleB)}
				}

				status := "confirmed"
				if dismiss {
					status = "dismissed"
				}
				rec, relPath, err := state.SaveFinding(dir, state.FindingRecord{
					RuleA:   ruleA,
					HashA:   hashA,
					RuleB:   ruleB,
					HashB:   hashB,
					Status:  status,
					Message: message,
				}, time.Now().UTC())
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}

				if *jsonOutput {
					return encodeJSON(cmd, map[string]any{
						"finding": rec,
						"path":    relPath,
					})
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s finding %s (%s vs %s) -> %s\n", status, rec.ID, rec.RuleA, rec.RuleB, relPath)
				return err
			}

			if candidates {
				res, err := resolve.Resolve(dir, resolve.Options{})
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				existing, err := state.ListFindings(dir)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				cands := state.FindCandidates(res.View, existing)
				if *jsonOutput {
					return encodeJSON(cmd, map[string]any{"candidates": cands})
				}
				if len(cands) == 0 {
					_, err = fmt.Fprintln(cmd.OutOrStdout(), "No semantic conflict candidates found.")
					return err
				}
				for _, c := range cands {
					fmt.Fprintf(cmd.OutOrStdout(), "%s (%s) <-> %s (%s): %s\n", c.RuleA, c.LevelA, c.RuleB, c.LevelB, c.Reason)
				}
				return nil
			}

			findings, err := state.ListFindings(dir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				return encodeJSON(cmd, map[string]any{"findings": findings})
			}
			if len(findings) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No conflict findings recorded.")
				return err
			}
			for _, f := range findings {
				fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] %s <-> %s: %s\n", f.ID, f.Status, f.RuleA, f.RuleB, f.Message)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().BoolVar(&candidates, "candidates", false, "Discover candidate rule pairs for semantic conflict review")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm a semantic conflict between --rule-a and --rule-b")
	cmd.Flags().BoolVar(&dismiss, "dismiss", false, "Dismiss a candidate pair with current rule hashes")
	cmd.Flags().StringVar(&ruleA, "rule-a", "", "First rule ID")
	cmd.Flags().StringVar(&ruleB, "rule-b", "", "Second rule ID")
	cmd.Flags().StringVar(&message, "message", "", "Justification or conflict description")
	return cmd
}

func findRuleHash(rules []resolve.ResolvedRule, id string) (string, bool) {
	for _, r := range rules {
		if r.ID == id {
			return r.Provenance.Hash, true
		}
	}
	return "", false
}

func encodeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return &ExitErrorWithCode{Code: ExitError, Err: err}
	}
	return nil
}

var _ = os.ErrNotExist
