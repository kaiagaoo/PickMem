package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameGroupMovesNotesAndSubgroups(t *testing.T) {
	s := newVault(t)
	mustAdd := func(label, group string) *Note {
		n, err := s.Add(&Note{Frontmatter: Frontmatter{Label: label, Group: group}, Body: "b"})
		if err != nil {
			t.Fatalf("add %s: %v", label, err)
		}
		return n
	}
	a := mustAdd("salary", "finance/income")
	b := mustAdd("rent", "finance/bills")
	c := mustAdd("vim", "about/preferences") // unrelated, must not move

	moved, err := s.RenameGroup("finance", "money")
	if err != nil {
		t.Fatalf("RenameGroup: %v", err)
	}
	if moved != 2 {
		t.Errorf("moved = %d, want 2", moved)
	}
	if n, _ := s.Get(a.ID); n.Group != "money/income" {
		t.Errorf("a.group = %q, want money/income", n.Group)
	}
	if n, _ := s.Get(b.ID); n.Group != "money/bills" {
		t.Errorf("b.group = %q, want money/bills", n.Group)
	}
	if n, _ := s.Get(c.ID); n.Group != "about/preferences" {
		t.Errorf("c.group changed to %q", n.Group)
	}
	// Old folder tree should be gone; new one present.
	if _, err := os.Stat(filepath.Join(s.Root, "finance")); !os.IsNotExist(err) {
		t.Errorf("old finance/ folder still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "money", "income")); err != nil {
		t.Errorf("new money/income folder missing: %v", err)
	}
}

func TestRenameEmptyGroup(t *testing.T) {
	s := newVault(t)
	if err := s.EnsureGroup("hobbies/music"); err != nil {
		t.Fatal(err)
	}
	moved, err := s.RenameGroup("hobbies", "interests")
	if err != nil {
		t.Fatalf("RenameGroup: %v", err)
	}
	if moved != 0 {
		t.Errorf("moved = %d, want 0", moved)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "interests", "music")); err != nil {
		t.Errorf("renamed empty folder missing: %v", err)
	}
}

func TestRenameGroupRejectsMoveIntoSelf(t *testing.T) {
	s := newVault(t)
	if _, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "x", Group: "work"}, Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenameGroup("work", "work/sub"); err == nil {
		t.Fatal("expected error moving a group into itself")
	}
}

func TestDeleteGroupPrunesRoutingRules(t *testing.T) {
	s := newVault(t)
	if err := s.SaveConfig(Config{
		Version: 1,
		RoutingRules: []RoutingRule{
			{Keyword: "salary", Group: "finance/income"},
			{Keyword: "vim", Group: "about/preferences"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteGroup("finance"); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	cfg, _ := s.LoadConfig()
	for _, r := range cfg.RoutingRules {
		if r.Group == "finance/income" {
			t.Error("routing rule for deleted group survived")
		}
	}
	// The deleted group should no longer appear in the taxonomy.
	for _, g := range s.KnownGroups() {
		if g == "finance/income" {
			t.Error("deleted group still in KnownGroups via routing rule")
		}
	}
}

func TestRenameGroupRepointsRoutingRules(t *testing.T) {
	s := newVault(t)
	if err := s.SaveConfig(Config{
		Version:      1,
		RoutingRules: []RoutingRule{{Keyword: "salary", Group: "finance/income"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RenameGroup("finance", "money"); err != nil {
		t.Fatalf("RenameGroup: %v", err)
	}
	cfg, _ := s.LoadConfig()
	if len(cfg.RoutingRules) != 1 || cfg.RoutingRules[0].Group != "money/income" {
		t.Errorf("routing rule not repointed: %+v", cfg.RoutingRules)
	}
}

func TestDeleteGroupRemovesNotesAndFolder(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "salary", Group: "finance/income"}, Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	keep, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "vim", Group: "about"}, Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := s.DeleteGroup("finance")
	if err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	if _, ok := s.Get(n.ID); ok {
		t.Error("deleted note still in index")
	}
	if _, ok := s.Get(keep.ID); !ok {
		t.Error("unrelated note was removed")
	}
	if _, err := os.Stat(filepath.Join(s.Root, "finance")); !os.IsNotExist(err) {
		t.Errorf("finance/ folder still exists: %v", err)
	}
}
