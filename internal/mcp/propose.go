package mcp

import (
	"context"

	"github.com/qwgao/pickmem/internal/ingest"
	"github.com/qwgao/pickmem/internal/routing"
	"github.com/qwgao/pickmem/internal/vault"
)

// ProposeResult reports what propose_memories did without echoing the
// caller's chat text back into the response. We stage first, summarize
// second.
type ProposeResult struct {
	Staged    int      `json:"staged"`
	Duplicate int      `json:"duplicate_skipped"`
	Labels    []string `json:"labels"`
}

// ProposeFromChat splits chat_text into memory candidates and stages
// each as a `status: pending` note in the inbox with `source: extract`.
// It never activates — that's the picker's job. Extraction is
// deterministic and rules-based; AI classification is opt-in in the
// import path (M4 --allow-ai), not here.
//
// Splitting: paragraphs separated by blank lines (ingest.SplitParagraphs),
// dropping anything under ingest.MinLen chars.
//
// De-dupe: normalized content hash over the whole vault's pending notes,
// so re-running propose_memories on the same conversation is idempotent.
func ProposeFromChat(s *vault.Store, chatText string) (ProposeResult, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return ProposeResult{}, err
	}
	router := routing.New(routing.NewRules(cfg))

	candidates := ingest.SplitParagraphs(chatText)

	// Existing pending items form the dedupe set.
	seen := map[string]bool{}
	for _, n := range s.ListPending() {
		seen[ingest.ContentHash(n.Body)] = true
	}

	// Known groups from the current vault — passed to classifiers so an
	// AI can't invent taxonomy. Rules ignore this; kept for interface
	// consistency.
	groups := KnownGroupNames(s)

	result := ProposeResult{Labels: []string{}}
	for _, body := range candidates {
		h := ingest.ContentHash(body)
		if seen[h] {
			result.Duplicate++
			continue
		}
		seen[h] = true

		label := ingest.DeriveLabel(body)
		group := router.Suggest(context.Background(), body, groups)

		n := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label:          label,
				SuggestedGroup: group,
				Source:         vault.SourceExtract,
			},
			Body: body,
		}
		if _, err := s.AddInbox(n); err != nil {
			return ProposeResult{}, err
		}
		result.Staged++
		result.Labels = append(result.Labels, label)
	}
	return result, nil
}

// sortStrings is inline-tiny to avoid a `sort` import just for one call.
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}
