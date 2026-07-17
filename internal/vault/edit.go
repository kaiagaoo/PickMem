package vault

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NoteEdit carries the mutable fields a management UI (the web app or a
// future CLI editor) may change on an existing active note. Callers send the
// full desired value of each field, not a delta: Label/Group/Body replace
// outright, and Tags replaces the whole slice.
type NoteEdit struct {
	Label string
	Group string
	Body  string
	Tags  []string
}

// EditNote rewrites an existing active note in place, the one place PickMem
// deliberately steps past its create-only default (the web app is a
// sanctioned editing surface). It still honors the same guard Update uses:
// it refuses to write if the file changed on disk since we last indexed it,
// so a concurrent Obsidian/CLI edit is never silently clobbered — callers
// should Reload() immediately before to pick up outside changes.
//
// When the group changes, the note's file moves to the new group folder
// (write-new-then-remove-old, mirroring AcceptInbox). A label change alone
// does NOT rename the file: the filename is a cosmetic slug, and renaming
// would churn the vault and break Obsidian links for no functional gain.
func (s *Store) EditNote(id string, e NoteEdit) (*Note, error) {
	if strings.TrimSpace(e.Label) == "" {
		return nil, errors.New("note requires a label")
	}
	if strings.TrimSpace(e.Group) == "" {
		return nil, errors.New("note requires a group")
	}

	s.mu.Lock()
	n, ok := s.notes[id]
	prevHash := s.tracked[id]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("note %s not found", id)
	}
	if n.Status != StatusActive {
		return nil, fmt.Errorf("only active notes can be edited (status=%s)", n.Status)
	}

	oldRel := n.RelPath
	oldFull := filepath.Join(s.Root, filepath.FromSlash(oldRel))
	current, err := os.ReadFile(oldFull)
	if err != nil {
		return nil, fmt.Errorf("read note for edit: %w", err)
	}
	if sha256.Sum256(current) != prevHash {
		return nil, fmt.Errorf("refusing to overwrite %s: file changed on disk since load (edit the outside change first, then retry)", oldRel)
	}

	// Mutate a copy so a mid-flight failure leaves the in-memory index
	// exactly as it was.
	updated := *n
	updated.Label = e.Label
	updated.Group = e.Group
	updated.Body = e.Body
	if len(e.Tags) == 0 {
		updated.Tags = nil
	} else {
		updated.Tags = append([]string(nil), e.Tags...)
	}

	newRel := oldRel
	moved := false
	if GroupToPath(e.Group) != GroupToPath(n.Group) {
		nr, err := s.uniquePathFor(id, e.Group, Slugify(e.Label))
		if err != nil {
			return nil, err
		}
		newRel = filepath.ToSlash(nr)
		moved = true
	}
	updated.RelPath = newRel
	newFull := filepath.Join(s.Root, filepath.FromSlash(newRel))

	data, err := updated.Serialize()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(newFull), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir group: %w", err)
	}
	if err := writeFileAtomic(newFull, data, 0o644); err != nil {
		return nil, err
	}
	// Only drop the old file once the new one is safely on disk.
	if moved {
		if err := os.Remove(oldFull); err != nil {
			return nil, fmt.Errorf("remove old note file: %w", err)
		}
	}

	s.mu.Lock()
	s.notes[id] = &updated
	s.tracked[id] = sha256.Sum256(data)
	s.mu.Unlock()
	return &updated, nil
}
