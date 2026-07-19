package config

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteSettings creates the file 0600 (it may hold the Voyage key in plaintext)
// and any missing parent dir 0700, then the loader reads it back losslessly.
func TestWriteSettingsCreatesPrivateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "settings.toml") // parent does not exist yet
	s := &Settings{
		Data:       DataSettings{DSN: "sqlite:///written.db"},
		Embeddings: EmbeddingsSettings{VoyageAPIKey: "pa-secret", VoyageRPM: 30},
		Net:        NetSettings{Port: "8443"},
	}

	if err := WriteSettings(path, s); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("file mode: got %v, want 0600", fi.Mode().Perm())
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Errorf("dir mode: got %v, want 0700", di.Mode().Perm())
	}

	// Round-trips through the loader.
	clearMnemeEnv(t)
	t.Setenv("MNEME_CONFIG", path)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != "sqlite:///written.db" || c.VoyageKey != "pa-secret" || c.VoyageRPM != 30 || c.Port != "8443" {
		t.Errorf("round-trip mismatch: dsn=%q key=%q rpm=%d port=%q", c.DSN, c.VoyageKey, c.VoyageRPM, c.Port)
	}
}

// Overwriting a pre-existing world-readable file tightens it back to 0600 —
// os.WriteFile alone would keep the looser mode.
func TestWriteSettingsEnforcesModeOnOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(path, []byte("[data]\ndsn=\"old\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteSettings(path, &Settings{Data: DataSettings{DSN: "sqlite:///new.db"}}); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("overwrite mode: got %v, want 0600", fi.Mode().Perm())
	}
}
