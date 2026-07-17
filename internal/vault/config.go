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
	// SuggestedTags is the vault's set of quick-pick tag chips. Empty means
	// "use the built-in defaults" (see DefaultSuggestedTags), so old vaults and
	// fresh ones behave identically until the user customizes it.
	SuggestedTags []string `json:"suggested_tags,omitempty"`
	// LegacyNoteTypes reads the pre-tags `note_types` key so an existing
	// vault's customized vocabulary carries over as suggested tags. Never
	// written back (see SaveConfig normalization is unnecessary — omitempty
	// keeps it out once SuggestedTags is set).
	LegacyNoteTypes []string `json:"note_types,omitempty"`
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
	// Migrate a pre-tags vault: an old `note_types` list becomes the
	// suggested-tag chips. Cleared here so the next SaveConfig drops the key.
	if len(c.SuggestedTags) == 0 && len(c.LegacyNoteTypes) > 0 {
		c.SuggestedTags = c.LegacyNoteTypes
	}
	c.LegacyNoteTypes = nil
	return c, nil
}

// SaveConfig writes pickmem/config.json atomically (via rename).
func (s *Store) SaveConfig(c Config) error {
	return writeJSONAtomic(s.configPath(), c)
}

// SuggestedTags returns the vault's quick-pick tag chips: the user's
// configured list, or the built-in defaults when none is set.
func (s *Store) SuggestedTags() []string {
	cfg, err := s.LoadConfig()
	if err != nil || len(cfg.SuggestedTags) == 0 {
		return DefaultSuggestedTags()
	}
	return normalizeTagList(cfg.SuggestedTags)
}

// normalizeTagList trims, de-dupes, and drops empties, preserving order.
func normalizeTagList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
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
