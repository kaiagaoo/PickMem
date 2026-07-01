package ingest

import (
	"context"
	"fmt"
	"os"

	"github.com/qwgao/pickmem/internal/routing"
	"github.com/qwgao/pickmem/internal/vault"
)

// ImportResult is what a caller learns about an import run without
// having to re-scan the inbox afterward. Kept small — detailed listings
// live in the inbox itself, viewable via `pickmem list --pending` or the
// review TUI.
type ImportResult struct {
	Parsed     int      `json:"parsed"`
	Staged     int      `json:"staged"`
	Duplicate  int      `json:"duplicate_skipped"`
	Routed     int      `json:"routed"` // Staged notes with a non-empty SuggestedGroup
	SkipReason []string `json:"skip_reasons,omitempty"`
}

// ImportFile reads a file, parses it, and stages each candidate as a
// pending inbox note with `source: import`. De-dupes across the whole
// vault (active + pending) via content hash. Optionally uses the
// provided Router to fill `suggested_group`.
//
// The Router is optional: pass nil for "rules-only, from the vault
// config." Pass a fully-populated Router when the caller wants AI in the
// chain — this keeps AI-consent (--allow-ai) at the CLI boundary rather
// than sprawled here.
func ImportFile(ctx context.Context, s *vault.Store, path string, format Format, router *routing.Router) (ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	return ImportBytes(ctx, s, data, format, router)
}

// ImportBytes is the split-for-testing variant. Same contract as
// ImportFile — takes raw bytes instead of a path.
func ImportBytes(ctx context.Context, s *vault.Store, data []byte, format Format, router *routing.Router) (ImportResult, error) {
	candidates := Parse(data, format)
	if router == nil {
		cfg, err := s.LoadConfig()
		if err != nil {
			return ImportResult{}, err
		}
		router = routing.New(routing.NewRules(cfg))
	}

	// Dedupe pool: everything the vault already knows about. Two sources —
	// pending inbox items (partially-imported earlier) and active notes
	// (accepted or created directly). Both count.
	seen := map[string]bool{}
	for _, n := range s.ListPending() {
		seen[ContentHash(n.Body)] = true
	}
	for _, n := range s.ListActive() {
		seen[ContentHash(n.Body)] = true
	}

	groups := knownGroups(s)

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

// knownGroups returns the sorted list of active groups. Same shape as
// mcp/propose.go's helper — kept private here to avoid a cross-package
// dependency for one 6-line function.
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
