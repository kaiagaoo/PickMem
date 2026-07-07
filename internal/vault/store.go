package vault

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store is the in-memory index over a vault directory. All CRUD goes
// through here so the create-only invariant (see EXECUTION.md §4) is
// enforced in one place.
type Store struct {
	// Root is the absolute path to the vault directory.
	Root string

	mu       sync.RWMutex
	notes    map[string]*Note    // id -> note (active and pending)
	tracked  map[string][32]byte // id -> sha256 of last-known bytes on disk
	warnings []string            // per-file problems found during the last reindex
}

// Open loads the vault at root. Missing pickmem/ subdirectories are treated
// as empty, so a freshly-initialized folder is valid.
func Open(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve vault path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat vault: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vault path is not a directory: %s", abs)
	}
	s := &Store{
		Root:    abs,
		notes:   map[string]*Note{},
		tracked: map[string][32]byte{},
	}
	if err := s.reindex(); err != nil {
		return nil, err
	}
	return s, nil
}

// Init scaffolds a fresh vault: creates pickmem/, pickmem/inbox/, and empty
// config/lenses/active files. Returns an already-loaded Store. If the
// vault already has a pickmem/ dir, Init is a no-op on those files and
// just returns Open(root).
func Init(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(abs, InboxDir), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir inbox: %w", err)
	}
	s := &Store{Root: abs, notes: map[string]*Note{}, tracked: map[string][32]byte{}}
	// Only write the machine-managed files if they don't already exist,
	// so re-running init on a populated vault is safe.
	if _, err := os.Stat(s.configPath()); errors.Is(err, fs.ErrNotExist) {
		if err := s.SaveConfig(DefaultConfig()); err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(s.lensesPath()); errors.Is(err, fs.ErrNotExist) {
		if err := s.SaveLenses(nil); err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(s.activePath()); errors.Is(err, fs.ErrNotExist) {
		if err := s.SaveActive(Active{ItemIDs: []string{}}); err != nil {
			return nil, err
		}
	}
	if err := s.reindex(); err != nil {
		return nil, err
	}
	return s, nil
}

// reindex walks the vault, parses every .md file with frontmatter, and
// rebuilds the id-keyed index. Files without frontmatter are silently
// skipped so a user's other Obsidian notes don't confuse PickMem.
//
// A file with a malformed frontmatter block (or a duplicate id) is
// skipped with a warning rather than failing the whole load — one
// half-typed note in Obsidian must not brick every command. Warnings
// accumulate in s.warnings (read via Warnings) so callers can surface
// them once. Only real I/O failures abort the walk.
func (s *Store) reindex() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notes = map[string]*Note{}
	s.tracked = map[string][32]byte{}
	s.warnings = nil

	return filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// Skip common hidden/system directories.
			if path != s.Root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		// Files without a `---` block get skipped, not errored — they're
		// the user's plain Obsidian notes.
		if !hasFrontmatter(data) {
			return nil
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		n, err := ParseNote(data)
		if err != nil {
			s.warnings = append(s.warnings, fmt.Sprintf("skipped %s: %v (fix or delete the file)", relSlash, err))
			return nil
		}
		n.RelPath = relSlash
		if existing, dup := s.notes[n.ID]; dup {
			s.warnings = append(s.warnings, fmt.Sprintf("skipped %s: duplicate id %s (already used by %s)", relSlash, n.ID, existing.RelPath))
			return nil
		}
		s.notes[n.ID] = n
		s.tracked[n.ID] = sha256.Sum256(data)
		return nil
	})
}

// Warnings returns the per-file problems found during the most recent
// (re)index: malformed frontmatter, duplicate ids. The affected files are
// skipped, not fatal — surface these to the user so they can fix them.
func (s *Store) Warnings() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.warnings))
	copy(out, s.warnings)
	return out
}

// Reload re-reads the vault from disk, discarding the in-memory index.
// Cheap enough to call after any external change (Obsidian edit, git
// checkout, etc.).
func (s *Store) Reload() error { return s.reindex() }

// Get returns a note by id.
func (s *Store) Get(id string) (*Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	return n, ok
}

// List returns all notes sorted by id (which is chronological for ULIDs).
// Callers should not mutate the returned slice's elements.
func (s *Store) List() []*Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Note, 0, len(s.notes))
	for _, n := range s.notes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListActive returns only status=active notes, sorted by id.
func (s *Store) ListActive() []*Note {
	all := s.List()
	out := all[:0]
	for _, n := range all {
		if n.Status == StatusActive {
			out = append(out, n)
		}
	}
	return out
}

// ListPending returns only status=pending notes, sorted by id.
func (s *Store) ListPending() []*Note {
	all := s.List()
	out := all[:0]
	for _, n := range all {
		if n.Status == StatusPending {
			out = append(out, n)
		}
	}
	return out
}

// Groups returns active notes bucketed by their frontmatter `group` field.
// Groups are sorted alphabetically; notes within each group by id.
func (s *Store) Groups() map[string][]*Note {
	out := map[string][]*Note{}
	for _, n := range s.ListActive() {
		out[n.Group] = append(out[n.Group], n)
	}
	return out
}

