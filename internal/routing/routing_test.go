package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/qwgao/pickmem/internal/vault"
)

func TestRulesFirstMatchWins(t *testing.T) {
	cfg := vault.Config{RoutingRules: []vault.RoutingRule{
		{Keyword: "mortgage", Group: "financial"},
		{Keyword: "mortgage broker", Group: "financial/loans"}, // more specific but declared later
	}}
	c := NewRules(cfg)
	got, err := c.Classify(context.Background(), "call the mortgage broker back", nil)
	if err != nil {
		t.Fatal(err)
	}
	// First rule wins — this documents the "order rules by specificity" contract.
	if got != "financial" {
		t.Errorf("got %q, want financial (first-match-wins)", got)
	}
}

func TestRulesCaseInsensitive(t *testing.T) {
	c := NewRules(vault.Config{RoutingRules: []vault.RoutingRule{{Keyword: "Doctor", Group: "health"}}})
	got, _ := c.Classify(context.Background(), "book DOCTOR appointment", nil)
	if got != "health" {
		t.Errorf("case-insensitive match failed: %q", got)
	}
}

func TestRulesNoMatchReturnsEmpty(t *testing.T) {
	c := NewRules(vault.Config{RoutingRules: []vault.RoutingRule{{Keyword: "specific", Group: "x"}}})
	got, _ := c.Classify(context.Background(), "totally unrelated body", nil)
	if got != "" {
		t.Errorf("expected empty on miss, got %q", got)
	}
}

func TestRulesSkipsEmptyKeyword(t *testing.T) {
	c := NewRules(vault.Config{RoutingRules: []vault.RoutingRule{
		{Keyword: "", Group: "junk"},
		{Keyword: "  ", Group: "junk2"},
		{Keyword: "real", Group: "ok"},
	}})
	got, _ := c.Classify(context.Background(), "really real match", nil)
	if got != "ok" {
		t.Errorf("empty-keyword rules leaked: got %q", got)
	}
}

// ---------- Router ----------

// mockClassifier lets us drive the router without external services.
type mockClassifier struct {
	name string
	fn   func(body string) (string, error)
}

func (m mockClassifier) Name() string { return m.name }
func (m mockClassifier) Classify(_ context.Context, body string, _ []string) (string, error) {
	return m.fn(body)
}

func TestRouterFallsThroughOnEmpty(t *testing.T) {
	a := mockClassifier{"a", func(string) (string, error) { return "", nil }}
	b := mockClassifier{"b", func(string) (string, error) { return "hit", nil }}
	r := New(a, b)
	if got := r.Suggest(context.Background(), "x", nil); got != "hit" {
		t.Errorf("router did not fall through: %q", got)
	}
}

func TestRouterStopsAtFirstHit(t *testing.T) {
	called := ""
	a := mockClassifier{"a", func(string) (string, error) { return "first", nil }}
	b := mockClassifier{"b", func(string) (string, error) { called = "b"; return "second", nil }}
	r := New(a, b)
	got := r.Suggest(context.Background(), "x", nil)
	if got != "first" {
		t.Errorf("router took second hit: %q", got)
	}
	if called == "b" {
		t.Error("router consulted b after a hit")
	}
}

func TestRouterSwallowsClassifierErrors(t *testing.T) {
	// AI-side errors must not fail the whole import — the reviewer will
	// pick a group by hand instead.
	a := mockClassifier{"a", func(string) (string, error) { return "", errors.New("api down") }}
	b := mockClassifier{"b", func(string) (string, error) { return "ok", nil }}
	r := New(a, b)
	if got := r.Suggest(context.Background(), "x", nil); got != "ok" {
		t.Errorf("router did not skip erroring classifier: %q", got)
	}
}

func TestRouterIgnoresNilClassifiers(t *testing.T) {
	// The CLI conditionally appends AIClassifier only when --allow-ai;
	// nil entries in that list must not blow up.
	r := New(nil, mockClassifier{"a", func(string) (string, error) { return "hit", nil }}, nil)
	if got := r.Suggest(context.Background(), "x", nil); got != "hit" {
		t.Errorf("nil-tolerance broken: %q", got)
	}
}

func TestRouterEmptyChainReturnsEmpty(t *testing.T) {
	r := New()
	if got := r.Suggest(context.Background(), "x", nil); got != "" {
		t.Errorf("empty chain returned %q, want empty", got)
	}
}
