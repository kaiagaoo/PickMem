package mcp

import (
	"testing"

	"github.com/qwgao/pickmem/internal/vault"
)

// These lock the exact byte format — mirror
// extension/test/assemble.test.ts fixture-for-fixture. If you change the
// format on either side, change both and keep the two suites in sync.

func newAssembleFixture(t *testing.T) *vault.Store {
	t.Helper()
	s, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func addNote(t *testing.T, s *vault.Store, label, group, body string) *vault.Note {
	t.Helper()
	n, err := s.Add(&vault.Note{
		Frontmatter: vault.Frontmatter{Label: label, Group: group},
		Body:        body,
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAssembleEmptySelection(t *testing.T) {
	s := newAssembleFixture(t)
	got := assemble(s, vault.Active{ItemIDs: nil})
	want := "--- pickmem: no memory selected ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssembleEmptySelectionUnderLens(t *testing.T) {
	s := newAssembleFixture(t)
	got := assemble(s, vault.Active{ActiveLens: "Job-Hunt", ItemIDs: nil})
	want := "--- pickmem: lens \"Job-Hunt\" is empty ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssembleSingleItem(t *testing.T) {
	s := newAssembleFixture(t)
	n := addNote(t, s, "salary", "financial", "monthly base $8k")
	got := assemble(s, vault.Active{ItemIDs: []string{n.ID}})
	want := "--- pickmem: selected memory ---\nsalary (financial): monthly base $8k\n--- end pickmem memory ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssembleMultipleItemsSingleBlankLineBetween(t *testing.T) {
	s := newAssembleFixture(t)
	a := addNote(t, s, "salary", "financial", "monthly base $8k")
	b := addNote(t, s, "kickoff", "work", "Aug 1")
	got := assemble(s, vault.Active{ItemIDs: []string{a.ID, b.ID}})
	want := "--- pickmem: selected memory ---\n" +
		"salary (financial): monthly base $8k\n" +
		"\n" +
		"kickoff (work): Aug 1\n" +
		"--- end pickmem memory ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssembleLensNameInHeader(t *testing.T) {
	s := newAssembleFixture(t)
	n := addNote(t, s, "x", "g", "body")
	got := assemble(s, vault.Active{ActiveLens: "Weekend", ItemIDs: []string{n.ID}})
	want := "--- pickmem: selected memory (lens: Weekend) ---\nx (g): body\n--- end pickmem memory ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssembleStaleIDsSkippedSilently(t *testing.T) {
	s := newAssembleFixture(t)
	n := addNote(t, s, "x", "g", "body")
	got := assemble(s, vault.Active{ItemIDs: []string{n.ID, "01Z-deleted"}})
	want := "--- pickmem: selected memory ---\nx (g): body\n--- end pickmem memory ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAssembleTrailingNewlinesNormalized(t *testing.T) {
	s := newAssembleFixture(t)
	n := addNote(t, s, "x", "g", "body\n\n\n\n")
	got := assemble(s, vault.Active{ItemIDs: []string{n.ID}})
	want := "--- pickmem: selected memory ---\nx (g): body\n--- end pickmem memory ---\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
