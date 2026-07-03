// Package templates ships the starter taxonomy: a tree of group folders,
// a pickmem/config.json seeded with routing rules, and a vault README —
// enough scaffolding that a new user has somewhere for their first note
// to land and a rough map of how to organize.
package templates

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultName is the template `pickmem init` applies unless --bare is
// passed. There's exactly one starter template; the one-vault strategy
// (see PROPOSAL.md) made per-persona templates unnecessary — nested
// groups cover personal/work/research in a single taxonomy instead.
const DefaultName = "starter"

//go:embed all:starter
var files embed.FS

// Available returns the names of shipped templates, sorted.
func Available() []string {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// Apply copies the named template into targetVault. Existing files are
// never overwritten — this is create-only.
func Apply(name, targetVault string) error {
	entries, err := files.ReadDir(name)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("unknown template %q (available: %s)", name, strings.Join(Available(), ", "))
	}
	return fs.WalkDir(files, name, func(embedPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(embedPath, name+"/")
		if rel == "" {
			return nil
		}
		// A convention: the embedded name `_gitkeep` becomes `.gitkeep`
		// on disk, so empty group folders can be tracked in git.
		outRel := strings.ReplaceAll(rel, "_gitkeep", ".gitkeep")
		outPath := filepath.Join(targetVault, outRel)

		if _, err := os.Stat(outPath); err == nil {
			return nil // never clobber
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		data, err := files.ReadFile(embedPath)
		if err != nil {
			return err
		}
		return os.WriteFile(outPath, data, 0o644)
	})
}
