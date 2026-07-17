package picker

import (
	"strings"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

// rowKind distinguishes the two things that appear in the scrollable list:
// group section headers and note rows. Both are now selectable — a header
// toggles every note in its subtree.
type rowKind int

const (
	kindHeader rowKind = iota
	kindNote
)

// row is the unified element of the picker's list.
//
// A note row points at a Note. A header row represents one segment of the
// group tree: `header` is its own segment for display ("stack"), `group`
// is the full path down to it ("developer/stack"), and `descendants` is
// the ids of every note anywhere in its subtree — so toggling the header
// can select/deselect them all and its checkbox can reflect their state.
type row struct {
	kind        rowKind
	group       string // full group path, both kinds
	header      string // leaf segment, header rows only
	depth       int    // nesting level, 0 = top-level
	note        *vault.Note
	descendants []string // note ids in this subtree, header rows only
}

// selectable reports whether the cursor can land on and toggle this row.
// Both kinds are selectable now: notes toggle themselves, headers toggle
// their whole subtree.
func (r row) selectable() bool { return true }

// matchText is the haystack the fuzzy filter searches. For notes we
// concatenate label + tags + body so a user typing a phrase they remember
// from the body finds it. For headers we return the full group path so
// filtering on a group name keeps that section.
func (r row) matchText() string {
	if r.kind == kindHeader {
		return r.group
	}
	var b strings.Builder
	b.WriteString(r.note.Label)
	b.WriteByte(' ')
	b.WriteString(strings.Join(r.note.Tags, " ")) // so typing a tag filters by it
	b.WriteByte(' ')
	b.WriteString(r.note.Body)
	return b.String()
}

// groupNode is one segment of the group hierarchy, built by splitting each
// note's Group on "/". A vault with only flat, single-segment groups (the
// common case) degenerates to one level of children under root.
type groupNode struct {
	name     string
	fullPath string
	children map[string]*groupNode
	notes    []*vault.Note
}

// buildRows flattens the notes into a depth-first walk of the group tree:
// a header, then that group's own notes, then each child group (recursively
// the same shape). Groups are sorted alphabetically at every level; notes
// keep the store's id order.
func buildRows(notes []*vault.Note) []row {
	root := &groupNode{children: map[string]*groupNode{}}
	for _, n := range notes {
		cur := root
		path := ""
		for _, seg := range strings.Split(n.Group, "/") {
			if path == "" {
				path = seg
			} else {
				path = path + "/" + seg
			}
			child, ok := cur.children[seg]
			if !ok {
				child = &groupNode{name: seg, fullPath: path, children: map[string]*groupNode{}}
				cur.children[seg] = child
			}
			cur = child
		}
		cur.notes = append(cur.notes, n)
	}

	out := make([]row, 0, 2*len(notes))
	var walk func(node *groupNode, depth int)
	walk = func(node *groupNode, depth int) {
		for _, n := range node.notes {
			out = append(out, row{kind: kindNote, group: node.fullPath, depth: depth, note: n})
		}
		names := make([]string, 0, len(node.children))
		for name := range node.children {
			names = append(names, name)
		}
		sortStrings(names)
		for _, name := range names {
			child := node.children[name]
			out = append(out, row{
				kind:        kindHeader,
				group:       child.fullPath,
				header:      child.name,
				depth:       depth,
				descendants: subtreeNoteIDs(child),
			})
			walk(child, depth+1)
		}
	}
	walk(root, 0)
	return out
}

// subtreeNoteIDs collects the ids of every note at or below node, in the
// same depth-first order buildRows emits them.
func subtreeNoteIDs(node *groupNode) []string {
	var ids []string
	var visit func(n *groupNode)
	visit = func(n *groupNode) {
		for _, note := range n.notes {
			ids = append(ids, note.ID)
		}
		names := make([]string, 0, len(n.children))
		for name := range n.children {
			names = append(names, name)
		}
		sortStrings(names)
		for _, name := range names {
			visit(n.children[name])
		}
	}
	visit(node)
	return ids
}

// firstSelectableIndex returns the index of the first row the cursor may
// land on, or -1 if the list is empty. Every row is selectable now, so
// this is just "first row if any."
func firstSelectableIndex(rows []row) int {
	if len(rows) == 0 {
		return -1
	}
	return 0
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
