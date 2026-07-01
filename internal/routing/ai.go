package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AIConfig configures the Anthropic-backed classifier.
type AIConfig struct {
	// APIKey — Anthropic API key. Required. Typically pulled from
	// $ANTHROPIC_API_KEY by the caller.
	APIKey string
	// Model — a small, fast Claude model is the right default for
	// classification. Overridable so users on a specific plan or eval
	// setup can point elsewhere.
	Model string
	// Endpoint — override for testing or for an OpenAI-compatible proxy.
	// Defaults to the official Anthropic Messages API.
	Endpoint string
	// HTTPClient — override for testing. Defaults to a client with a
	// short timeout; classification of one note should not stall an
	// entire import.
	HTTPClient *http.Client
}

// AIClassifier calls the Anthropic Messages API to pick a group from an
// existing taxonomy. Guardrails:
//   - Only proposes groups that appear in the passed `groups` list; the
//     response is validated against that set, so it can't invent taxonomy.
//   - "none" (or any unrecognized answer) returns "" so the Router falls
//     through cleanly.
//   - All errors return ("", err) — the Router swallows them, so an API
//     outage or bad key silently downgrades to rules-only.
type AIClassifier struct {
	cfg AIConfig
}

// NewAI builds an AIClassifier. Returns nil (safe for use with New(...))
// if the API key is empty, so callers can wire it conditionally without
// a branch.
func NewAI(cfg AIConfig) *AIClassifier {
	if cfg.APIKey == "" {
		return nil
	}
	if cfg.Model == "" {
		cfg.Model = "claude-haiku-4-5-20251001"
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.anthropic.com/v1/messages"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &AIClassifier{cfg: cfg}
}

func (c *AIClassifier) Name() string { return "ai" }

// anthropicRequest / anthropicResponse mirror the Messages API subset we
// use. Kept private so we don't leak SDK-shaped types into the interface.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Classify asks the model to name the best-fitting group. If groups is
// empty (no active taxonomy), it returns "" immediately — no point in
// spending a request when the reviewer has to pick from scratch anyway.
func (c *AIClassifier) Classify(ctx context.Context, body string, groups []string) (string, error) {
	if len(groups) == 0 {
		return "", nil
	}
	system := "You are a classifier. You will be given a short piece of text and a fixed list of category names. Reply with exactly one category name from the list, or the single word `none` if no category clearly fits. No punctuation, no explanation."

	user := fmt.Sprintf("categories:\n- %s\n\ntext:\n%s\n\ncategory:",
		strings.Join(groups, "\n- "), body)

	reqBody, err := json.Marshal(anthropicRequest{
		Model:     c.cfg.Model,
		MaxTokens: 32,
		System:    system,
		Messages:  []anthropicMessage{{Role: "user", Content: user}},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("anthropic api %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var out anthropicResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", errors.New(out.Error.Message)
	}
	answer := ""
	for _, block := range out.Content {
		if block.Type == "text" {
			answer += block.Text
		}
	}
	answer = strings.TrimSpace(strings.Trim(answer, "`\"' \n"))
	if answer == "" || strings.EqualFold(answer, "none") {
		return "", nil
	}
	// Guardrail: only accept answers that match an actual group in the
	// caller's taxonomy. Anything else silently degrades to "" so the
	// Router keeps walking the chain (or the reviewer picks by hand).
	for _, g := range groups {
		if strings.EqualFold(answer, g) {
			return g, nil
		}
	}
	return "", nil
}

// truncate keeps error messages small so they don't blow up the terminal
// when the API returns a huge HTML error page.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
