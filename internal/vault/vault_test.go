package vault

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ---------- Frontmatter round-trip ----------

func TestNoteRoundTrip(t *testing.T) {
	orig := &Note{
		Frontmatter: Frontmatter{
			ID:        "01JAX5D9KX3M8VYZ8T5EK5JY7C",
			Label:     "income — freelance + salary",
			Group:     "financial",
			Tags:      []string{"money", "recurring"},
			Source:    SourceManual,
			Status:    StatusActive,
			CreatedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		},
		Body: "Freelance ~= $5k/mo, salary $8k/mo.\nBonuses land in March.",
	}
	data, err := orig.Serialize()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	got, err := ParseNote(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(orig.Frontmatter, got.Frontmatter) {
		t.Errorf("frontmatter drifted:\n want %+v\n got  %+v", orig.Frontmatter, got.Frontmatter)
	}
	if strings.TrimSpace(orig.Body) != strings.TrimSpace(got.Body) {
		t.Errorf("body drifted:\n want %q\n got  %q", orig.Body, got.Body)
	}
}

func TestNoteTypeRoundTripAndFactOmission(t *testing.T) {
	base := Frontmatter{
		ID:        "01JAX5D9KX3M8VYZ8T5EK5JY7C",
		Label:     "sailing idea",
		Group:     "hobbies",
		Source:    SourceManual,
		Status:    StatusActive,
		CreatedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// A non-default kind round-trips and appears on disk.
	idea := &Note{Frontmatter: base, Body: "try a solo overnight sail"}
	idea.Type = TypeIdea
	data, err := idea.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "type: idea") {
		t.Errorf("idea type not serialized:\n%s", data)
	}
	got, err := ParseNote(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind() != TypeIdea {
		t.Errorf("kind = %q, want idea", got.Kind())
	}

	// The default kind is omitted on disk but still reads back as fact.
	fact := &Note{Frontmatter: base, Body: "keeps a sailboat"}
	fact.Type = TypeFact
	data, err = fact.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "type:") {
		t.Errorf("fact type should be omitted on disk:\n%s", data)
	}
	got, _ = ParseNote(data)
	if got.Kind() != TypeFact {
		t.Errorf("kind = %q, want fact (default)", got.Kind())
	}
}

func TestParseNoteRejectsMissingRequiredFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"no id", "---\nlabel: x\ngroup: g\nsource: manual\nstatus: active\ncreated_at: 2026-07-01T12:00:00Z\n---\n\nhi"},
		{"no label", "---\nid: 01JAX5D9KX3M8VYZ8T5EK5JY7C\ngroup: g\nsource: manual\nstatus: active\ncreated_at: 2026-07-01T12:00:00Z\n---\n\nhi"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseNote([]byte(tc.body)); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// ---------- ULID stability ----------

func TestNewIDUniqueAndValid(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if !ValidID(id) {
			t.Fatalf("NewID produced invalid ULID: %s", id)
		}
		if seen[id] {
			t.Fatalf("duplicate ULID: %s", id)
		}
		seen[id] = true
	}
}

// ---------- Slug ----------

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Client Acme — kickoff notes": "client-acme-kickoff-notes",
		"  weird   spacing!!  ":       "weird-spacing",
		"Español ñoño":                "espa-ol-o-o",
		"":                            "note",
		"---":                         "note",
		"very " + strings.Repeat("a", 100) + " long": "very-" + strings.Repeat("a", 55),
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// ---------- Store: init, add, list, show, remove ----------

func newVault(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return s
}

func TestStoreInitCreatesLayout(t *testing.T) {
	s := newVault(t)
	for _, rel := range []string{
		filepath.Join(PickmemDir, "inbox"),
		filepath.Join(PickmemDir, ConfigFile),
		filepath.Join(PickmemDir, LensesFile),
		filepath.Join(PickmemDir, ActiveFile),
	} {
		if _, err := os.Stat(filepath.Join(s.Root, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}

func TestStoreInitIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Add a lens, then re-init — the lens must survive.
	if err := s.SaveLenses([]Lens{{Name: "Job-Hunt", ItemIDs: []string{"a"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(dir); err != nil {
		t.Fatalf("re-init: %v", err)
	}
	ls, err := s.LoadLenses()
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 1 || ls[0].Name != "Job-Hunt" {
		t.Errorf("lens lost on re-init: %+v", ls)
	}
}

func TestStoreAddListShowRemove(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "salary", Group: "financial"},
		Body:        "monthly base $8k",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if n.ID == "" {
		t.Fatal("Add did not assign ID")
	}
	if n.Status != StatusActive {
		t.Errorf("Add did not set status=active, got %s", n.Status)
	}
	if !strings.HasPrefix(n.RelPath, "financial/") {
		t.Errorf("wrong RelPath: %s", n.RelPath)
	}
	got, ok := s.Get(n.ID)
	if !ok || got.Label != "salary" {
		t.Errorf("Get after Add failed: %+v ok=%v", got, ok)
	}
	if all := s.List(); len(all) != 1 {
		t.Errorf("List size = %d, want 1", len(all))
	}
	if err := s.Remove(n.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := s.Get(n.ID); ok {
		t.Error("Get returned removed note")
	}
	if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(n.RelPath))); err == nil {
		t.Error("note file still on disk after Remove")
	}
}

func TestStoreAddDisambiguatesSlugCollision(t *testing.T) {
	s := newVault(t)
	n1, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "salary", Group: "financial"}, Body: "one"})
	if err != nil {
		t.Fatal(err)
	}
	n2, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "salary", Group: "financial"}, Body: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if n1.RelPath == n2.RelPath {
		t.Errorf("Add clobbered on slug collision: both at %s", n1.RelPath)
	}
	// Both files must still exist on disk.
	for _, n := range []*Note{n1, n2} {
		if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(n.RelPath))); err != nil {
			t.Errorf("missing on disk: %s (%v)", n.RelPath, err)
		}
	}
}

