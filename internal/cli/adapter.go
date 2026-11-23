package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/adapters"
	"github.com/Milosauro/gradus-ai/internal/adapter"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

type adapterListEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Source     string `json:"source"`
	TargetFile string `json:"target_file"`
	Bound      bool   `json:"bound"`
}

func newAdapterCmd(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adapter",
		Short: "List, synchronize, and verify declarative provider adapters",
	}

	cmd.AddCommand(newAdapterListCmd(jsonOutput))
	cmd.AddCommand(newAdapterSyncCmd(jsonOutput))
	cmd.AddCommand(newAdapterVerifyCmd(jsonOutput))
	return cmd
}

func newAdapterListCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available official and custom adapters",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			reg, err := adapter.LoadRegistry(dir, adapters.FS)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			res, err := resolve.Resolve(dir, resolve.Options{OfflineOnly: true})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			bound := make(map[string]bool)
			for _, b := range res.View.Bindings {
				bound[b.Provider] = true
			}

			ids := make([]string, 0, len(reg))
			for id := range reg {
				ids = append(ids, id)
			}
			sort.Strings(ids)

			entries := make([]adapterListEntry, 0, len(ids))
			for _, id := range ids {
				m := reg[id]
				entries = append(entries, adapterListEntry{
					ID:         m.ID,
					Name:       m.Name,
					Source:     m.Source,
					TargetFile: m.TargetFile,
					Bound:      bound[m.ID],
				})
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{"adapters": entries})
			}

			out := cmd.OutOrStdout()
			for _, e := range entries {
				fmt.Fprintf(out, "%s\t%s\t%s\tbound=%t\n", e.ID, e.Source, e.TargetFile, e.Bound)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}

func newAdapterSyncCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var provider string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Inject resolved Gradus rules into bound provider constitution files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			reg, err := adapter.LoadRegistry(dir, adapters.FS)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			res, err := resolve.Resolve(dir, resolve.Options{})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			targets, _, err := selectAdapterTargets(reg, res.View, provider)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			results := make([]adapter.SyncResult, 0, len(targets))
			for _, id := range targets {
				sr, err := adapter.Sync(dir, reg[id], res.View)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				results = append(results, sr)
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{"synced": results})
			}

			out := cmd.OutOrStdout()
			if len(results) == 0 {
				fmt.Fprintln(out, "No bound adapters to sync.")
				return nil
			}
			for _, sr := range results {
				state := "unchanged"
				if sr.Changed {
					state = "updated"
				}
				fmt.Fprintf(out, "%s: %s (%s)\n", sr.Provider, sr.TargetFile, state)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&provider, "provider", "", "Specific adapter provider ID to sync")
	return cmd
}

func newAdapterVerifyCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var provider string
	var skipEvidence bool

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify bound provider files for drift, CLI version constraints, and evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			reg, err := adapter.LoadRegistry(dir, adapters.FS)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			res, err := resolve.Resolve(dir, resolve.Options{})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			targets, reqsByProvider, err := selectAdapterTargets(reg, res.View, provider)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			results := make([]adapter.VerifyResult, 0, len(targets))
			verdict := "pass"
			failedCount := 0
			for _, id := range targets {
				vr := adapter.Verify(dir, reg[id], res.View, reqsByProvider[id], skipEvidence)
				if vr.Status != "pass" {
					verdict = "fail"
					failedCount++
				}
				results = append(results, vr)
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(map[string]any{
					"verdict": verdict,
					"results": results,
				}); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			} else {
				out := cmd.OutOrStdout()
				if len(results) == 0 {
					fmt.Fprintln(out, "adapter verify passed (no bound adapters)")
				}
				for _, vr := range results {
					fmt.Fprintf(out, "%s (%s): %s\n", vr.Provider, vr.TargetFile, vr.Status)
					for _, issue := range vr.Issues {
						fmt.Fprintf(out, "  - %s\n", issue)
					}
				}
			}

			if failedCount > 0 {
				if *jsonOutput {
					return &ExitErrorWithCode{Code: ExitViolation}
				}
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("adapter verification failed for %d provider(s)", failedCount),
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&provider, "provider", "", "Specific adapter provider ID to verify")
	cmd.Flags().BoolVar(&skipEvidence, "skip-evidence", false, "Skip running provider evidence_command checks")
	return cmd
}

func selectAdapterTargets(reg map[string]adapter.Manifest, view resolve.ResolvedView, explicitProvider string) ([]string, map[string]map[string]string, error) {
	reqsByProvider := make(map[string]map[string]string)
	boundSet := make(map[string]bool)

	for _, b := range view.Bindings {
		boundSet[b.Provider] = true
		if len(b.Requires) > 0 {
			if reqsByProvider[b.Provider] == nil {
				reqsByProvider[b.Provider] = make(map[string]string)
			}
			for k, v := range b.Requires {
				reqsByProvider[b.Provider][k] = v
			}
		}
	}

	if p := strings.TrimSpace(explicitProvider); p != "" {
		if _, ok := reg[p]; !ok {
			return nil, nil, fmt.Errorf("unknown adapter provider %q", p)
		}
		return []string{p}, reqsByProvider, nil
	}

	targets := make([]string, 0, len(boundSet))
	for p := range boundSet {
		if _, ok := reg[p]; !ok {
			return nil, nil, fmt.Errorf("bound provider %q has no registered adapter", p)
		}
		targets = append(targets, p)
	}
	sort.Strings(targets)
	return targets, reqsByProvider, nil
}
