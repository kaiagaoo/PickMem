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
		// Header: keep only if the immediately-following notes contain a
		// kept one before the next header.
		if hasKeptChild(rows, i, kept) {
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

// hasKeptChild returns true if any row between headerIdx (exclusive) and
// the next header (or end of list) is in `kept`.
func hasKeptChild(rows []row, headerIdx int, kept map[int]int) bool {
	for j := headerIdx + 1; j < len(rows); j++ {
		if rows[j].kind == kindHeader {
			return false
		}
		if _, ok := kept[j]; ok {
			return true
		}
	}
	return false
}
