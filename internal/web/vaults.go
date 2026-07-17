package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/userconf"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// vaultRefDTO is one entry in the vault switcher: a known vault path, its
// display name, whether it still exists on disk, and whether it's the one
// currently open.
type vaultRefDTO struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Exists  bool   `json:"exists"`
	Current bool   `json:"current"`
}

// recentVaultRefs resolves the MRU vault list (from user config) into display
// refs, guaranteeing the currently-open vault appears first.
func recentVaultRefs(currentPath string) []vaultRefDTO {
	cfg, _ := userconf.Load()
	seen := map[string]bool{}
	out := []vaultRefDTO{}
	add := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil || seen[abs] {
			return
		}
		seen[abs] = true
		info, statErr := os.Stat(abs)
		out = append(out, vaultRefDTO{
			Path:    abs,
			Name:    vaultDisplayName(abs),
			Exists:  statErr == nil && info.IsDir(),
			Current: abs == currentPath,
		})
	}
	add(currentPath)
	for _, p := range cfg.RecentVaults {
		add(p)
	}
	return out
}

// vaultDisplayName reads a vault's own pickmem/config.json for its name,
// falling back to the folder's base name.
func vaultDisplayName(vaultPath string) string {
	data, err := os.ReadFile(filepath.Join(vaultPath, vault.PickmemDir, vault.ConfigFile))
	if err == nil {
		var c struct {
			VaultName string `json:"vault_name"`
		}
		if json.Unmarshal(data, &c) == nil && strings.TrimSpace(c.VaultName) != "" {
			return c.VaultName
		}
	}
	return filepath.Base(vaultPath)
}

// expandPath resolves a leading ~ and returns an absolute path, so users can
// paste "~/vaults/work" into the switcher.
func expandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("path required")
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	return filepath.Abs(p)
}

// swapStore replaces the open store and records the switch in user config.
// Caller must hold s.mu.
func (s *Server) swapStore(st *vault.Store) {
	s.store = st
	_ = userconf.SetCurrent(st.Root)
}

// withLock is like withVault but does NOT reload the current store first —
// used by vault-management handlers that are about to replace the store
// entirely, so reloading the outgoing vault would be wasted work.
func (s *Server) withLock(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		h(w, r)
	}
}

// handleSwitchVault opens an existing directory as the active vault.
func (s *Server) handleSwitchVault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, err := expandPath(req.Path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		writeErr(w, http.StatusBadRequest, "not a folder: "+abs)
		return
	}
	st, err := vault.Open(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "open vault: "+err.Error())
		return
	}
	s.swapStore(st)
	s.writeState(w)
}

// handleCreateVault initializes a brand-new empty vault at path and switches
// to it.
func (s *Server) handleCreateVault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, err := expandPath(req.Path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	st, err := vault.Init(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "create vault: "+err.Error())
		return
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		if cfg, err := st.LoadConfig(); err == nil {
			cfg.VaultName = name
			_ = st.SaveConfig(cfg)
		}
	}
	s.swapStore(st)
	s.writeState(w)
}

// handleImportVaultAsNew creates a new vault at path from a portable JSON
// blob (a whole-vault export) and switches to it.
func (s *Server) handleImportVaultAsNew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string        `json:"path"`
		Name  string        `json:"name"`
		Vault portableVault `json:"vault"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, err := expandPath(req.Path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	st, err := vault.Init(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "create vault: "+err.Error())
		return
	}
	if _, err := importVault(st, req.Vault); err != nil {
		writeErr(w, http.StatusBadRequest, "import: "+err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.Vault.VaultName)
	}
	if name != "" {
		if cfg, err := st.LoadConfig(); err == nil {
			cfg.VaultName = name
			_ = st.SaveConfig(cfg)
		}
	}
	s.swapStore(st)
	s.writeState(w)
}

// handleForgetVault removes a path from the recent list (files untouched).
func (s *Server) handleForgetVault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &req) {
		return
	}
	_ = userconf.Forget(req.Path)
	s.writeState(w)
}
