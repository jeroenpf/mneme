package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Scripted-input smoke test of the flow: drive the real form in accessible mode
// with canned line input, then run the deterministic tail. Accessible mode runs
// every field in order (hide funcs are a TTY-only nicety) and ignores the
// password field's no-TTY error, so the script is: backend=1(sqlite),
// sqlite-path=blank, postgres-dsn=blank, embeddings=n, [key: no read], rpm=0,
// net=1(localhost), port=blank.
func TestInitFlowScriptedInput(t *testing.T) {
	script := strings.Join([]string{"1", "", "", "n", "0", "1", ""}, "\n") + "\n"

	answers, err := collectAnswers(strings.NewReader(script), io.Discard, true)
	if err != nil {
		t.Fatalf("collectAnswers: %v", err)
	}
	if answers.Backend != "sqlite" {
		t.Errorf("backend: got %q, want sqlite", answers.Backend)
	}
	if answers.Embeddings {
		t.Errorf("embeddings should be off, got on")
	}
	if answers.NetMode != "localhost" {
		t.Errorf("net mode: got %q, want localhost", answers.NetMode)
	}
	if answers.VoyageRPM != 0 {
		t.Errorf("rpm: got %d, want 0", answers.VoyageRPM)
	}

	// The collected answers drive a real settings.toml through the flow tail.
	path := filepath.Join(t.TempDir(), "settings.toml")
	if _, err := writeInit(answers, path, io.Discard); err != nil {
		t.Fatalf("writeInit: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings.toml not written: %v", err)
	}
}
