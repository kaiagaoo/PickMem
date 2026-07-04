package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaiagaoo/PickMem/internal/vault"
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

func TestImportStagesAsPending(t *testing.T) {
	s := newVault(t)
	data := []byte(`[
      "Salary is monthly base $8k plus quarterly bonus.",
      "Prefers meetings in the morning; blocks afternoons for deep work.",
      "Avoids seafood — allergic to shellfish."
    ]`)
	r, err := ImportBytes(context.Background(), s, data, FormatAuto)
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
		if n.Status != vault.StatusPending || n.Source != vault.SourceImport {
			t.Errorf("bad staged note: status=%s source=%s", n.Status, n.Source)
		}
	}
	if a, _ := s.LoadActive(); len(a.ItemIDs) != 0 {
		t.Errorf("import activated something: %+v", a)
	}
}

func TestImportAppliesRoutingRules(t *testing.T) {
	s := newVault(t)
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
	r, err := ImportBytes(context.Background(), s, data, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if r.Routed != 2 {
		t.Errorf("Routed = %d, want 2: %+v", r.Routed, r)
	}
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

func TestImportDedupesAcrossActiveAndPending(t *testing.T) {
	s := newVault(t)
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
	r, err := ImportBytes(context.Background(), s, data, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if r.Staged != 1 || r.Duplicate != 2 {
		t.Errorf("result = %+v, want Staged=1 Duplicate=2", r)
	}
}

func TestImportFile(t *testing.T) {
	s := newVault(t)
	tmp := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(tmp, []byte(`["A memory long enough to keep in the inbox."]`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ImportFile(context.Background(), s, tmp, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if r.Staged != 1 {
		t.Errorf("ImportFile Staged=%d, want 1", r.Staged)
	}
}
