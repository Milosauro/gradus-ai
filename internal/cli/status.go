package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Milosauro/gradus-ai/internal/lock"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/internal/state"
)

type statusReport struct {
	LockState      string   `json:"lock_state"` // ok | missing | invalid | mismatch
	DefinitionHash string   `json:"definition_hash,omitempty"`
	ViewHash       string   `json:"view_hash,omitempty"`
	Conflicts      int      `json:"conflicts"`
	PendingLessons int      `json:"pending_lessons"`
	Drift          int      `json:"drift"`
	UserRules      []string `json:"user_rules,omitempty"`
}

func newStatusCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var userDir string
	var brief bool
	var antigravityHook bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a fast local summary of lock state, conflicts, drift, and pending lessons",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			if antigravityHook {
				// Antigravity executes hooks with CWD set to the directory containing hooks.json (.agents/).
				if _, err := os.Stat(filepath.Join(dir, "gradus.yaml")); os.IsNotExist(err) {
					if abs, aerr := filepath.Abs(dir); aerr == nil {
						parent := filepath.Dir(abs)
						if _, perr := os.Stat(filepath.Join(parent, "gradus.yaml")); perr == nil {
							dir = parent
						}
					}
				}
			}
			rep, exitCode, err := buildStatusReport(dir, userDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if antigravityHook {
				msg := fmt.Sprintf("gradus: lock=%s conflicts=%d drift=%d lessons=%d",
					rep.LockState, rep.Conflicts, rep.Drift, rep.PendingLessons)
				envelope := map[string]any{
					"injectSteps": []any{
						map[string]any{
							"systemMessage": map[string]any{
								"systemMessage": msg,
							},
						},
					},
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				if err := enc.Encode(envelope); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				return nil
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(rep); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else if brief {
				fmt.Fprintf(cmd.OutOrStdout(), "gradus: lock=%s conflicts=%d drift=%d lessons=%d\n",
					rep.LockState, rep.Conflicts, rep.Drift, rep.PendingLessons)
			} else {
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "Lock state:      %s\n", rep.LockState)
				if rep.ViewHash != "" {
					fmt.Fprintf(out, "View hash:       %s\n", rep.ViewHash)
				}
				fmt.Fprintf(out, "Conflicts:       %d\n", rep.Conflicts)
				fmt.Fprintf(out, "Drift items:     %d\n", rep.Drift)
				fmt.Fprintf(out, "Pending lessons: %d\n", rep.PendingLessons)
			}

			if exitCode != ExitPass {
				return &ExitErrorWithCode{
					Code: exitCode,
					Err:  fmt.Errorf("status check reported lock=%s conflicts=%d", rep.LockState, rep.Conflicts),
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&userDir, "user-dir", "", "Override user-level rules directory (defaults to ~/.gradus/levels/user)")
	cmd.Flags().BoolVar(&brief, "brief", false, "Emit compact single-line summary")
	cmd.Flags().BoolVar(&antigravityHook, "antigravity-hook", false, "Emit Antigravity SessionStart injectSteps JSON")
	return cmd
}

// buildStatusReport performs zero-network local inspection (< 100ms).
func buildStatusReport(rootDir, userDir string) (statusReport, int, error) {
	var rep statusReport

	cfgBytes, err := os.ReadFile(filepath.Join(rootDir, "gradus.yaml"))
	if err != nil {
		return rep, ExitError, fmt.Errorf("failed to read gradus.yaml: %w", err)
	}
	cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
	if err != nil {
		return rep, ExitError, err
	}

	lockPath := filepath.Join(rootDir, "gradus.lock")
	lf, lockErr := lock.ReadLockfile(lockPath)
	if lockErr != nil {
		if os.IsNotExist(lockErr) {
			rep.LockState = "missing"
		} else {
			rep.LockState = "invalid"
			return rep, ExitError, lockErr
		}
	} else {
		rep.LockState = "ok"
		rep.DefinitionHash = lf.DefinitionHash
		rep.ViewHash = lf.ViewHash

		// Fast local lock check for local locked levels (zero network I/O).
		for _, lvl := range cfgRes.Value.Levels {
			if !lvl.Locked || lvl.Source.Path == "" || strings.HasPrefix(lvl.Source.Path, "~/") {
				continue
			}
			entry, ok := lf.FindEntry("level", lvl.ID)
			if !ok {
				rep.LockState = "mismatch"
				break
			}
			dirPath := lvl.Source.Path
			if !filepath.IsAbs(dirPath) {
				dirPath = filepath.Join(rootDir, filepath.FromSlash(dirPath))
			}
			actualHash, err := lock.HashDirectory(dirPath)
			if err != nil || actualHash != entry.Hash {
				rep.LockState = "mismatch"
				break
			}
		}
	}

	rep.Conflicts = countConfirmedFindings(filepath.Join(rootDir, ".gradus", "findings"))
	if res, rerr := resolve.Resolve(rootDir, resolve.Options{OfflineOnly: true}); rerr == nil {
		rep.Conflicts = len(res.View.Conflicts)
	}
	rep.PendingLessons = countPendingLessons(rootDir)
	rep.UserRules = loadUserRuleIDs(userDir)

	if rep.LockState == "mismatch" {
		return rep, ExitError, nil
	}
	if rep.Conflicts > 0 || rep.Drift > 0 {
		return rep, ExitViolation, nil
	}
	return rep, ExitPass, nil
}

func countConfirmedFindings(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yaml") && !strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var f struct {
			Status string `yaml:"status"`
		}
		if yaml.Unmarshal(data, &f) == nil && f.Status == "confirmed" {
			count++
		}
	}
	return count
}

func countPendingLessons(rootDir string) int {
	lessonsDir, err := state.LessonsDir(rootDir, "")
	if err != nil {
		return 0
	}
	items, err := state.ListLessons(lessonsDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, l := range items {
		if l.Status == "open" || l.Status == "proposed" {
			count++
		}
	}
	return count
}

func loadUserRuleIDs(userDir string) []string {
	dir := strings.TrimSpace(userDir)
	if dir == "" {
		if os.Getenv("CI") != "" {
			return nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		dir = filepath.Join(home, ".gradus", "levels", "user")
	}
	rulesDir := filepath.Join(dir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		if parsed, err := schema.ParseRuleFile(filepath.Join(rulesDir, e.Name()), "user", "user"); err == nil {
			if parsed.Value.Status != "draft" && parsed.Value.Status != "retired" {
				ids = append(ids, parsed.Value.ID)
			}
		}
	}
	return ids
}
