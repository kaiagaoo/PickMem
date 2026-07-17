package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TaxonomyGroups returns the vault's category taxonomy derived from its
// FOLDER TREE: every directory under the vault root is a group, named by
// its forward-slash path. This is the taxonomy the user curates directly
// in Obsidian — make a folder, get a category — rather than being limited
// to the starter set.
//
// Excluded (never appear as categories, and never leave the disk via
// list_groups):
//   - the pickmem/ management dir (inbox, config, lenses — not categories)
//   - hidden dirs (dotdirs: .git, .obsidian, .trash) and node_modules
//   - any dir whose name starts with "_" — a PRIVATE category. You can
//     still file notes there by hand, but it's kept out of the list shared
//     with a model, so a sensitive folder name never gets exposed.
func (s *Store) TaxonomyGroups() []string {
	set := map[string]bool{}
	_ = filepath.WalkDir(s.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path == s.Root {
			return nil
		}
		if isExcludedTaxonomyDir(d.Name()) {
			return filepath.SkipDir
		}
		rel, relErr := filepath.Rel(s.Root, path)
		if relErr != nil {
			return nil
		}
		group := filepath.ToSlash(rel)
		if group == PickmemDir || strings.HasPrefix(group, PickmemDir+"/") {
			return filepath.SkipDir
		}
		set[group] = true
		return nil
	})
	return sortedKeys(set)
}

// KnownGroups is the authoritative taxonomy for classification and review:
// the folder tree (what the user curates), unioned with any group a note
// is actually filed under and any routing-rule target. The union keeps a
// note whose folder was renamed/removed from vanishing, and lets rules
// pre-declare a target before its folder exists. Folders are the primary
// source, so an empty folder created in Obsidian shows up immediately.
func (s *Store) KnownGroups() []string {
	set := map[string]bool{}
	for _, g := range s.TaxonomyGroups() {
		set[g] = true
	}
	for g := range s.Groups() {
		if g != "" {
			set[g] = true
		}
	}
	if cfg, err := s.LoadConfig(); err == nil {
		for _, r := range cfg.RoutingRules {
			if r.Group != "" {
				set[r.Group] = true
			}
		}
	}
	return sortedKeys(set)
}

// EnsureGroup creates an empty group folder so a group can exist before it
// holds any note (onboarding seeds groups this way). The path is sanitized
// to stay inside the vault and out of the managed pickmem/ dir. A no-op if
// the folder already exists.
func (s *Store) EnsureGroup(group string) error {
	clean, err := cleanGroup(group)
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(s.Root, filepath.FromSlash(clean)), 0o755)
}

func isExcludedTaxonomyDir(name string) bool {
	return strings.HasPrefix(name, ".") ||
		strings.HasPrefix(name, "_") ||
		name == "node_modules"
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