// Add creates a new active memory note in the group folder derived from
// n.Group. Fills in id/created_at if missing. Forces status=active. Fails
// if a note with the same id already exists.
func (s *Store) Add(n *Note) (*Note, error) {
	if n.Label == "" {
		return nil, errors.New("note requires a label")
	}
	if n.Group == "" {
		return nil, errors.New("note requires a group")
	}
	if n.ID == "" {
		n.ID = NewID()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	if n.Source == "" {
		n.Source = SourceManual
	}
	n.Status = StatusActive
	n.SuggestedGroup = ""

	s.mu.RLock()
	_, exists := s.notes[n.ID]
	s.mu.RUnlock()
	if exists {
		return nil, fmt.Errorf("id %s already exists", n.ID)
	}

	stem := Slugify(n.Label)
	rel, err := s.uniquePathFor(n.ID, n.Group, stem)
	if err != nil {
		return nil, err
	}
	full := filepath.Join(s.Root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir group: %w", err)
	}
	data, err := n.Serialize()
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(full, data, 0o644); err != nil {
		return nil, err
	}
	n.RelPath = filepath.ToSlash(rel)
	s.mu.Lock()
	s.notes[n.ID] = n
	s.tracked[n.ID] = sha256.Sum256(data)
	s.mu.Unlock()
	return n, nil
}

// Remove deletes the note file for the given id. Only removes files
// PickMem tracks; user-authored files without frontmatter aren't in the
// index and can never be reached here.
func (s *Store) Remove(id string) error {
	s.mu.RLock()
	n, ok := s.notes[id]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("note %s not found", id)
	}
	full := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("remove note file: %w", err)
	}
	s.mu.Lock()
	delete(s.notes, id)
	delete(s.tracked, id)
	s.mu.Unlock()
	return nil
}

// Update mutates a note through a caller-supplied function, then re-
// serializes it. Refuses to write if the on-disk file has changed since
// we last observed it (protects user edits — the create-only invariant).
// The updater must NOT change the note's ID or Status; those are managed
// explicitly by Add / Accept / Reject.
func (s *Store) Update(id string, mutate func(*Note) error) (*Note, error) {
	s.mu.Lock()
	n, ok := s.notes[id]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("note %s not found", id)
	}
	prevHash := s.tracked[id]
	s.mu.Unlock()

	full := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	current, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("read note for update: %w", err)
	}
	if sha256.Sum256(current) != prevHash {
		return nil, fmt.Errorf("refusing to overwrite %s: file changed on disk since load (create-only invariant)", n.RelPath)
	}
	origID, origStatus := n.ID, n.Status
	if err := mutate(n); err != nil {
		return nil, err
	}
	if n.ID != origID {
		return nil, errors.New("update must not change note id")
	}
	if n.Status != origStatus {
		return nil, errors.New("update must not change note status")
	}
	data, err := n.Serialize()
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(full, data, 0o644); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.tracked[id] = sha256.Sum256(data)
	s.mu.Unlock()
	return n, nil
}

// AbsPath returns the absolute filesystem path of the note (useful for
// `edit` which launches $EDITOR).
func (s *Store) AbsPath(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	if !ok {
		return "", false
	}
	return filepath.Join(s.Root, filepath.FromSlash(n.RelPath)), true
}

// uniquePathFor returns a vault-relative path for a note in the given
// group whose slug is `stem`. If the naïve path is taken, we append a
// short suffix from the id so writes never clobber an existing file.
func (s *Store) uniquePathFor(id, group, stem string) (string, error) {
	dir := GroupToPath(group)
	tryRel := filepath.Join(dir, stem+".md")
	if _, err := os.Stat(filepath.Join(s.Root, tryRel)); errors.Is(err, fs.ErrNotExist) {
		return tryRel, nil
	}
	// Collision — append the tail of the id (ULIDs sort by time, so the
	// tail is the random portion and stays short).
	suffix := id
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	tryRel = filepath.Join(dir, stem+"-"+strings.ToLower(suffix)+".md")
	if _, err := os.Stat(filepath.Join(s.Root, tryRel)); errors.Is(err, fs.ErrNotExist) {
		return tryRel, nil
	}
	// Extremely unlikely, but keep going with a longer suffix.
	return filepath.Join(dir, stem+"-"+strings.ToLower(id)+".md"), nil
}

// hasFrontmatter cheaply detects a leading `---\n` block without parsing
// the whole file.
func hasFrontmatter(data []byte) bool {
	trimmed := data
	// Optional UTF-8 BOM.
	if len(trimmed) >= 3 && trimmed[0] == 0xEF && trimmed[1] == 0xBB && trimmed[2] == 0xBF {
		trimmed = trimmed[3:]
	}
	return strings.HasPrefix(string(trimmed), "---\n") || strings.HasPrefix(string(trimmed), "---\r\n")
}

// trackFor returns the sha256 of the file at path. Wrapped as a helper so
// callers who just wrote the file don't need to re-read it.
func trackFor(_ string, data []byte) [32]byte { return sha256.Sum256(data) }

// writeFileAtomic writes to a temp file in the same directory, fsyncs it,
// then renames over the target. Prevents partial writes on crash.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".pickmem-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// writeJSONAtomic serializes v as pretty JSON and writes it atomically.
func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}
