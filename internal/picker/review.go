package picker

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

// ReviewOutcome is one row's fate after the review session ends.
type ReviewOutcome int

const (
	OutcomePending  ReviewOutcome = iota // user did nothing
	OutcomeAccepted                      // move to group folder, flip active
	OutcomeRejected                      // delete inbox file
)

// ReviewDecision is the caller's marching order for one pending id.
type ReviewDecision struct {
	ID      string
	Group   string // for accepted rows
	Outcome ReviewOutcome
}

// ReviewResult is what the caller reads after the Bubble Tea program
// exits. Confirmed=false means the user cancelled — do nothing.
type ReviewResult struct {
	Confirmed bool
	Decisions []ReviewDecision
}

// reviewMode gates which key handler runs in Update.
type reviewMode int

const (
	rmBrowse reviewMode = iota
	rmFilter
	rmReassign // group overlay
)

// ReviewModel is a bulk-review TUI over pending inbox notes. Reuses the
// picker's Theme so it feels consistent, but has its own key map and
// state machine. Kept separate from Model because the two flows share
// almost none of their business logic — trying to unify them would just
// mean more conditionals.
type ReviewModel struct {
	store *vault.Store
	theme Theme

	// items is the ordered pending set at open time. We snapshot so
	// filtering/marking doesn't fight with a mid-review re-scan.
	items   []*vault.Note
	visible []int // indices into items after filter

	// per-item state
	selected  map[string]bool // id -> selected
	decisions map[string]ReviewDecision

	cursor int // index into visible
	scroll int

	width, height int

	mode reviewMode

	// filter mode
	query       string
	filterInput textinput.Model

	// reassign overlay
	reassignInput textinput.Model
	groupSuggests []string
	groupCursor   int

	Result ReviewResult
}

// NewReview loads the vault's pending inbox and returns a ready model.
// Empty inbox is the caller's problem to gate on — the model handles it
// gracefully (empty view) but the CLI prints a friendlier message.
func NewReview(store *vault.Store) (ReviewModel, error) {
	pending := store.ListPending()

	fi := textinput.New()
	fi.Placeholder = "filter"
	fi.Prompt = "/ "
	fi.CharLimit = 200

	gi := textinput.New()
	gi.Placeholder = "group"
	gi.Prompt = "group: "
	gi.CharLimit = 80

	m := ReviewModel{
		store:         store,
		theme:         DefaultTheme(),
		items:         pending,
		selected:      map[string]bool{},
		decisions:     map[string]ReviewDecision{},
		width:         80,
		height:        24,
		filterInput:   fi,
		reassignInput: gi,
	}
	m.applyFilter() // populates visible
	m.groupSuggests = m.knownGroups()
	return m, nil
}

func (m ReviewModel) Init() tea.Cmd { return nil }

// ---------- Update ----------

func (m ReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case rmFilter:
			return m.updateFilter(msg)
		case rmReassign:
			return m.updateReassign(msg)
		default:
			return m.updateBrowse(msg)
		}
	}
	return m, nil
}

func (m ReviewModel) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		m.Result = ReviewResult{Confirmed: false}
		return m, tea.Quit
	case "enter":
		m.Result = ReviewResult{Confirmed: true, Decisions: m.finalDecisions()}
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.visible)-1 {
			m.cursor++
			m.ensureVisible()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
	case " ", "space":
		m.toggleAtCursor()
	case "a":
		// Accept selected — or the cursor row if nothing is selected.
		if len(m.selected) == 0 {
			m.acceptID(m.currentID())
		} else {
			for id := range m.selected {
				m.acceptID(id)
			}
			m.selected = map[string]bool{}
		}
	case "A":
		// Accept every remaining pending row that has a suggested_group.
		// Rows with an empty suggestion stay pending — the reviewer must
		// pick a group first.
		for _, n := range m.items {
			if _, done := m.decisions[n.ID]; done {
				continue
			}
			if n.SuggestedGroup == "" {
				continue
			}
			m.acceptID(n.ID)
		}
	case "r":
		if len(m.selected) == 0 {
			m.rejectID(m.currentID())
		} else {
			for id := range m.selected {
				m.rejectID(id)
			}
			m.selected = map[string]bool{}
		}
	case "g":
		if m.currentID() != "" || len(m.selected) > 0 {
			m.mode = rmReassign
			m.reassignInput.SetValue("")
			m.reassignInput.Focus()
			m.groupCursor = 0
		}
	case "/":
		m.mode = rmFilter
		m.filterInput.SetValue(m.query)
		m.filterInput.Focus()
	}
	return m, nil
}

func (m ReviewModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.query = ""
		m.filterInput.SetValue("")
		m.applyFilter()
		m.mode = rmBrowse
		return m, nil
	case "enter":
		m.mode = rmBrowse
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.query = m.filterInput.Value()
	m.applyFilter()
	return m, cmd
}

