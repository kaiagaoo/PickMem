package cli

import (
	"errors"
	"fmt"

	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rm <id-or-suffix>",
		Short: "Delete a memory note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			n, err := resolveNote(s, args[0])
			if err != nil {
				return err
			}
			if !yes {
				fmt.Fprintf(cmd.OutOrStdout(), "About to delete %s (%s).\nRun again with --yes to confirm.\n", n.Label, n.RelPath)
				return errors.New("aborted (missing --yes)")
			}
			if err := s.Remove(n.ID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed: %s\n", n.RelPath)
			// Also scrub from any lens or active selection that referenced it,
			// so we don't leave stale ids pointing at nothing.
			return sweepReferences(s, n.ID)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "confirm the deletion")
	return cmd
}

// sweepReferences removes a deleted note's id from lenses.json and
// active.json so dangling references don't accumulate.
func sweepReferences(s *vault.Store, id string) error {
	ls, err := s.LoadLenses()
	if err != nil {
		return err
	}
	changed := false
	for i, l := range ls {
		before := len(l.ItemIDs)
		l.ItemIDs = without(l.ItemIDs, id)
		if len(l.ItemIDs) != before {
			ls[i] = l
			changed = true
		}
	}
	if changed {
		if err := s.SaveLenses(ls); err != nil {
			return err
		}
	}
	a, err := s.LoadActive()
	if err != nil {
		return err
	}
	before := len(a.ItemIDs)
	a.ItemIDs = without(a.ItemIDs, id)
	if len(a.ItemIDs) != before {
		return s.SaveActive(a)
	}
	return nil
}

func without(ids []string, drop string) []string {
	out := ids[:0]
	for _, id := range ids {
		if id != drop {
			out = append(out, id)
		}
	}
	return out
}
