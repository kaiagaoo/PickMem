package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kaiagaoo/PickMem/internal/vault"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------- fixture ----------

// newFixture builds a small vault + running server. The client and server
// speak over an in-memory transport so tests exercise the real MCP
// dispatch path without touching stdio.
func newFixture(t *testing.T) (*vault.Store, *sdkmcp.ClientSession, []*vault.Note) {
	t.Helper()
	dir := t.TempDir()
	s, err := vault.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	notes := seedNotes(t, s)

	// Also stage one lens so use_lens has something to activate.
	if err := s.SaveLenses([]vault.Lens{
		{Name: "Work", ItemIDs: []string{notes[2].ID}},
	}); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	clientT, serverT := sdkmcp.NewInMemoryTransports()

	srv := NewServer(s, "test")
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return s, cs, notes
}

func seedNotes(t *testing.T, s *vault.Store) []*vault.Note {
	t.Helper()
	var notes []*vault.Note
	adds := []struct {
		label, group, body string
	}{
		{"salary", "financial", "monthly base $8k"},
		{"bills", "financial", "rent, utilities, internet"},
		{"client-acme kickoff", "work", "kickoff meeting Aug 1"},
	}
	for _, a := range adds {
		n, err := s.Add(&vault.Note{
			Frontmatter: vault.Frontmatter{Label: a.label, Group: a.group},
			Body:        a.body,
		})
		if err != nil {
			t.Fatal(err)
		}
		notes = append(notes, n)
	}
	return notes
}

// ---------- resource ----------

func TestActiveResourceEmptyByDefault(t *testing.T) {
	_, cs, _ := newFixture(t)
	res, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{URI: ActiveResourceURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Contents) != 1 {
		t.Fatalf("want 1 content, got %d", len(res.Contents))
	}
	text := res.Contents[0].Text
	if !strings.Contains(text, "no memory selected") {
		t.Errorf("empty resource missing default marker: %q", text)
	}
}

func TestActiveResourceReflectsPickedItems(t *testing.T) {
	s, cs, notes := newFixture(t)
	if err := s.SaveActive(vault.Active{
		ItemIDs: []string{notes[0].ID, notes[2].ID},
	}); err != nil {
		t.Fatal(err)
	}
	res, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{URI: ActiveResourceURI})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Contents[0].Text
	for _, want := range []string{"salary", "monthly base $8k", "client-acme kickoff", "kickoff meeting Aug 1"} {
		if !strings.Contains(text, want) {
			t.Errorf("assembled block missing %q; got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "bills") {
		t.Errorf("unpicked note leaked into resource:\n%s", text)
	}
}

// ---------- get_active_memory tool ----------

func TestGetActiveMemoryTool(t *testing.T) {
	s, cs, notes := newFixture(t)
	if err := s.SaveActive(vault.Active{ItemIDs: []string{notes[0].ID}}); err != nil {
		t.Fatal(err)
	}
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "get_active_memory"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %v", res.Content)
	}
	text := textOf(t, res)
	if !strings.Contains(text, "salary") || !strings.Contains(text, "monthly base") {
		t.Errorf("get_active_memory did not include picked note: %s", text)
	}
}

// ---------- list_lenses ----------

