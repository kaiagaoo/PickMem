// Package routing turns a raw memory body into a suggested group.
// Structure: pluggable Classifiers behind a common interface; a Router
// consults them in order and returns the first non-empty suggestion.
//
// Locked defaults (EXECUTION.md §1): rules are the default, AI is opt-in.
// The Router is safe to use with an empty classifier list — it just
// returns "" for everything, which the reviewer resolves by hand.
package routing

import (
	"context"
	"strings"

	"github.com/qwgao/pickmem/internal/vault"
)

// Classifier proposes a group for a memory body. Implementations receive
// the ordered list of groups that currently exist in the vault so an AI
// classifier can't invent new taxonomy behind the user's back — the spec
// says "proposing from the *existing* taxonomy" (§M4).
//
// Returning "" means "no confident suggestion"; the Router treats that as
// a miss and moves on to the next classifier.
type Classifier interface {
	Name() string
	Classify(ctx context.Context, body string, groups []string) (string, error)
}

// Router composes classifiers in priority order. The Router itself is
// stateless — construction is cheap and callers can rebuild between
// imports if config changed.
type Router struct {
	classifiers []Classifier
}

// New returns a Router with the given classifier chain. First non-empty
// suggestion wins. Nil entries are ignored so callers can conditionally
// append classifiers.
func New(cs ...Classifier) *Router {
	out := make([]Classifier, 0, len(cs))
	for _, c := range cs {
		if c != nil {
			out = append(out, c)
		}
	}
	return &Router{classifiers: out}
}

// Suggest walks the chain and returns the first classifier's non-empty
// answer. If every classifier returns "" or errors, Suggest returns "".
// Errors from AI classifiers are surfaced but non-fatal: the caller
// (import.go) uses this as a hint, not a hard requirement.
func (r *Router) Suggest(ctx context.Context, body string, groups []string) string {
	for _, c := range r.classifiers {
		g, err := c.Classify(ctx, body, groups)
		if err != nil {
			// Silently move on. Import shouldn't fail because someone's
			// API key expired; the reviewer will pick a group by hand.
			continue
		}
		if g != "" {
			return g
		}
	}
	return ""
}

// ---------- RulesClassifier ----------

// RulesClassifier applies vault.Config.RoutingRules — case-insensitive
// substring matches over the body. First matching rule wins so users can
// order rules by specificity.
type RulesClassifier struct {
	rules []vault.RoutingRule
}

// NewRules builds a RulesClassifier from the vault config. Blank keywords
// are skipped so a partially-edited config doesn't accidentally match
// every body.
func NewRules(cfg vault.Config) *RulesClassifier {
	// Copy so later config edits don't mutate our snapshot.
	rules := make([]vault.RoutingRule, 0, len(cfg.RoutingRules))
	for _, r := range cfg.RoutingRules {
		if strings.TrimSpace(r.Keyword) == "" || r.Group == "" {
			continue
		}
		rules = append(rules, r)
	}
	return &RulesClassifier{rules: rules}
}

func (c *RulesClassifier) Name() string { return "rules" }

// Classify performs case-insensitive substring matches. The `groups`
// parameter is ignored because rules are pre-declared with their target
// group — a rules author already chose which group a match maps to.
func (c *RulesClassifier) Classify(_ context.Context, body string, _ []string) (string, error) {
	hay := strings.ToLower(body)
	for _, r := range c.rules {
		if strings.Contains(hay, strings.ToLower(r.Keyword)) {
			return r.Group, nil
		}
	}
	return "", nil
}
