package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/qwgao/pickmem/internal/ingest"
	"github.com/qwgao/pickmem/internal/routing"
	"github.com/qwgao/pickmem/internal/vault"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var (
		format  string
		allowAI bool
		aiModel string
	)

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import memories from a ChatGPT/Claude export or a generic list",
		Long: `Parse a file of memories and stage each item as a pending inbox note.
Auto-detects JSON, bulleted lists, and paragraph-separated text; override with --format.

Nothing goes active — accept from ` + "`pickmem review`" + ` (or ` + "`pickmem pick`" + ` after moving items to a group) once the inbox is routed.

By default only the vault's routing rules run. With --allow-ai, an
Anthropic classifier proposes groups for items the rules miss, picking
only from the vault's existing taxonomy. Requires $ANTHROPIC_API_KEY.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}
			cfg, err := s.LoadConfig()
			if err != nil {
				return err
			}
			// Compose classifier chain: rules first, AI second when
			// consented AND a key is present. routing.NewAI returns nil
			// on an empty key so the chain silently degrades without a
			// branch.
			chain := []routing.Classifier{routing.NewRules(cfg)}
			if allowAI {
				key := os.Getenv("ANTHROPIC_API_KEY")
				if key == "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "warning: --allow-ai set but ANTHROPIC_API_KEY is empty; using rules only.")
				} else {
					ai := routing.NewAI(routing.AIConfig{APIKey: key, Model: aiModel})
					if ai != nil {
						chain = append(chain, ai)
					}
				}
			}
			router := routing.New(chain...)

			f := ingest.ResolveFormat(format)
			result, err := ingest.ImportFile(cmd.Context(), s, args[0], f, router)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Parsed:    %d\n", result.Parsed)
			fmt.Fprintf(out, "Staged:    %d\n", result.Staged)
			fmt.Fprintf(out, "Routed:    %d (with a suggested_group)\n", result.Routed)
			fmt.Fprintf(out, "Duplicate: %d (already in vault, skipped)\n", result.Duplicate)
			if result.Staged > 0 {
				fmt.Fprintln(out, "\nReview + accept with: pickmem review")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "auto", "auto | json | bullets | paragraphs (chatgpt/claude/list are aliases for auto)")
	cmd.Flags().BoolVar(&allowAI, "allow-ai", false, "add an Anthropic classifier to route items rules miss (requires $ANTHROPIC_API_KEY)")
	cmd.Flags().StringVar(&aiModel, "ai-model", "", "override Anthropic model id (default: a small, fast Claude)")
	return cmd
}

// Silence linter — context helper for future subcommands. Keeps the
// import shape consistent with the rest of the CLI.
var _ = context.Background
