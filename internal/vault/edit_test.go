package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditNoteInPlace(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "salary", Group: "finance/income"},
		Body:        "monthly base $8k",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	origPath := n.RelPath

	edited, err := s.EditNote(n.ID, NoteEdit{
		Label: "salary",
		Group: "finance/income",
		Body:  "monthly base $9k + bonus",
		Tags:  []string{"idea", "money"},
	})
	if err != nil {
		t.Fatalf("EditNote: %v", err)
	}
	// Body/tags changed; a same-group, same-label edit keeps the file.
	if edited.RelPath != origPath {
		t.Errorf("in-place edit moved the file: %s -> %s", origPath, edited.RelPath)
	}
	if edited.Body != "monthly base $9k + bonus" {
		t.Errorf("body not updated: %q", edited.Body)
	}
	if strings.Join(edited.Tags, ",") != "idea,money" {
		t.Errorf("tags not updated: %v", edited.Tags)
	}

	// Re-open from disk to prove the write landed.
	s2, err := Open(s.Root)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, ok := s2.Get(n.ID)
	if !ok {
		t.Fatal("edited note missing after reopen")
	}
	// Parsed bodies keep a trailing newline (existing round-trip behavior).
	if strings.TrimSpace(got.Body) != "monthly base $9k + bonus" || strings.Join(got.Tags, ",") != "idea,money" {
		t.Errorf("on-disk note not updated: body=%q tags=%v", got.Body, got.Tags)
	}
}

func TestEditNoteMovesOnGroupChange(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "vim", Group: "about/preferences"},
		Body:        "uses vim",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	oldFull := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))

	edited, err := s.EditNote(n.ID, NoteEdit{
		Label: "vim",
		Group: "work/tools",
		Body:  "uses vim",
	})
	if err != nil {
		t.Fatalf("EditNote: %v", err)
	}
	if edited.Group != "work/tools" {
		t.Errorf("group not updated: %q", edited.Group)
	}
	if _, err := os.Stat(oldFull); !os.IsNotExist(err) {
		t.Errorf("old file still present after group move: %v", err)
	}
	newFull := filepath.Join(s.Root, filepath.FromSlash(edited.RelPath))
	if _, err := os.Stat(newFull); err != nil {
		t.Errorf("new file missing after group move: %v", err)
	}
	if filepath.Dir(edited.RelPath) != "work/tools" {
		t.Errorf("moved note not under new group folder: %s", edited.RelPath)
	}
}

func TestEditNoteRefusesOutsideChange(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "x", Group: "misc"},
		Body:        "original",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Simulate an Obsidian edit that the store hasn't reindexed.
	full := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	if err := os.WriteFile(full, []byte("---\nid: "+n.ID+"\nlabel: x\ngroup: misc\nsource: manual\nstatus: active\ncreated_at: 2026-01-01T00:00:00Z\n---\n\ntouched outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditNote(n.ID, NoteEdit{Label: "x", Group: "misc", Body: "clobber"}); err == nil {
		t.Fatal("expected EditNote to refuse a note changed on disk")
	}
}
