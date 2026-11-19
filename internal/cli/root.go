package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/version"
)

// Exit codes following gradus-design.md D-73.
const (
	ExitPass      = 0
	ExitViolation = 1
	ExitError     = 2
)

// NewRootCmd builds the root cobra command for gradusctl.
func NewRootCmd() *cobra.Command {
	var jsonOutput bool

	rootCmd := &cobra.Command{
		Use:   "gradusctl",
		Short: "gradusctl resolves, verifies, and compiles layered rules for AI coding agents",
		Long: `Gradus organizes rules for AI coding agents into a governed pyramid of levels.
gradusctl deterministically resolves sources, enforces the restriction invariant,
guards actions, verifies lockfiles, and compiles target agent instructions.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Emit structured machine-readable JSON")

	rootCmd.AddCommand(newVersionCmd(&jsonOutput))

	return rootCmd
}

// Execute runs the root command and exits with the standard exit code.
func Execute() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
		os.Exit(ExitError)
	}
}

func newVersionCmd(jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print gradusctl version and build metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			if *jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "gradusctl %s (commit: %s, built: %s)\n", info.Version, info.Commit, info.Date)
			return err
		},
	}
}
