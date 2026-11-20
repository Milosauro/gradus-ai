package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/hooks"
	"github.com/Milosauro/gradus-ai/internal/schema"
)

func newHooksCmd(jsonOutput *bool) *cobra.Command {
	var dir, manager string
	var allowNative bool

	cmd := &cobra.Command{
		Use:   "hooks [print|install]",
		Short: "Detect git hook manager and print or install Gradus pre-commit hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			action := "print"
			if len(args) == 1 {
				action = args[0]
			}
			if action != "print" && action != "install" {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("unknown hooks action %q (expected print or install)", action)}
			}

			configured := manager
			if configured == "" {
				configured = "auto"
				if cfgBytes, err := os.ReadFile(filepath.Join(dir, "gradus.yaml")); err == nil {
					if parsed, parseErr := schema.ParsePyramidConfig(cfgBytes); parseErr == nil && parsed.Value.Hooks != nil && parsed.Value.Hooks.Manager != "" {
						configured = parsed.Value.Hooks.Manager
					}
				}
			}

			detected := hooks.DetectManager(dir, configured)
			snippet, err := hooks.RenderSnippet(detected)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			var updatedFile string
			if action == "install" {
				updatedFile, err = hooks.InstallHooks(dir, detected, allowNative)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
			}

			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"manager":      detected,
					"action":       action,
					"snippet":      snippet,
					"updated_file": updatedFile,
				})
			}

			out := cmd.OutOrStdout()
			if action == "print" {
				fmt.Fprintf(out, "# Detected git hook manager: %s\n%s", detected, snippet)
			} else {
				fmt.Fprintf(out, "Installed %s hooks in %s\n", detected, updatedFile)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&manager, "manager", "", "Override hook manager (auto|pre-commit|lefthook|husky|native|none)")
	cmd.Flags().BoolVar(&allowNative, "native", false, "Confirm modifying git core.hooksPath when using native hooks")
	return cmd
}
