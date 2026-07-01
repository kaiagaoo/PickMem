package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/qwgao/pickmem/internal/routing"
	"github.com/qwgao/pickmem/internal/vault"
)

func newVault(t *testing.T) *vault.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestImportBytesStagesAsPending(t *testing.T) {
	s := newVault(t)
	// Bare JSON string array — the simplest realistic export.
	data := []byte(`[
      "Salary is monthly base $8k plus quarterly bonus.",
      "Prefers meetings in the morning; blocks afternoons for deep work.",
      "Avoids seafood — allergic to shellfish."
    ]`)
	r, err := ImportBytes(context.Background(), s, data, FormatAuto, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Parsed != 3 || r.Staged != 3 {
		t.Errorf("result = %+v, want Parsed=3 Staged=3", r)
	}
	pending := s.ListPending()
	if len(pending) != 3 {
		t.Fatalf("inbox has %d pending, want 3", len(pending))
	}
	for _, n := range pending {
		if n.Status != vault.StatusPending {
			t.Errorf("staged note not pending: %s", n.Status)
		}
		if n.Source != vault.SourceImport {
			t.Errorf("staged note source=%s, want import", n.Source)
		}
	}
	// Active must be untouched.
	if a, _ := s.LoadActive(); len(a.ItemIDs) != 0 {
		t.Errorf("import activated something: %+v", a)
	}
}

func TestImportDedupesAcrossActiveAndPending(t *testing.T) {
	s := newVault(t)
	// Pre-seed one active and one pending with the same body as our import.
	if _, err := s.Add(&vault.Note{
		Frontmatter: vault.Frontmatter{Label: "active dup", Group: "personal"},
		Body:        "Avoids seafood — allergic to shellfish.",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddInbox(&vault.Note{
		Frontmatter: vault.Frontmatter{Label: "pending dup"},
		Body:        "Prefers meetings in the morning; blocks afternoons for deep work.",
	}); err != nil {
		t.Fatal(err)
	}

	data := []byte(`[
      "Salary is monthly base $8k plus quarterly bonus.",
      "Prefers meetings in the morning; blocks afternoons for deep work.",
      "Avoids seafood — allergic to shellfish."
    ]`)
	r, err := ImportBytes(context.Background(), s, data, FormatAuto, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Staged != 1 || r.Duplicate != 2 {
		t.Errorf("result = %+v, want Staged=1 Duplicate=2", r)
	}
}

func TestImportAppliesRoutingRules(t *testing.T) {
	s := newVault(t)
	// Seed routing rules so we can verify SuggestedGroup gets filled.
	cfg, _ := s.LoadConfig()
	cfg.RoutingRules = []vault.RoutingRule{
		{Keyword: "salary", Group: "financial"},
		{Keyword: "meeting", Group: "work"},
	}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	data := []byte(`[
      "Salary is monthly base $8k plus quarterly bonus.",
      "Prefers meetings in the morning.",
      "Loves plants and enamel pins."
    ]`)
	r, err := ImportBytes(context.Background(), s, data, FormatAuto, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Routed != 2 {
		t.Errorf("Routed = %d, want 2 (salary+meeting matched, plants unmatched): %+v", r.Routed, r)
	}
	// Verify each note landed with the right suggestion.
	got := map[string]string{}
	for _, n := range s.ListPending() {
		got[n.Label] = n.SuggestedGroup
	}
	if got["Salary is monthly base $8k plus quarterly bonus"] != "financial" {
		t.Errorf("salary not routed to financial: %v", got)
	}
	if got["Prefers meetings in the morning"] != "work" {
		t.Errorf("meeting not routed to work: %v", got)
	}
	if got["Loves plants and enamel pins"] != "" {
		t.Errorf("unmatched note got a suggestion: %v", got)
	}
}

func TestImportRouterAppliedInsteadOfRules(t *testing.T) {
	// Injected Router overrides the default. This is how --allow-ai will
	// wire the AI classifier in for unmatched items.
	s := newVault(t)
	stub := stubClassifier{group: "stubbed"}
	router := routing.New(&stub)
	data := []byte(`["Any body will do because the stub always returns stubbed."]`)
	if _, err := ImportBytes(context.Background(), s, data, FormatAuto, router); err != nil {
		t.Fatal(err)
	}
	pending := s.ListPending()
	if len(pending) != 1 || pending[0].SuggestedGroup != "stubbed" {
		t.Errorf("injected router not applied: %+v", pending)
	}
}

func TestImportFile(t *testing.T) {
	s := newVault(t)
	tmp := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(tmp, []byte(`["A memory long enough to keep in the inbox."]`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ImportFile(context.Background(), s, tmp, FormatAuto, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Staged != 1 {
		t.Errorf("ImportFile Staged=%d, want 1", r.Staged)
	}
}

// The DoD test: a 30+ item export imports cleanly, stages to inbox, and
// nothing activates.
func TestImport30ItemFixtureStagesAll(t *testing.T) {
	s := newVault(t)
	items := make([]string, 0, 34)
	for i := 0; i < 34; i++ {
		items = append(items, fmt.Sprintf(
			`Memory #%d: this line is long enough to survive the 12-char filter and unique enough to dodge the dedupe.`, i))
	}
	// Serialize as bare JSON array.
	data := []byte(`[`)
	for i, s := range items {
		if i > 0 {
			data = append(data, ',')
		}
		data = append(data, '"')
		data = append(data, s...)
		data = append(data, '"')
	}
	data = append(data, ']')

	r, err := ImportBytes(context.Background(), s, data, FormatJSON, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Staged != 34 || r.Parsed != 34 {
		t.Errorf("30+ import: %+v; want Parsed=Staged=34", r)
	}
	if len(s.ListPending()) != 34 {
		t.Errorf("inbox has %d pending, want 34", len(s.ListPending()))
	}
	if a, _ := s.LoadActive(); len(a.ItemIDs) != 0 {
		t.Errorf("import activated notes: %+v", a)
	}
}

// ---------- helpers ----------

type stubClassifier struct{ group string }

func (s stubClassifier) Name() string { return "stub" }
func (s stubClassifier) Classify(_ context.Context, _ string, _ []string) (string, error) {
	return s.group, nil
}
