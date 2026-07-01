package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config is the on-disk pickmem/config.json. Kept minimal in M1; ingestion
// (routing rules) and extension prefs land here in later milestones.
type Config struct {
	// TemplateName records which starter template `init` used. Purely for
	// diagnostics; PickMem never reads it back to make decisions.
	TemplateName string `json:"template_name,omitempty"`
	// Version is a schema version for the on-disk contract. Bump on
	// breaking layout changes so future migrations can detect old vaults.
	Version int `json:"version"`
	// RoutingRules maps keyword substrings to a target group. Populated by
	// M4's import router; unused in M1 but reserved to keep the shape
	// stable.
	RoutingRules []RoutingRule `json:"routing_rules,omitempty"`
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
