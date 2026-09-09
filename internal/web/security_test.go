package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalRequestGuardAllowsLoopback(t *testing.T) {
	h := localRequestGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:4577/api/state", nil)
	req.Host = "127.0.0.1:4577"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestLocalRequestGuardRejectsCrossOriginMutation(t *testing.T) {
	called := false
	h := localRequestGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:4577/api/vault/clear", nil)
	req.Host = "127.0.0.1:4577"
	req.Header.Set("Origin", "https://malicious.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%v, want forbidden before handler", rr.Code, called)
	}
}

func TestLocalRequestGuardRejectsDNSRebindingHost(t *testing.T) {
	h := localRequestGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "http://attacker.example/api/state", nil)
	req.Host = "attacker.example"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want forbidden", rr.Code)
	}
}
