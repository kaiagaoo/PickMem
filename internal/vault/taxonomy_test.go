package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaxonomyGroupsFromFolders(t *testing.T) {
	dir := t.TempDir()
	s, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	// User curates taxonomy in Obsidian: create folders, including a
	// nested one and a couple that must be excluded.
	for _, d := range []string{
		"work/side-projects", // new category, no notes yet
		"reading",
		"_private/debt",   // private: excluded from the shared list
		".obsidian/cache", // hidden: excluded
	} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	groups := s.TaxonomyGroups()
	set := map[string]bool{}
	for _, g := range groups {
		set[g] = true
	}

	// Empty folders show up immediately — the whole point.
	if !set["work"] || !set["work/side-projects"] || !set["reading"] {
		t.Errorf("expected folder-created groups present, got %v", groups)
	}
	// The pickmem/ management dir is never a category.
	for _, g := range groups {
		if g == "pickmem" || strings.HasPrefix(g, "pickmem/") {
			t.Errorf("pickmem/ leaked into taxonomy: %q", g)
		}
	}
	// Private + hidden folders never leave the disk via the shared list.
	if set["_private"] || set["_private/debt"] {
		t.Errorf("_private folder exposed in taxonomy: %v", groups)
	}
	for _, g := range groups {
		if g == ".obsidian" || strings.HasPrefix(g, ".obsidian") {
			t.Errorf("hidden dir leaked into taxonomy: %q", g)
		}
	}
}

func TestKnownGroupsUnionsFoldersNotesAndRules(t *testing.T) {
	dir := t.TempDir()
	s, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	// A rule target whose folder doesn't exist yet.
	cfg, _ := s.LoadConfig()
	cfg.RoutingRules = []RoutingRule{{Keyword: "gym", Group: "about/health"}}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	// An empty folder the user made in Obsidian (no notes in it).
	if err := os.MkdirAll(filepath.Join(dir, "career"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A note filed into a group that has no folder of its own.
	if _, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "sailboat", Group: "hobbies/sailing"},
		Body:        "keeps a sailboat",
	}); err != nil {
		t.Fatal(err)
	}

	known := map[string]bool{}
	for _, g := range s.KnownGroups() {
		known[g] = true
	}
	if !known["about/health"] {
		t.Error("rule target missing from KnownGroups")
	}
	if !known["hobbies/sailing"] {
		t.Error("note group missing from KnownGroups")
	}
	// An empty folder still counts (folder-sourced) — the whole point.
	if !known["career"] {
		t.Error("empty folder group missing from KnownGroups")
	}
}
