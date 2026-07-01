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
func assemble(s *vault.Store, active vault.Active) string {
	if len(active.ItemIDs) == 0 {
		return emptyMarker(active.ActiveLens)
	}
	var b strings.Builder
	if active.ActiveLens != "" {
		fmt.Fprintf(&b, "<!-- pickmem lens: %s -->\n\n", active.ActiveLens)
	}
	for i, id := range active.ItemIDs {
		n, ok := s.Get(id)
		if !ok {
			// Stale id — the note was deleted since active.json was
			// written. Skip silently rather than error; the picker sweeps
			// these on next open, but we don't want a broken pipe here.
			continue
		}
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		fmt.Fprintf(&b, "# %s  ·  %s\n\n", n.Label, n.Group)
		b.WriteString(strings.TrimRight(n.Body, "\n"))
		b.WriteByte('\n')
	}
	return b.String()
}

func emptyMarker(lens string) string {
	if lens != "" {
		return fmt.Sprintf("<!-- pickmem: lens %q is empty -->\n", lens)
	}
	return "<!-- pickmem: no memory selected -->\n"
}
