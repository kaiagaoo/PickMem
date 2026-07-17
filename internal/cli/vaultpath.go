// Package cli wires the pickmem cobra commands. It's separated from
// cmd/pickmem so cobra wiring is independently testable.
package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kaiagaoo/PickMem/internal/userconf"
)

// EnvVaultPath is the environment variable that overrides the user config.
const EnvVaultPath = "PICKMEM_VAULT"

// UserConfig / LoadUserConfig / SaveUserConfig are thin re-exports of the
// shared internal/userconf package, kept so existing CLI call sites (init.go)
// don't churn.
type UserConfig = userconf.Config

func LoadUserConfig() (UserConfig, error) { return userconf.Load() }
func SaveUserConfig(c UserConfig) error   { return userconf.Save(c) }

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
	uc, err := userconf.Load()
	if err != nil {
		return "", err
	}
	if uc.VaultPath != "" {
		return filepath.Abs(uc.VaultPath)
	}
	return "", errors.New("no vault path set: pass --vault, set $PICKMEM_VAULT, or run `pickmem init <path>`")
}
