// Package retrieval ranks vault notes against a task description without
// changing the user's active memory. It is deliberately local and
// deterministic: suggestions help the user curate context, but never bypass
// PickMem's explicit approval boundary.
package retrieval

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

// Match is one ranked memory suggestion with an explanation suitable for a
// CLI or UI. Score is only meaningful relative to other matches for the same
// query.
type Match struct {
	Note         *vault.Note `json:"-"`
	Score        float64     `json:"score"`
	MatchedTerms []string    `json:"matched_terms"`
}

type document struct {
	note   *vault.Note
	terms  map[string]float64
	length float64
}

// Rank applies a field-weighted BM25 scorer. Labels receive the greatest
// weight, followed by tags and group names, while body text supplies recall.
// Notes with no query-term match are omitted.
func Rank(query string, notes []*vault.Note, limit int) []Match {
	queryTerms := unique(tokenize(query))
	if len(queryTerms) == 0 || limit == 0 {
		return []Match{}
	}

	docs := make([]document, 0, len(notes))
	df := make(map[string]int, len(queryTerms))
	var totalLength float64
	for _, n := range notes {
		if n == nil || n.Status != vault.StatusActive {
			continue
		}
		d := buildDocument(n)
		docs = append(docs, d)
		totalLength += d.length
		for _, term := range queryTerms {
			if d.terms[term] > 0 {
				df[term]++
			}
		}
	}
	if len(docs) == 0 {
		return []Match{}
	}

	avgLength := totalLength / float64(len(docs))
	const k1, b = 1.5, 0.75
	results := make([]Match, 0, len(docs))
	for _, d := range docs {
		var score float64
		matched := make([]string, 0, len(queryTerms))
		for _, term := range queryTerms {
			tf := d.terms[term]
			if tf == 0 {
				continue
			}
			matched = append(matched, term)
			idf := math.Log(1 + (float64(len(docs)-df[term])+0.5)/(float64(df[term])+0.5))
			norm := tf + k1*(1-b+b*d.length/avgLength)
			score += idf * (tf * (k1 + 1)) / norm
		}
		if score == 0 {
			continue
		}
		// A direct phrase match is a useful precision signal, especially for
		// short labels such as "response style" or "typescript stack".
		phrase := strings.ToLower(strings.TrimSpace(query))
		if len(phrase) >= 4 && (strings.Contains(strings.ToLower(d.note.Label), phrase) ||
			strings.Contains(strings.ToLower(d.note.Body), phrase)) {
			score *= 1.25
		}
		results = append(results, Match{Note: d.note, Score: score, MatchedTerms: matched})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Note.ID < results[j].Note.ID
		}
		return results[i].Score > results[j].Score
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// RankOverlap is a deliberately simple baseline for offline evaluation: it
// counts unique query terms present in each note, without field weights, IDF,
// or document-length normalization.
func RankOverlap(query string, notes []*vault.Note, limit int) []Match {
	queryTerms := unique(tokenize(query))
	results := make([]Match, 0, len(notes))
	for _, n := range notes {
		if n == nil || n.Status != vault.StatusActive {
			continue
		}
		present := map[string]bool{}
		for _, term := range tokenize(n.Label + " " + n.Group + " " + strings.Join(n.Tags, " ") + " " + n.Body) {
			present[term] = true
		}
		matched := make([]string, 0, len(queryTerms))
		for _, term := range queryTerms {
			if present[term] {
				matched = append(matched, term)
			}
		}
		if len(matched) > 0 {
			results = append(results, Match{Note: n, Score: float64(len(matched)), MatchedTerms: matched})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Note.ID < results[j].Note.ID
		}
		return results[i].Score > results[j].Score
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

func buildDocument(n *vault.Note) document {
	terms := map[string]float64{}
	add := func(text string, weight float64) {
		for _, term := range tokenize(text) {
			terms[term] += weight
		}
	}
	add(n.Label, 4)
	add(strings.Join(n.Tags, " "), 3)
	add(strings.ReplaceAll(n.Group, "/", " "), 2)
	add(n.Body, 1)
	var length float64
	for _, tf := range terms {
		length += tf
	}
	if length == 0 {
		length = 1
	}
	return document{note: n, terms: terms, length: length}
}

func tokenize(s string) []string {
	raw := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(raw))
	for _, term := range raw {
		if len(term) > 4 && strings.HasSuffix(term, "s") && !strings.HasSuffix(term, "ss") {
			term = strings.TrimSuffix(term, "s")
		}
		if !stopwords[term] {
			out = append(out, term)
		}
	}
	return out
}

var stopwords = map[string]bool{
	"a": true, "about": true, "an": true, "and": true, "are": true, "as": true,
	"at": true, "be": true, "for": true, "from": true, "help": true,
	"i": true, "in": true, "is": true, "it": true, "me": true,
	"my": true, "of": true, "on": true, "or": true, "the": true,
	"this": true, "to": true, "with": true,
}

func unique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
