package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/ingest"
	"github.com/kaiagaoo/PickMem/internal/routing"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// StageItem is one memory candidate the model has already extracted and
// condensed from the conversation. This is the counterpart of
// propose_memories for callers that can do the judgment themselves: the
// model decides *what* is memory-worthy, PickMem decides what happens to
// it (staged pending, never activated).
type StageItem struct {
	Label string `json:"label,omitempty" jsonschema:"short human-readable title; derived from the body if omitted"`
	Body  string `json:"body" jsonschema:"the memory itself: one self-contained item, stated in third person"`
	// Type is what kind of note this is, independent of its group. Unknown
	// or empty defaults to fact.
	Type string `json:"type,omitempty" jsonschema:"kind of note: fact (a stable fact, the default), idea (a proposal/concept), thought (a fleeting reflection), or reference (external material)"`
	// SuggestedGroup must name a group that already exists in the vault —
	// staging can't invent taxonomy. Invalid or missing values fall back
	// to the vault's keyword routing rules, then to unrouted.
	SuggestedGroup string `json:"suggested_group,omitempty" jsonschema:"an existing vault group for this item (see list_groups); invalid values are dropped, not created"`
}

// StagedItem reports what happened to one input item, in input order.
type StagedItem struct {
	Label string `json:"label"`
	// Outcome is "staged", "duplicate", or "skipped".
	Outcome        string `json:"outcome"`
	Type           string `json:"type,omitempty"`
	SuggestedGroup string `json:"suggested_group,omitempty"`
	// Warning explains a downgrade the model should learn from, e.g. a
	// suggested_group that doesn't exist in the vault.
	Warning string `json:"warning,omitempty"`
}

// StageResult summarizes a stage_memories call without echoing bodies back.
type StageResult struct {
	Staged    int          `json:"staged"`
	Duplicate int          `json:"duplicate_skipped"`
	Skipped   int          `json:"skipped"`
	Items     []StagedItem `json:"items"`
}

// StageMemories stages model-extracted items into pickmem/inbox/ as
// `status: pending`, `source: extract` notes. Rules mirror the import
// pipeline: content-hash de-dupe against the whole vault (active and
// pending), keyword-rule routing as the fallback classifier. It never
// activates anything — `pickmem review` is the only gate into memory.
func StageMemories(s *vault.Store, items []StageItem) (StageResult, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return StageResult{}, err
	}
	router := routing.New(routing.NewRules(cfg))

	known := map[string]bool{}
	for _, g := range KnownGroupNames(s) {
		known[g] = true
	}

	// De-dupe against everything already in the vault, like import does:
	// re-saving an accepted memory is a duplicate, not a new pending item.
	seen := map[string]bool{}
	for _, n := range s.List() {
		seen[ingest.ContentHash(n.Body)] = true
	}

	result := StageResult{Items: make([]StagedItem, 0, len(items))}
	for _, item := range items {
		body := strings.TrimSpace(item.Body)
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = ingest.DeriveLabel(body)
		}
		out := StagedItem{Label: label}

		if body == "" {
			out.Outcome = "skipped"
			out.Warning = "empty body"
			result.Skipped++
			result.Items = append(result.Items, out)
			continue
		}
		h := ingest.ContentHash(body)
		if seen[h] {
			out.Outcome = "duplicate"
			result.Duplicate++
			result.Items = append(result.Items, out)
			continue
		}

		group := strings.Trim(item.SuggestedGroup, "/ ")
		if group != "" && !known[group] {
			out.Warning = fmt.Sprintf("suggested_group %q does not exist in the vault; fell back to routing rules", group)
			group = ""
		}
		if group == "" {
			group = router.Suggest(context.Background(), body, KnownGroupNames(s))
		}

		kind := vault.NormalizeType(item.Type)
		n := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label:          label,
				Type:           kind,
				SuggestedGroup: group,
				Source:         vault.SourceExtract,
			},
			Body: body,
		}
		if _, err := s.AddInbox(n); err != nil {
			return StageResult{}, err
		}
		seen[h] = true
		out.Outcome = "staged"
		out.Type = kind
		out.SuggestedGroup = group
		result.Staged++
		result.Items = append(result.Items, out)
	}
	return result, nil
}

// KnownGroupNames is the taxonomy a classifier (the model, via
// list_groups) may suggest from. It's the user's curated folder tree
// unioned with note groups and rule targets — see vault.Store.KnownGroups.
func KnownGroupNames(s *vault.Store) []string {
	return s.KnownGroups()
}
