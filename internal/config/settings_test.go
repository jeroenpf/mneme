package config

import (
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// The Settings schema serializes to the friendly [data]/[embeddings]/[net]
// TOML shape the wizard writes and viper reads, and round-trips without loss.
func TestSettingsTOMLRoundTrip(t *testing.T) {
	in := Settings{
		Data:       DataSettings{DSN: "sqlite:///home/x/.mneme/mneme.db"},
		Embeddings: EmbeddingsSettings{VoyageAPIKey: "pa-abc", VoyageModel: "voyage-4-large", VoyageRPM: 60},
		Net: NetSettings{
			Port:           "8443",
			AllowedOrigins: []string{"http://localhost:5273", "https://mneme.dev:8443"},
			TLSCert:        "/certs/c.pem",
			TLSKey:         "/certs/k.pem",
		},
	}

	raw, err := toml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	for _, want := range []string{"[data]", "dsn =", "[embeddings]", "voyage_api_key =", "voyage_rpm =", "[net]", "port =", "allowed_origins ="} {
		if !strings.Contains(got, want) {
			t.Errorf("TOML missing %q; got:\n%s", want, got)
		}
	}

	var back Settings
	if err := toml.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Data.DSN != in.Data.DSN || back.Embeddings != in.Embeddings ||
		back.Net.Port != in.Net.Port || back.Net.TLSCert != in.Net.TLSCert ||
		back.Net.TLSKey != in.Net.TLSKey || len(back.Net.AllowedOrigins) != 2 {
		t.Errorf("round-trip mismatch:\n in=%+v\nout=%+v", in, back)
	}
}

// Optional/empty fields are omitted so a minimal settings.toml stays clean.
func TestSettingsOmitsEmpty(t *testing.T) {
	raw, err := toml.Marshal(Settings{Data: DataSettings{DSN: "sqlite:///x.db"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "voyage_api_key") {
		t.Errorf("empty voyage_api_key should be omitted; got:\n%s", raw)
	}
}

// The built-in default DSN is a self-contained SQLite file under ~/.mneme —
// the "just works with no config" path for a fresh tester.
func TestDefaultDSNIsSQLiteUnderHome(t *testing.T) {
	got := defaultDSN()
	if !strings.HasPrefix(got, "sqlite://") {
		t.Errorf("default DSN should be a sqlite:// URL, got %q", got)
	}
	if !strings.HasSuffix(got, "/.mneme/mneme.db") {
		t.Errorf("default DSN should point at ~/.mneme/mneme.db, got %q", got)
	}
}

// The settings file lives at ~/.mneme/settings.toml, overridable via
// MNEME_CONFIG so tests (and advanced users) can point elsewhere.
func TestSettingsPathOverride(t *testing.T) {
	t.Setenv("MNEME_CONFIG", "/tmp/custom/settings.toml")
	if got := SettingsPath(); got != "/tmp/custom/settings.toml" {
		t.Errorf("MNEME_CONFIG override: got %q", got)
	}
}
