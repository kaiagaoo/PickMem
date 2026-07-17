package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config is the on-disk pickmem/config.json. Kept minimal in M1; ingestion
// (routing rules) and extension prefs land here in later milestones.
type Config struct {
	// TemplateName records which starter template `init` used. Purely for
	// diagnostics; PickMem never reads it back to make decisions.
	TemplateName string `json:"template_name,omitempty"`
	// VaultName is an optional human label for the vault, shown in the web
	// UI header and Settings. Cosmetic only — the vault is identified by its
	// path, never by this name.
	VaultName string `json:"vault_name,omitempty"`
	// Version is a schema version for the on-disk contract. Bump on
	// breaking layout changes so future migrations can detect old vaults.
	Version int `json:"version"`
	// RoutingRules maps keyword substrings to a target group. Populated by
	// M4's import router; unused in M1 but reserved to keep the shape
	// stable.
	RoutingRules []RoutingRule `json:"routing_rules,omitempty"`
	// NoteTypes is the vault's type vocabulary (the kinds a note can be).
	// Empty means "use the built-in defaults" (see DefaultNoteTypes), so old
	// vaults and fresh ones behave identically until the user customizes it.
	NoteTypes []string `json:"note_types,omitempty"`
}

// RoutingRule assigns a suggested group when a keyword substring (case-
// insensitive) appears in an imported item's label or body.
type RoutingRule struct {
	Keyword string `json:"keyword"`
	Group   string `json:"group"`
}

// DefaultConfig is what `init` writes when no template supplies its own.
func DefaultConfig() Config {
	return Config{Version: 1}
}

func (s *Store) configPath() string {
	return filepath.Join(s.Root, PickmemDir, ConfigFile)
}

// LoadConfig reads pickmem/config.json. Missing file returns DefaultConfig()
// with no error — a partial vault is still usable.
func (s *Store) LoadConfig() (Config, error) {
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return c, nil
}

// SaveConfig writes pickmem/config.json atomically (via rename).
func (s *Store) SaveConfig(c Config) error {
	return writeJSONAtomic(s.configPath(), c)
}

// NoteTypes returns the vault's type vocabulary: the user's configured list,
// or the built-in defaults when none is set. TypeFact is always guaranteed to
// be present and first, since it's the canonical default.
func (s *Store) NoteTypes() []string {
	cfg, err := s.LoadConfig()
	if err != nil || len(cfg.NoteTypes) == 0 {
		return DefaultNoteTypes()
	}
	return normalizeTypeList(cfg.NoteTypes)
}

// RenameNoteType renames a type in the vault's vocabulary and rewrites every
// active note currently using it. The default type (TypeFact) can't be
// renamed, and the new name must not already exist. Returns the number of
// notes updated.
func (s *Store) RenameNoteType(from, to string) (int, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return 0, errors.New("type name required")
	}
	if from == TypeFact {
		return 0, errors.New("the default type (fact) can't be renamed")
	}
	if from == to {
		return 0, nil
	}
	list := s.NoteTypes() // materializes defaults if the list is empty
	found := false
	for _, t := range list {
		if t == to {
			return 0, fmt.Errorf("type %q already exists", to)
		}
		if t == from {
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("type %q not found", from)
	}

	next := make([]string, len(list))
	for i, t := range list {
		if t == from {
			next[i] = to
		} else {
			next[i] = t
		}
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		return 0, err
	}
	cfg.NoteTypes = next
	if err := s.SaveConfig(cfg); err != nil {
		return 0, err
	}

	updated := 0
	for _, n := range s.ListActive() {
		if n.Kind() != from {
			continue
		}
		if _, err := s.EditNote(n.ID, NoteEdit{
			Label: n.Label, Group: n.Group, Body: n.Body, Type: to, Tags: n.Tags,
		}); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

// normalizeTypeList trims, de-dupes, drops empties, and guarantees TypeFact is
// present and first — so the default type can never be configured away.
func normalizeTypeList(in []string) []string {
	out := []string{TypeFact}
	seen := map[string]bool{TypeFact: true}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
