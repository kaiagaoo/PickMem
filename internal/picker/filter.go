package picker

import (
	"strings"

	"github.com/sahilm/fuzzy"
)

// filterRows applies a fuzzy query to note rows and returns the surviving
// rows plus their group headers. Empty query is a pass-through.
//
// Group headers are always retained if any of their notes match, so the
// user sees the section they're filtering within. If no notes in a group
// match, the header drops too — otherwise the list would be visually
// noisy.
func filterRows(rows []row, query string) []row {
	q := strings.TrimSpace(query)
	if q == "" {
		return rows
	}

	// Score every note row via fuzzy; keep those that match.
	type indexed struct {
		i    int
		row  row
		rank int
	}
	var candidates []indexed
	haystack := make([]string, 0, len(rows))
	noteIdx := make([]int, 0, len(rows))
	for i, r := range rows {
		if r.kind == kindNote {
			haystack = append(haystack, r.matchText())
			noteIdx = append(noteIdx, i)
		}
	}
	matches := fuzzy.Find(q, haystack)
	// A "kept notes" set for cheap group-header inclusion below.
	kept := make(map[int]int, len(matches))
	for _, m := range matches {
		kept[noteIdx[m.Index]] = m.Score
	}
	for i, r := range rows {
		if r.kind == kindNote {
			if score, ok := kept[i]; ok {
				candidates = append(candidates, indexed{i: i, row: r, rank: score})
			}
			continue
		}
		// Header: keep if any note anywhere in its subtree was kept, so
		// ancestor headers survive when a deeply-nested note matches.
		if hasKeptDescendant(rows, i, kept) {
			candidates = append(candidates, indexed{i: i, row: r, rank: 0})
		}
	}
	// Preserve original row order (which is groups-alphabetical + notes-by-id).
	out := make([]row, len(candidates))
	for j, c := range candidates {
		out[j] = c.row
	}
	// candidates are already in original-index order because we walked
	// rows linearly, so no re-sort needed.
	return out
}

// hasKeptDescendant returns true if any kept note lives in the subtree of
// the header at headerIdx. A header's subtree is every following row with
// a greater depth, up to the next row at the same or shallower depth — so
// this correctly retains ancestor headers (e.g. "about") when a grandchild
// note ("about/health/chronic") matches.
func hasKeptDescendant(rows []row, headerIdx int, kept map[int]int) bool {
	headerDepth := rows[headerIdx].depth
	for j := headerIdx + 1; j < len(rows); j++ {
		if rows[j].depth <= headerDepth {
			return false // left this header's subtree
		}
		if rows[j].kind == kindNote {
			if _, ok := kept[j]; ok {
				return true
			}
		}
	}
	return false
}
