// Package web serves the PickMem management UI: a small HTTP+JSON API over
// internal/vault.Store plus an embedded single-page app. It's a third
// surface alongside the TUI picker and the MCP server, and writes the same
// pickmem/active.json + lenses.json, so what the model sees never depends on
// which surface you used to pick it.
//
// The server is single-user and localhost-only by design. Every API request
// reloads the vault from disk first (so an Obsidian/CLI edit is reflected
// immediately) under one process-wide lock, which serializes mutations —
// fine for one person clicking around, and it keeps the create-only /
// edit-guard invariants honest.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

//go:embed all:static
var staticFS embed.FS

// Server is the HTTP handler for the web UI. Hold it via Handler().
type Server struct {
	store *vault.Store
	mu    sync.Mutex // serializes reload+operation across concurrent requests
	mux   *http.ServeMux
}

// NewServer builds a Server over an already-open vault store.
func NewServer(store *vault.Store) *Server {
	s := &Server{store: store, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the root http.Handler (API + embedded SPA).
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	// API. Each handler runs under withVault, which takes the lock and
	// reloads the store before calling through.
	s.mux.HandleFunc("GET /api/state", s.withVault(s.handleState))
	s.mux.HandleFunc("POST /api/notes", s.withVault(s.handleAddNote))
	s.mux.HandleFunc("PATCH /api/notes/{id}", s.withVault(s.handleEditNote))
	s.mux.HandleFunc("DELETE /api/notes/{id}", s.withVault(s.handleDeleteNote))
	s.mux.HandleFunc("PUT /api/active", s.withVault(s.handleSetActive))
	s.mux.HandleFunc("POST /api/inbox/{id}/accept", s.withVault(s.handleAcceptInbox))
	s.mux.HandleFunc("POST /api/inbox/{id}/reject", s.withVault(s.handleRejectInbox))
	s.mux.HandleFunc("PUT /api/lenses/{name}", s.withVault(s.handleSaveLens))
	s.mux.HandleFunc("POST /api/lenses/{name}/use", s.withVault(s.handleUseLens))
	s.mux.HandleFunc("DELETE /api/lenses/{name}", s.withVault(s.handleDeleteLens))
	s.mux.HandleFunc("POST /api/groups", s.withVault(s.handleCreateGroup))
	s.mux.HandleFunc("POST /api/groups/rename", s.withVault(s.handleRenameGroup))
	s.mux.HandleFunc("POST /api/groups/delete", s.withVault(s.handleDeleteGroup))
	s.mux.HandleFunc("PUT /api/vault/name", s.withVault(s.handleSetVaultName))
	s.mux.HandleFunc("GET /api/export", s.withVault(s.handleExport))
	s.mux.HandleFunc("POST /api/import", s.withVault(s.handleImport))
	s.mux.HandleFunc("POST /api/vault/clear", s.withVault(s.handleClearVault))
	s.mux.HandleFunc("POST /api/vaults/switch", s.withLock(s.handleSwitchVault))
	s.mux.HandleFunc("POST /api/vaults/create", s.withLock(s.handleCreateVault))
	s.mux.HandleFunc("POST /api/vaults/import", s.withLock(s.handleImportVaultAsNew))
	s.mux.HandleFunc("POST /api/vaults/forget", s.withVault(s.handleForgetVault))

	// Everything else: the embedded SPA, with a fallback to index.html so
	// client-side navigation works.
	s.mux.Handle("/", s.spaHandler())
}

// withVault wraps a handler: it takes the process lock, reloads the vault
// from disk, and on success invokes the handler. A reload failure is a 500
// — the vault is unreadable, so nothing downstream is trustworthy.
func (s *Server) withVault(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.store.Reload(); err != nil {
			writeErr(w, http.StatusInternalServerError, "reload vault: "+err.Error())
			return
		}
		h(w, r)
	}
}

// --- handlers -------------------------------------------------------------

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	s.writeState(w)
}

type addNoteReq struct {
	Label string   `json:"label"`
	Group string   `json:"group"`
	Body  string   `json:"body"`
	Type  string   `json:"type"`
	Tags  []string `json:"tags"`
	// ToInbox routes a new note to the inbox (status=pending) for later
	// review instead of activating it immediately. The blueprint's
	// de-emphasized "send to inbox instead" option, and the seam the future
	// AI pipeline writes into.
	ToInbox bool `json:"to_inbox"`
}

