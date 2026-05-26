package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/web"
)

func TestHandlerServesIndex(t *testing.T) {
	srv := httptest.NewServer(web.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type: got %q, want text/html…", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `<div id="app">`) {
		t.Errorf("body missing app root: %s", body)
	}
}

func TestHandlerFallsBackToIndexForUnknownPath(t *testing.T) {
	srv := httptest.NewServer(web.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/doc/vehicle-api")
	if err != nil {
		t.Fatalf("GET /doc/vehicle-api: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `<div id="app">`) {
		t.Errorf("SPA fallback didn't serve index.html: %s", body)
	}
}

func TestHandlerReturns404ForMissingAsset(t *testing.T) {
	// /assets/ misses must 404. Masking them behind index.html hides
	// bundler errors. Test holds regardless of whether the embedded
	// dist contains a real build or just the placeholder.
	srv := httptest.NewServer(web.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/assets/does-not-exist.js")
	if err != nil {
		t.Fatalf("GET /assets/x: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing asset: got %d, want 404", resp.StatusCode)
	}
}
