package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MNEME_DSN", "")
	t.Setenv("MNEME_PORT", "")
	t.Setenv("MNEME_ENV", "")
	t.Setenv("MNEME_CORS_ORIGINS", "")

	// Empty env vars are *set* (t.Setenv), so getenv returns "" — for
	// CORS this means an empty slice, but DSN empty causes Load() to
	// reject. Unset them to exercise the defaults path.
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

func TestTLSDisabledByDefault(t *testing.T) {
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

func TestLoadRejectsEmptyDSN(t *testing.T) {
	t.Setenv("MNEME_DSN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for empty DSN, got nil")
	}
}

func TestSearchMaxDist(t *testing.T) {
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
