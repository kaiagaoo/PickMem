// Package userconf is the user-level (machine-wide) config for PickMem,
// stored at ~/.config/pickmem/config.json (or $XDG_CONFIG_HOME). It records
// the current vault and a most-recently-used list so the web app's vault
// switcher can move between vaults, and daily CLI commands don't need
// --vault. It is distinct from a vault's own pickmem/config.json.
package userconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config is the user-level pointer to vaults.
type Config struct {
	// VaultPath is the current/default vault (absolute path).
	VaultPath string `json:"vault_path,omitempty"`
	// RecentVaults is the MRU list of vault paths (absolute), most-recent
	// first, so the web switcher can list them.
	RecentVaults []string `json:"recent_vaults,omitempty"`
}

// Dir returns the config directory, honoring $XDG_CONFIG_HOME.
func Dir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "pickmem"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pickmem"), nil
}

func path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads the config, returning an empty struct if the file is absent.
func Load() (Config, error) {
	p, err := path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("decode user config %s: %w", p, err)
	}
	return c, nil
}

// Save writes the config, creating the directory if needed.
func Save(c Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(d, "config.json"), data, 0o644)
}

// SetCurrent marks path as the current vault and moves it to the front of
// the recent list (de-duplicated, capped). Absolute paths only.
func SetCurrent(vaultPath string) error {
	abs, err := filepath.Abs(vaultPath)
	if err != nil {
		return err
	}
	c, err := Load()
	if err != nil {
		return err
	}
	c.VaultPath = abs
	c.RecentVaults = pushRecent(c.RecentVaults, abs)
	return Save(c)
}

// Forget removes a path from the recent list (the vault files are untouched).
func Forget(vaultPath string) error {
	abs, err := filepath.Abs(vaultPath)
	if err != nil {
		return err
	}
	c, err := Load()
	if err != nil {
		return err
	}
	out := c.RecentVaults[:0]
	for _, p := range c.RecentVaults {
		if p != abs {
			out = append(out, p)
		}
	}
	c.RecentVaults = out
	return Save(c)
}

// pushRecent moves abs to the front, de-dupes, and caps the list at 12.
func pushRecent(list []string, abs string) []string {
	out := []string{abs}
	for _, p := range list {
		if p != abs {
			out = append(out, p)
		}
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}
