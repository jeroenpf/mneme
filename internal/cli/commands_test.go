package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The command surface is locked early: init, doctor, export, and import all
// exist as subcommands with a runnable body, even where that body is still a
// stub. This pins the surface so later phases fill in behaviour without moving
// commands around.
func TestCommandSurfaceRegistered(t *testing.T) {
	for _, name := range []string{"server", "init", "doctor", "export", "import"} {
		root := newRootCmd()
		sub, _, err := root.Find([]string{name})
		if err != nil {
			t.Errorf("find %q subcommand: %v", name, err)
			continue
		}
		if sub.Name() != name {
			t.Errorf("expected %q subcommand, got %q", name, sub.Name())
		}
		if sub.RunE == nil {
			t.Errorf("%q subcommand has no RunE", name)
		}
	}
}

// export and import are deferred stubs: they must not error, and they must say
// so plainly rather than pretending to have done work.
func TestDeferredStubsAnnounceThemselves(t *testing.T) {
	for _, name := range []string{"export", "import"} {
		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{name})
		if err := root.Execute(); err != nil {
			t.Errorf("%q stub should not error: %v", name, err)
		}
		if !strings.Contains(strings.ToLower(out.String()), "not yet") {
			t.Errorf("%q stub output should flag it as unimplemented; got: %q", name, out.String())
		}
	}
}
