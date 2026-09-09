package retrieval

import (
	"testing"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

func note(id, label, group, body string, tags ...string) *vault.Note {
	return &vault.Note{Frontmatter: vault.Frontmatter{
		ID: id, Label: label, Group: group, Tags: tags, Status: vault.StatusActive,
	}, Body: body}
}

func TestRankPrefersLabelAndTagMatches(t *testing.T) {
	notes := []*vault.Note{
		note("a", "Travel", "personal", "Uses TypeScript at work."),
		note("b", "TypeScript stack", "work/stack", "Uses strict mode.", "typescript"),
	}
	got := Rank("typescript stack", notes, 2)
	if len(got) != 2 || got[0].Note.ID != "b" {
		t.Fatalf("Rank() = %+v, want b first", got)
	}
}

func TestRankOmitsUnmatchedAndPendingNotes(t *testing.T) {
	pending := note("p", "Go stack", "work", "Uses Go.")
	pending.Status = vault.StatusPending
	got := Rank("go", []*vault.Note{
		note("a", "Diet", "health", "Vegetarian meals."), pending,
	}, 5)
	if len(got) != 0 {
		t.Fatalf("Rank() returned irrelevant or pending notes: %+v", got)
	}
}

func TestRankIsDeterministicOnTies(t *testing.T) {
	notes := []*vault.Note{
		note("b", "Editor", "work", "Uses vim."),
		note("a", "Editor", "work", "Uses emacs."),
	}
	got := Rank("editor", notes, 2)
	if len(got) != 2 || got[0].Note.ID != "a" || got[1].Note.ID != "b" {
		t.Fatalf("tie order = %+v, want ids a,b", got)
	}
}

func TestRankRespectsLimit(t *testing.T) {
	notes := []*vault.Note{
		note("a", "Work", "work", "Project alpha"),
		note("b", "Work", "work", "Project beta"),
	}
	if got := Rank("work", notes, 1); len(got) != 1 {
		t.Fatalf("len(Rank()) = %d, want 1", len(got))
	}
}

func TestRankReturnsMatchedTerms(t *testing.T) {
	got := Rank("concise python", []*vault.Note{
		note("a", "Answer style", "preferences", "Keep answers concise.", "python"),
	}, 5)
	if len(got) != 1 || len(got[0].MatchedTerms) != 2 {
		t.Fatalf("Rank() = %+v, want two matched terms", got)
	}
}

func TestRankIgnoresStopwords(t *testing.T) {
	got := Rank("prepare for a discussion about the architecture", []*vault.Note{
		note("a", "Unrelated", "about", "This is a generic note."),
	}, 5)
	if len(got) != 0 {
		t.Fatalf("Rank() matched only stopwords: %+v", got)
	}
}
