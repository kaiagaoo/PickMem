package cli

import (
	"fmt"

	"github.com/kaiagaoo/PickMem/internal/picker"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the vault and what the model currently sees",
		Long: `One-screen summary of the vault: where it is, how many notes it holds,
and — the part that matters — the current active selection: which lens (if
any), how many items, and a rough token estimate. Doubles as a diagnostic:
if an MCP client seems to read the wrong memory, compare its output with
this. Print the full assembled block with ` + "`pickmem context`" + `.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			active := s.ListActive()
			pending := s.ListPending()
			groups := s.Groups()
			lenses, err := s.LoadLenses()
			if err != nil {
				return err
			}
			sel, err := s.LoadActive()
			if err != nil {
				return err
			}

			// Resolve the selection against live notes; count and estimate
			// tokens the same way the picker footer does.
			var bodies []string
			live := 0
			for _, id := range sel.ItemIDs {
				if n, ok := s.Get(id); ok {
					bodies = append(bodies, n.Body)
					live++
				}
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Vault:     %s\n", s.Root)
			fmt.Fprintf(out, "Notes:     %d active · %d pending in inbox · %d groups\n", len(active), len(pending), len(groups))
			fmt.Fprintf(out, "Lenses:    %d\n", len(lenses))

			label := "custom"
			if sel.ActiveLens != "" {
				label = sel.ActiveLens + " (lens)"
			}
			if live == 0 {
				fmt.Fprintf(out, "Selection: none — the model sees nothing (run `pickmem pick`)\n")
			} else {
				fmt.Fprintf(out, "Selection: %s · %d items · ~%d tokens\n", label, live, picker.EstimateTokens(bodies))
			}
			if dangling := len(sel.ItemIDs) - live; dangling > 0 {
				fmt.Fprintf(out, "           (%d selected id(s) point at deleted notes and are ignored)\n", dangling)
			}
			if len(pending) > 0 {
				fmt.Fprintf(out, "\nInbox has %d pending item(s) — review with `pickmem review`.\n", len(pending))
			}
			return nil
		},
	}
}
