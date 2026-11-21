package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/schema"
)

type explainResult struct {
	Rule         schema.Rule     `json:"rule"`
	Charter      *schema.Charter `json:"charter,omitempty"`
	OverriddenBy []string        `json:"overridden_by,omitempty"`
	Notes        []string        `json:"notes,omitempty"`
}

func newExplainCmd(jsonOutput *bool) *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "explain <rule-id>",
		Short: "Display origin, level, enforcement, rationale, and examples for a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			ruleID := args[0]

			pyramid, err := schema.LoadLocalPyramid(dir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			var found *schema.Rule
			var overriddenBy []string
			for i := range pyramid.Rules {
				r := &pyramid.Rules[i]
				if r.ID == ruleID {
					found = r
				}
				if r.Overrides == ruleID {
					overriddenBy = append(overriddenBy, r.ID)
				}
			}

			if found == nil {
				return &ExitErrorWithCode{
					Code: ExitError,
					Err:  fmt.Errorf("rule %q not found in local pyramid", ruleID),
				}
			}

			var levelCharter *schema.Charter
			if lc, ok := pyramid.Levels[found.Level]; ok {
				levelCharter = lc.Charter
			}

			res := explainResult{
				Rule:         *found,
				Charter:      levelCharter,
				OverriddenBy: overriddenBy,
				Notes:        pyramid.Notes,
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s: %s\n", found.ID, found.Title)
			fmt.Fprintf(out, "Level:       %s\n", found.Level)
			if found.Source != "" {
				fmt.Fprintf(out, "Source:      %s\n", found.Source)
			} else {
				fmt.Fprintf(out, "Source:      %s\n", found.ProvenanceSource)
			}
			fmt.Fprintf(out, "Severity:    %s\n", found.Severity)
			fmt.Fprintf(out, "Enforcement: %s\n", found.Enforcement)
			fmt.Fprintf(out, "Scope:       %s\n", strings.Join(found.Scope, ", "))
			if found.Overrides != "" {
				fmt.Fprintf(out, "Overrides:   %s\n", found.Overrides)
			}
			if len(overriddenBy) > 0 {
				fmt.Fprintf(out, "Overridden by: %s\n", strings.Join(overriddenBy, ", "))
			}
			if levelCharter != nil && levelCharter.Purpose != "" {
				fmt.Fprintf(out, "Charter:     %s\n", levelCharter.Purpose)
			}
			fmt.Fprintf(out, "\nStatement:\n  %s\n", found.Statement)
			if found.Why != "" {
				fmt.Fprintf(out, "\nWhy:\n  %s\n", found.Why)
			}
			if len(found.Rationalizations) > 0 {
				fmt.Fprintf(out, "\nRejected Rationalizations:\n")
				for _, rat := range found.Rationalizations {
					fmt.Fprintf(out, "  - %s\n", rat)
				}
			}
			if len(found.Examples.Compliant) > 0 {
				fmt.Fprintf(out, "\nCompliant Examples:\n")
				for _, ex := range found.Examples.Compliant {
					fmt.Fprintf(out, "%s\n", ex)
				}
			}
			if len(found.Examples.NonCompliant) > 0 {
				fmt.Fprintf(out, "\nNon-compliant Examples:\n")
				for _, ex := range found.Examples.NonCompliant {
					fmt.Fprintf(out, "%s\n", ex)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	return cmd
}
