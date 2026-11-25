package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/mcp"
)

func newServeCmd() *cobra.Command {
	var dir string
	var useMCP, offline bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the resolved Gradus pyramid over a protocol interface (e.g. --mcp)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !useMCP {
				return &ExitErrorWithCode{
					Code: ExitError,
					Err:  fmt.Errorf("missing required mode flag: specify --mcp to start the MCP stdio server"),
				}
			}
			if dir == "" {
				dir = "."
			}
			srv := &mcp.Server{
				RootDir: dir,
				Offline: offline,
			}
			if err := srv.ServeStdio(cmd.InOrStdin(), cmd.OutOrStdout()); err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&useMCP, "mcp", false, "Run the MCP JSON-RPC 2.0 server over stdin/stdout")
	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().BoolVar(&offline, "offline", true, "Resolve remote levels from local cache only")
	return cmd
}
