package picker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/qwgao/pickmem/internal/vault"
)

// mode is the picker's top-level state. Only one mode is active at a
// time — filter mode disables list keys, lens overlay disables browse
// keys, and save-lens replaces the lens overlay.
type mode int

const (
	modeBrowse mode = iota
	modeFilter
	modeLens
	modeSaveLens
)

// Result is what the picker returns on confirm. If Confirmed is false,
// the caller must not touch active.json — the user cancelled.
type Result struct {
	Confirmed  bool
	ActiveLens string
	ItemIDs    []string
}

// Model is the Bubble Tea model for the picker. Kept as a value so
// bubbletea's copy-on-update discipline doesn't fight us; mutators
// return a new Model rather than sharing pointers.
type Model struct {
	store *vault.Store
	theme Theme
	keys  keys

	// rows is every group header + selectable note, in display order.
	rows []row
	// visible is rows after fuzzy filter is applied. When query is empty
	// visible == rows.
	visible []row

	// selected tracks note ids currently ticked.
	selected map[string]bool

	// cursor is an index into visible.
	cursor int
	// scroll is the index of the first row rendered (top of viewport).
	scroll int

	width, height int

	mode mode

	// filter mode state.
	query       string
	filterInput textinput.Model

	// lens overlay + save-as state.
	lenses     []vault.Lens
	activeLens string
	lensCursor int
	saveInput  textinput.Model

	// result populated on Confirm; the caller reads it after tea.Run.
	Result Result
}

// New builds a picker Model bound to the given store. Loads lenses and
// active.json to seed initial state, so re-running the picker inside a
// session picks up where it left off.
func New(store *vault.Store) (Model, error) {
	notes := store.ListActive()
	rows := buildRows(notes)

	lenses, err := store.LoadLenses()
	if err != nil {
		return Model{}, err
	}
	active, err := store.LoadActive()
	if err != nil {
		return Model{}, err
	}

	selected := make(map[string]bool, len(active.ItemIDs))
	for _, id := range active.ItemIDs {
		selected[id] = true
	}

	fi := textinput.New()
	fi.Placeholder = "filter"
	fi.Prompt = "/ "
	fi.CharLimit = 200

	si := textinput.New()
	si.Placeholder = "lens name"
	si.Prompt = "name: "
	si.CharLimit = 60

	m := Model{
		store:       store,
		theme:       DefaultTheme(),
		keys:        defaultKeys(),
		rows:        rows,
		visible:     rows,
		selected:    selected,
		cursor:      firstSelectableIndex(rows),
		width:       80,
		height:      24,
		lenses:      lenses,
		activeLens:  active.ActiveLens,
		filterInput: fi,
		saveInput:   si,
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m, nil
}

// Init is called once by bubbletea; nothing async to kick off here.
func (m Model) Init() tea.Cmd { return nil }

// Update is the reducer. Every keystroke lands here and returns a new
// Model + optional Cmd. Modes gate which key handler runs first.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeFilter:
			return m.updateFilter(msg)
		case modeLens:
			return m.updateLens(msg)
		case modeSaveLens:
			return m.updateSaveLens(msg)
		default:
			return m.updateBrowse(msg)
		}
	}
	return m, nil
}

// ---------- browse mode ----------

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.Result = Result{Confirmed: false}
		return m, tea.Quit
	case key.Matches(msg, m.keys.Confirm):
		m.Result = Result{
			Confirmed:  true,
			ActiveLens: m.activeLens,
			ItemIDs:    m.selectedIDs(),
		}
		return m, tea.Quit
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(+1)
	case key.Matches(msg, m.keys.Toggle):
		m.toggleAtCursor()
	case key.Matches(msg, m.keys.Filter):
		m.mode = modeFilter
		m.filterInput.SetValue(m.query)
		m.filterInput.Focus()
	case key.Matches(msg, m.keys.Lens):
		if len(m.lenses) > 0 {
			m.mode = modeLens
			m.lensCursor = 0
		}
	case key.Matches(msg, m.keys.SaveLens):
		if len(m.selectedIDs()) > 0 {
			m.mode = modeSaveLens
			m.saveInput.SetValue("")
			m.saveInput.Focus()
		}
	}
	return m, nil
}

// moveCursor moves by delta, skipping non-selectable rows (headers).
// Wraps at the edges by clamping — no cursor-off-screen.
func (m *Model) moveCursor(delta int) {
	if len(m.visible) == 0 {
		return
	}
	i := m.cursor + delta
	for i >= 0 && i < len(m.visible) {
		if m.visible[i].selectable() {
			m.cursor = i
			m.ensureVisible()
			return
		}
		i += delta
	}
	// Nothing selectable in that direction — cursor stays put.
}

