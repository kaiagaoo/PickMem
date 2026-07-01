package cli

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags (defaults to dev).
var Version = "dev"

// NewRoot builds the root cobra command with all subcommands registered.
// Kept as a constructor (not a global) so tests can build a fresh tree.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "pickmem",
		Short:         "PickMem — a memory-curation layer for LLMs",
		Long:          "PickMem is a local-first tool that lets you pick which memory items reach the model, per session. Your brain lives in an Obsidian vault; you choose the slice.",
		SilenceUsage:  true, // don't dump usage on runtime errors
		SilenceErrors: true, // cobra's own error print is redundant with fmt.Fprintln in main
		Version:       Version,
	}
	// Every non-init subcommand accepts --vault as a persistent override.
	root.PersistentFlags().String("vault", "", "path to vault (overrides $PICKMEM_VAULT and user config)")

	root.AddCommand(
		newInitCmd(),
		newAddCmd(),
		newListCmd(),
		newShowCmd(),
		newEditCmd(),
		newRmCmd(),
		newPickCmd(),
	)
	return root
}

// vaultFlag pulls --vault off any command, deferring to Resolve for env +
// user-config fallback.
func vaultFlag(cmd *cobra.Command) (string, error) {
	flagVal, _ := cmd.Flags().GetString("vault")
	return ResolveVaultPath(flagVal)
}
