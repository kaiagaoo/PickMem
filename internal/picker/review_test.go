package picker

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qwgao/pickmem/internal/vault"
)

// newReviewFixture stages three pending items with varying suggested
// groups so tests can exercise accept-with-suggestion, reassign, and
// reject flows.
func newReviewFixture(t *testing.T) (*vault.Store, []*vault.Note) {
	t.Helper()
	dir := t.TempDir()
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []*vault.Note
	adds := []struct{ label, suggested, body string }{
		{"salary is monthly base 8k", "financial", "salary is monthly base $8k plus bonus"},
		{"kickoff acme aug 1", "work", "kickoff meeting with Acme is on Aug 1"},
		{"loves plants and enamel pins", "", "loves plants and enamel pins"},
	}
	for _, a := range adds {
		n, err := s.AddInbox(&vault.Note{
			Frontmatter: vault.Frontmatter{Label: a.label, SuggestedGroup: a.suggested},
			Body:        a.body,
		})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return s, out
}

func TestReviewAcceptWithSuggestedGroup(t *testing.T) {
	s, notes := newReviewFixture(t)
	m, err := NewReview(s)
	if err != nil {
		t.Fatal(err)
	}
	// Cursor is on notes[0] (salary → financial). Press 'a'.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = next.(ReviewModel)
	d, ok := m.decisions[notes[0].ID]
	if !ok || d.Outcome != OutcomeAccepted || d.Group != "financial" {
		t.Errorf("accept did not stick: %+v", d)
	}
}

func TestReviewRejectSelected(t *testing.T) {
	s, notes := newReviewFixture(t)
	m, _ := NewReview(s)
	// Select all three, then reject.
	m.selected = map[string]bool{notes[0].ID: true, notes[1].ID: true, notes[2].ID: true}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = next.(ReviewModel)
	for _, n := range notes {
		if m.decisions[n.ID].Outcome != OutcomeRejected {
			t.Errorf("reject missed %s: %+v", n.ID, m.decisions[n.ID])
		}
	}
	// Selection should have been cleared after the bulk action.
	if len(m.selected) != 0 {
		t.Errorf("selection not cleared after bulk reject: %v", m.selected)
	}
}

func TestReviewAcceptAllSkipsUnrouted(t *testing.T) {
	s, notes := newReviewFixture(t)
	m, _ := NewReview(s)
	// 'A' accepts every row that has a SuggestedGroup. notes[2] has none.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	m = next.(ReviewModel)
	if m.decisions[notes[0].ID].Outcome != OutcomeAccepted {
		t.Errorf("A missed salary")
	}
	if m.decisions[notes[1].ID].Outcome != OutcomeAccepted {
		t.Errorf("A missed acme")
	}
	if _, ok := m.decisions[notes[2].ID]; ok {
		t.Errorf("A accepted unrouted note: %+v", m.decisions[notes[2].ID])
	}
}

func TestReviewReassignGroupAcceptsPreviouslyUnrouted(t *testing.T) {
	s, notes := newReviewFixture(t)
	m, _ := NewReview(s)
	// Move cursor to notes[2] (unrouted).
	for i, idx := range m.visible {
		if m.items[idx].ID == notes[2].ID {
			m.cursor = i
			break
		}
	}
	// Trigger reassign, type "personal", enter.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = next.(ReviewModel)
	if m.mode != rmReassign {
		t.Fatalf("g did not enter reassign mode; got mode=%d", m.mode)
	}
	m.reassignInput.SetValue("personal")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(ReviewModel)
	// The row should now have SuggestedGroup=personal on the item struct
	// (so 'A' picks it up) — accept still requires a subsequent 'a'.
	item := m.findItem(notes[2].ID)
	if item.SuggestedGroup != "personal" {
		t.Errorf("reassign did not update item: %+v", item)
	}
	// Now 'A' should accept it.
	m.cursor = 0
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	m = next.(ReviewModel)
	if m.decisions[notes[2].ID].Group != "personal" {
		t.Errorf("A after reassign didn't accept notes[2]: %+v", m.decisions[notes[2].ID])
	}
}

func TestReviewEnterProducesDecisionsForEveryItem(t *testing.T) {
	s, notes := newReviewFixture(t)
	m, _ := NewReview(s)
	// Accept the first, reject the second, leave the third pending.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = next.(ReviewModel)
	m.cursor = 1
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = next.(ReviewModel)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(ReviewModel)
	if cmd == nil {
		t.Fatal("enter did not return tea.Quit")
	}
	if !m.Result.Confirmed {
		t.Fatal("Result.Confirmed = false")
	}
	if len(m.Result.Decisions) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(m.Result.Decisions))
	}
	byID := map[string]ReviewDecision{}
	for _, d := range m.Result.Decisions {
		byID[d.ID] = d
	}
	if byID[notes[0].ID].Outcome != OutcomeAccepted {
		t.Errorf("notes[0] not accepted: %+v", byID[notes[0].ID])
	}
	if byID[notes[1].ID].Outcome != OutcomeRejected {
		t.Errorf("notes[1] not rejected: %+v", byID[notes[1].ID])
	}
	if byID[notes[2].ID].Outcome != OutcomePending {
		t.Errorf("notes[2] should be pending (untouched): %+v", byID[notes[2].ID])
	}
}

func TestReviewCancelReturnsUnconfirmed(t *testing.T) {
	s, _ := newReviewFixture(t)
	m, _ := NewReview(s)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(ReviewModel)
	if cmd == nil {
		t.Fatal("esc did not return tea.Quit")
	}
	if m.Result.Confirmed {
		t.Error("cancel should not confirm")
	}
}
