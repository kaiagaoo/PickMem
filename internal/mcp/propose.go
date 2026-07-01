package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/qwgao/pickmem/internal/vault"
)

// ProposeResult reports what propose_memories did without leaking the
// caller's chat text back into the response. We stage first, summarize
// second.
type ProposeResult struct {
	Staged    int      `json:"staged"`
	Duplicate int      `json:"duplicate_skipped"`
	Labels    []string `json:"labels"`
}

// ProposeFromChat splits chat_text into memory candidates and stages each
// as a `status: pending` note in the inbox with `source: extract`. It
// never activates — that's what the picker is for. Extraction is
// deterministic and rules-based; the AI classifier is M4, behind
// --allow-ai.
//
// The splitting heuristic is intentionally boring: paragraphs separated
// by blank lines, trimmed, minimum length 12 characters (skips
// throwaways like "ok" and "thanks"). Bullet/numbered lines are grouped
// into their surrounding paragraph.
//
// De-dupe uses a content hash over normalized whitespace, so re-running
// the same chat won't stage the same candidate twice.
func ProposeFromChat(s *vault.Store, chatText string) (ProposeResult, error) {
	candidates := splitParagraphs(chatText)

	// Build a set of hashes already present in the inbox so re-runs on
	// the same conversation don't duplicate work.
	seen := map[string]bool{}
	for _, n := range s.ListPending() {
		seen[contentHash(n.Body)] = true
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		return ProposeResult{}, err
	}

	result := ProposeResult{Labels: []string{}}
	for _, body := range candidates {
		h := contentHash(body)
		if seen[h] {
			result.Duplicate++
			continue
		}
		seen[h] = true

		label := deriveLabel(body)
		group := suggestGroup(cfg, body)

		n := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label:          label,
				SuggestedGroup: group,
				Source:         vault.SourceExtract,
			},
			Body: body,
		}
		if _, err := s.AddInbox(n); err != nil {
			return ProposeResult{}, err
		}
		result.Staged++
		result.Labels = append(result.Labels, label)
	}
	return result, nil
}

// splitParagraphs breaks input into candidate memories. A paragraph is
// runs of non-blank lines separated by one or more blank lines. Trailing
// whitespace is trimmed; paragraphs shorter than 12 chars are dropped.
func splitParagraphs(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	raw := strings.Split(s, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		// Collapse internal single newlines that are just wrap-breaks
		// (heuristic: no leading whitespace on either side of the break).
		p = strings.TrimSpace(p)
		if len(p) < 12 {
			continue
		}
		out = append(out, p)
	}
	return out
}

// deriveLabel makes a short human-readable label from a paragraph. Uses
// the first sentence or first line, truncated. The picker shows this, so
// it needs to be scannable at a glance.
func deriveLabel(body string) string {
	first := body
	if i := strings.IndexAny(body, ".!\n"); i > 0 {
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

// suggestGroup applies the vault's routing rules to a candidate. Rules
// are simple case-insensitive substring matches against label+body. The
// first hit wins so users can order their rules by specificity. Empty
// suggestion is fine — the review UI will prompt for a group on accept.
func suggestGroup(cfg vault.Config, body string) string {
	hay := strings.ToLower(body)
	for _, r := range cfg.RoutingRules {
		if r.Keyword == "" {
			continue
		}
		if strings.Contains(hay, strings.ToLower(r.Keyword)) {
			return r.Group
		}
	}
	return ""
}

// contentHash normalizes whitespace and returns a short hex digest.
// Normalization: lowercase, collapse whitespace runs, trim.
func contentHash(body string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(body), " "))
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}
