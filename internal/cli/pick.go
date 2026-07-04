package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaiagaoo/PickMem/internal/picker"
	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

func newPickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pick",
		Short: "Full-screen picker: choose which memory items the model sees",
		Long: `Open the grouped multi-select picker. Space toggles, / filters, l opens the lens overlay, s saves the selection as a new lens, enter confirms (writes active.json), q cancels.

The default is nothing — the picker opens with your last active selection (or nothing on a fresh vault). Confirming with no items selected clears active.json, matching the "user decides relevance" invariant.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}
			if len(s.ListActive()) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No active notes yet. Add some with `pickmem add` first.")
				return nil
			}
			m, err := picker.New(s)
			if err != nil {
				return err
			}
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.ErrOrStderr()))
			final, err := p.Run()
			if err != nil {
				return err
			}
			result := final.(picker.Model).Result
			if !result.Confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			if err := s.SaveActive(vault.Active{
				ActiveLens: result.ActiveLens,
				ItemIDs:    result.ItemIDs,
			}); err != nil {
				return err
			}
			label := "custom"
			if result.ActiveLens != "" {
				label = result.ActiveLens
			}
			if len(result.ItemIDs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Active selection cleared.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Active: %s · %d items\n", label, len(result.ItemIDs))
			return nil
		},
	}
}
