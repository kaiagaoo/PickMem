package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaiagaoo/PickMem/internal/picker"
	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Bulk-review pending inbox notes",
		Long: `Open a TUI over the pending inbox: mark items for accept/reject, reassign groups in bulk, and apply the outcomes on confirm.

Keys:
  space      select at cursor
  a          accept selected (or cursor)
  A          accept every remaining item that has a suggested_group
  r          reject selected (or cursor)
  g          reassign group (overlay: type new, tab/↓ to browse existing)
  /          filter
  enter      apply the decisions
  q/esc      cancel — inbox unchanged

Accept moves the file from pickmem/inbox/ into its group folder and flips status to active.
Reject deletes the inbox file. Pending items (no decision) stay in the inbox for later.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}
			if len(s.ListPending()) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Inbox is empty. Try: pickmem import <file>")
				return nil
			}
			m, err := picker.NewReview(s)
			if err != nil {
				return err
			}
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.ErrOrStderr()))
			final, err := p.Run()
			if err != nil {
				return err
			}
			result := final.(picker.ReviewModel).Result
			if !result.Confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			accepted, rejected, skipped := 0, 0, 0
			for _, d := range result.Decisions {
				switch d.Outcome {
				case picker.OutcomeAccepted:
					if _, err := s.AcceptInbox(d.ID, d.Group); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "accept %s: %v\n", d.ID, err)
						continue
					}
					accepted++
				case picker.OutcomeRejected:
					if err := s.RejectInbox(d.ID); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "reject %s: %v\n", d.ID, err)
						continue
					}
					rejected++
				default:
					skipped++
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Accepted: %d  Rejected: %d  Left pending: %d\n", accepted, rejected, skipped)
			return nil
		},
	}
}
