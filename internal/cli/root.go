package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// ExitErrorWithCode carries an explicit CLI exit code alongside an error message.
type ExitErrorWithCode struct {
	Code int
	Err  error
}

func (e *ExitErrorWithCode) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitErrorWithCode) Unwrap() error {
	return e.Err
}

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
	rootCmd.AddCommand(newInitCmd(&jsonOutput))
	rootCmd.AddCommand(newExplainCmd(&jsonOutput))
	rootCmd.AddCommand(newResolveCmd(&jsonOutput))
	rootCmd.AddCommand(newExceptionCmd(&jsonOutput))
	rootCmd.AddCommand(newStatusCmd(&jsonOutput))
	rootCmd.AddCommand(newDoctorCmd(&jsonOutput))
	rootCmd.AddCommand(newTestCmd(&jsonOutput))
	rootCmd.AddCommand(newSimulateCmd(&jsonOutput))
	rootCmd.AddCommand(newHandbookCmd(&jsonOutput))
	rootCmd.AddCommand(newCompileCmd(&jsonOutput))
	rootCmd.AddCommand(newEvidenceCmd(&jsonOutput))
	rootCmd.AddCommand(newCheckCmd(&jsonOutput))
	rootCmd.AddCommand(newVerifyCmd(&jsonOutput))
	rootCmd.AddCommand(newHooksCmd(&jsonOutput))
	rootCmd.AddCommand(newGuardCmd(&jsonOutput))
	rootCmd.AddCommand(newConflictsCmd(&jsonOutput))
	rootCmd.AddCommand(newLessonsCmd(&jsonOutput))
	rootCmd.AddCommand(newReconcileCmd(&jsonOutput))

	return rootCmd
}

// ExecuteCode runs the CLI with the given arguments and streams, returning the exit code (0, 1, or 2).
func ExecuteCode(args []string, stdout, stderr io.Writer) int {
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		var coded *ExitErrorWithCode
		if errors.As(err, &coded) {
			if coded.Err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", coded.Err)
			}
			return coded.Code
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}
	return ExitPass
}

// Execute runs the root command and exits with the standard exit code.
func Execute() {
	code := ExecuteCode(os.Args[1:], os.Stdout, os.Stderr)
	if code != ExitPass {
		os.Exit(code)
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
