// Package cli wires the pickmem cobra commands. It's separated from
// cmd/pickmem so cobra wiring is independently testable.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EnvVaultPath is the environment variable that overrides the user config.
const EnvVaultPath = "PICKMEM_VAULT"

// UserConfig is stored at ~/.config/pickmem/config.json (or $XDG_CONFIG_HOME).
// It's a tiny pointer to the last `init`-ed vault so daily commands don't
// need --vault.
type UserConfig struct {
	VaultPath string `json:"vault_path,omitempty"`
}

func userConfigDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "pickmem"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pickmem"), nil
}

func userConfigPath() (string, error) {
	d, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// LoadUserConfig reads the user-level config, returning an empty struct if
// the file doesn't exist yet.
func LoadUserConfig() (UserConfig, error) {
	p, err := userConfigPath()
	if err != nil {
		return UserConfig{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return UserConfig{}, nil
		}
		return UserConfig{}, err
	}
	var c UserConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return UserConfig{}, fmt.Errorf("decode user config %s: %w", p, err)
	}
	return c, nil
}

// SaveUserConfig writes the user-level config, creating the directory if
// needed.
func SaveUserConfig(c UserConfig) error {
	d, err := userConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	p := filepath.Join(d, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p, data, 0o644)
}

// ResolveVaultPath picks the vault path with precedence:
//  1. --vault flag (passed in as flagVal, "" if unset)
//  2. $PICKMEM_VAULT env var
//  3. user config's VaultPath
//
// Returns an error only if all three are empty.
func ResolveVaultPath(flagVal string) (string, error) {
	if flagVal != "" {
		return filepath.Abs(flagVal)
	}
	if v := os.Getenv(EnvVaultPath); v != "" {
		return filepath.Abs(v)
	}
	uc, err := LoadUserConfig()
	if err != nil {
		return "", err
	}
	if uc.VaultPath != "" {
		return filepath.Abs(uc.VaultPath)
	}
	return "", errors.New("no vault path set: pass --vault, set $PICKMEM_VAULT, or run `pickmem init <path>`")
}
