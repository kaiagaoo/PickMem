package picker

import (
	"testing"

	"github.com/qwgao/pickmem/internal/vault"
)

func nestedFixtureNotes() []*vault.Note {
	mk := func(id, label, group string) *vault.Note {
		return &vault.Note{Frontmatter: vault.Frontmatter{ID: id, Label: label, Group: group}}
	}
	return []*vault.Note{
		mk("1", "chronic", "about/health"),
		mk("2", "ethnicity", "about/identity"),
		mk("3", "salary", "finance"),
	}
}

func TestBuildRowsNestsByPathSegment(t *testing.T) {
	rows := buildRows(nestedFixtureNotes())

	type got struct {
		kind  rowKind
		text  string // header segment, or note label
		depth int
	}
	var out []got
	for _, r := range rows {
		if r.kind == kindHeader {
			out = append(out, got{r.kind, r.header, r.depth})
		} else {
			out = append(out, got{r.kind, r.note.Label, r.depth})
		}
	}

	want := []got{
		{kindHeader, "about", 0},
		{kindHeader, "health", 1},
		{kindNote, "chronic", 2},
		{kindHeader, "identity", 1},
		{kindNote, "ethnicity", 2},
		{kindHeader, "finance", 0},
		{kindNote, "salary", 1},
	}
	if len(out) != len(want) {
		t.Fatalf("got %d rows, want %d:\n got  %+v\n want %+v", len(out), len(want), out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v\nfull: %+v", i, out[i], want[i], out)
		}
	}
}

func TestBuildRowsHeaderDescendantsSpanWholeSubtree(t *testing.T) {
	rows := buildRows(nestedFixtureNotes())
	for _, r := range rows {
		if r.kind == kindHeader && r.header == "about" {
			// "about" has two grandchildren notes (chronic, ethnicity),
			// no direct notes of its own.
			if len(r.descendants) != 2 {
				t.Errorf("about descendants = %v, want the 2 grandchild note ids", r.descendants)
			}
		}
		if r.kind == kindHeader && r.header == "health" {
			if len(r.descendants) != 1 || r.descendants[0] != "1" {
				t.Errorf("health descendants = %v, want [1]", r.descendants)
			}
		}
	}
}

func TestBuildRowsHeaderGroupIsFullPath(t *testing.T) {
	rows := buildRows(nestedFixtureNotes())
	found := map[string]bool{}
	for _, r := range rows {
		if r.kind == kindHeader {
			found[r.group] = true
		}
	}
	for _, want := range []string{"about", "about/health", "about/identity", "finance"} {
		if !found[want] {
			t.Errorf("missing header with full path %q; saw %v", want, found)
		}
	}
}

func TestBuildRowsFlatVaultIsSingleLevel(t *testing.T) {
	// No "/"-nested groups: every header at depth 0, every note at depth 1.
	notes := []*vault.Note{
		{Frontmatter: vault.Frontmatter{ID: "1", Label: "a", Group: "financial"}},
		{Frontmatter: vault.Frontmatter{ID: "2", Label: "b", Group: "health"}},
	}
	for _, r := range buildRows(notes) {
		if r.kind == kindHeader && r.depth != 0 {
			t.Errorf("flat header at non-zero depth: %+v", r)
		}
		if r.kind == kindNote && r.depth != 1 {
			t.Errorf("flat note at unexpected depth: %+v", r)
		}
	}
}

func TestRowIndentGrowsWithDepth(t *testing.T) {
	cases := map[int]string{0: "  ", 1: "    ", 2: "      "}
	for depth, want := range cases {
		if got := rowIndent(depth); got != want {
			t.Errorf("rowIndent(%d) = %q, want %q", depth, got, want)
		}
	}
}

func TestGroupStateTriState(t *testing.T) {
	sel := map[string]bool{"a": true}
	if s := groupState(sel, []string{"a", "b"}); s != checkSome {
		t.Errorf("one of two selected = %v, want checkSome", s)
	}
	if s := groupState(sel, []string{"a"}); s != checkAll {
		t.Errorf("only selected id = %v, want checkAll", s)
	}
	if s := groupState(sel, []string{"b", "c"}); s != checkNone {
		t.Errorf("none selected = %v, want checkNone", s)
	}
	if s := groupState(sel, nil); s != checkNone {
		t.Errorf("empty group = %v, want checkNone", s)
	}
}

func TestParentHeaderTogglesAllNestedNotes(t *testing.T) {
	// Toggling the top "about" header must select the grandchildren under
	// both about/health and about/identity, not just direct children.
	rows := buildRows(nestedFixtureNotes())
	var about row
	for _, r := range rows {
		if r.kind == kindHeader && r.header == "about" {
			about = r
		}
	}
	sel := map[string]bool{}
	if groupState(sel, about.descendants) != checkNone {
		t.Fatal("fixture should start unselected")
	}
	for _, id := range about.descendants {
		sel[id] = true
	}
	if !sel["1"] || !sel["2"] {
		t.Errorf("toggling 'about' did not select both nested notes: %v", sel)
	}
}
