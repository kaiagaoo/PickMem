package picker

import (
	"reflect"
	"sort"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qwgao/pickmem/internal/vault"
)

// ---------- fixture ----------

// newFixture spins up a vault with a small deterministic corpus so tests
// can drive Update without touching real disk state repeatedly.
func newFixture(t *testing.T) (*vault.Store, []*vault.Note) {
	t.Helper()
	dir := t.TempDir()
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	var notes []*vault.Note
	adds := []struct {
		label, group, body string
		tags               []string
	}{
		{"salary", "financial", "monthly base $8k", []string{"money"}},
		{"bills", "financial", "rent, utilities, internet", nil},
		{"client-acme kickoff", "work", "kickoff meeting notes", []string{"acme"}},
		{"gift ideas for sister", "personal", "plants, enamel pins", []string{"gifts"}},
		{"solar panel research", "home", "3 quotes, need to compare inverters", nil},
	}
	for _, a := range adds {
		n, err := s.Add(&vault.Note{
			Frontmatter: vault.Frontmatter{Label: a.label, Group: a.group, Tags: a.tags},
			Body:        a.body,
		})
		if err != nil {
			t.Fatal(err)
		}
		notes = append(notes, n)
	}
	return s, notes
}

// ---------- rows / filter / tokens (pure) ----------

func TestBuildRowsGroupsAlphabetically(t *testing.T) {
	s, _ := newFixture(t)
	rows := buildRows(s.ListActive())
	// Expected header order: financial, home, personal, work.
	var headers []string
	for _, r := range rows {
		if r.kind == kindHeader {
			headers = append(headers, r.group)
		}
	}
	want := []string{"financial", "home", "personal", "work"}
	if !reflect.DeepEqual(headers, want) {
		t.Errorf("group order: got %v want %v", headers, want)
	}
}

func TestFilterFuzzyOverBodyAndTags(t *testing.T) {
	s, _ := newFixture(t)
	rows := buildRows(s.ListActive())

	// "inverter" only appears in the solar body — filter must find it.
	filtered := filterRows(rows, "inverter")
	if !containsNoteLabel(filtered, "solar panel research") {
		t.Errorf("body match missed: %v", labelsOf(filtered))
	}

	// "money" is a tag — must survive.
	filtered = filterRows(rows, "money")
	if !containsNoteLabel(filtered, "salary") {
		t.Errorf("tag match missed: %v", labelsOf(filtered))
	}

	// Fuzzy on label.
	filtered = filterRows(rows, "gft")
	if !containsNoteLabel(filtered, "gift ideas for sister") {
		t.Errorf("fuzzy label miss: %v", labelsOf(filtered))
	}
}

