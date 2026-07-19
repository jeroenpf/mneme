package cli

import (
	"strings"
	"testing"

	"github.com/jeroenpfeil/mneme/internal/config"
)

func TestBuildSettingsSQLiteDefaultPath(t *testing.T) {
	s, err := buildSettings(wizardAnswers{Backend: "sqlite", NetMode: "localhost"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	// Empty SQLitePath → the built-in default file under ~/.mneme.
	if !strings.HasPrefix(s.Data.DSN, "sqlite://") || !strings.HasSuffix(s.Data.DSN, "/.mneme/mneme.db") {
		t.Errorf("sqlite default DSN: got %q", s.Data.DSN)
	}
}

func TestBuildSettingsSQLiteCustomPath(t *testing.T) {
	s, err := buildSettings(wizardAnswers{Backend: "sqlite", SQLitePath: "/data/knowledge.db", NetMode: "localhost"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Data.DSN != "sqlite:///data/knowledge.db" {
		t.Errorf("sqlite custom DSN: got %q", s.Data.DSN)
	}
}

func TestBuildSettingsPostgres(t *testing.T) {
	dsn := "postgres://u:p@h:5432/mneme?sslmode=disable"
	s, err := buildSettings(wizardAnswers{Backend: "postgres", PostgresDSN: dsn, NetMode: "localhost"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Data.DSN != dsn {
		t.Errorf("postgres DSN: got %q", s.Data.DSN)
	}
}

func TestBuildSettingsPostgresRequiresDSN(t *testing.T) {
	if _, err := buildSettings(wizardAnswers{Backend: "postgres", NetMode: "localhost"}); err == nil {
		t.Fatal("postgres backend with empty DSN should error")
	}
}

func TestBuildSettingsEmbeddingsEnabled(t *testing.T) {
	s, err := buildSettings(wizardAnswers{
		Backend: "sqlite", NetMode: "localhost",
		Embeddings: true, VoyageKey: "pa-key", VoyageRPM: 60,
	})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Embeddings.VoyageAPIKey != "pa-key" || s.Embeddings.VoyageRPM != 60 {
		t.Errorf("embeddings: key=%q rpm=%d", s.Embeddings.VoyageAPIKey, s.Embeddings.VoyageRPM)
	}
	if s.Embeddings.VoyageModel != "voyage-4-large" {
		t.Errorf("embeddings model default: got %q", s.Embeddings.VoyageModel)
	}
}

func TestBuildSettingsLexicalOnly(t *testing.T) {
	// Embeddings disabled → no key stored (FTS-only mode), even if a key was typed.
	s, err := buildSettings(wizardAnswers{Backend: "sqlite", NetMode: "localhost", Embeddings: false, VoyageKey: "pa-leftover"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Embeddings.VoyageAPIKey != "" {
		t.Errorf("lexical-only should not store a key, got %q", s.Embeddings.VoyageAPIKey)
	}
}

func TestBuildSettingsEmbeddingsRequiresKey(t *testing.T) {
	if _, err := buildSettings(wizardAnswers{Backend: "sqlite", NetMode: "localhost", Embeddings: true, VoyageKey: ""}); err == nil {
		t.Fatal("embeddings enabled with empty key should error")
	}
}

func TestBuildSettingsLocalhostMode(t *testing.T) {
	s, err := buildSettings(wizardAnswers{Backend: "sqlite", NetMode: "localhost"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Net.Port != "8080" {
		t.Errorf("localhost default port: got %q, want 8080", s.Net.Port)
	}
	if s.Net.TLSCert != "" || s.Net.TLSKey != "" {
		t.Errorf("localhost mode must not set TLS paths: cert=%q key=%q", s.Net.TLSCert, s.Net.TLSKey)
	}
	if len(s.Net.AllowedOrigins) == 0 || !strings.HasPrefix(s.Net.AllowedOrigins[0], "http://localhost") {
		t.Errorf("localhost origins: got %v", s.Net.AllowedOrigins)
	}
}

func TestBuildSettingsMnemeDevMode(t *testing.T) {
	s, err := buildSettings(wizardAnswers{Backend: "sqlite", NetMode: "mneme.dev"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Net.Port != "8443" {
		t.Errorf("mneme.dev default port: got %q, want 8443", s.Net.Port)
	}
	cert, key := config.CertPaths()
	if s.Net.TLSCert != cert || s.Net.TLSKey != key {
		t.Errorf("mneme.dev TLS paths: cert=%q key=%q, want %q / %q", s.Net.TLSCert, s.Net.TLSKey, cert, key)
	}
	if len(s.Net.AllowedOrigins) == 0 || !strings.HasPrefix(s.Net.AllowedOrigins[0], "https://mneme.dev") {
		t.Errorf("mneme.dev origins: got %v", s.Net.AllowedOrigins)
	}
}

func TestBuildSettingsCustomPort(t *testing.T) {
	s, err := buildSettings(wizardAnswers{Backend: "sqlite", NetMode: "localhost", Port: "9000"})
	if err != nil {
		t.Fatalf("buildSettings: %v", err)
	}
	if s.Net.Port != "9000" {
		t.Errorf("custom port: got %q", s.Net.Port)
	}
	if s.Net.AllowedOrigins[0] != "http://localhost:9000" {
		t.Errorf("origin should reflect custom port: got %v", s.Net.AllowedOrigins)
	}
}
