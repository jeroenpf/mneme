package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The root command, run bare, prints help listing the subcommands — it does
// not start a server or error. This is the "bare `mneme` prints help" contract.
func TestRootBarePrintsHelp(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("bare root should not error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"Usage:", "server", "Available Commands"} {
		if !strings.Contains(got, want) {
			t.Errorf("bare root help missing %q; got:\n%s", want, got)
		}
	}
}

// `mneme server` is registered as a subcommand and carries a RunE (the server
// lifecycle) — we assert its presence without starting it (which would bind a
// port).
func TestServerSubcommandRegistered(t *testing.T) {
	root := newRootCmd()
	sub, _, err := root.Find([]string{"server"})
	if err != nil {
		t.Fatalf("find server subcommand: %v", err)
	}
	if sub.Name() != "server" {
		t.Fatalf("expected server subcommand, got %q", sub.Name())
	}
	if sub.RunE == nil {
		t.Error("server subcommand has no RunE (would not run the server)")
	}
}

// The binary identifies itself as `mneme` so help/usage and completions render
// the intended command name.
func TestRootUseIsMneme(t *testing.T) {
	if got := newRootCmd().Name(); got != "mneme" {
		t.Errorf("root command name: got %q, want mneme", got)
	}
}
