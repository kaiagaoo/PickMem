package templates

import (
	"strings"
	"testing"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

func TestSeedNotesCreatesOnePerEntry(t *testing.T) {
	dir := t.TempDir()
	if err := Apply(DefaultName, dir); err != nil {
		t.Fatal(err)
	}
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	n, err := SeedNotes(s)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(StarterNotes) {
		t.Errorf("seeded %d, want %d", n, len(StarterNotes))
	}
	active := s.ListActive()
	if len(active) != len(StarterNotes) {
		t.Fatalf("vault has %d active notes, want %d", len(active), len(StarterNotes))
	}
	for _, note := range active {
		if len(note.Tags) != 1 || note.Tags[0] != StarterTag {
			t.Errorf("%s missing the %q tag: %v", note.Label, StarterTag, note.Tags)
		}
		if !strings.Contains(note.Body, "____") {
			t.Errorf("%s has no blank to fill: %q", note.Label, note.Body)
		}
	}
}

func TestSeedNotesIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SeedNotes(s); err != nil {
		t.Fatal(err)
	}
	// The user fills one in (body change keeps group+label).
	filled := s.ListActive()[0]
	if _, err := s.Update(filled.ID, func(n *vault.Note) error {
		n.Body = "Name: Kaia\nBased in: SF"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Re-seeding (init --force) must create nothing and not touch the
	// filled-in note.
	n, err := SeedNotes(s)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("re-seed created %d notes, want 0", n)
	}
	got, _ := s.Get(filled.ID)
	if !strings.Contains(got.Body, "Kaia") {
		t.Errorf("re-seed clobbered a filled note: %q", got.Body)
	}
}

// Every starter note must land in a folder the starter taxonomy actually
// creates — otherwise seeding invents groups the vault README doesn't
// document.
func TestStarterNotesMatchStarterTaxonomy(t *testing.T) {
	dir := t.TempDir()
	if err := Apply(DefaultName, dir); err != nil {
		t.Fatal(err)
	}
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, g := range s.KnownGroups() {
		known[g] = true
	}
	seen := map[string]bool{}
	for _, sn := range StarterNotes {
		if !known[sn.Group] {
			t.Errorf("starter note %q targets group %q, which the starter taxonomy doesn't create", sn.Label, sn.Group)
		}
		key := sn.Group + "/" + sn.Label
		if seen[key] {
			t.Errorf("duplicate starter note %q", key)
		}
		seen[key] = true
		if sn.Label == "" || sn.Body == "" {
			t.Errorf("starter note in %q has empty label or body", sn.Group)
		}
	}
}
