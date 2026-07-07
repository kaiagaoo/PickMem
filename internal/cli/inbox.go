package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Manage pending inbox items",
	}
	cmd.AddCommand(newInboxClearCmd())
	return cmd
}

func newInboxClearCmd() *cobra.Command {
	var yes bool
	var source string

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete pending inbox items in bulk (e.g. undo an import)",
		Long: `Delete pending inbox items without opening the review TUI. Only pending
items are eligible — active notes can never be touched by this command.

Scope with --source to undo one pipeline's staging:
  pickmem inbox clear --source import --yes    # undo a pickmem import
  pickmem inbox clear --source extract --yes   # drop model-staged items
  pickmem inbox clear --yes                    # everything pending`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			pending := s.ListPending()
			targets := pending[:0]
			for _, n := range pending {
				if source == "" || n.Source == source {
					targets = append(targets, n)
				}
			}
			if len(targets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Nothing to clear.")
				return nil
			}
			if !yes {
				scope := "all pending items"
				if source != "" {
					scope = fmt.Sprintf("pending items with source=%s", source)
				}
				return fmt.Errorf("would delete %d inbox item(s) (%s) — re-run with --yes to confirm", len(targets), scope)
			}
			cleared := 0
			for _, n := range targets {
				if err := s.RejectInbox(n.ID); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "clear %s: %v\n", n.ID, err)
					continue
				}
				cleared++
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cleared %d inbox item(s). Active notes are untouched.\n", cleared)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm the deletion")
	cmd.Flags().StringVar(&source, "source", "", "only clear items with this source: import | extract | manual")
	return cmd
}
