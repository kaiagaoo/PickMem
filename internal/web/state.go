package web

import (
	"github.com/kaiagaoo/PickMem/internal/mcp"
	"github.com/kaiagaoo/PickMem/internal/picker"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// noteDTO is the wire shape of a single memory note. It flattens the note's
// frontmatter + body into the JSON the SPA consumes.
type noteDTO struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Group     string   `json:"group"`
	Tags      []string `json:"tags"`
	Body      string   `json:"body"`
	Source    string   `json:"source"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at"`
	RelPath   string   `json:"rel_path"`
	Tokens    int      `json:"tokens"`
}

func toDTO(n *vault.Note) noteDTO {
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	return noteDTO{
		ID:        n.ID,
		Label:     n.Label,
		Group:     n.Group,
		Tags:      tags,
		Body:      n.Body,
		Source:    n.Source,
		Status:    n.Status,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		RelPath:   n.RelPath,
		Tokens:    picker.EstimateTokens([]string{n.Body}),
	}
}

type lensDTO struct {
	Name    string   `json:"name"`
	ItemIDs []string `json:"item_ids"`
	Count   int      `json:"count"`
}

type activeDTO struct {
	ActiveLens string   `json:"active_lens"`
	ItemIDs    []string `json:"item_ids"`
}

// stateDTO is the single bootstrap payload the SPA loads and re-loads after
// every mutation, so the whole client is a pure function of one fetch.
type stateDTO struct {
	VaultPath     string        `json:"vault_path"`
	VaultName     string        `json:"vault_name"`
	Notes         []noteDTO     `json:"notes"`
	Pending       []noteDTO     `json:"pending"`
	Groups        []string      `json:"groups"`
	Lenses        []lensDTO     `json:"lenses"`
	Active        activeDTO     `json:"active"`
	Context       string        `json:"context"`
	Tokens        int           `json:"tokens"`
	SuggestedTags []string      `json:"suggested_tags"`
	Warnings      []string      `json:"warnings"`
	Vaults        []vaultRefDTO `json:"vaults"`
}

// buildState assembles the full client state from the store. The store is
// assumed freshly reloaded (see the reload middleware), so this is a pure
// read.
func buildState(s *vault.Store) (stateDTO, error) {
	active, err := s.LoadActive()
	if err != nil {
		return stateDTO{}, err
	}
	lenses, err := s.LoadLenses()
	if err != nil {
		return stateDTO{}, err
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		return stateDTO{}, err
	}

	notes := []noteDTO{}
	for _, n := range s.ListActive() {
		notes = append(notes, toDTO(n))
	}
	pending := []noteDTO{}
	for _, n := range s.ListPending() {
		pending = append(pending, toDTO(n))
	}

	lensDTOs := []lensDTO{}
	for _, l := range lenses {
		ids := l.ItemIDs
		if ids == nil {
			ids = []string{}
		}
		lensDTOs = append(lensDTOs, lensDTO{Name: l.Name, ItemIDs: ids, Count: len(ids)})
	}

	// Token estimate over the live (non-dangling) selection, matching the
	// picker footer and `pickmem status`.
	var bodies []string
	for _, id := range active.ItemIDs {
		if n, ok := s.Get(id); ok {
			bodies = append(bodies, n.Body)
		}
	}

	ctx, err := mcp.AssembleActive(s)
	if err != nil {
		return stateDTO{}, err
	}

	ids := active.ItemIDs
	if ids == nil {
		ids = []string{}
	}

	return stateDTO{
		VaultPath:     s.Root,
		VaultName:     cfg.VaultName,
		Notes:         notes,
		Pending:       pending,
		Groups:        s.KnownGroups(),
		Lenses:        lensDTOs,
		Active:        activeDTO{ActiveLens: active.ActiveLens, ItemIDs: ids},
		Context:       ctx,
		Tokens:        picker.EstimateTokens(bodies),
		SuggestedTags: s.SuggestedTags(),
		Warnings:      s.Warnings(),
		Vaults:        recentVaultRefs(s.Root),
	}, nil
}
