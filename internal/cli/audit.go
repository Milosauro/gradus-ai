package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/audit"
	"github.com/Milosauro/gradus-ai/internal/resolve"
)

func newAuditCmd(jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Report framework control coverage and export structured compliance audit bundles",
	}
	cmd.AddCommand(newAuditCoverageCmd(jsonOutput))
	cmd.AddCommand(newAuditExportCmd(jsonOutput))
	return cmd
}

func newAuditCoverageCmd(jsonOutput *bool) *cobra.Command {
	var dir, framework string
	var offline bool

	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Show compliance framework control coverage across active rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			res, err := resolve.Resolve(dir, resolve.Options{OfflineOnly: offline})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			bundle, err := audit.BuildBundle(dir, res.View, audit.Options{Framework: framework})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"view_hash":      bundle.ViewHash,
					"frameworks":     bundle.Frameworks,
					"unmapped_rules": bundle.UnmappedRules,
				})
			}
			if len(bundle.Frameworks) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No framework controls mapped.")
			} else {
				for _, fw := range bundle.Frameworks {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: %d control(s), %d rule(s)\n", fw.Framework, fw.ControlCount, fw.RuleCount)
					for _, c := range fw.Controls {
						ids := make([]string, 0, len(c.Rules))
						for _, r := range c.Rules {
							ids = append(ids, r.RuleID)
						}
						fmt.Fprintf(cmd.OutOrStdout(), "  - %s: %s\n", c.Control, strings.Join(ids, ", "))
					}
				}
			}
			if len(bundle.UnmappedRules) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Unmapped active rules (%d): %s\n", len(bundle.UnmappedRules), strings.Join(bundle.UnmappedRules, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&framework, "framework", "", "Filter coverage to a specific framework prefix (e.g. ISO27001)")
	cmd.Flags().BoolVar(&offline, "offline", true, "Use cached remote levels only")
	return cmd
}

func newAuditExportCmd(jsonOutput *bool) *cobra.Command {
	var dir, framework, format, outputPath string
	var offline bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a self-contained JSON or Markdown audit bundle (rules, controls, exceptions, evidence)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			effectiveFormat := strings.ToLower(strings.TrimSpace(format))
			if effectiveFormat == "" {
				if *jsonOutput {
					effectiveFormat = "json"
				} else {
					effectiveFormat = "markdown"
				}
			}
			if effectiveFormat != "json" && effectiveFormat != "markdown" {
				return &ExitErrorWithCode{
					Code: ExitError,
					Err:  fmt.Errorf("unsupported --format %q (expected json or markdown)", format),
				}
			}

			res, err := resolve.Resolve(dir, resolve.Options{OfflineOnly: offline})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			bundle, err := audit.BuildBundle(dir, res.View, audit.Options{Framework: framework})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			var rendered []byte
			if effectiveFormat == "json" {
				rendered, err = json.MarshalIndent(bundle, "", "  ")
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				rendered = append(rendered, '\n')
			} else {
				rendered = []byte(audit.RenderMarkdown(bundle))
			}

			if strings.TrimSpace(outputPath) != "" {
				target := outputPath
				if !filepath.IsAbs(target) {
					target = filepath.Join(dir, target)
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if err := os.WriteFile(target, rendered, 0o644); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if *jsonOutput {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					return enc.Encode(map[string]any{
						"output":     target,
						"format":     effectiveFormat,
						"view_hash":  bundle.ViewHash,
						"frameworks": len(bundle.Frameworks),
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s audit export to %s\n", effectiveFormat, target)
				return nil
			}

			_, err = cmd.OutOrStdout().Write(rendered)
			return err
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&framework, "framework", "", "Filter export to a specific framework prefix")
	cmd.Flags().StringVar(&format, "format", "", "Output format: json or markdown (defaults to json when --json is set, else markdown)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Optional output file path")
	cmd.Flags().BoolVar(&offline, "offline", true, "Use cached remote levels only")
	return cmd
}
