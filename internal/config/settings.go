package config

import (
	"os"
	"path/filepath"
)

// Settings is the typed, friendly front-door schema serialized to
// ~/.mneme/settings.toml. It mirrors the runtime Config but groups keys into
// the sections the init wizard presents (data, embeddings, net). Env vars and
// CLI flags still override it (see Load); the file is just the pleasant default.
//
// Struct tags carry both `toml` (for the write helper) and `mapstructure` (for
// viper's Unmarshal), kept in sync. Optional fields are `omitempty` so a
// minimal file stays clean.
type Settings struct {
	Data       DataSettings       `toml:"data" mapstructure:"data"`
	Embeddings EmbeddingsSettings `toml:"embeddings" mapstructure:"embeddings"`
	Net        NetSettings        `toml:"net" mapstructure:"net"`
}

// DataSettings selects the storage backend: a sqlite:// file path (the
// self-contained default) or a postgres:// DSN.
type DataSettings struct {
	DSN string `toml:"dsn" mapstructure:"dsn"`
}

// EmbeddingsSettings configures the optional Voyage embedding provider. An
// empty VoyageAPIKey means lexical-only (FTS) mode — no external calls.
type EmbeddingsSettings struct {
	VoyageAPIKey string `toml:"voyage_api_key,omitempty" mapstructure:"voyage_api_key"`
	VoyageModel  string `toml:"voyage_model,omitempty" mapstructure:"voyage_model"`
	VoyageRPM    int    `toml:"voyage_rpm,omitempty" mapstructure:"voyage_rpm"`
}

// NetSettings configures the listener. TLSCert/TLSKey set selects HTTPS mode
// (mneme.dev); left empty is plain HTTP on loopback (localhost). AllowedOrigins
// is the CORS allow-list.
type NetSettings struct {
	Port           string   `toml:"port,omitempty" mapstructure:"port"`
	AllowedOrigins []string `toml:"allowed_origins,omitempty" mapstructure:"allowed_origins"`
	TLSCert        string   `toml:"tls_cert,omitempty" mapstructure:"tls_cert"`
	TLSKey         string   `toml:"tls_key,omitempty" mapstructure:"tls_key"`
}

// mnemeHome returns ~/.mneme, the per-user Mneme state directory. It falls back
// to a relative ".mneme" only if the home dir cannot be resolved (rare).
func mnemeHome() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".mneme"
	}
	return filepath.Join(home, ".mneme")
}

// SettingsPath is the location of the config file: ~/.mneme/settings.toml,
// overridable via the MNEME_CONFIG env var (also bound to the --config flag).
func SettingsPath() string {
	if p := os.Getenv("MNEME_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(mnemeHome(), "settings.toml")
}

// defaultDSN is the built-in storage default: a self-contained SQLite database
// at ~/.mneme/mneme.db, so a freshly-downloaded binary runs with zero config.
func defaultDSN() string {
	return "sqlite://" + filepath.Join(mnemeHome(), "mneme.db")
}
