package routing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAINilOnEmptyKey(t *testing.T) {
	// Callers do: routing.New(..., routing.NewAI(cfg)). Passing a nil is
	// legal — the router filters nils. This lets the CLI append without
	// branching on --allow-ai.
	if c := NewAI(AIConfig{}); c != nil {
		t.Errorf("NewAI with empty key should return nil, got %T", c)
	}
}

func TestAIClassifyReturnsGroupFromTaxonomy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "financial") {
			t.Errorf("request body missing taxonomy: %s", body)
		}
		// Simulate the model naming an existing group.
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"financial"}]}`))
	}))
	defer srv.Close()

	c := NewAI(AIConfig{APIKey: "k", Endpoint: srv.URL, HTTPClient: srv.Client()})
	got, err := c.Classify(context.Background(), "reviewed my mortgage refi options this week", []string{"financial", "personal"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "financial" {
		t.Errorf("Classify = %q, want financial", got)
	}
}

func TestAIRejectsAnswerOutsideTaxonomy(t *testing.T) {
	// Guardrail: even if the model returns a plausible-sounding group
	// that isn't in the caller's list, we drop it so the AI can't invent
	// new taxonomy (EXECUTION.md §M4).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"business"}]}`))
	}))
	defer srv.Close()
	c := NewAI(AIConfig{APIKey: "k", Endpoint: srv.URL, HTTPClient: srv.Client()})
	got, _ := c.Classify(context.Background(), "x", []string{"financial", "personal"})
	if got != "" {
		t.Errorf("Classify accepted made-up group: %q", got)
	}
}

func TestAINoneReplyReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"none"}]}`))
	}))
	defer srv.Close()
	c := NewAI(AIConfig{APIKey: "k", Endpoint: srv.URL, HTTPClient: srv.Client()})
	got, err := c.Classify(context.Background(), "x", []string{"financial"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("`none` reply should map to empty, got %q", got)
	}
}

func TestAIEmptyTaxonomySkipsCall(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	c := NewAI(AIConfig{APIKey: "k", Endpoint: srv.URL, HTTPClient: srv.Client()})
	got, err := c.Classify(context.Background(), "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" || called {
		t.Errorf("empty taxonomy should short-circuit; got=%q called=%v", got, called)
	}
}

func TestAIErrorSurfacesForRouterToSwallow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit","message":"slow down"}}`))
	}))
	defer srv.Close()
	c := NewAI(AIConfig{APIKey: "k", Endpoint: srv.URL, HTTPClient: srv.Client()})
	_, err := c.Classify(context.Background(), "x", []string{"g"})
	if err == nil {
		t.Errorf("HTTP 429 should return an error so the Router can swallow it")
	}
}

// Belt-and-suspenders: the request body matches the shape the Messages API expects.
func TestAIRequestShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("missing anthropic-version header")
		}
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role string `json:"role"`
			} `json:"messages"`
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("body not JSON: %v", err)
		}
		if req.Model == "" {
			t.Error("model missing from request")
		}
		if len(req.Messages) == 0 || req.Messages[0].Role != "user" {
			t.Errorf("expected one user message, got %+v", req.Messages)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"none"}]}`))
	}))
	defer srv.Close()
	c := NewAI(AIConfig{APIKey: "k", Endpoint: srv.URL, HTTPClient: srv.Client()})
	_, _ = c.Classify(context.Background(), "x", []string{"g"})
}
