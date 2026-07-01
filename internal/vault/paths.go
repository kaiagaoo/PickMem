package vault

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// PickmemDir is the vault subdirectory that PickMem manages: inbox notes,
// lenses.json, active.json, config.json. Everything else in the vault is
// user-owned.
const PickmemDir = "pickmem"

// InboxDir is where pending notes stage before being accepted into a group.
var InboxDir = filepath.Join(PickmemDir, "inbox")

// Vault-level filenames.
const (
	LensesFile = "lenses.json"
	ActiveFile = "active.json"
	ConfigFile = "config.json"
)

// slugRE keeps ASCII letters, digits, and dashes. Everything else collapses
// to a single dash; runs of dashes collapse; leading/trailing dashes drop.
var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a label into a filesystem-safe filename stem. It's a hint,
// not a key — collisions are resolved by appending a short id suffix at
// write time (see Store.pathFor).
func Slugify(label string) string {
	var b strings.Builder
	for _, r := range label {
		switch {
		case unicode.IsLetter(r):
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	s := slugRE.ReplaceAllString(strings.TrimSpace(b.String()), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "note"
	}
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// GroupToPath converts a frontmatter group like "work/Client-Acme" into a
// filesystem-relative directory. Empty group is legal and means the vault
// root.
func GroupToPath(group string) string {
	group = strings.Trim(group, "/")
	if group == "" {
		return ""
	}
	return filepath.FromSlash(group)
}

// PathToGroup is the inverse: given a directory relative to the vault root,
// return the canonical group name (forward-slash separated). Used only for
// diagnostics — frontmatter is the source of truth for grouping.
func PathToGroup(dir string) string {
	dir = filepath.ToSlash(dir)
	dir = strings.Trim(dir, "/")
	return dir
}
