package cli

import (
	"fmt"

	"github.com/qwgao/pickmem/internal/vault"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id-or-suffix>",
		Short: "Open a note in $EDITOR",
		Long:  "Launches $EDITOR (or vi) on the note file. PickMem does not rewrite bytes itself, preserving the create-only invariant — you are the one making the change.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}
			n, err := resolveNote(s, args[0])
			if err != nil {
				return err
			}
			full, ok := s.AbsPath(n.ID)
			if !ok {
				return fmt.Errorf("note %s has no path", n.ID)
			}
			editor := editorEnv()
			if err := launch(editor, full); err != nil {
				return fmt.Errorf("editor: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Edited: %s\n", n.RelPath)
			return nil
		},
	}
}

// editorEnv resolves $EDITOR with a vi fallback.
func editorEnv() string {
	if v := getenvNonEmpty("VISUAL"); v != "" {
		return v
	}
	if v := getenvNonEmpty("EDITOR"); v != "" {
		return v
	}
	return "vi"
}

func getenvNonEmpty(name string) string {
	return osGetenv(name)
}
