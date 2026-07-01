// Package ingest converts raw text (chat transcripts, memory exports,
// generic lists) into memory-note candidates for the inbox. The text/
// helpers here are the shared bits — parsers in parse.go and the import
// pipeline in import.go build on them.
package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SplitParagraphs breaks input into candidate memories separated by one or
// more blank lines. Paragraphs shorter than MinLen are dropped as noise
// (things like "ok" and "thanks" that shouldn't become memory items).
func SplitParagraphs(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	raw := strings.Split(s, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if len(p) < MinLen {
			continue
		}
		out = append(out, p)
	}
	return out
}

// MinLen is the minimum character count for a paragraph to be treated as
// a memory candidate. Keeps chat filler ("ok", "yes", "thanks") from
// polluting the inbox. Exported so callers can tune it (tests, imports
// from short-item exports).
var MinLen = 12

// DeriveLabel makes a short human-readable label from a memory body.
// Uses the first sentence (or first line) truncated to 80 chars. Chosen
// over more complex heuristics because the reviewer sees it immediately
// and can edit before accepting — cheap and predictable beats clever.
func DeriveLabel(body string) string {
	first := body
	if i := strings.IndexAny(body, ".!?\n"); i > 0 {
		first = body[:i]
	}
	first = strings.TrimSpace(first)
	if len(first) > 80 {
		first = strings.TrimSpace(first[:80])
	}
	if first == "" {
		first = "memory candidate"
	}
	return first
}

// ContentHash returns a short hex digest keyed on whitespace-normalized,
// lowercased content. Used by import + propose_memories to skip already-
// staged items on re-runs. Not a cryptographic guarantee — just enough
// to avoid stupid duplicates.
func ContentHash(body string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(body), " "))
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}
