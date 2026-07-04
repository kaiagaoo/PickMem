package ingest

import (
	"context"
	"fmt"
	"os"

	"github.com/kaiagaoo/PickMem/internal/routing"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// ImportResult is what a caller learns about an import run without having
// to re-scan the inbox afterward.
type ImportResult struct {
	Parsed    int `json:"parsed"`    // chunks the parser recognized
	Staged    int `json:"staged"`    // written to the inbox as pending
	Duplicate int `json:"duplicate"` // skipped: content already in the vault
	Routed    int `json:"routed"`    // staged items with a non-empty suggested_group
}

// ImportFile reads a file and stages each parsed memory as a pending inbox
// note. See ImportBytes.
func ImportFile(ctx context.Context, s *vault.Store, path string, format Format) (ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	return ImportBytes(ctx, s, data, format)
}

// ImportBytes parses raw bytes (JSON / bullets / paragraphs, auto-detected
// unless format says otherwise), routes each chunk with the vault's keyword
// rules, de-duplicates against active + pending notes on a content hash,
// and stages the survivors as `status: pending`. Nothing activates.
func ImportBytes(ctx context.Context, s *vault.Store, data []byte, format Format) (ImportResult, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return ImportResult{}, err
	}
	router := routing.New(routing.NewRules(cfg))
	groups := knownGroups(s)

	// Dedupe pool: everything the vault already knows about.
	seen := map[string]bool{}
	for _, n := range s.ListPending() {
		seen[ContentHash(n.Body)] = true
	}
	for _, n := range s.ListActive() {
		seen[ContentHash(n.Body)] = true
	}

	candidates := Parse(data, format)
	result := ImportResult{Parsed: len(candidates)}
	for _, body := range candidates {
		h := ContentHash(body)
		if seen[h] {
			result.Duplicate++
			continue
		}
		seen[h] = true

		suggested := router.Suggest(ctx, body, groups)
		note := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label:          DeriveLabel(body),
				SuggestedGroup: suggested,
				Source:         vault.SourceImport,
			},
			Body: body,
		}
		if _, err := s.AddInbox(note); err != nil {
			return result, fmt.Errorf("stage %q: %w", note.Label, err)
		}
		result.Staged++
		if suggested != "" {
			result.Routed++
		}
	}
	return result, nil
}

// knownGroups returns the sorted list of active groups.
func knownGroups(s *vault.Store) []string {
	groups := s.Groups()
	out := make([]string, 0, len(groups))
	for g := range groups {
		out = append(out, g)
	}
	sortStrings(out)
	return out
}

func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}
