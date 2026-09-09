package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

func TestSuggestionsEndpointRanksWithoutActivating(t *testing.T) {
	store, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n, err := store.Add(&vault.Note{
		Frontmatter: vault.Frontmatter{Label: "Go backend", Group: "work/stack", Tags: []string{"golang"}},
		Body:        "Builds backend services in Go.",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/suggestions",
		bytes.NewBufferString(`{"query":"debug my Go backend","limit":3}`))
	req.Host = "127.0.0.1"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	NewServer(store).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Items []suggestionDTO `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != n.ID {
		t.Fatalf("suggestions = %+v, want note %s", body.Items, n.ID)
	}
	active, err := store.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if len(active.ItemIDs) != 0 {
		t.Fatalf("suggestions changed active memory: %+v", active)
	}
}
