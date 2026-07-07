package cli

import (
	"fmt"

	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

// openVault resolves the vault path (--vault → $PICKMEM_VAULT → user
// config), opens the store, and prints any per-file load warnings to
// stderr — the one place every command surfaces "a note was skipped"
// without failing.
func openVault(cmd *cobra.Command) (*vault.Store, error) {
	root, err := vaultFlag(cmd)
	if err != nil {
		return nil, err
	}
	s, err := vault.Open(root)
	if err != nil {
		return nil, err
	}
	for _, w := range s.Warnings() {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
	}
	return s, nil
}
