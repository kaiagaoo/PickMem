package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/retrieval"
	"github.com/spf13/cobra"
)

type suggestionDTO struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Group        string   `json:"group"`
	Score        float64  `json:"score"`
	MatchedTerms []string `json:"matched_terms"`
}

func newSuggestCmd() *cobra.Command {
	var top int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "suggest <task description>",
		Short: "Recommend relevant memories without activating them",
		Long: `Rank active vault notes against a task description using a local,
deterministic BM25-style scorer. This command is read-only: suggestions never
change active.json, preserving the user's approval boundary.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if top < 1 {
				return fmt.Errorf("--top must be at least 1")
			}
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")
			matches := retrieval.Rank(query, s.ListActive(), top)
			items := make([]suggestionDTO, 0, len(matches))
			for _, match := range matches {
				items = append(items, suggestionDTO{
					ID: match.Note.ID, Label: match.Note.Label, Group: match.Note.Group,
					Score: match.Score, MatchedTerms: match.MatchedTerms,
				})
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(items)
			}
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No relevant memories found. Nothing was activated.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Suggested memories for %q (read-only):\n", query)
			for i, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%d. %s  [%s]  score=%.3f  matched=%s\n",
					i+1, item.Label, item.Group, item.Score, strings.Join(item.MatchedTerms, ","))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Run `pickmem pick` to approve a selection.")
			return nil
		},
	}
	cmd.Flags().IntVar(&top, "top", 5, "maximum number of suggestions")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit structured JSON")
	return cmd
}