func TestFilterKeepsGroupHeaderWhenChildMatches(t *testing.T) {
	s, _ := newFixture(t)
	rows := buildRows(s.ListActive())
	filtered := filterRows(rows, "salary")
	// The "financial" header must survive because salary is in it.
	var sawHeader bool
	for _, r := range filtered {
		if r.kind == kindHeader && r.group == "financial" {
			sawHeader = true
		}
	}
	if !sawHeader {
		t.Errorf("financial header dropped when its child matched: %v", labelsOf(filtered))
	}
}

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		in   []string
		want int
	}{
		{nil, 0},
		{[]string{""}, 0},
		{[]string{"1234"}, 1},          // 4 chars -> 1
		{[]string{"12345"}, 2},         // 5 chars -> ceil(5/4) = 2
		{[]string{"12", "34", "5"}, 2}, // 5 chars total
	}
	for _, c := range cases {
		if got := EstimateTokens(c.in); got != c.want {
			t.Errorf("EstimateTokens(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// ---------- Model updates ----------

func TestToggleMarksAndUnmarks(t *testing.T) {
	s, notes := newFixture(t)
	m, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	// Cursor starts on the first selectable row = first note under
	// "financial" (which is `bills` after id-sort, since the second Add
	// was `bills` and ULIDs are chronological). Rather than depend on
	// that, drive the cursor to a known note.
	target := notes[0] // "salary"
	moveTo(&m, target.ID)
	next, _ := m.Update(spaceKey())
	m = next.(Model)
	if !m.selected[target.ID] {
		t.Errorf("toggle did not select %s", target.ID)
	}
	next, _ = m.Update(spaceKey())
	m = next.(Model)
	if m.selected[target.ID] {
		t.Errorf("second toggle did not deselect %s", target.ID)
	}
}

func TestToggleOnHeaderIsNoop(t *testing.T) {
	s, _ := newFixture(t)
	m, _ := New(s)
	// Force cursor onto a header.
	for i, r := range m.visible {
		if r.kind == kindHeader {
			m.cursor = i
			break
		}
	}
	before := len(m.selected)
	next, _ := m.Update(spaceKey())
	m = next.(Model)
	if len(m.selected) != before {
		t.Errorf("space on header changed selection: %d -> %d", before, len(m.selected))
	}
}

func TestApplyLensReplacesSelection(t *testing.T) {
	s, notes := newFixture(t)
	lens := vault.Lens{Name: "Job-Hunt", ItemIDs: []string{notes[0].ID, notes[2].ID}}
	if err := s.SaveLenses([]vault.Lens{lens}); err != nil {
		t.Fatal(err)
	}
	m, _ := New(s)

	// Pre-select something not in the lens; applyLens must clear it.
	m.selected = map[string]bool{notes[4].ID: true}
	m.applyLens(lens)

	got := selectedSlice(m.selected)
	want := []string{notes[0].ID, notes[2].ID}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("applyLens selection = %v, want %v", got, want)
	}
	if m.activeLens != "Job-Hunt" {
		t.Errorf("activeLens = %q, want %q", m.activeLens, "Job-Hunt")
	}
}

func TestApplyLensSkipsDeletedIDs(t *testing.T) {
	s, notes := newFixture(t)
	// Delete one note, then apply a lens that references it.
	toKeep := notes[0].ID
	toDelete := notes[1].ID
	if err := s.Remove(toDelete); err != nil {
		t.Fatal(err)
	}
	// Re-open so the picker sees the post-delete state.
	m, _ := New(s)
	m.applyLens(vault.Lens{Name: "stale", ItemIDs: []string{toKeep, toDelete}})
	if m.selected[toDelete] {
		t.Errorf("applyLens selected a deleted note %s", toDelete)
	}
	if !m.selected[toKeep] {
		t.Errorf("applyLens dropped a live id %s", toKeep)
	}
}

func TestConfirmProducesResultWithSelectedIDs(t *testing.T) {
	s, notes := newFixture(t)
	m, _ := New(s)
	m.selected = map[string]bool{notes[0].ID: true, notes[3].ID: true}
	next, cmd := m.Update(enterKey())
	m = next.(Model)
	if cmd == nil {
		t.Fatal("enter did not return tea.Quit cmd")
	}
	if !m.Result.Confirmed {
		t.Error("Result.Confirmed = false after enter")
	}
	got := m.Result.ItemIDs
	sort.Strings(got)
	want := []string{notes[0].ID, notes[3].ID}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Result.ItemIDs = %v, want %v", got, want)
	}
}

func TestConfirmWithEmptySelectionIsAllowed(t *testing.T) {
	s, _ := newFixture(t)
	m, _ := New(s)
	m.selected = map[string]bool{}
	next, _ := m.Update(enterKey())
	m = next.(Model)
	if !m.Result.Confirmed {
		t.Error("empty enter should confirm (matches 'default is nothing')")
	}
	if len(m.Result.ItemIDs) != 0 {
		t.Errorf("empty enter should produce no ids, got %v", m.Result.ItemIDs)
	}
}

func TestCancelReturnsUnconfirmed(t *testing.T) {
	s, notes := newFixture(t)
	m, _ := New(s)
	m.selected = map[string]bool{notes[0].ID: true}
	next, cmd := m.Update(escKey())
	m = next.(Model)
	if cmd == nil {
		t.Fatal("esc did not return tea.Quit cmd")
	}
	if m.Result.Confirmed {
		t.Error("cancel should not set Confirmed")
	}
}

// ---------- Active selection persistence: end-to-end ----------

func TestActiveSelectionRoundTripThroughPicker(t *testing.T) {
	s, notes := newFixture(t)
	// Seed active.json so New() loads it.
	if err := s.SaveActive(vault.Active{
		ActiveLens: "existing",
		ItemIDs:    []string{notes[0].ID, notes[2].ID},
	}); err != nil {
		t.Fatal(err)
	}
	m, _ := New(s)
	if m.activeLens != "existing" {
		t.Errorf("activeLens not loaded: %q", m.activeLens)
	}
	if !m.selected[notes[0].ID] || !m.selected[notes[2].ID] {
		t.Errorf("selection not loaded: %v", m.selected)
	}
	// Confirm — Result.ItemIDs should match, in display order.
	next, _ := m.Update(enterKey())
	m = next.(Model)
	if len(m.Result.ItemIDs) != 2 {
		t.Errorf("Result.ItemIDs len = %d, want 2", len(m.Result.ItemIDs))
	}
	if m.Result.ActiveLens != "existing" {
		t.Errorf("Result.ActiveLens = %q, want existing", m.Result.ActiveLens)
	}
}

// ---------- helpers ----------

func moveTo(m *Model, id string) {
	for i, r := range m.visible {
		if r.kind == kindNote && r.note.ID == id {
			m.cursor = i
			return
		}
	}
}

func spaceKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}} }
func enterKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }
func escKey() tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyEsc} }

func selectedSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	return out
}

func containsNoteLabel(rs []row, label string) bool {
	for _, r := range rs {
		if r.kind == kindNote && r.note.Label == label {
			return true
		}
	}
	return false
}

func labelsOf(rs []row) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		if r.kind == kindNote {
			out = append(out, r.note.Label)
		} else {
			out = append(out, "["+r.group+"]")
		}
	}
	return out
}
