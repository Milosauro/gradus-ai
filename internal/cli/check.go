package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/check"
	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

type checkReport struct {
	Verdict         string             `json:"verdict"`
	Files           []string           `json:"files"`
	Findings        []evidence.Finding `json:"findings"`
	EvidenceWritten []string           `json:"evidence_written,omitempty"`
}

func newCheckCmd(jsonOutput *bool) *cobra.Command {
	var dir, base string
	var staged, writeEvidence bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run deterministic code checks on staged, branch, or repository files",
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

			checkRes, err := check.RunChecks(dir, resolved.View, check.Options{
				Staged: staged,
				Base:   base,
			})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			verdict := "pass"
			if len(checkRes.Findings) > 0 {
				verdict = "fail"
			}

			var written []string
			if writeEvidence {
				rec := evidence.NewRecord(dir, "check", verdict, base, resolved.View.LockHash, resolved.View.ViewHash, checkRes.Findings, time.Now().UTC())
				written, err = evidence.WriteRecord(dir, cfgRes.Value.Evidence, rec, time.Now().UTC())
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			}

			rep := checkReport{
				Verdict:         verdict,
				Files:           checkRes.Files,
				Findings:        checkRes.Findings,
				EvidenceWritten: written,
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				if len(checkRes.Findings) == 0 {
					fmt.Fprintf(out, "check passed (%d file(s) scanned)\n", len(checkRes.Files))
				} else {
					for _, f := range checkRes.Findings {
						fmt.Fprintf(out, "✗ %s  %s  %s:%d\n  %s\n  Level: %s · Source: %s\n",
							f.RuleID, f.Severity, f.Path, f.Line, f.Message, f.Level, f.Source)
					}
					fmt.Fprintf(out, "%d violation(s).\n", len(checkRes.Findings))
				}
			}

			if len(checkRes.Findings) > 0 {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("%d check violation(s) found", len(checkRes.Findings)),
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().BoolVar(&staged, "staged", false, "Check only git staged files")
	cmd.Flags().StringVar(&base, "base", "", "Check files changed relative to base git ref")
	cmd.Flags().BoolVar(&writeEvidence, "write-evidence", false, "Persist check evidence to configured sinks")
	return cmd
}
