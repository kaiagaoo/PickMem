package web

import (
	"net/http"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

// handleCreateGroup makes an empty group folder (onboarding seeds groups
// before they hold items).
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		writeErr(w, http.StatusBadRequest, "group path required")
		return
	}
	if err := s.store.EnsureGroup(strings.TrimSpace(req.Path)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

// handleRenameGroup moves a group (and everything filed under it) to a new
// path. Uses a JSON body rather than a path param because group paths contain
// slashes.
func (s *Server) handleRenameGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.From) == "" || strings.TrimSpace(req.To) == "" {
		writeErr(w, http.StatusBadRequest, "from and to are required")
		return
	}
	if _, err := s.store.RenameGroup(req.From, req.To); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

// handleDeleteGroup deletes a group folder and all notes under it.
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		writeErr(w, http.StatusBadRequest, "path required")
		return
	}
	if _, err := s.store.DeleteGroup(req.Path); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

// handleSetNoteTypes persists the vault's type vocabulary into config.json.
// The store normalizes the list (fact guaranteed, de-duped) on read, so we
// just store what the client sends.
func (s *Server) handleSetNoteTypes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Types []string `json:"types"`
	}
	if !decode(w, r, &req) {
		return
	}
	cfg, err := s.store.LoadConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.NoteTypes = req.Types
	if err := s.store.SaveConfig(cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeState(w)
}

// handleRenameNoteType renames a type and updates every note using it.
func (s *Server) handleRenameNoteType(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !decode(w, r, &req) {
		return
	}
	if _, err := s.store.RenameNoteType(req.From, req.To); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

// handleSetVaultName persists a cosmetic vault name into config.json.
func (s *Server) handleSetVaultName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &req) {
		return
	}
	cfg, err := s.store.LoadConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.VaultName = strings.TrimSpace(req.Name)
	if err := s.store.SaveConfig(cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeState(w)
}

// handleImport materializes a portable blob into new notes (merge mode).
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	var pv portableVault
	if !decode(w, r, &pv) {
		return
	}
	if _, err := importVault(s.store, pv); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

// handleClearVault deletes every PickMem-tracked note (active + pending) and
// resets the active selection and lenses. Destructive — the UI gates it
// behind a typed confirmation. Non-PickMem files (the user's other Obsidian
// notes without frontmatter) are never touched, since they aren't tracked.
func (s *Server) handleClearVault(w http.ResponseWriter, r *http.Request) {
	for _, n := range s.store.List() {
		if err := s.store.Remove(n.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := s.store.SaveActive(vault.Active{ItemIDs: []string{}}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.store.SaveLenses(nil); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeState(w)
}
