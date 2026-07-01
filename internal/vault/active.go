package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Active is the picked slice: which memory items the model gets to see
// right now, and (optionally) which lens they came from.
type Active struct {
	ActiveLens string   `json:"active_lens,omitempty"`
	ItemIDs    []string `json:"item_ids"`
}

func (s *Store) activePath() string {
	return filepath.Join(s.Root, PickmemDir, ActiveFile)
}

// LoadActive reads pickmem/active.json. Missing file returns empty Active
// (invariant §4: default is nothing).
func (s *Store) LoadActive() (Active, error) {
	data, err := os.ReadFile(s.activePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Active{ItemIDs: []string{}}, nil
		}
		return Active{}, fmt.Errorf("read active: %w", err)
	}
	var a Active
	if err := json.Unmarshal(data, &a); err != nil {
		return Active{}, fmt.Errorf("decode active: %w", err)
	}
	if a.ItemIDs == nil {
		a.ItemIDs = []string{}
	}
	return a, nil
}

// SaveActive atomically writes pickmem/active.json.
func (s *Store) SaveActive(a Active) error {
	if a.ItemIDs == nil {
		a.ItemIDs = []string{}
	}
	return writeJSONAtomic(s.activePath(), a)
}
