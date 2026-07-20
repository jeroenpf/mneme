package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/appinfo"
)

func TestInstallHandlerServesInfo(t *testing.T) {
	info := appinfo.Info{
		Version:     "0.1.1",
		Mode:        "localhost",
		URL:         "http://localhost:8901",
		MCPEndpoint: "http://localhost:8901/mcp",
		DB:          appinfo.DBInfo{Driver: "sqlite", Path: "/home/u/.mneme/mneme.db"},
		Embeddings:  appinfo.EmbInfo{Enabled: false},
	}
	h := &InstallHandler{Info: info}

	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/v1/install", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type: got %q", ct)
	}
	var got appinfo.Info
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.MCPEndpoint != info.MCPEndpoint {
		t.Errorf("mcp_endpoint: got %q", got.MCPEndpoint)
	}
	if got.DB.Path != info.DB.Path {
		t.Errorf("db.path: got %q", got.DB.Path)
	}
}

func TestInstallHandlerBodyHasNoSecrets(t *testing.T) {
	// The Info struct has no field for the Voyage key or DB credentials, so the
	// serialized body structurally cannot contain them. Assert the contract so a
	// future field addition can't silently leak.
	h := &InstallHandler{Info: appinfo.Info{
		Version:    "0.1.1",
		Embeddings: appinfo.EmbInfo{Enabled: true, Model: "voyage-4-large"},
	}}
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/v1/install", nil))

	body := strings.ToLower(rec.Body.String())
	for _, forbidden := range []string{"api_key", "voyage_api_key", "token", "password", "secret"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("body must not contain %q: %s", forbidden, rec.Body.String())
		}
	}
	if !strings.Contains(body, "mcp_endpoint") {
		t.Errorf("body should expose mcp_endpoint: %s", rec.Body.String())
	}
}
