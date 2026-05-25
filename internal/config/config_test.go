package config

import (
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

func TestLoadRejectsEmptyDSN(t *testing.T) {
	t.Setenv("MNEME_DSN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for empty DSN, got nil")
	}
}
