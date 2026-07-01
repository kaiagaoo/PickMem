package picker

import (
	"strings"

	"github.com/qwgao/pickmem/internal/vault"
)

// rowKind distinguishes the two things that appear in the scrollable list:
// group section headers (unselectable) and note rows (selectable).
type rowKind int

const (
	kindHeader rowKind = iota
	kindNote
)

// row is the unified element of the picker's list. Headers carry only a
// label; note rows point at a Note. Kept as a single flat type so scroll
// math and rendering share one path.
type row struct {
	kind  rowKind
	group string // set for both kinds
	note  *vault.Note
}

func (r row) selectable() bool { return r.kind == kindNote }

// matchText is the haystack the fuzzy filter searches. For notes we
// concatenate label + tags + body so a user typing a phrase they remember
// from the body finds it. For headers we return the group name so the
// filter doesn't hide a whole section when the group name itself matches.
func (r row) matchText() string {
	if r.kind == kindHeader {
		return r.group
	}
	var b strings.Builder
	b.WriteString(r.note.Label)
	b.WriteByte(' ')
	b.WriteString(strings.Join(r.note.Tags, " "))
	b.WriteByte(' ')
	b.WriteString(r.note.Body)
	return b.String()
}

// buildRows takes a set of notes and flattens them into (header, item,
// item, header, item, ...) rows sorted by group name, then by note id
// within each group.
func buildRows(notes []*vault.Note) []row {
	// The store already sorts notes by id; we just need to bucket by group
	// while preserving that inner order.
	buckets := map[string][]*vault.Note{}
	for _, n := range notes {
		buckets[n.Group] = append(buckets[n.Group], n)
	}
	groups := make([]string, 0, len(buckets))
	for g := range buckets {
		groups = append(groups, g)
	}
	// Alphabetical group order feels natural for a "table of contents"
	// browsing UI.
	sortStrings(groups)

	out := make([]row, 0, len(notes)+len(groups))
	for _, g := range groups {
		out = append(out, row{kind: kindHeader, group: g})
		for _, n := range buckets[g] {
			out = append(out, row{kind: kindNote, group: g, note: n})
		}
	}
	return out
}

// firstSelectableIndex returns the index of the first note row, or -1 if
// the list is empty. Used to place the cursor on open so it doesn't sit
// on a non-selectable header.
func firstSelectableIndex(rows []row) int {
	for i, r := range rows {
		if r.selectable() {
			return i
		}
	}
	return -1
}

// sortStrings is a tiny wrapper so we don't spread `sort` imports across
// small files.
func sortStrings(ss []string) {
	// Simple insertion sort — the list of groups is tiny.
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1] > ss[j]; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}
