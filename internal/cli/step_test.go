package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// runStep runs the action and returns its error; on a non-TTY writer it prints
// the title (no animation) instead of a spinner.
func TestRunStepRunsActionAndReports(t *testing.T) {
	var buf bytes.Buffer
	called := false
	if err := runStep(&buf, "doing the thing", func() error { called = true; return nil }); err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if !called {
		t.Fatal("action was not run")
	}
	if !strings.Contains(buf.String(), "doing the thing") {
		t.Errorf("title not printed; got %q", buf.String())
	}
}

func TestRunStepPropagatesError(t *testing.T) {
	var buf bytes.Buffer
	want := errors.New("boom")
	if err := runStep(&buf, "x", func() error { return want }); !errors.Is(err, want) {
		t.Errorf("runStep should return the action error: got %v", err)
	}
}
