package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	isolateConfig(t)
	t.Setenv("MNEME_PORT", "")
	t.Setenv("MNEME_ENV", "")
	t.Setenv("MNEME_CORS_ORIGINS", "")

	// An explicit DSN passes straight through, ahead of the file/default layers.
	t.Setenv("MNEME_DSN", "postgres://x@y/z")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != "postgres://x@y/z" {
		t.Errorf("DSN: got %q", c.DSN)
	}
}

func TestLoadFromEnv(t *testing.T) {
	isolateConfig(t)
	t.Setenv("MNEME_DSN", "postgres://u:p@h/d?sslmode=disable")
	t.Setenv("MNEME_PORT", "9090")
	t.Setenv("MNEME_ENV", "production")
	t.Setenv("MNEME_CORS_ORIGINS", "http://a.example , http://b.example,, http://c.example")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != "postgres://u:p@h/d?sslmode=disable" {
		t.Errorf("DSN: got %q", c.DSN)
	}
	if c.Port != "9090" {
		t.Errorf("Port: got %q", c.Port)
	}
	if c.Env != "production" {
		t.Errorf("Env: got %q", c.Env)
	}
	want := []string{"http://a.example", "http://b.example", "http://c.example"}
	if !reflect.DeepEqual(c.CORSOrigins, want) {
		t.Errorf("CORSOrigins: got %v, want %v (trimming + empty-skip)", c.CORSOrigins, want)
	}
}

// The server binds loopback by default (local-only), overridable via MNEME_HOST
// (Docker sets 0.0.0.0 so its port mapping works).
func TestHostDefaultsToLoopback(t *testing.T) {
	isolateConfig(t)
	clearMnemeEnv(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Host != "127.0.0.1" {
		t.Errorf("default Host: got %q, want 127.0.0.1", c.Host)
	}

	t.Setenv("MNEME_HOST", "0.0.0.0")
	c, _ = Load()
	if c.Host != "0.0.0.0" {
		t.Errorf("MNEME_HOST override: got %q", c.Host)
	}
}

func TestListenAddr(t *testing.T) {
	c := &Config{Host: "127.0.0.1", Port: "8080"}
	if got := c.ListenAddr(); got != "127.0.0.1:8080" {
		t.Errorf("ListenAddr: got %q, want 127.0.0.1:8080", got)
	}
}

func TestTLSDisabledByDefault(t *testing.T) {
	isolateConfig(t)
	t.Setenv("MNEME_TLS_CERT", "")
	t.Setenv("MNEME_TLS_KEY", "")
	t.Setenv("MNEME_DSN", "postgres://x@y/z")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.TLSCert != "" || c.TLSKey != "" {
		t.Errorf("TLS paths: got %q / %q, want empty defaults", c.TLSCert, c.TLSKey)
	}
	if c.TLSEnabled() {
		t.Error("TLSEnabled: got true with no TLS env set")
	}
}

func TestTLSEnabled(t *testing.T) {
	dir := t.TempDir()
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")
	for _, p := range []string{cert, key} {
		if err := os.WriteFile(p, []byte("pem"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	missing := filepath.Join(dir, "nope.pem")

	cases := []struct {
		name string
		cert string
		key  string
		want bool
	}{
		{"both files exist", cert, key, true},
		{"cert only", cert, "", false},
		{"key only", "", key, false},
		{"cert file missing", missing, key, false},
		{"key file missing", cert, missing, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			t.Setenv("MNEME_DSN", "postgres://x@y/z")
			t.Setenv("MNEME_TLS_CERT", tc.cert)
			t.Setenv("MNEME_TLS_KEY", tc.key)

			c, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := c.TLSEnabled(); got != tc.want {
				t.Errorf("TLSEnabled: got %v, want %v", got, tc.want)
			}
		})
	}
}

// An empty MNEME_DSN is treated as "unset" and falls back to the self-contained
// SQLite default — the loader never yields an empty DSN, so a fresh binary with
// no configuration still boots. (Superseded the old "empty DSN is an error":
// with a built-in default there is nothing to reject.)
func TestEmptyDSNFallsBackToDefault(t *testing.T) {
	isolateConfig(t)
	t.Setenv("MNEME_DSN", "")
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != defaultDSN() {
		t.Errorf("empty MNEME_DSN should fall back to the sqlite default; got %q", c.DSN)
	}
}

func TestSearchMaxDist(t *testing.T) {
	isolateConfig(t)
	t.Setenv("MNEME_DSN", "postgres://x@y/z")

	// Default is the semantic relevance floor (cosine distance).
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.SearchMaxDist != 0.65 {
		t.Errorf("default SearchMaxDist: got %v, want 0.65", c.SearchMaxDist)
	}

	// Override parses a float; a bad value falls back to the default.
	t.Setenv("MNEME_SEARCH_MAX_DIST", "0.6")
	c, _ = Load()
	if c.SearchMaxDist != 0.6 {
		t.Errorf("env SearchMaxDist: got %v, want 0.6", c.SearchMaxDist)
	}
	t.Setenv("MNEME_SEARCH_MAX_DIST", "not-a-number")
	c, _ = Load()
	if c.SearchMaxDist != 0.65 {
		t.Errorf("bad SearchMaxDist should fall back to 0.65, got %v", c.SearchMaxDist)
	}
}
