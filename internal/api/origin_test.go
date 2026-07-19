package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginGuard(t *testing.T) {
	allowed := []string{"http://localhost:8765", "https://mneme.dev:8443"}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guard := OriginGuard(allowed)(next)

	cases := []struct {
		name   string
		origin string
		want   int
	}{
		// Native clients (Claude Code, curl) and browser navigations send no
		// Origin — they must pass, or the MCP transport and healthchecks break.
		{"no origin", "", http.StatusOK},
		{"allowed localhost", "http://localhost:8765", http.StatusOK},
		{"allowed mneme.dev", "https://mneme.dev:8443", http.StatusOK},
		// DNS-rebinding: a page on another origin scripting the local server.
		{"foreign origin blocked", "http://evil.example", http.StatusForbidden},
		// An allowed host but wrong scheme/port is still a different origin.
		{"wrong scheme blocked", "https://localhost:8765", http.StatusForbidden},
		{"wrong port blocked", "http://localhost:9999", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			guard.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("origin %q: got %d, want %d", tc.origin, rec.Code, tc.want)
			}
		})
	}
}

// An empty allow-list still permits no-Origin requests (native clients) while
// rejecting any browser origin — a safe closed default.
func TestOriginGuardEmptyAllowList(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	guard := OriginGuard(nil)(next)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("no-origin with empty allow-list should pass: got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rec = httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("origin with empty allow-list should be blocked: got %d", rec.Code)
	}
}
