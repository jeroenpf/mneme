package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/migrations"
)

// The count>0 path is unit-tested against a fake client in
// internal/embed (TestWorkerRetryFailedNow); the CLI tests cover the
// wiring: the no-key refusal and the zero-failure report (which makes
// no provider calls, so a dummy key is safe).
func TestEmbedRetryCommand(t *testing.T) {
	dsn := "sqlite:" + filepath.Join(t.TempDir(), "mneme.db")
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	run := func() (string, error) {
		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"embed-retry", "--dsn", dsn})
		err := root.ExecuteContext(context.Background())
		return out.String(), err
	}

	// Without a Voyage key the command refuses with a pointer at the env var.
	t.Setenv("MNEME_VOYAGE_API_KEY", "")
	if _, err := run(); err == nil || !strings.Contains(err.Error(), "MNEME_VOYAGE_API_KEY") {
		t.Errorf("expected disabled-embeddings error, got %v", err)
	}

	// With a key and an empty failure bucket it reports nothing to retry.
	t.Setenv("MNEME_VOYAGE_API_KEY", "dummy")
	out, err := run()
	if err != nil {
		t.Fatalf("execute embed-retry: %v", err)
	}
	if !strings.Contains(out, "nothing to retry") {
		t.Errorf("output = %q, want nothing-to-retry report", out)
	}
}