func TestListLensesTool(t *testing.T) {
	_, cs, _ := newFixture(t)
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "list_lenses"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	var got []struct {
		Name  string `json:"name"`
		Items int    `json:"items"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Work" || got[0].Items != 1 {
		t.Errorf("list_lenses = %+v, want [{Work,1}]", got)
	}
}

// ---------- use_lens ----------

func TestUseLensSwitchesActive(t *testing.T) {
	s, cs, notes := newFixture(t)
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "use_lens",
		Arguments: map[string]any{"name": "Work"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("use_lens error: %v", res.Content)
	}
	// Response should include the assembled block.
	text := textOf(t, res)
	if !strings.Contains(text, "client-acme kickoff") {
		t.Errorf("use_lens response missing lens content: %s", text)
	}
	// active.json should now reflect the lens.
	a, err := s.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if a.ActiveLens != "Work" || !reflect.DeepEqual(a.ItemIDs, []string{notes[2].ID}) {
		t.Errorf("active.json wrong after use_lens: %+v", a)
	}
}

func TestUseLensRejectsUnknown(t *testing.T) {
	_, cs, _ := newFixture(t)
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "use_lens",
		Arguments: map[string]any{"name": "does-not-exist"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Errorf("expected IsError on unknown lens; got content: %s", textOf(t, res))
	}
}

// ---------- propose_memories ----------

func TestProposeMemoriesStagesToInboxOnly(t *testing.T) {
	s, cs, _ := newFixture(t)
	chat := `I've been thinking about groceries. I want to try more grains and less processed food.

Also, the mortgage refinance is due for review — rates dropped again this week.

Random. Hi.

The kickoff meeting with Acme is at 10am on Aug 1. Bring the whiteboard markers.`

	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "propose_memories",
		Arguments: map[string]any{"chat_text": chat},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("propose error: %v", res.Content)
	}
	var pr ProposeResult
	if err := json.Unmarshal([]byte(textOf(t, res)), &pr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// "Random. Hi." is 10 chars → under the 12-char floor → dropped.
	if pr.Staged != 3 {
		t.Errorf("staged = %d, want 3 (the two-word 'Random. Hi.' should be filtered): %+v", pr.Staged, pr)
	}
	// Inbox should contain exactly those, all pending, all source=extract,
	// and NONE of them active.
	pending := s.ListPending()
	if len(pending) != 3 {
		t.Errorf("inbox has %d pending, want 3", len(pending))
	}
	for _, n := range pending {
		if n.Status != vault.StatusPending {
			t.Errorf("staged note not pending: %s", n.Status)
		}
		if n.Source != vault.SourceExtract {
			t.Errorf("staged note source=%s, want extract", n.Source)
		}
	}
	// Active must be untouched.
	a, err := s.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ItemIDs) != 0 {
		t.Errorf("propose activated something: %+v", a)
	}
}

func TestProposeDeDupesOnRepeatedRun(t *testing.T) {
	_, cs, _ := newFixture(t)
	chat := "The kickoff meeting with Acme is at 10am on Aug 1. Bring the whiteboard markers."

	call := func() ProposeResult {
		res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
			Name:      "propose_memories",
			Arguments: map[string]any{"chat_text": chat},
		})
		if err != nil {
			t.Fatal(err)
		}
		var pr ProposeResult
		if err := json.Unmarshal([]byte(textOf(t, res)), &pr); err != nil {
			t.Fatal(err)
		}
		return pr
	}

	first := call()
	second := call()
	if first.Staged != 1 || first.Duplicate != 0 {
		t.Errorf("first run: %+v, want staged=1 dup=0", first)
	}
	if second.Staged != 0 || second.Duplicate != 1 {
		t.Errorf("second run: %+v, want staged=0 dup=1", second)
	}
}

func TestProposeAppliesRoutingRules(t *testing.T) {
	s, cs, _ := newFixture(t)
	// Seed a routing rule so we can verify suggestGroup fires.
	cfg, _ := s.LoadConfig()
	cfg.RoutingRules = []vault.RoutingRule{{Keyword: "mortgage", Group: "financial"}}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	_, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "propose_memories",
		Arguments: map[string]any{"chat_text": "The mortgage refi is due for review this month."},
	})
	if err != nil {
		t.Fatal(err)
	}
	pending := s.ListPending()
	if len(pending) != 1 || pending[0].SuggestedGroup != "financial" {
		labels := make([]string, 0, len(pending))
		sort.Slice(pending, func(i, j int) bool { return pending[i].Label < pending[j].Label })
		for _, n := range pending {
			labels = append(labels, n.Label+"→"+n.SuggestedGroup)
		}
		t.Errorf("routing rule not applied: %v", labels)
	}
}

// ---------- helpers ----------

func textOf(t *testing.T, res *sdkmcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent: %T", res.Content[0])
	}
	return tc.Text
}