func (m *Model) toggleAtCursor() {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return
	}
	r := m.visible[m.cursor]
	if !r.selectable() {
		return
	}
	id := r.note.ID
	if m.selected[id] {
		delete(m.selected, id)
	} else {
		m.selected[id] = true
	}
	// Custom selection breaks the "this is a lens" state.
	m.activeLens = ""
}

// ensureVisible scrolls the viewport so the cursor stays inside it.
func (m *Model) ensureVisible() {
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

// ---------- filter mode ----------

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.query = ""
		m.filterInput.SetValue("")
		m.applyFilter()
		m.mode = modeBrowse
		return m, nil
	case "enter", "/":
		m.mode = modeBrowse
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.query = m.filterInput.Value()
	m.applyFilter()
	return m, cmd
}

func (m *Model) applyFilter() {
	m.visible = filterRows(m.rows, m.query)
	// Reset cursor to the first selectable visible row so we never land
	// on a header (or off the end of a shrunken list).
	m.cursor = firstSelectableIndex(m.visible)
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.scroll = 0
}

// ---------- lens overlay ----------

func (m Model) updateLens(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeBrowse
	case "up", "k":
		if m.lensCursor > 0 {
			m.lensCursor--
		}
	case "down", "j":
		if m.lensCursor < len(m.lenses)-1 {
			m.lensCursor++
		}
	case "enter":
		if m.lensCursor >= 0 && m.lensCursor < len(m.lenses) {
			m.applyLens(m.lenses[m.lensCursor])
		}
		m.mode = modeBrowse
	}
	return m, nil
}

// applyLens replaces the current selection with the lens's item ids and
// stamps activeLens so the footer + saved active.json reflect it.
func (m *Model) applyLens(l vault.Lens) {
	m.selected = map[string]bool{}
	for _, id := range l.ItemIDs {
		// Only include ids the vault still knows about, so a lens
		// pointing at a since-deleted note doesn't ghost-select nothing.
		if _, ok := m.store.Get(id); ok {
			m.selected[id] = true
		}
	}
	m.activeLens = l.Name
}

// ---------- save-as-lens prompt ----------

func (m Model) updateSaveLens(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBrowse
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.saveInput.Value())
		if name == "" {
			m.mode = modeBrowse
			return m, nil
		}
		ids := m.selectedIDs()
		m.lenses = vault.UpsertLens(m.lenses, vault.Lens{Name: name, ItemIDs: ids})
		if err := m.store.SaveLenses(m.lenses); err != nil {
			// Best-effort: fall back to browse. The user can retry.
			m.mode = modeBrowse
			return m, nil
		}
		m.activeLens = name
		m.mode = modeBrowse
		return m, nil
	}
	var cmd tea.Cmd
	m.saveInput, cmd = m.saveInput.Update(msg)
	return m, cmd
}

// selectedIDs returns selected note ids in the display order (grouped +
// stable), which is what we persist to active.json.
func (m Model) selectedIDs() []string {
	out := make([]string, 0, len(m.selected))
	for _, r := range m.rows {
		if r.kind != kindNote {
			continue
		}
		if m.selected[r.note.ID] {
			out = append(out, r.note.ID)
		}
	}
	return out
}

// ---------- View ----------

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.viewHeader())
	if m.mode == modeFilter {
		b.WriteString(m.theme.FilterBar.Render(m.filterInput.View()))
		b.WriteByte('\n')
	}
	b.WriteString(m.viewList())
	b.WriteString(m.viewFooter())

	base := b.String()
	switch m.mode {
	case modeLens:
		return overlay(base, m.viewLensBox(), m.width, m.height)
	case modeSaveLens:
		return overlay(base, m.viewSavePrompt(), m.width, m.height)
	}
	return base
}

func (m Model) viewHeader() string {
	root := filepath.Base(m.store.Root)
	line := m.theme.Title.Render("PickMem") + m.theme.Dim.Render("  "+root)
	return line + "\n"
}

// listHeight is how many rows the scrollable body gets. Header eats one
// line; filter bar eats one if active; footer eats two lines.
func (m Model) listHeight() int {
	h := m.height - 1 - 2 // header + footer
	if m.mode == modeFilter {
		h--
	}
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) viewList() string {
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
		b.WriteString(m.renderRow(m.visible[i], i == m.cursor))
		b.WriteByte('\n')
	}
	// Pad to fixed height so the footer sits at the same spot when the
	// list shrinks.
	for i := end - m.scroll; i < viewport; i++ {
		b.WriteByte('\n')
	}
	return b.String()
}