func (m ReviewModel) updateReassign(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = rmBrowse
		return m, nil
	case "enter":
		group := strings.TrimSpace(m.reassignInput.Value())
		if group == "" && m.groupCursor >= 0 && m.groupCursor < len(m.groupSuggests) {
			group = m.groupSuggests[m.groupCursor]
		}
		if group == "" {
			m.mode = rmBrowse
			return m, nil
		}
		m.setGroupForSelection(group)
		m.mode = rmBrowse
		return m, nil
	case "tab", "down":
		if m.groupCursor < len(m.groupSuggests)-1 {
			m.groupCursor++
		}
		return m, nil
	case "up":
		if m.groupCursor > 0 {
			m.groupCursor--
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.reassignInput, cmd = m.reassignInput.Update(msg)
	return m, cmd
}

// ---------- actions ----------

func (m *ReviewModel) toggleAtCursor() {
	id := m.currentID()
	if id == "" {
		return
	}
	if m.selected[id] {
		delete(m.selected, id)
	} else {
		m.selected[id] = true
	}
}

func (m *ReviewModel) acceptID(id string) {
	if id == "" {
		return
	}
	n := m.findItem(id)
	if n == nil {
		return
	}
	group := n.SuggestedGroup
	// If we already have a decision with a group, use it (came from
	// a reassign).
	if prev, ok := m.decisions[id]; ok && prev.Group != "" {
		group = prev.Group
	}
	if group == "" {
		// No suggestion + no manual group → force reassign flow instead of
		// silently dropping the row.
		return
	}
	m.decisions[id] = ReviewDecision{ID: id, Group: group, Outcome: OutcomeAccepted}
}

func (m *ReviewModel) rejectID(id string) {
	if id == "" {
		return
	}
	m.decisions[id] = ReviewDecision{ID: id, Outcome: OutcomeRejected}
}

// setGroupForSelection applies a group to every selected row (or the
// cursor row if nothing is selected). If a row was already accepted, we
// update its group; otherwise this just records the group and leaves the
// row pending until 'a' or 'A' accepts it.
func (m *ReviewModel) setGroupForSelection(group string) {
	targets := m.selectedIDs()
	if len(targets) == 0 {
		if id := m.currentID(); id != "" {
			targets = []string{id}
		}
	}
	for _, id := range targets {
		if _, ok := m.decisions[id]; !ok {
			// Also mutate the in-memory item's SuggestedGroup so the row
			// renders with the new group and 'A' picks it up.
			if n := m.findItem(id); n != nil {
				n.SuggestedGroup = group
			}
			continue
		}
		d := m.decisions[id]
		d.Group = group
		if d.Outcome == OutcomePending {
			d.Outcome = OutcomeAccepted
		}
		m.decisions[id] = d
	}
}

// ---------- selectors / helpers ----------

func (m ReviewModel) currentID() string {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return ""
	}
	return m.items[m.visible[m.cursor]].ID
}

