package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AddInbox stages a new pending note under pickmem/inbox/. If the note has
// no id/created_at yet, they're filled in. Status is forced to pending.
// The file is created; if a slug collision occurs the id suffix is appended.
func (s *Store) AddInbox(n *Note) (*Note, error) {
	if n.ID == "" {
		n.ID = NewID()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	if n.Source == "" {
		n.Source = SourceManual
	}
	n.Status = StatusPending

	inbox := filepath.Join(s.Root, InboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir inbox: %w", err)
	}
	stem := Slugify(n.Label)
	rel := filepath.Join(InboxDir, stem+".md")
	full := filepath.Join(s.Root, rel)
	if _, err := os.Stat(full); err == nil {
		// slug collision -> disambiguate with a short id suffix
		suffix := n.ID
		if len(suffix) > 6 {
			suffix = suffix[len(suffix)-6:]
		}
		rel = filepath.Join(InboxDir, stem+"-"+suffix+".md")
		full = filepath.Join(s.Root, rel)
	}
	n.RelPath = filepath.ToSlash(rel)

	data, err := n.Serialize()
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(full, data, 0o644); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.notes[n.ID] = n
	s.tracked[n.ID] = trackFor(full, data)
	s.mu.Unlock()
	return n, nil
}

// AcceptInbox moves a pending note out of pickmem/inbox/ into its group
// folder and flips status to active. Group defaults to the note's
// SuggestedGroup if the caller passes an empty string.
func (s *Store) AcceptInbox(id, group string) (*Note, error) {
	s.mu.Lock()
	n, ok := s.notes[id]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("note %s not found", id)
	}
	if n.Status != StatusPending {
		return nil, fmt.Errorf("note %s is not pending (status=%s)", id, n.Status)
	}
	if group == "" {
		group = n.SuggestedGroup
	}
	if group == "" {
		return nil, fmt.Errorf("no group provided and no suggested_group on note")
	}
	oldFull := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	// Flip fields before serialize so the on-disk representation reflects
	// the accepted state (status=active, group=<accepted>, no suggested_group).
	n.Group = group
	n.Status = StatusActive
	n.SuggestedGroup = ""

	stem := Slugify(n.Label)
	newRel, err := s.uniquePathFor(n.ID, group, stem)
	if err != nil {
		return nil, err
	}
	newFull := filepath.Join(s.Root, newRel)
	if err := os.MkdirAll(filepath.Dir(newFull), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir group: %w", err)
	}
	data, err := n.Serialize()
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(newFull, data, 0o644); err != nil {
		return nil, err
	}
	// Only remove the old inbox file after the new one is safely written.
	if err := os.Remove(oldFull); err != nil {
		return nil, fmt.Errorf("remove inbox file: %w", err)
	}
	n.RelPath = filepath.ToSlash(newRel)
	s.mu.Lock()
	s.tracked[n.ID] = trackFor(newFull, data)
	s.mu.Unlock()
	return n, nil
}

// RejectInbox deletes a pending note. Only pending notes are eligible so
// this can never destroy accepted memory.
func (s *Store) RejectInbox(id string) error {
	s.mu.Lock()
	n, ok := s.notes[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("note %s not found", id)
	}
	if n.Status != StatusPending {
		return fmt.Errorf("only pending notes can be rejected (status=%s)", n.Status)
	}
	full := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("remove inbox file: %w", err)
	}
	s.mu.Lock()
	delete(s.notes, id)
	delete(s.tracked, id)
	s.mu.Unlock()
	return nil
}
