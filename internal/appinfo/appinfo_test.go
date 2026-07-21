package appinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/config"
)

func TestSummarizeLocalhost(t *testing.T) {
	cfg := &config.Config{Host: "127.0.0.1", Port: "8901", DSN: "sqlite:///home/u/.mneme/mneme.db"}
	got := Summarize(cfg, "0.1.1", false)

	if got.Mode != "localhost" {
		t.Errorf("mode: got %q, want localhost", got.Mode)
	}
	if got.URL != "http://localhost:8901" {
		t.Errorf("url: got %q, want http://localhost:8901", got.URL)
	}
	if got.MCPEndpoint != "http://localhost:8901/mcp" {
		t.Errorf("mcp: got %q", got.MCPEndpoint)
	}
	if got.Version != "0.1.1" {
		t.Errorf("version: got %q", got.Version)
	}
	if got.DB.Driver != "sqlite" || got.DB.Path != "/home/u/.mneme/mneme.db" {
		t.Errorf("db: got %+v", got.DB)
	}
	if got.Embeddings.Enabled {
		t.Errorf("embeddings should be disabled")
	}
	// The friendly URL must use localhost, never the 127.0.0.1 bind address —
	// the bug that started this feature.
	if strings.Contains(got.URL, "127.0.0.1") {
		t.Errorf("url must not use the bind address: %q", got.URL)
	}
}

func TestSummarizeMnemeDevWithTLS(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")
	for _, p := range []string{cert, key} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &config.Config{Host: "127.0.0.1", Port: "8443", DSN: "sqlite:///d/mneme.db", TLSCert: cert, TLSKey: key}
	got := Summarize(cfg, "0.1.1", false)

	if got.Mode != "mnemedev" {
		t.Errorf("mode: got %q, want mnemedev", got.Mode)
	}
	if got.URL != "https://mneme.dev:8443" {
		t.Errorf("url: got %q, want https://mneme.dev:8443", got.URL)
	}
	if got.MCPEndpoint != "https://mneme.dev:8443/mcp" {
		t.Errorf("mcp: got %q", got.MCPEndpoint)
	}
}

func TestSummarizeEmbeddingsEnabled(t *testing.T) {
	cfg := &config.Config{Port: "8901", DSN: "sqlite:///d/mneme.db", VoyageModel: "voyage-4-large"}
	got := Summarize(cfg, "0.1.1", true)

	if !got.Embeddings.Enabled {
		t.Errorf("embeddings should be enabled")
	}
	if got.Embeddings.Model != "voyage-4-large" {
		t.Errorf("model: got %q, want voyage-4-large", got.Embeddings.Model)
	}
}

func TestSummarizePostgresHidesCredentials(t *testing.T) {
	cfg := &config.Config{Port: "8901", DSN: "postgres://user:secretpass@host:5432/mnemedb"}
	got := Summarize(cfg, "0.1.1", false)

	if got.DB.Driver != "postgres" {
		t.Errorf("driver: got %q, want postgres", got.DB.Driver)
	}
	if strings.Contains(got.DB.Path, "secretpass") {
		t.Errorf("db path must not leak credentials: %q", got.DB.Path)
	}
}

func TestSummarizePublicURLOverridesDerived(t *testing.T) {
	// Behind a proxy/port-remap the bound host:port isn't reachable; PublicURL
	// wins for the advertised URL + MCP endpoint (trailing slash trimmed).
	cfg := &config.Config{
		Host: "127.0.0.1", Port: "8080", DSN: "sqlite:///d/mneme.db",
		PublicURL: "https://mneme.example.com/",
	}
	got := Summarize(cfg, "0.1.2", false)

	if got.URL != "https://mneme.example.com" {
		t.Errorf("url: got %q, want https://mneme.example.com", got.URL)
	}
	if got.MCPEndpoint != "https://mneme.example.com/mcp" {
		t.Errorf("mcp: got %q", got.MCPEndpoint)
	}
}
