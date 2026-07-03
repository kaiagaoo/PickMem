package cli

import (
	"fmt"

	"github.com/qwgao/pickmem/internal/ingest"
	"github.com/qwgao/pickmem/internal/vault"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import a batch of memories from an export or a list",
		Long: `Read a file and stage each memory as a pending inbox note. Nothing goes
active — accept from ` + "`pickmem review`" + ` afterward.

The file is parsed by shape — JSON, a bulleted list, or blank-line
paragraphs (auto-detected; override with --format) — and each chunk is
routed with the vault's keyword rules. Everything is de-duplicated against
what's already in the vault.`,
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

			f := ingest.ResolveFormat(format)
			result, err := ingest.ImportFile(cmd.Context(), s, args[0], f)
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
	cmd.Flags().StringVar(&format, "format", "auto", "auto | json | bullets | paragraphs")
	return cmd
}