func (s *Server) handleAddNote(w http.ResponseWriter, r *http.Request) {
	var req addNoteReq
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Label) == "" || strings.TrimSpace(req.Group) == "" {
		writeErr(w, http.StatusBadRequest, "label and group are required")
		return
	}
	n := &vault.Note{
		Frontmatter: vault.Frontmatter{
			Label: strings.TrimSpace(req.Label),
			Group: strings.TrimSpace(req.Group),
			Type:  vault.NormalizeType(req.Type),
			Tags:  cleanTags(req.Tags),
		},
		Body: req.Body,
	}
	var err error
	if req.ToInbox {
		n.Source = vault.SourceManual
		n.SuggestedGroup = strings.TrimSpace(req.Group)
		_, err = s.store.AddInbox(n)
	} else {
		_, err = s.store.Add(n)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

func (s *Server) handleEditNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req addNoteReq // same shape as add
	if !decode(w, r, &req) {
		return
	}
	_, err := s.store.EditNote(id, vault.NoteEdit{
		Label: strings.TrimSpace(req.Label),
		Group: strings.TrimSpace(req.Group),
		Body:  req.Body,
		Type:  req.Type,
		Tags:  cleanTags(req.Tags),
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Remove(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

type setActiveReq struct {
	ItemIDs    []string `json:"item_ids"`
	ActiveLens string   `json:"active_lens"`
}

// handleSetActive persists a manual selection. Manually editing the
// selection detaches it from any lens unless the caller names one, matching
// the TUI: once you toggle items, the footer reads "custom".
func (s *Server) handleSetActive(w http.ResponseWriter, r *http.Request) {
	var req setActiveReq
	if !decode(w, r, &req) {
		return
	}
	ids := req.ItemIDs
	if ids == nil {
		ids = []string{}
	}
	if err := s.store.SaveActive(vault.Active{ActiveLens: req.ActiveLens, ItemIDs: ids}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeState(w)
}

type acceptReq struct {
	Group string `json:"group"`
}

func (s *Server) handleAcceptInbox(w http.ResponseWriter, r *http.Request) {
	var req acceptReq
	if !decode(w, r, &req) {
		return
	}
	if _, err := s.store.AcceptInbox(r.PathValue("id"), strings.TrimSpace(req.Group)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

func (s *Server) handleRejectInbox(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RejectInbox(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeState(w)
}

type saveLensReq struct {
	// ItemIDs to store. If nil, the lens captures the current active
	// selection (the "save what I picked" flow).
	ItemIDs []string `json:"item_ids"`
}

func (s *Server) handleSaveLens(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" {
		writeErr(w, http.StatusBadRequest, "lens name required")
		return
	}
	var req saveLensReq
	if !decode(w, r, &req) {
		return
	}
	ids := req.ItemIDs
	if ids == nil {
		active, err := s.store.LoadActive()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		ids = active.ItemIDs
	}
	if ids == nil {
		ids = []string{}
	}
	ls, err := s.store.LoadLenses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	ls = vault.UpsertLens(ls, vault.Lens{Name: name, ItemIDs: ids})
	if err := s.store.SaveLenses(ls); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeState(w)
}

// handleUseLens mirrors the MCP use_lens tool: activate the lens, dropping
// ids of since-deleted notes so we never persist dangling references.
func (s *Server) handleUseLens(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	ls, err := s.store.LoadLenses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lens, ok := vault.FindLens(ls, name)
	if !ok {
		writeErr(w, http.StatusNotFound, "lens not found: "+name)
		return
	}
	live := make([]string, 0, len(lens.ItemIDs))
	for _, id := range lens.ItemIDs {
		if _, ok := s.store.Get(id); ok {
			live = append(live, id)
		}
	}
	if err := s.store.SaveActive(vault.Active{ActiveLens: lens.Name, ItemIDs: live}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeState(w)
}

func (s *Server) handleDeleteLens(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	ls, err := s.store.LoadLenses()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	ls = vault.RemoveLens(ls, name)
	if err := s.store.SaveLenses(ls); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// If the active selection was riding this lens, detach the label so the
	// UI stops claiming a lens that no longer exists (ids are left intact).
	if active, err := s.store.LoadActive(); err == nil && active.ActiveLens == name {
		active.ActiveLens = ""
		_ = s.store.SaveActive(active)
	}
	s.writeState(w)
}

// --- helpers --------------------------------------------------------------

func (s *Server) writeState(w http.ResponseWriter) {
	state, err := buildState(s.store)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// decode reads a JSON request body into v. An empty body is legal — it
// leaves v at its zero value — because several endpoints have all-optional
// fields (e.g. save-lens capturing the current active selection).
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		return true
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		writeErr(w, http.StatusBadRequest, "bad JSON: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func cleanTags(in []string) []string {
	out := make([]string, 0, len(in))
	for _, t := range in {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// spaHandler serves the embedded static build, falling back to index.html
// for any path that isn't a real asset (so a refresh on a client route still
// loads the app).
func (s *Server) spaHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// Should never happen — the embed path is a constant.
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	index, _ := fs.ReadFile(sub, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if st, err := fs.Stat(sub, p); err != nil || st.IsDir() {
			if len(index) == 0 {
				http.Error(w, "web UI not built (run `npm run build` in webapp/)", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
