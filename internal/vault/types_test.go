package vault

import (
	"reflect"
	"testing"
)

func TestNormalizeTypePreservesCustom(t *testing.T) {
	cases := map[string]string{
		"":         TypeFact, // empty defaults to fact
		"  ":       TypeFact, // whitespace-only too
		"fact":     TypeFact,
		"idea":     TypeIdea,
		"task":     "task",     // user-defined type is preserved
		" task ":   "task",     // trimmed
		"question": "question", // preserved, not coerced to fact
	}
	for in, want := range cases {
		if got := NormalizeType(in); got != want {
			t.Errorf("NormalizeType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenameNoteType(t *testing.T) {
	s := newVault(t)
	// A note typed "idea" should follow a rename to "concept".
	n, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "solo sail", Group: "projects", Type: TypeIdea}, Body: "b"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.RenameNoteType("idea", "concept")
	if err != nil {
		t.Fatalf("RenameNoteType: %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}
	if got, _ := s.Get(n.ID); got.Kind() != "concept" {
		t.Errorf("note type = %q, want concept", got.Kind())
	}
	types := s.NoteTypes()
	has := func(x string) bool {
		for _, t := range types {
			if t == x {
				return true
			}
		}
		return false
	}
	if has("idea") || !has("concept") {
		t.Errorf("vocabulary not updated: %v", types)
	}

	// Guards: can't rename the default, and can't collide with an existing type.
	if _, err := s.RenameNoteType("fact", "core"); err == nil {
		t.Error("expected error renaming the default type")
	}
	if _, err := s.RenameNoteType("concept", "thought"); err == nil {
		t.Error("expected error renaming onto an existing type")
	}
}

func TestNoteTypesDefaultsAndCustom(t *testing.T) {
	s := newVault(t)

	// A vault with no configured types uses the built-in defaults.
	if got := s.NoteTypes(); !reflect.DeepEqual(got, DefaultNoteTypes()) {
		t.Errorf("default NoteTypes = %v, want %v", got, DefaultNoteTypes())
	}

	// A custom list is honored, but fact is always guaranteed first and
	// duplicates/blanks are dropped.
	cfg, _ := s.LoadConfig()
	cfg.NoteTypes = []string{"task", "fact", "", "task", "question"}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	want := []string{"fact", "task", "question"}
	if got := s.NoteTypes(); !reflect.DeepEqual(got, want) {
		t.Errorf("custom NoteTypes = %v, want %v", got, want)
	}
}
