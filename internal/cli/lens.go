package cli

import (
	"fmt"

	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

func newLensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lens",
		Short: "Manage saved lenses (named selections)",
		Long: `A lens is a named, saved selection of memory items. Create one from the
picker (` + "`pickmem pick`" + `, key s) or the extension; these subcommands make
switching scriptable — e.g. alias ` + "`pickmem lens use Job-Hunt`" + ` per task.`,
	}
	cmd.AddCommand(newLensListCmd(), newLensUseCmd(), newLensRmCmd())
	return cmd
}

func newLensListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved lenses",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			lenses, err := s.LoadLenses()
			if err != nil {
				return err
			}
			if len(lenses) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No lenses yet. Save one from the picker: `pickmem pick`, then press s.")
				return nil
			}
			active, err := s.LoadActive()
			if err != nil {
				return err
			}
			for _, l := range lenses {
				marker := "  "
				if l.Name == active.ActiveLens {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%-24s %d items\n", marker, l.Name, len(l.ItemIDs))
			}
			return nil
		},
	}
}

func newLensUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Activate a lens (replaces the current selection)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			lenses, err := s.LoadLenses()
			if err != nil {
				return err
			}
			lens, ok := vault.FindLens(lenses, args[0])
			if !ok {
				return fmt.Errorf("lens %q not found (see `pickmem lens list`)", args[0])
			}
			// Drop ids of since-deleted notes — same behavior as the TUI
			// and the MCP use_lens tool.
			live := make([]string, 0, len(lens.ItemIDs))
			for _, id := range lens.ItemIDs {
				if _, ok := s.Get(id); ok {
					live = append(live, id)
				}
			}
			if err := s.SaveActive(vault.Active{ActiveLens: lens.Name, ItemIDs: live}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Active: %s · %d items\n", lens.Name, len(live))
			if dropped := len(lens.ItemIDs) - len(live); dropped > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "(%d item(s) in this lens point at deleted notes and were skipped)\n", dropped)
			}
			return nil
		},
	}
}

func newLensRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a saved lens (notes are untouched)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			lenses, err := s.LoadLenses()
			if err != nil {
				return err
			}
			if _, ok := vault.FindLens(lenses, args[0]); !ok {
				return fmt.Errorf("lens %q not found (see `pickmem lens list`)", args[0])
			}
			if err := s.SaveLenses(vault.RemoveLens(lenses, args[0])); err != nil {
				return err
			}
			// If the deleted lens was active, the selection stays as-is but
			// is no longer "from a lens" — it's just a custom pick now.
			active, err := s.LoadActive()
			if err != nil {
				return err
			}
			if active.ActiveLens == args[0] {
				active.ActiveLens = ""
				if err := s.SaveActive(active); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted lens %q. Notes are untouched.\n", args[0])
			return nil
		},
	}
}
