package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/config"
)

// writeInit writes a private 0600 file and a summary naming the resolved
// storage, search mode, and address; the file loads back through config.Load.
func TestWriteInitSQLiteLexicalLocalhost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	var buf bytes.Buffer

	s, err := writeInit(wizardAnswers{Backend: "sqlite", NetMode: "localhost"}, path, &buf)
	if err != nil {
		t.Fatalf("writeInit: %v", err)
	}

	if fi, err := os.Stat(path); err != nil {
		t.Fatalf("stat: %v", err)
	} else if fi.Mode().Perm() != 0o600 {
		t.Errorf("mode: got %v, want 0600", fi.Mode().Perm())
	}

	out := buf.String()
	for _, want := range []string{path, "sqlite://", "lexical", "http://localhost:8765"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q; got:\n%s", want, out)
		}
	}

	// Round-trips through the loader.
	t.Setenv("MNEME_CONFIG", path)
	t.Setenv("MNEME_DSN", "")
	c, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DSN != s.Data.DSN || c.Port != "8765" {
		t.Errorf("round-trip: dsn=%q port=%q", c.DSN, c.Port)
	}
}

// The summary reports semantic search and the HTTPS address for mneme.dev mode,
// and never echoes the Voyage key.
func TestWriteInitEmbeddingsMnemeDevMasksKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	var buf bytes.Buffer

	a := wizardAnswers{Backend: "sqlite", Embeddings: true, VoyageKey: "pa-super-secret", NetMode: "mneme.dev"}
	if _, err := writeInit(a, path, &buf); err != nil {
		t.Fatalf("writeInit: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "pa-super-secret") {
		t.Errorf("summary must not echo the Voyage key; got:\n%s", out)
	}
	for _, want := range []string{"semantic", "https://mneme.dev:8443"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q; got:\n%s", want, out)
		}
	}
}

// A bad answer combination surfaces the builder error and writes no file.
func TestWriteInitPropagatesBuildErrorWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	var buf bytes.Buffer

	if _, err := writeInit(wizardAnswers{Backend: "postgres", NetMode: "localhost"}, path, &buf); err == nil {
		t.Fatal("expected build error for postgres with no DSN")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("no file should be written on error; stat err=%v", err)
	}
}
