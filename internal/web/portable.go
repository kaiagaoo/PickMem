package web

import (
	"fmt"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

// The portable vault format: a whole vault as one JSON object, used to import
// an exported/backed-up vault. It is a VIEW over the Markdown vault, not the
// vault's storage — the source of truth stays the folder of notes. Import
// materializes items back into real notes.

type portableItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Body  string `json:"body"`
	// Type is a legacy field: older exports carried a single note type. It's
	// folded into Tags on import so nothing is lost (see importVault).
	Type      string   `json:"type,omitempty"`
	Group     string   `json:"group"`
	Source    string   `json:"source"`
	Status    string   `json:"status"` // "active" | "inbox"
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// portableVault is the import shape. Unknown fields in an incoming blob (e.g.
// a full export's lenses/active) are ignored by the JSON decoder.
type portableVault struct {
	FormatVersion int            `json:"format_version"`
	VaultName     string         `json:"vault_name,omitempty"`
	Items         []portableItem `json:"items"`
}

// importVault materializes a portable blob into the vault. Every incoming item
// is created as a NEW note (fresh id), so import can never overwrite or delete
// an existing note — consistent with PickMem's create-only default. Lenses and
// the active selection are intentionally not imported, because the new ids
// wouldn't line up with the blob's old ids. Returns the count created.
func importVault(s *vault.Store, pv portableVault) (int, error) {
	if pv.FormatVersion != 1 {
		return 0, fmt.Errorf("unsupported format_version %d (expected 1)", pv.FormatVersion)
	}
	created := 0
	for _, it := range pv.Items {
		if it.Label == "" {
			continue // skip malformed
		}
		group := it.Group
		if group == "" {
			group = "imported"
		}
		tags := it.Tags
		// Fold a legacy note type into tags (dropping the old default "fact").
		if t := it.Type; t != "" && t != vault.TagFact {
			tags = append([]string{t}, tags...)
		}
		n := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label:  it.Label,
				Group:  group,
				Tags:   tags,
				Source: vault.SourceImport,
			},
			Body: it.Body,
		}
		if it.Status == "inbox" {
			n.SuggestedGroup = group
			if _, err := s.AddInbox(n); err != nil {
				return created, err
			}
		} else {
			if _, err := s.Add(n); err != nil {
				return created, err
			}
		}
		created++
	}
	return created, nil
}
