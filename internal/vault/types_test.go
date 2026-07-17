package vault

import (
	"reflect"
	"testing"
)

func TestSuggestedTagsDefaultsAndCustom(t *testing.T) {
	s := newVault(t)

	// A vault with no configured list uses the built-in defaults.
	if got := s.SuggestedTags(); !reflect.DeepEqual(got, DefaultSuggestedTags()) {
		t.Errorf("default SuggestedTags = %v, want %v", got, DefaultSuggestedTags())
	}

	// A custom list is honored; duplicates and blanks are dropped, order kept.
	cfg, _ := s.LoadConfig()
	cfg.SuggestedTags = []string{"task", "fact", "", "task", "question"}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	want := []string{"task", "fact", "question"}
	if got := s.SuggestedTags(); !reflect.DeepEqual(got, want) {
		t.Errorf("custom SuggestedTags = %v, want %v", got, want)
	}
}

// TestLegacyNoteTypesMigrateToSuggestedTags proves an old vault's `note_types`
// config carries over as suggested tags, and is dropped from config on save.
func TestLegacyNoteTypesMigrateToSuggestedTags(t *testing.T) {
	s := newVault(t)
	cfg, _ := s.LoadConfig()
	cfg.LegacyNoteTypes = []string{"idea", "task"}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if got := s.SuggestedTags(); !reflect.DeepEqual(got, []string{"idea", "task"}) {
		t.Errorf("migrated SuggestedTags = %v, want [idea task]", got)
	}
	// After a load+save the legacy key is gone and the value lives under
	// suggested_tags.
	reloaded, _ := s.LoadConfig()
	if len(reloaded.LegacyNoteTypes) != 0 {
		t.Errorf("legacy note_types not cleared: %v", reloaded.LegacyNoteTypes)
	}
}

// TestLegacyTypeFoldsIntoTags proves a note file that still has a `type:` line
// migrates that value into the tag list on parse (dropping the old fact
// default), so no data is lost and the field quietly disappears on next save.
func TestLegacyTypeFoldsIntoTags(t *testing.T) {
	legacy := []byte("---\n" +
		"id: 01JABCDEFGHJKMNPQRSTVWXYZ0\n" +
		"label: solo sail\n" +
		"group: projects\n" +
		"type: idea\n" +
		"tags:\n  - sailing\n" +
		"source: manual\n" +
		"status: active\n" +
		"created_at: 2026-01-01T00:00:00Z\n" +
		"---\n\nwants to try a solo overnight sail\n")
	n, err := ParseNote(legacy)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if !reflect.DeepEqual(n.Tags, []string{"idea", "sailing"}) {
		t.Errorf("legacy type not folded into tags: %v", n.Tags)
	}
	// A fact type is the old default and carries no information — dropped.
	factNote := []byte("---\nid: 01JABCDEFGHJKMNPQRSTVWXYZ1\nlabel: x\ngroup: misc\ntype: fact\nsource: manual\nstatus: active\ncreated_at: 2026-01-01T00:00:00Z\n---\n\nbody\n")
	fn, err := ParseNote(factNote)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	if len(fn.Tags) != 0 {
		t.Errorf("fact type should not become a tag: %v", fn.Tags)
	}
}
