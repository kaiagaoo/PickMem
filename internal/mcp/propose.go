package mcp

import (
	"context"

	"github.com/qwgao/pickmem/internal/ingest"
	"github.com/qwgao/pickmem/internal/routing"
	"github.com/qwgao/pickmem/internal/vault"
)

// ProposeResult reports what propose_memories did without echoing the
// caller's chat text back into the response.
type ProposeResult struct {
	Staged    int      `json:"staged"`
	Duplicate int      `json:"duplicate_skipped"`
	Labels    []string `json:"labels"`
}

// ProposeFromChat splits chatText into candidates and stages each as a
// `status: pending` inbox note with `source: extract`. It never activates
// anything — that stays the picker's job.
//
// Extraction is deterministic and rules-based: paragraphs separated by
// blank lines, routed with the vault's keyword rules, de-duplicated
// against the pending inbox on a content hash.
func ProposeFromChat(s *vault.Store, chatText string) (ProposeResult, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return ProposeResult{}, err
	}
	router := routing.New(routing.NewRules(cfg))
	groups := KnownGroupNames(s)

	seen := map[string]bool{}
	for _, n := range s.ListPending() {
		seen[ingest.ContentHash(n.Body)] = true
	}

	result := ProposeResult{Labels: []string{}}
	for _, body := range ingest.SplitParagraphs(chatText) {
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

func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}
