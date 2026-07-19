package config

import (
	"os"
	"path/filepath"
	"testing"
)

// clearMnemeEnv blanks every MNEME_* override so a test exercises a known
// precedence layer rather than whatever the ambient shell exported.
func clearMnemeEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"MNEME_DSN", "MNEME_PORT", "MNEME_ENV", "MNEME_CORS_ORIGINS",
		"MNEME_TLS_CERT", "MNEME_TLS_KEY", "MNEME_VOYAGE_API_KEY",
		"MNEME_VOYAGE_MODEL", "MNEME_VOYAGE_RPM",
	} {
		t.Setenv(k, "")
	}
}

// isolateConfig points the loader at a fresh temp settings path so tests never
// read (or write) the developer's real ~/.mneme/settings.toml.
func isolateConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.toml")
	t.Setenv("MNEME_CONFIG", path)
	return path
}

// With no env and no file, the loader resolves the self-contained defaults:
// the SQLite DSN under ~/.mneme, port 8080, and the standard Voyage model.
func TestLoadDefaultsResolved(t *testing.T) {
	clearMnemeEnv(t)
	isolateConfig(t) // path does not exist → file layer is empty

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != defaultDSN() {
		t.Errorf("DSN: got %q, want default %q", c.DSN, defaultDSN())
	}
	if c.Port != "8080" {
		t.Errorf("Port: got %q, want 8080", c.Port)
	}
	if c.VoyageModel != "voyage-4-large" {
		t.Errorf("VoyageModel: got %q", c.VoyageModel)
	}
}

// settings.toml supplies values when no env overrides are present (file > default).
func TestLoadFileOnly(t *testing.T) {
	clearMnemeEnv(t)
	path := isolateConfig(t)
	body := `[data]
dsn = "sqlite:///file/only.db"

[embeddings]
voyage_api_key = "pa-from-file"
voyage_rpm = 42

[net]
port = "7777"
allowed_origins = ["http://file.example"]
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != "sqlite:///file/only.db" {
		t.Errorf("DSN from file: got %q", c.DSN)
	}
	if c.Port != "7777" {
		t.Errorf("Port from file: got %q", c.Port)
	}
	if c.VoyageKey != "pa-from-file" || c.VoyageRPM != 42 {
		t.Errorf("embeddings from file: key=%q rpm=%d", c.VoyageKey, c.VoyageRPM)
	}
	if len(c.CORSOrigins) != 1 || c.CORSOrigins[0] != "http://file.example" {
		t.Errorf("CORS from file: got %v", c.CORSOrigins)
	}
}

// An env var beats the same key set in the file (env > file).
func TestLoadEnvOverridesFile(t *testing.T) {
	clearMnemeEnv(t)
	path := isolateConfig(t)
	if err := os.WriteFile(path, []byte("[data]\ndsn = \"sqlite:///file.db\"\n[net]\nport = \"7777\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MNEME_DSN", "postgres://env/wins")
	t.Setenv("MNEME_PORT", "9999")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != "postgres://env/wins" {
		t.Errorf("env should override file DSN: got %q", c.DSN)
	}
	if c.Port != "9999" {
		t.Errorf("env should override file port: got %q", c.Port)
	}
}

// A malformed settings.toml is surfaced as an error, not silently ignored.
func TestLoadRejectsMalformedFile(t *testing.T) {
	clearMnemeEnv(t)
	path := isolateConfig(t)
	if err := os.WriteFile(path, []byte("this is = = not toml ]["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("expected error for malformed settings.toml, got nil")
	}
}
