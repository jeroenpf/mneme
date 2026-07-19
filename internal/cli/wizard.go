package cli

import (
	"fmt"

	"github.com/jeroenpfeil/mneme/internal/config"
)

// wizardAnswers is the raw set of choices the init form collects. It is a plain
// value with no IO so the mapping to Settings (buildSettings) can be unit-tested
// independently of the interactive huh layer.
type wizardAnswers struct {
	Backend     string // "sqlite" | "postgres"
	SQLitePath  string // sqlite backend; empty → the ~/.mneme default
	PostgresDSN string // postgres backend; required when Backend == "postgres"

	Embeddings bool   // true → Voyage; false → lexical-only (FTS) mode
	VoyageKey  string // required when Embeddings is true
	VoyageRPM  int    // optional proactive throttle; 0 = off

	NetMode string // "localhost" (plain HTTP) | "mneme.dev" (HTTPS)
	Port    string // empty → per-mode default (8080 localhost, 8443 mneme.dev)
}

// buildSettings maps validated wizard answers to a config.Settings. It is pure:
// no files touched, no mkcert run — that IO lives in the command layer. Returns
// an error for the two inconsistent combinations (postgres with no DSN,
// embeddings enabled with no key) so the form can re-prompt.
func buildSettings(a wizardAnswers) (config.Settings, error) {
	var s config.Settings

	switch a.Backend {
	case "postgres":
		if a.PostgresDSN == "" {
			return s, fmt.Errorf("postgres backend requires a DSN")
		}
		s.Data.DSN = a.PostgresDSN
	case "sqlite", "":
		path := a.SQLitePath
		if path == "" {
			path = config.DefaultSQLitePath()
		}
		s.Data.DSN = "sqlite://" + path
	default:
		return s, fmt.Errorf("unknown backend %q", a.Backend)
	}

	if a.Embeddings {
		if a.VoyageKey == "" {
			return s, fmt.Errorf("embeddings enabled but no Voyage API key provided")
		}
		s.Embeddings.VoyageAPIKey = a.VoyageKey
		s.Embeddings.VoyageModel = "voyage-4-large"
		s.Embeddings.VoyageRPM = a.VoyageRPM
	}

	switch a.NetMode {
	case "mneme.dev":
		s.Net.Port = orDefault(a.Port, "8443")
		s.Net.TLSCert, s.Net.TLSKey = config.CertPaths()
		s.Net.AllowedOrigins = []string{"https://mneme.dev:" + s.Net.Port}
	case "localhost", "":
		s.Net.Port = orDefault(a.Port, "8080")
		s.Net.AllowedOrigins = []string{"http://localhost:" + s.Net.Port}
	default:
		return s, fmt.Errorf("unknown network mode %q", a.NetMode)
	}

	return s, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
