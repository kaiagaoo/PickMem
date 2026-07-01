package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/qwgao/pickmem/internal/vault"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		groupFilter string
		showPending bool
		showAll     bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List memory notes grouped by frontmatter group",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}

			var notes []*vault.Note
			switch {
			case showAll:
				notes = s.List()
			case showPending:
				notes = s.ListPending()
			default:
				notes = s.ListActive()
			}
			if groupFilter != "" {
				notes = filterByGroup(notes, groupFilter)
			}
			if len(notes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no notes)")
				return nil
			}

			// Bucket by group for a friendlier layout.
			buckets := map[string][]*vault.Note{}
			for _, n := range notes {
				g := n.Group
				if n.Status == vault.StatusPending {
					g = "inbox"
				}
				buckets[g] = append(buckets[g], n)
			}
			groups := make([]string, 0, len(buckets))
			for g := range buckets {
				groups = append(groups, g)
			}
			sort.Strings(groups)

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, g := range groups {
				fmt.Fprintf(tw, "\n[%s]\n", g)
				for _, n := range buckets[g] {
					fmt.Fprintf(tw, "  %s\t%s\t%s\n", shortID(n.ID), n.Label, tagString(n.Tags))
				}
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVarP(&groupFilter, "group", "g", "", "only show notes whose frontmatter group matches (prefix match)")
	cmd.Flags().BoolVar(&showPending, "pending", false, "show only pending inbox notes")
	cmd.Flags().BoolVar(&showAll, "all", false, "show both active and pending notes")
	return cmd
}

// shortID returns the last 6 chars of a ULID — enough to disambiguate a
// personal vault at a glance without making the terminal noisy.
func shortID(id string) string {
	if len(id) <= 6 {
		return id
	}
	return "…" + id[len(id)-6:]
}

func tagString(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return "#" + strings.Join(tags, " #")
}

func filterByGroup(notes []*vault.Note, prefix string) []*vault.Note {
	out := notes[:0]
	for _, n := range notes {
		if strings.HasPrefix(n.Group, prefix) {
			out = append(out, n)
		}
	}
	return out
}
