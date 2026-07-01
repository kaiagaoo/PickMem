package cli

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	pmmcp "github.com/qwgao/pickmem/internal/mcp"
	"github.com/qwgao/pickmem/internal/vault"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server over stdio",
		Long: `Run the PickMem MCP server. Reads/writes JSON-RPC on stdin/stdout, so it's typically launched by a client (Claude Desktop, Cursor, Cline) rather than run interactively.

Exposes:
  - resource pickmem://active       the currently-picked memory block
  - tool     get_active_memory      same block via a tool call
  - tool     list_lenses            saved lens names + item counts
  - tool     use_lens(name)         switch active selection to a lens
  - tool     propose_memories(text) stage candidates to inbox (no activate)

Use ` + "`pickmem install <client>`" + ` to wire this into a client's config.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}
			srv := pmmcp.NewServer(s, Version)
			// Signals are handled by the SDK's stdio transport when the
			// parent closes its end of the pipe.
			return srv.Run(context.Background(), &sdkmcp.StdioTransport{})
		},
		// stdio needs a clean stdout — cobra prints usage to stderr on error
		// but we still want to suppress the auto-usage dump on runtime
		// failures so we don't corrupt the JSON-RPC stream if a client
		// somehow keeps reading past the exit.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}