// ---------- Group resolution: frontmatter beats folder ----------

func TestGroupResolutionFrontmatterWinsOverFolder(t *testing.T) {
	s := newVault(t)
	// Handcraft a note file placed under "misc/" but declaring group=financial.
	body := []byte(`---
id: 01JAX5D9KX3M8VYZ8T5EK5JY7C
label: misplaced
group: financial
source: manual
status: active
created_at: 2026-07-01T12:00:00Z
---

body
`)
	target := filepath.Join(s.Root, "misc", "misplaced.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	groups := s.Groups()
	if _, ok := groups["financial"]; !ok {
		t.Errorf("expected group=financial from frontmatter, got %v", keys(groups))
	}
	if _, ok := groups["misc"]; ok {
		t.Errorf("folder name leaked into groups: %v", keys(groups))
	}
}

// ---------- Non-frontmatter files are ignored ----------

func TestReloadSkipsNonFrontmatterMarkdown(t *testing.T) {
	s := newVault(t)
	if err := os.WriteFile(filepath.Join(s.Root, "random.md"), []byte("just a note.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := len(s.List()); got != 0 {
		t.Errorf("plain markdown got indexed: %d notes", got)
	}
}

// ---------- Lens ops ----------

func TestLensRoundTripAndOps(t *testing.T) {
	s := newVault(t)
	orig := []Lens{
		{Name: "Job-Hunt", ItemIDs: []string{"a", "b"}},
		{Name: "Client-Acme", ItemIDs: []string{"c"}},
	}
	if err := s.SaveLenses(orig); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadLenses()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Errorf("round-trip drift:\n want %+v\n got  %+v", orig, got)
	}
	l, ok := FindLens(got, "Job-Hunt")
	if !ok || len(l.ItemIDs) != 2 {
		t.Errorf("FindLens: %+v ok=%v", l, ok)
	}
	got = UpsertLens(got, Lens{Name: "Job-Hunt", ItemIDs: []string{"a"}})
	if l, _ := FindLens(got, "Job-Hunt"); len(l.ItemIDs) != 1 {
		t.Errorf("Upsert did not replace: %+v", l)
	}
	got = UpsertLens(got, Lens{Name: "Groceries", ItemIDs: []string{"z"}})
	if _, ok := FindLens(got, "Groceries"); !ok {
		t.Error("Upsert did not append new lens")
	}
	got = RemoveLens(got, "Client-Acme")
	if _, ok := FindLens(got, "Client-Acme"); ok {
		t.Error("RemoveLens did not remove")
	}
}

// ---------- Active selection round-trip ----------

func TestActiveRoundTrip(t *testing.T) {
	s := newVault(t)
	orig := Active{ActiveLens: "Job-Hunt", ItemIDs: []string{"a", "b"}}
	if err := s.SaveActive(orig); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(orig, got) {
		t.Errorf("Active drift: want %+v got %+v", orig, got)
	}
}

func TestLoadActiveMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, PickmemDir), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ItemIDs) != 0 || a.ActiveLens != "" {
		t.Errorf("expected empty, got %+v", a)
	}
}

// ---------- Inbox lifecycle ----------

func TestInboxAcceptFlow(t *testing.T) {
	s := newVault(t)
	n, err := s.AddInbox(&Note{
		Frontmatter: Frontmatter{Label: "gift ideas for sis", SuggestedGroup: "personal"},
		Body:        "loves plants and enamel pins",
	})
	if err != nil {
		t.Fatalf("AddInbox: %v", err)
	}
	if n.Status != StatusPending {
		t.Errorf("AddInbox status=%s, want pending", n.Status)
	}
	if !strings.HasPrefix(n.RelPath, filepath.ToSlash(InboxDir)+"/") {
		t.Errorf("inbox path wrong: %s", n.RelPath)
	}
	// Snapshot the inbox path — AcceptInbox mutates n.RelPath in place.
	inboxRel := n.RelPath
	accepted, err := s.AcceptInbox(n.ID, "" /* use suggested */)
	if err != nil {
		t.Fatalf("AcceptInbox: %v", err)
	}
	if accepted.Status != StatusActive {
		t.Errorf("accepted status=%s, want active", accepted.Status)
	}
	if accepted.Group != "personal" {
		t.Errorf("accepted group=%s, want personal", accepted.Group)
	}
	if strings.Contains(accepted.RelPath, filepath.ToSlash(InboxDir)) {
		t.Errorf("accepted still under inbox: %s", accepted.RelPath)
	}
	// Old inbox file must be gone.
	if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(inboxRel))); err == nil {
		t.Error("old inbox file still exists after accept")
	}
	// New file must exist on disk.
	if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(accepted.RelPath))); err != nil {
		t.Errorf("new file missing: %v", err)
	}
}

