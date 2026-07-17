package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cleanGroup normalizes and validates a group path, rejecting anything that
// would escape the vault or collide with the managed pickmem/ dir.
func cleanGroup(group string) (string, error) {
	g := strings.Trim(strings.TrimSpace(group), "/")
	if g == "" {
		return "", errors.New("empty group")
	}
	clean := filepath.ToSlash(filepath.Clean(g))
	if clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("invalid group path: %s", group)
	}
	if clean == PickmemDir || strings.HasPrefix(clean, PickmemDir+"/") {
		return "", fmt.Errorf("group cannot live under %s/", PickmemDir)
	}
	return clean, nil
}

// RenameGroup moves a group to a new path, taking every note filed under it
// (and its subgroups) along. Each affected note's `group` frontmatter is
// rewritten and its file relocated via EditNote, so the create-only edit
// guard still applies. Empty subfolders are moved/cleaned up too, so a group
// with no notes can still be renamed. Returns the number of notes moved.
func (s *Store) RenameGroup(from, to string) (int, error) {
	fromG, err := cleanGroup(from)
	if err != nil {
		return 0, err
	}
	toG, err := cleanGroup(to)
	if err != nil {
		return 0, err
	}
	if fromG == toG {
		return 0, nil
	}
	if toG == fromG+"/" || strings.HasPrefix(toG, fromG+"/") {
		return 0, fmt.Errorf("cannot move a group into itself")
	}

	moved := 0
	for _, n := range s.ListActive() {
		if n.Group != fromG && !strings.HasPrefix(n.Group, fromG+"/") {
			continue
		}
		newGroup := toG + n.Group[len(fromG):]
		if _, err := s.EditNote(n.ID, NoteEdit{
			Label: n.Label,
			Group: newGroup,
			Body:  n.Body,
			Type:  n.Kind(),
			Tags:  n.Tags,
		}); err != nil {
			return moved, fmt.Errorf("move %q: %w", n.Label, err)
		}
		moved++
	}

	// Repoint any routing rules that target the old group, so a renamed group
	// keeps its import routing and doesn't leave a ghost entry in the
	// taxonomy (KnownGroups unions routing-rule targets).
	if cfg, err := s.LoadConfig(); err == nil {
		changed := false
		for i, r := range cfg.RoutingRules {
			if r.Group == fromG || strings.HasPrefix(r.Group, fromG+"/") {
				cfg.RoutingRules[i].Group = toG + r.Group[len(fromG):]
				changed = true
			}
		}
		if changed {
			_ = s.SaveConfig(cfg)
		}
	}

	// Relocate any physical folder left behind: the whole tree when the group
	// had no notes, or just the now-empty dirs after the notes moved out.
	oldDir := filepath.Join(s.Root, filepath.FromSlash(fromG))
	newDir := filepath.Join(s.Root, filepath.FromSlash(toG))
	if info, statErr := os.Stat(oldDir); statErr == nil && info.IsDir() {
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(newDir), 0o755); err != nil {
				return moved, err
			}
			if err := os.Rename(oldDir, newDir); err != nil {
				return moved, fmt.Errorf("rename folder: %w", err)
			}
		} else {
			removeEmptyDirs(oldDir)
		}
	}
	return moved, nil
}

// DeleteGroup deletes every note filed under a group (and its subgroups) and
// removes the group's folder tree. Only PickMem-tracked notes are removed
// from the index; the RemoveAll clears the folder on disk. Returns the number
// of notes deleted.
func (s *Store) DeleteGroup(group string) (int, error) {
	g, err := cleanGroup(group)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, n := range s.ListActive() {
		if n.Group == g || strings.HasPrefix(n.Group, g+"/") {
			if err := s.Remove(n.ID); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	dir := filepath.Join(s.Root, filepath.FromSlash(g))
	if err := os.RemoveAll(dir); err != nil {
		return deleted, fmt.Errorf("remove group folder: %w", err)
	}

	// Drop routing rules that target the deleted group, so it fully
	// disappears from the taxonomy instead of lingering as an empty entry
	// (KnownGroups unions routing-rule targets).
	if cfg, err := s.LoadConfig(); err == nil {
		kept := cfg.RoutingRules[:0]
		changed := false
		for _, r := range cfg.RoutingRules {
			if r.Group == g || strings.HasPrefix(r.Group, g+"/") {
				changed = true
				continue
			}
			kept = append(kept, r)
		}
		if changed {
			cfg.RoutingRules = kept
			_ = s.SaveConfig(cfg)
		}
	}
	return deleted, nil
}

// removeEmptyDirs deletes empty directories bottom-up starting at root. A
// directory holding only a `.gitkeep` placeholder (the starter template ships
// these to keep empty group folders in git) counts as empty and is removed
// along with the placeholder. A directory with any real file (e.g. the user's
// own non-PickMem notes) is left in place.
func removeEmptyDirs(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			removeEmptyDirs(filepath.Join(root, e.Name()))
		}
	}
	entries, err = os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() != ".gitkeep" {
			return // holds real content — keep it
		}
	}
	for _, e := range entries { // only .gitkeep placeholders remain
		_ = os.Remove(filepath.Join(root, e.Name()))
	}
	_ = os.Remove(root)
}
