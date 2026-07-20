package cli

import (
	"bytes"
	"strings"
	"testing"
)

// `mneme --version` reports the build version. Unbuilt (no ldflags) it defaults
// to "dev"; GoReleaser overrides internal/cli.version at link time via
// -X github.com/jeroenpf/mneme/internal/cli.version=<tag>. The Homebrew formula
// smoke-tests this command, so the flag must exist and print a version line.
func TestRootVersionFlag(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("mneme --version should not error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "mneme") {
		t.Errorf("--version output %q does not name the binary", got)
	}
	if !strings.Contains(got, "dev") {
		t.Errorf("--version output %q missing default version %q", got, "dev")
	}
}