func TestInboxRejectDeletes(t *testing.T) {
	s := newVault(t)
	n, err := s.AddInbox(&Note{Frontmatter: Frontmatter{Label: "junk", SuggestedGroup: "personal"}, Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RejectInbox(n.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(n.ID); ok {
		t.Error("rejected note still in index")
	}
	if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(n.RelPath))); err == nil {
		t.Error("rejected file still on disk")
	}
}

func TestReindexSkipsMalformedFrontmatterWithWarning(t *testing.T) {
	s := newVault(t)
	good, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "salary", Group: "finance"},
		Body:        "monthly base $8k",
	})
	if err != nil {
		t.Fatal(err)
	}
	// A half-typed Obsidian note: frontmatter block, no id.
	bad := filepath.Join(s.Root, "finance", "draft.md")
	if err := os.WriteFile(bad, []byte("---\nlabel: half-typed\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The load must succeed, keep the good note, and record a warning.
	if err := s.Reload(); err != nil {
		t.Fatalf("one malformed file failed the whole load: %v", err)
	}
	if _, ok := s.Get(good.ID); !ok {
		t.Error("good note lost after reload with a malformed sibling")
	}
	ws := s.Warnings()
	if len(ws) != 1 || !strings.Contains(ws[0], "finance/draft.md") {
		t.Errorf("warnings = %v, want one mentioning finance/draft.md", ws)
	}
	// A clean reload clears the warning.
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	if ws := s.Warnings(); len(ws) != 0 {
		t.Errorf("warnings not cleared after clean reload: %v", ws)
	}
}

func TestReindexSkipsDuplicateIDWithWarning(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "salary", Group: "finance"},
		Body:        "monthly base $8k",
	})
	if err != nil {
		t.Fatal(err)
	}
	// A stray copy of the note (e.g. a user's manual file duplicate).
	src := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dup := filepath.Join(s.Root, "finance", "salary-copy.md")
	if err := os.WriteFile(dup, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("duplicate id failed the whole load: %v", err)
	}
	if _, ok := s.Get(n.ID); !ok {
		t.Error("note with duplicated id missing from index entirely")
	}
	ws := s.Warnings()
	if len(ws) != 1 || !strings.Contains(ws[0], "duplicate id") {
		t.Errorf("warnings = %v, want one duplicate-id warning", ws)
	}
}

// ---------- The create-only invariant: user-authored files are inviolable ----------

// TestCreateOnlyNeverRewritesUserAuthoredFile is the load-bearing test for
// EXECUTION.md §4 invariant #1. We plant a user-authored markdown file
// (no frontmatter — the user's own Obsidian note) and drive the store
// through every operation. The file's bytes must be identical before and
// after.
func TestCreateOnlyNeverRewritesUserAuthoredFile(t *testing.T) {
	s := newVault(t)
	userPath := filepath.Join(s.Root, "personal", "diary.md")
	original := []byte("# my diary\n\ni had eggs for breakfast.\n")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	// Also plant a note with the SAME slug (`diary`) in the same group to
	// force the disambiguation path.
	if _, err := s.Add(&Note{
		Frontmatter: Frontmatter{Label: "diary", Group: "personal"},
		Body:        "pickmem-tracked diary",
	}); err != nil {
		t.Fatal(err)
	}
	// The user file must be byte-for-byte unchanged.
	after, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, after) {
		t.Errorf("user file was modified!\n before: %q\n after:  %q", original, after)
	}
}

// TestUpdateRefusesIfFileChangedOnDisk covers the tracked-hash guard.
func TestUpdateRefusesIfFileChangedOnDisk(t *testing.T) {
	s := newVault(t)
	n, err := s.Add(&Note{Frontmatter: Frontmatter{Label: "temp", Group: "personal"}, Body: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	// Simulate user editing the file directly (as in Obsidian).
	full := filepath.Join(s.Root, filepath.FromSlash(n.RelPath))
	if err := os.WriteFile(full, []byte("garbage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = s.Update(n.ID, func(n *Note) error {
		n.Body = "v2"
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "create-only") {
		t.Errorf("expected create-only refusal, got: %v", err)
	}
}

// ---------- helpers ----------

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
