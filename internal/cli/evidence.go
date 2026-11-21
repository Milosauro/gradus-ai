package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/evidence"
	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func newEvidenceCmd(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evidence",
		Short: "Write or inspect verifiable evidence records",
	}
	cmd.AddCommand(newEvidenceListCmd(jsonOutput))
	cmd.AddCommand(newEvidenceWriteCmd(jsonOutput))
	cmd.AddCommand(newEvidencePruneCmd(jsonOutput))
	return cmd
}

func newEvidenceListCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local evidence records sorted newest-first",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			records, err := evidence.ListLocalRecords(dir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(records)
			}
			for _, r := range records {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %-6s  %s  findings=%d\n",
					r.ID, r.Timestamp, r.Verdict, r.Kind, len(r.Findings))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}

func newEvidenceWriteCmd(jsonOutput *bool) *cobra.Command {
	var dir, kind, verdict, base, findingsFile string

	cmd := &cobra.Command{
		Use:   "write",
		Short: "Write an evidence record to configured sinks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			switch kind {
			case "verify", "check", "review", "semantic-conflicts", "exception-use":
			default:
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid evidence kind %q", kind)}
			}
			switch verdict {
			case "pass", "fail", "error":
			default:
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid evidence verdict %q", verdict)}
			}

			var evCfg *schema.EvidenceConfig
			if cfgBytes, err := os.ReadFile(filepath.Join(dir, "gradus.yaml")); err == nil {
				if parsed, parseErr := schema.ParsePyramidConfig(cfgBytes); parseErr == nil {
					evCfg = parsed.Value.Evidence
				}
			}

			lockHash, viewHash := readLockHashes(dir)

			var findings []evidence.Finding
			if findingsFile != "" {
				data, err := os.ReadFile(findingsFile)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("failed to read findings file: %w", err)}
				}
				if err := json.Unmarshal(data, &findings); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid findings JSON: %w", err)}
				}
			}

			rec := evidence.NewRecord(dir, kind, verdict, base, lockHash, viewHash, findings, time.Now().UTC())
			written, err := evidence.WriteRecord(dir, evCfg, rec, time.Now().UTC())
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"record":  rec,
					"written": written,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Recorded evidence %s (%s: %s) to %v\n", rec.ID, rec.Kind, rec.Verdict, written)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&kind, "kind", "check", "Evidence kind (verify|check|review|semantic-conflicts|exception-use)")
	cmd.Flags().StringVar(&verdict, "verdict", "pass", "Evidence verdict (pass|fail|error)")
	cmd.Flags().StringVar(&base, "base", "", "Base git ref if applicable")
	cmd.Flags().StringVar(&findingsFile, "findings", "", "Optional JSON file containing findings array")
	return cmd
}

func newEvidencePruneCmd(jsonOutput *bool) *cobra.Command {
	var dir, before string
	var all bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove local evidence records under .gradus/evidence/",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if !all && before == "" {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("specify --all or --before YYYY-MM-DD")}
			}
			if before != "" {
				if _, err := time.Parse("2006-01-02", before); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("invalid --before date %q (expected YYYY-MM-DD)", before)}
				}
			}

			evDir := filepath.Join(dir, ".gradus", "evidence")
			entries, err := os.ReadDir(evDir)
			if err != nil && !os.IsNotExist(err) {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			removed, remaining := 0, 0
			for _, e := range entries {
				if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
					continue
				}
				full := filepath.Join(evDir, e.Name())
				shouldDelete := all
				if !shouldDelete && before != "" {
					if raw, rerr := os.ReadFile(full); rerr == nil {
						var doc struct {
							Timestamp string `json:"timestamp"`
						}
						if json.Unmarshal(raw, &doc) == nil && len(doc.Timestamp) >= 10 && doc.Timestamp[:10] < before {
							shouldDelete = true
						}
					}
				}
				if shouldDelete {
					if rerr := os.Remove(full); rerr != nil {
						return &ExitErrorWithCode{Code: ExitError, Err: rerr}
					}
					removed++
				} else {
					remaining++
				}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]int{"removed": removed, "remaining": remaining})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Pruned %d evidence record(s) (%d remaining)\n", removed, remaining)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&before, "before", "", "Delete records with timestamp before YYYY-MM-DD")
	cmd.Flags().BoolVar(&all, "all", false, "Delete all local evidence records")
	return cmd
}

func readLockHashes(dir string) (lockHash string, viewHash string) {
	lockPath := filepath.Join(dir, "gradus.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return "", ""
	}
	lockHash = lock.HashBytes(data)
	if lf, err := lock.ReadLockfile(lockPath); err == nil {
		viewHash = lf.ViewHash
	}
	return lockHash, viewHash
}
