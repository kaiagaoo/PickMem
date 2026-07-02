// Package mcp exposes the picked slice of the vault to MCP clients
// (Claude Desktop, Cursor, Cline, …) over stdio. Everything here is a
// thin adapter over internal/vault — no data lives in this package.
package mcp

import (
	"fmt"
	"strings"

	"github.com/qwgao/pickmem/internal/vault"
)

// AssembleActive builds the context block the model sees: the bodies of
// currently-picked notes, each preceded by a lightweight provenance
// header. Deterministic — id lookup only, no similarity search.
//
// If the active selection is empty (the "default is nothing" state), we
// return a short marker string instead of "" so the model can tell the
// difference between "no memory selected" and "memory failed to load."
func AssembleActive(s *vault.Store) (string, error) {
	active, err := s.LoadActive()
	if err != nil {
		return "", err
	}
	return assemble(s, active), nil
}

// assemble is the pure part of AssembleActive — split out so tests can
// drive it with a hand-built Active struct without touching disk.
//
// Format: plain markdown, deliberately boring. An earlier version wrapped
// each item in XML-style tags on the theory that Claude follows tag
// boundaries well — but that's Claude-specific guidance, and this block
// also gets pasted verbatim into ChatGPT/Gemini by the extension, where
// there's no comparable evidence for tag-following behavior. Plain,
// provider-neutral markdown with a closing "--- end pickmem memory ---"
// line reads equally well to any model (and to a human glancing at the
// chat input before hitting send) without leaning on an unproven,
// model-specific claim.
//
// Single blank line between items, nowhere else. The closing line is
// also what gives the extension's Insert/Copy flow a clean boundary
// against whatever the user types next in the same input — no separate
// divider needed on the extension side.
//
// Kept byte-identical with extension/src/vault/assemble.ts — see
// CLAUDE.md's M5 note before changing either side.
func assemble(s *vault.Store, active vault.Active) string {
	if len(active.ItemIDs) == 0 {
		return emptyBlock(active.ActiveLens)
	}
	var b strings.Builder
	b.WriteString("--- pickmem: selected memory")
	if active.ActiveLens != "" {
		fmt.Fprintf(&b, " (lens: %s)", active.ActiveLens)
	}
	b.WriteString(" ---\n")
	first := true
	for _, id := range active.ItemIDs {
		n, ok := s.Get(id)
		if !ok {
			// Stale id — the note was deleted since active.json was
			// written. Skip silently rather than error; the picker sweeps
			// these on next open, but we don't want a broken pipe here.
			continue
		}
		if !first {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s (%s): %s\n", n.Label, n.Group, strings.TrimRight(n.Body, "\n"))
		first = false
	}
	b.WriteString("--- end pickmem memory ---\n")
	return b.String()
}

func emptyBlock(lens string) string {
	if lens != "" {
		return fmt.Sprintf("--- pickmem: lens %q is empty ---\n", lens)
	}
	return "--- pickmem: no memory selected ---\n"
}