func (m Model) renderRow(r row, isCursor bool) string {
	if r.kind == kindHeader {
		return m.theme.GroupHeader.Render("  " + r.group)
	}
	glyphFull := "[x]"
	glyphEmpty := "[ ]"
	checked := m.selected[r.note.ID]
	var glyph string
	if checked {
		glyph = m.theme.Checkbox.Render(glyphFull)
	} else {
		glyph = m.theme.CheckboxDim.Render(glyphEmpty)
	}
	line := "    " + glyph + " " + r.note.Label
	if len(r.note.Tags) > 0 {
		line += "  " + m.theme.Tag.Render("#"+strings.Join(r.note.Tags, " #"))
	}
	switch {
	case isCursor && checked:
		return m.theme.Cursor.Render(line)
	case isCursor:
		return m.theme.Cursor.Render(line)
	case checked:
		return m.theme.Selected.Render(line)
	default:
		return m.theme.Item.Render(line)
	}
}

func (m Model) viewFooter() string {
	// Compute the "Active: ..." + count + tokens summary.
	label := "custom"
	if m.activeLens != "" {
		label = m.activeLens
	}
	if len(m.selected) == 0 {
		label = "none"
	}
	bodies := make([]string, 0, len(m.selected))
	for id := range m.selected {
		if n, ok := m.store.Get(id); ok {
			bodies = append(bodies, n.Body)
		}
	}
	tokens := EstimateTokens(bodies)

	summary := fmt.Sprintf("Active: %s · %d selected · ~%d tokens",
		label, len(m.selected), tokens)

	// Hints, styled — key names bold, action text dim.
	hints := []string{
		m.theme.FooterKey.Render("space") + m.theme.Footer.Render(" toggle"),
		m.theme.FooterKey.Render("/") + m.theme.Footer.Render(" filter"),
		m.theme.FooterKey.Render("l") + m.theme.Footer.Render(" lens"),
		m.theme.FooterKey.Render("s") + m.theme.Footer.Render(" save-lens"),
		m.theme.FooterKey.Render("enter") + m.theme.Footer.Render(" confirm"),
		m.theme.FooterKey.Render("q") + m.theme.Footer.Render(" cancel"),
	}
	return m.theme.Footer.Render(summary) + "\n" +
		strings.Join(hints, m.theme.Dim.Render("  "))
}

func (m Model) viewLensBox() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Lenses"))
	b.WriteByte('\n')
	for i, l := range m.lenses {
		line := fmt.Sprintf("  %s  (%d items)", l.Name, len(l.ItemIDs))
		if i == m.lensCursor {
			b.WriteString(m.theme.Cursor.Render(line))
		} else {
			b.WriteString(m.theme.OverlayItem.Render(line))
		}
		b.WriteByte('\n')
	}
	b.WriteString(m.theme.Dim.Render("enter: apply · esc: close"))
	return m.theme.OverlayBox.Render(b.String())
}

func (m Model) viewSavePrompt() string {
	body := m.theme.Title.Render("Save selection as lens") + "\n" +
		m.saveInput.View() + "\n" +
		m.theme.Dim.Render("enter: save · esc: cancel")
	return m.theme.OverlayBox.Render(body)
}

// overlay pastes the box onto the base view centered horizontally and
// vertically. Falls back to base if the box doesn't fit.
func overlay(base, box string, w, h int) string {
	boxLines := strings.Split(box, "\n")
	bw := 0
	for _, l := range boxLines {
		if n := lipgloss.Width(l); n > bw {
			bw = n
		}
	}
	bh := len(boxLines)
	if bw >= w || bh >= h {
		return base
	}
	baseLines := strings.Split(base, "\n")
	// pad base to h lines
	for len(baseLines) < h {
		baseLines = append(baseLines, "")
	}
	top := (h - bh) / 2
	left := (w - bw) / 2
	for i, bl := range boxLines {
		row := top + i
		if row >= h {
			break
		}
		// pad the base row to `left` columns, then overwrite.
		pad := left - lipgloss.Width(baseLines[row])
		if pad < 0 {
			pad = 0
		}
		baseLines[row] = baseLines[row] + strings.Repeat(" ", pad) + bl
	}
	return strings.Join(baseLines, "\n")
}
