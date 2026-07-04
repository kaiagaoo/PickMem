package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kaiagaoo/PickMem/internal/vault"
)

func callStage(t *testing.T, cs *sdkmcp.ClientSession, items []map[string]any) StageResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "stage_memories",
		Arguments: map[string]any{"items": items},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("stage error: %v", res.Content)
	}
	var sr StageResult
	if err := json.Unmarshal([]byte(textOf(t, res)), &sr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return sr
}

// ---------- list_groups ----------

func TestListGroupsUnionsNoteGroupsAndRuleTargets(t *testing.T) {
	s, cs, _ := newFixture(t)
	// A rule target that no note uses yet — must still be listed, because
	// on a fresh vault the starter taxonomy exists only as rules+folders.
	cfg, _ := s.LoadConfig()
	cfg.RoutingRules = []vault.RoutingRule{{Keyword: "gym", Group: "about/health"}}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "list_groups",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var groups []string
	if err := json.Unmarshal([]byte(textOf(t, res)), &groups); err != nil {
		t.Fatal(err)
	}
	want := []string{"about/health", "financial", "work"}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("list_groups = %v, want %v", groups, want)
	}
}

// ---------- stage_memories ----------

func TestStageMemoriesStagesPendingOnly(t *testing.T) {
	s, cs, _ := newFixture(t)
	sr := callStage(t, cs, []map[string]any{
		{
			"label":           "prefers vim",
			"body":            "Uses vim everywhere; don't suggest vscode extensions.",
			"suggested_group": "work",
		},
		{
			"body": "Allergic to penicillin.",
		},
	})

	if sr.Staged != 2 || sr.Duplicate != 0 || sr.Skipped != 0 {
		t.Fatalf("result = %+v, want 2 staged", sr)
	}
	// Second item had no label — derived from the body's first sentence.
	if sr.Items[1].Label != "Allergic to penicillin" {
		t.Errorf("derived label = %q", sr.Items[1].Label)
	}

	pending := s.ListPending()
	if len(pending) != 2 {
		t.Fatalf("inbox has %d pending, want 2", len(pending))
	}
	for _, n := range pending {
		if n.Status != vault.StatusPending || n.Source != vault.SourceExtract {
			t.Errorf("staged note status=%s source=%s, want pending/extract", n.Status, n.Source)
		}
		if !strings.HasPrefix(n.RelPath, "pickmem/inbox/") {
			t.Errorf("staged outside the inbox: %s", n.RelPath)
		}
	}

	// Active selection must be untouched — staging never activates.
	a, err := s.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ItemIDs) != 0 {
		t.Errorf("stage_memories activated something: %+v", a)
	}
}

func TestStageMemoriesRejectsUnknownGroupWithFallback(t *testing.T) {
	s, cs, _ := newFixture(t)
	cfg, _ := s.LoadConfig()
	cfg.RoutingRules = []vault.RoutingRule{{Keyword: "salary", Group: "financial"}}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	sr := callStage(t, cs, []map[string]any{
		{
			"label":           "raise",
			"body":            "Got a salary raise to $9k in July 2026.",
			"suggested_group": "money/raises", // does not exist
		},
	})
	if sr.Staged != 1 {
		t.Fatalf("result = %+v, want 1 staged", sr)
	}
	it := sr.Items[0]
	if it.SuggestedGroup != "financial" {
		t.Errorf("fallback group = %q, want financial (rules)", it.SuggestedGroup)
	}
	if !strings.Contains(it.Warning, "money/raises") {
		t.Errorf("warning should name the rejected group: %q", it.Warning)
	}
	pending := s.ListPending()
	if len(pending) != 1 || pending[0].SuggestedGroup != "financial" {
		t.Errorf("note carries wrong suggestion: %+v", pending)
	}
}

func TestStageMemoriesDeDupes(t *testing.T) {
	s, cs, notes := newFixture(t)
	_ = notes
	sr := callStage(t, cs, []map[string]any{
		// Duplicate of an ACTIVE note's body (seeded "monthly base $8k") —
		// re-saving accepted memory must not create a pending copy.
		{"label": "salary again", "body": "monthly base $8k"},
		// Same body twice within one call: second is a duplicate.
		{"label": "cat", "body": "Has a cat named Miso."},
		{"label": "cat 2", "body": "has a cat named MISO."},
	})
	if sr.Staged != 1 || sr.Duplicate != 2 {
		t.Fatalf("result = %+v, want staged=1 dup=2", sr)
	}
	if len(s.ListPending()) != 1 {
		t.Errorf("inbox has %d pending, want 1", len(s.ListPending()))
	}
}

func TestStageMemoriesSkipsEmptyBodies(t *testing.T) {
	_, cs, _ := newFixture(t)
	sr := callStage(t, cs, []map[string]any{
		{"label": "blank", "body": "   "},
		{"label": "fine", "body": "Prefers window seats on flights."},
	})
	if sr.Staged != 1 || sr.Skipped != 1 {
		t.Fatalf("result = %+v, want staged=1 skipped=1", sr)
	}
	if sr.Items[0].Outcome != "skipped" || sr.Items[0].Warning == "" {
		t.Errorf("empty body not reported: %+v", sr.Items[0])
	}
}

func TestStageMemoriesErrorsOnNoItems(t *testing.T) {
	_, cs, _ := newFixture(t)
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "stage_memories",
		Arguments: map[string]any{"items": []map[string]any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Errorf("empty items should be a tool error, got: %v", res.Content)
	}
}