func (m ReviewModel) selectedIDs() []string {
	out := make([]string, 0, len(m.selected))
	for id := range m.selected {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (m ReviewModel) findItem(id string) *vault.Note {
	for _, n := range m.items {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// finalDecisions returns the caller-facing list, including implicit
// pending outcomes for anything the user didn't touch.
func (m ReviewModel) finalDecisions() []ReviewDecision {
	out := make([]ReviewDecision, 0, len(m.items))
	for _, n := range m.items {
		if d, ok := m.decisions[n.ID]; ok {
			out = append(out, d)
			continue
		}
		out = append(out, ReviewDecision{ID: n.ID, Outcome: OutcomePending})
	}
	return out
}

func (m ReviewModel) knownGroups() []string {
	// The full curated taxonomy (folder tree + note groups + rule targets),
	// so the reassign overlay suggests every folder the user made in
	// Obsidian, not only groups that already hold a note.
	return m.store.KnownGroups()
}

func (m *ReviewModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.query))
	m.visible = m.visible[:0]
	for i, n := range m.items {
		if q == "" || strings.Contains(strings.ToLower(n.Label+" "+n.Body+" "+n.SuggestedGroup), q) {
			m.visible = append(m.visible, i)
		}
	}
	if m.cursor >= len(m.visible) {
		m.cursor = 0
	}
	m.scroll = 0
}

func (m *ReviewModel) ensureVisible() {
	viewport := m.listHeight()
	if viewport <= 0 {
		return
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	} else if m.cursor >= m.scroll+viewport {
		m.scroll = m.cursor - viewport + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// ---------- View ----------

func (m ReviewModel) View() string {
	var b strings.Builder
	b.WriteString(m.viewHeader())
	if m.mode == rmFilter {
		b.WriteString(m.theme.FilterBar.Render(m.filterInput.View()))
		b.WriteByte('\n')
	}
	b.WriteString(m.viewList())
	b.WriteString(m.viewFooter())

	base := b.String()
	if m.mode == rmReassign {
		return overlay(base, m.viewReassignBox(), m.width, m.height)
	}
	return base
}

func (m ReviewModel) viewHeader() string {
	root := filepath.Base(m.store.Root)
	line := m.theme.Title.Render("PickMem review") + m.theme.Dim.Render("  "+root+
		fmt.Sprintf("  ·  %d pending", len(m.items)))
	return line + "\n"
}

func (m ReviewModel) listHeight() int {
	h := m.height - 1 - 2
	if m.mode == rmFilter {
		h--
	}
	if h < 3 {
		return 3
	}
	return h
}

func (m ReviewModel) viewList() string {
	viewport := m.listHeight()
	if len(m.visible) == 0 {
		return m.theme.Dim.Render("  (no matches)") + "\n"
	}
	var b strings.Builder
	end := m.scroll + viewport
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := m.scroll; i < end; i++ {
		b.WriteString(m.renderRow(i == m.cursor, m.items[m.visible[i]]))
		b.WriteByte('\n')
	}
	for i := end - m.scroll; i < viewport; i++ {
		b.WriteByte('\n')
	}
	return b.String()
}

func (m ReviewModel) renderRow(isCursor bool, n *vault.Note) string {
	// Outcome glyph.
	glyph := "[ ]"
	if m.selected[n.ID] {
		glyph = "[·]"
	}
	if d, ok := m.decisions[n.ID]; ok {
		switch d.Outcome {
		case OutcomeAccepted:
			glyph = "[✓]"
		case OutcomeRejected:
			glyph = "[✗]"
		}
	}

	group := n.SuggestedGroup
	if d, ok := m.decisions[n.ID]; ok && d.Group != "" {
		group = d.Group
	}
	groupCol := "?"
	if group != "" {
		groupCol = group
	}

	line := fmt.Sprintf("    %s  %-25s  %s", glyph, truncateLabel(n.Label, 25), m.theme.Tag.Render("→ "+groupCol))

	// Styling by state (cursor > accepted > rejected > selected > default).
	if isCursor {
		return m.theme.Cursor.Render(line)
	}
	if d, ok := m.decisions[n.ID]; ok {
		switch d.Outcome {
		case OutcomeAccepted:
			return m.theme.Selected.Render(line)
		case OutcomeRejected:
			return m.theme.Danger.Render(line)
		}
	}
	if m.selected[n.ID] {
		return m.theme.Selected.Render(line)
	}
	return m.theme.Item.Render(line)
}

func (m ReviewModel) viewFooter() string {
	counts := countByOutcome(m.decisions)
	summary := fmt.Sprintf("%d accepted · %d rejected · %d pending",
		counts[OutcomeAccepted], counts[OutcomeRejected], len(m.items)-counts[OutcomeAccepted]-counts[OutcomeRejected])
	hints := []string{
		m.theme.FooterKey.Render("space") + m.theme.Footer.Render(" select"),
		m.theme.FooterKey.Render("a") + m.theme.Footer.Render(" accept"),
		m.theme.FooterKey.Render("A") + m.theme.Footer.Render(" accept-all"),
		m.theme.FooterKey.Render("r") + m.theme.Footer.Render(" reject"),
		m.theme.FooterKey.Render("g") + m.theme.Footer.Render(" group"),
		m.theme.FooterKey.Render("/") + m.theme.Footer.Render(" filter"),
		m.theme.FooterKey.Render("enter") + m.theme.Footer.Render(" apply"),
		m.theme.FooterKey.Render("q") + m.theme.Footer.Render(" cancel"),
	}
	return m.theme.Footer.Render(summary) + "\n" +
		strings.Join(hints, m.theme.Dim.Render("  "))
}

func (m ReviewModel) viewReassignBox() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Reassign group"))
	b.WriteByte('\n')
	b.WriteString(m.reassignInput.View())
	b.WriteByte('\n')
	if len(m.groupSuggests) > 0 {
		b.WriteString(m.theme.Dim.Render("existing groups:"))
		b.WriteByte('\n')
		for i, g := range m.groupSuggests {
			line := "  " + g
			if i == m.groupCursor {
				b.WriteString(m.theme.Cursor.Render(line))
			} else {
				b.WriteString(m.theme.OverlayItem.Render(line))
			}
			b.WriteByte('\n')
		}
	}
	b.WriteString(m.theme.Dim.Render("type new · tab/↓ browse · enter apply · esc cancel"))
	return m.theme.OverlayBox.Render(b.String())
}

// ---------- helpers ----------

func countByOutcome(m map[string]ReviewDecision) map[ReviewOutcome]int {
	out := map[ReviewOutcome]int{}
	for _, d := range m {
		out[d.Outcome]++
	}
	return out
}

func truncateLabel(s string, n int) string {
	if len(s) <= n {
		return s + strings.Repeat(" ", n-len(s))
	}
	return s[:n-1] + "…"
}
