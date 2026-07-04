package cli

import (
	"fmt"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "show <id-or-suffix>",
		Short: "Print a memory note by id (or its short suffix)",
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
			if raw {
				data, err := n.Serialize()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "id:       %s\n", n.ID)
			fmt.Fprintf(out, "label:    %s\n", n.Label)
			fmt.Fprintf(out, "group:    %s\n", n.Group)
			fmt.Fprintf(out, "status:   %s\n", n.Status)
			fmt.Fprintf(out, "source:   %s\n", n.Source)
			fmt.Fprintf(out, "created:  %s\n", n.CreatedAt.Format("2006-01-02 15:04:05 MST"))
			if len(n.Tags) > 0 {
				fmt.Fprintf(out, "tags:     %s\n", strings.Join(n.Tags, ", "))
			}
			fmt.Fprintf(out, "path:     %s\n\n", n.RelPath)
			fmt.Fprintln(out, n.Body)
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "print the raw file (frontmatter + body)")
	return cmd
}

// resolveNote takes either a full ULID or a 3+ character suffix and
// returns the unique matching note. Ambiguous suffixes error out.
func resolveNote(s *vault.Store, key string) (*vault.Note, error) {
	if n, ok := s.Get(key); ok {
		return n, nil
	}
	// Try suffix match.
	if len(key) < 3 {
		return nil, fmt.Errorf("id %q too short — use at least 3 chars", key)
	}
	var matches []*vault.Note
	up := strings.ToUpper(key)
	for _, n := range s.List() {
		if strings.HasSuffix(n.ID, up) {
			matches = append(matches, n)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no note matches %q", key)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous id %q matches %d notes — use more chars", key, len(matches))
	}
}
