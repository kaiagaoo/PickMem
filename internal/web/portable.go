package web

import (
	"fmt"
	"time"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

// The portable vault format (blueprint §4/§12): the whole vault as one JSON
// blob for export/import. It is a VIEW over the Markdown vault, not the
// vault's storage — the source of truth stays the folder of notes. Import
// materializes items back into real notes.

type portableItem struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Body      string   `json:"body"`
	Type      string   `json:"type"`
	Group     string   `json:"group"`
	Source    string   `json:"source"`
	Status    string   `json:"status"` // "active" | "inbox"
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

type portableLens struct {
	Name    string   `json:"name"`
	ItemIDs []string `json:"item_ids"`
}

type portableVault struct {
	FormatVersion int            `json:"format_version"`
	VaultName     string         `json:"vault_name,omitempty"`
	ExportedAt    string         `json:"exported_at"`
	Items         []portableItem `json:"items"`
	Lenses        []portableLens `json:"lenses"`
	ActiveItemIDs []string       `json:"active_item_ids"`
	ActiveLens    string         `json:"active_lens,omitempty"`
}

// statusFor maps a note's on-disk status to the portable "active"/"inbox"
// vocabulary the blueprint uses (pending -> inbox).
func statusFor(n *vault.Note) string {
	if n.Status == vault.StatusPending {
		return "inbox"
	}
	return "active"
}

func exportVault(s *vault.Store) (portableVault, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return portableVault{}, err
	}
	active, err := s.LoadActive()
	if err != nil {
		return portableVault{}, err
	}
	lenses, err := s.LoadLenses()
	if err != nil {
		return portableVault{}, err
	}

	items := []portableItem{}
	for _, n := range s.List() { // active + pending, id-sorted
		items = append(items, portableItem{
			ID:        n.ID,
			Label:     n.Label,
			Body:      n.Body,
			Type:      n.Kind(),
			Group:     n.Group,
			Source:    n.Source,
			Status:    statusFor(n),
			Tags:      n.Tags,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
		})
	}
	pls := []portableLens{}
	for _, l := range lenses {
		ids := l.ItemIDs
		if ids == nil {
			ids = []string{}
		}
		pls = append(pls, portableLens{Name: l.Name, ItemIDs: ids})
	}
	ids := active.ItemIDs
	if ids == nil {
		ids = []string{}
	}
	return portableVault{
		FormatVersion: 1,
		VaultName:     cfg.VaultName,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Items:         items,
		Lenses:        pls,
		ActiveItemIDs: ids,
		ActiveLens:    active.ActiveLens,
	}, nil
}

// importVault materializes a portable blob into the vault. v1 supports the
// "merge" mode only: every incoming item is created as a NEW note (fresh id),
// so import can never overwrite or delete an existing note — consistent with
// PickMem's create-only default. Lenses/active are intentionally not imported
// in merge mode, because the new ids wouldn't line up with the blob's old
// ids; membership is rebuilt by the user. Returns the count created.
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
		n := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label:  it.Label,
				Group:  group,
				Type:   vault.NormalizeType(it.Type),
				Tags:   it.Tags,
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
