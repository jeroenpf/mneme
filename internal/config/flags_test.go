package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

// serverFlags mirrors the flag set the `server` command exposes, so these tests
// exercise the real LoadWithFlags binding rather than a bespoke shape.
func serverFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("dsn", "", "")
	fs.String("port", "", "")
	fs.String("config", "", "")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return fs
}

// A set flag beats the same key in the environment (flag > env).
func TestLoadFlagOverridesEnv(t *testing.T) {
	clearMnemeEnv(t)
	isolateConfig(t)
	t.Setenv("MNEME_DSN", "postgres://env/loses")
	t.Setenv("MNEME_PORT", "1111")

	fs := serverFlags(t, "--dsn", "sqlite:///flag/wins.db", "--port", "2222")
	c, err := LoadWithFlags(fs)
	if err != nil {
		t.Fatalf("LoadWithFlags: %v", err)
	}
	if c.DSN != "sqlite:///flag/wins.db" {
		t.Errorf("flag should override env DSN: got %q", c.DSN)
	}
	if c.Port != "2222" {
		t.Errorf("flag should override env port: got %q", c.Port)
	}
}

// An unset flag must NOT clobber the env value — precedence keys on flag.Changed,
// not on the flag merely being defined with a zero default.
func TestUnsetFlagLeavesEnvIntact(t *testing.T) {
	clearMnemeEnv(t)
	isolateConfig(t)
	t.Setenv("MNEME_PORT", "5555")

	fs := serverFlags(t) // no args → nothing changed
	c, err := LoadWithFlags(fs)
	if err != nil {
		t.Fatalf("LoadWithFlags: %v", err)
	}
	if c.Port != "5555" {
		t.Errorf("unset flag should not clobber env port: got %q", c.Port)
	}
}

// --config redirects the settings file, ahead of the MNEME_CONFIG env default.
func TestConfigFlagRedirectsSettingsFile(t *testing.T) {
	clearMnemeEnv(t)
	// MNEME_CONFIG points one way; --config points at the file we actually wrote.
	t.Setenv("MNEME_CONFIG", filepath.Join(t.TempDir(), "ignored.toml"))
	real := filepath.Join(t.TempDir(), "real.toml")
	if err := os.WriteFile(real, []byte("[data]\ndsn = \"sqlite:///via-flag.db\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	fs := serverFlags(t, "--config", real)
	c, err := LoadWithFlags(fs)
	if err != nil {
		t.Fatalf("LoadWithFlags: %v", err)
	}
	if c.DSN != "sqlite:///via-flag.db" {
		t.Errorf("--config should redirect the settings file: got DSN %q", c.DSN)
	}
}
