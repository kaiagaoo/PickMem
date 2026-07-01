package ingest

import (
	"reflect"
	"strings"
	"testing"
)

// ---------- JSON ----------

func TestParseJSONBareStringArray(t *testing.T) {
	in := []byte(`["I like coffee dark roast, no sugar.","The kickoff meeting with Acme is on Aug 1."]`)
	got := Parse(in, FormatJSON)
	want := []string{
		"I like coffee dark roast, no sugar.",
		"The kickoff meeting with Acme is on Aug 1.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseJSONArrayOfObjects(t *testing.T) {
	in := []byte(`[
      {"memory": "moved to Portland in 2024"},
      {"text": "prefers meetings in the morning"},
      {"content": "avoids seafood — allergic to shellfish"}
    ]`)
	got := Parse(in, FormatJSON)
	if len(got) != 3 {
		t.Errorf("expected 3 memories, got %d: %v", len(got), got)
	}
	for _, s := range got {
		if len(s) < MinLen {
			t.Errorf("short candidate leaked: %q", s)
		}
	}
}

func TestParseJSONMemoriesWrapper(t *testing.T) {
	in := []byte(`{"memories": ["salary is monthly base $8k","favorite editor is vim"]}`)
	got := Parse(in, FormatJSON)
	if len(got) != 2 {
		t.Errorf("expected 2 memories, got %d: %v", len(got), got)
	}
}

// ---------- bullets ----------

func TestParseBulletsMultipleMarkers(t *testing.T) {
	in := `# My memories

- I like coffee dark roast, no sugar.
* Salary is monthly base $8k.
+ The kickoff meeting with Acme is on Aug 1.
1. Prefers meetings in the morning.
2) Avoids seafood — shellfish allergy.
`
	got := Parse([]byte(in), FormatBullets)
	if len(got) != 5 {
		t.Errorf("expected 5 bullets, got %d: %v", len(got), got)
	}
}

func TestParseBulletsFoldsContinuationLines(t *testing.T) {
	in := `- First memory, which continues
  on the next line without a marker.
- Second memory.
`
	got := Parse([]byte(in), FormatBullets)
	if len(got) != 2 {
		t.Fatalf("expected 2 bullets, got %d: %v", len(got), got)
	}
	if !strings.Contains(got[0], "continues on the next line") {
		t.Errorf("continuation not folded: %q", got[0])
	}
}

// ---------- auto ----------

func TestParseAutoRoutesToJSON(t *testing.T) {
	got := Parse([]byte(`["short one dropped","a memory that is definitely long enough to keep"]`), FormatAuto)
	// First entry ("short one dropped") is 17 chars; it survives. But
	// something shorter would drop.
	if len(got) < 1 {
		t.Errorf("auto→json missed items: %v", got)
	}
}

func TestParseAutoRoutesToBullets(t *testing.T) {
	in := `- one memory long enough to keep
- another memory long enough to keep
- third memory long enough to keep
`
	got := Parse([]byte(in), FormatAuto)
	if len(got) != 3 {
		t.Errorf("auto→bullets got %d, want 3: %v", len(got), got)
	}
}

func TestParseAutoRoutesToParagraphs(t *testing.T) {
	in := `First paragraph is here. It is a memory.

Second paragraph. Also a memory.

Third paragraph, also fine.`
	got := Parse([]byte(in), FormatAuto)
	if len(got) != 3 {
		t.Errorf("auto→paragraphs got %d, want 3: %v", len(got), got)
	}
}

func TestParseAutoFallsThroughOnMalformedJSON(t *testing.T) {
	// Looks like JSON but isn't — the parser must gracefully retry as
	// text so a user with a broken export doesn't get zero results.
	in := `["broken
- one memory long enough to keep
- another memory long enough
`
	got := Parse([]byte(in), FormatAuto)
	// We should end up with the bullet items (or paragraph fallback).
	if len(got) == 0 {
		t.Errorf("auto failed to recover from malformed JSON: %v", got)
	}
}

// ---------- format resolution ----------

func TestResolveFormat(t *testing.T) {
	cases := map[string]Format{
		"":           FormatAuto,
		"json":       FormatJSON,
		"JSON":       FormatJSON,
		"bullets":    FormatBullets,
		"paragraphs": FormatParagraphs,
		"auto":       FormatAuto,
		"chatgpt":    FormatAuto, // alias
		"claude":     FormatAuto,
		"list":       FormatAuto,
		"garbage":    FormatAuto, // unknown flags collapse to auto
	}
	for in, want := range cases {
		if got := ResolveFormat(in); got != want {
			t.Errorf("ResolveFormat(%q) = %q, want %q", in, got, want)
		}
	}
}

// ---------- filter ----------

func TestShortCandidatesDropped(t *testing.T) {
	in := []byte(`["ok","yes","hi","this one is long enough to keep"]`)
	got := Parse(in, FormatJSON)
	if len(got) != 1 || got[0] != "this one is long enough to keep" {
		t.Errorf("filter did not drop shorts: %v", got)
	}
}
