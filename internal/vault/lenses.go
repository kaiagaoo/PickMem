package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Lens is a saved, named selection of memory item ids. See PROPOSAL §2.
type Lens struct {
	Name    string   `json:"name"`
	ItemIDs []string `json:"item_ids"`
}

func (s *Store) lensesPath() string {
	return filepath.Join(s.Root, PickmemDir, LensesFile)
}

// LoadLenses reads pickmem/lenses.json. Missing file returns an empty slice
// (a fresh vault has no lenses yet).
func (s *Store) LoadLenses() ([]Lens, error) {
	data, err := os.ReadFile(s.lensesPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Lens{}, nil
		}
		return nil, fmt.Errorf("read lenses: %w", err)
	}
	var out []Lens
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode lenses: %w", err)
	}
	return out, nil
}

// SaveLenses atomically writes pickmem/lenses.json.
func (s *Store) SaveLenses(ls []Lens) error {
	if ls == nil {
		ls = []Lens{}
	}
	return writeJSONAtomic(s.lensesPath(), ls)
}

// FindLens returns a lens by name (case-sensitive).
func FindLens(ls []Lens, name string) (Lens, bool) {
	for _, l := range ls {
		if l.Name == name {
			return l, true
		}
	}
	return Lens{}, false
}

// UpsertLens replaces the lens with matching name, or appends if new.
func UpsertLens(ls []Lens, l Lens) []Lens {
	for i, existing := range ls {
		if existing.Name == l.Name {
			ls[i] = l
			return ls
		}
	}
	return append(ls, l)
}

// RemoveLens removes the named lens. Missing name is a no-op.
func RemoveLens(ls []Lens, name string) []Lens {
	out := ls[:0]
	for _, l := range ls {
		if l.Name != name {
			out = append(out, l)
		}
	}
	return out
}
