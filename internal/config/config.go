package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// viper keys for the front-door settings (data/embeddings/net sections). These
// map 1:1 to the Settings struct and to the MNEME_* env vars bound in newViper.
const (
	keyDSN         = "data.dsn"
	keyVoyageKey   = "embeddings.voyage_api_key"
	keyVoyageModel = "embeddings.voyage_model"
	keyVoyageRPM   = "embeddings.voyage_rpm"
	keyPort        = "net.port"
	keyCORS        = "net.allowed_origins"
	keyTLSCert     = "net.tls_cert"
	keyTLSKey      = "net.tls_key"
)

type Config struct {
	DSN         string
	Port        string
	Env         string
	CORSOrigins []string
	TLSCert     string
	TLSKey      string
	VoyageKey   string
	VoyageModel string
	VoyageRPM   int // optional proactive throttle; 0 = off (rely on 429 backoff)
	// ReconcileEveryMin is the interval, in minutes, for periodic embedding
	// reconciliation (orphan sweep + backfill of missed sources). A startup
	// pass always runs; <=0 disables the periodic passes after it.
	ReconcileEveryMin int
	// SearchMaxDist is the semantic relevance floor: hybrid search drops
	// vector candidates whose cosine distance to the query exceeds it, so a
	// vague/irrelevant query returns nothing rather than a ranked corpus.
	// <= 0 disables the floor (always-ranked). Keyword (FTS) matches always
	// pass regardless. Default 0.65 is calibrated for voyage-4-large query-
	// to-document distances (relevant hits measured at 0.45–0.60, unrelated
	// at 0.75+); tune via MNEME_SEARCH_MAX_DIST as the corpus grows.
	SearchMaxDist float64
}

// Load resolves the runtime config from settings.toml, MNEME_* env vars, and
// built-in defaults, in that precedence (env > file > default). It is the
// no-flags entrypoint used by tests and any caller without a cobra flag set.
func Load() (*Config, error) {
	v, err := newViper(SettingsPath())
	if err != nil {
		return nil, err
	}
	return fromViper(v)
}

// LoadWithFlags resolves config with CLI flags overlaid at the top of the
// precedence chain (flag > env > file > default). Recognised flags: --config
// (settings file path), --dsn, --port. Unset flags fall through untouched.
func LoadWithFlags(flags *pflag.FlagSet) (*Config, error) {
	path := SettingsPath()
	if f := flags.Lookup("config"); f != nil && f.Changed {
		path = f.Value.String()
	}
	v, err := newViper(path)
	if err != nil {
		return nil, err
	}
	bindFlags(v, flags)
	return fromViper(v)
}

// newViper builds a viper seeded with defaults and MNEME_* env bindings, then
// layers in the settings file at path. A missing file is fine (fresh install);
// a malformed file is an error. The front-door keys (data/embeddings/net) are
// file-aware; the internal knobs (Env, ReconcileEveryMin, SearchMaxDist) stay
// env-only via the getenv helpers in fromViper — they are not wizard fields.
func newViper(path string) (*viper.Viper, error) {
	v := viper.New()
	v.SetDefault(keyDSN, defaultDSN())
	v.SetDefault(keyPort, "8080")
	v.SetDefault(keyCORS, []string{"http://localhost:5273", "https://mneme.dev:8443"})
	v.SetDefault(keyVoyageModel, "voyage-4-large")
	v.SetDefault(keyVoyageRPM, 0)

	_ = v.BindEnv(keyDSN, "MNEME_DSN")
	_ = v.BindEnv(keyPort, "MNEME_PORT")
	_ = v.BindEnv(keyCORS, "MNEME_CORS_ORIGINS")
	_ = v.BindEnv(keyTLSCert, "MNEME_TLS_CERT")
	_ = v.BindEnv(keyTLSKey, "MNEME_TLS_KEY")
	_ = v.BindEnv(keyVoyageKey, "MNEME_VOYAGE_API_KEY")
	_ = v.BindEnv(keyVoyageModel, "MNEME_VOYAGE_MODEL")
	_ = v.BindEnv(keyVoyageRPM, "MNEME_VOYAGE_RPM")

	v.SetConfigFile(path)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		// A not-yet-created file is the normal first-run state; anything else
		// (malformed TOML, permission error) is a real problem worth surfacing.
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}
	return v, nil
}

// bindFlags overlays recognised cobra flags onto viper. BindPFlag only takes
// effect when the flag was actually set by the user (viper checks flag.Changed),
// so unset flags leave env/file/default precedence intact.
func bindFlags(v *viper.Viper, flags *pflag.FlagSet) {
	if f := flags.Lookup("dsn"); f != nil {
		_ = v.BindPFlag(keyDSN, f)
	}
	if f := flags.Lookup("port"); f != nil {
		_ = v.BindPFlag(keyPort, f)
	}
}

// fromViper projects a resolved viper into the runtime Config. CORS is read
// specially: an env var arrives as a CSV string (split + trimmed here) while
// file/default values are already string slices.
func fromViper(v *viper.Viper) (*Config, error) {
	c := &Config{
		DSN:         v.GetString(keyDSN),
		Port:        v.GetString(keyPort),
		TLSCert:     v.GetString(keyTLSCert),
		TLSKey:      v.GetString(keyTLSKey),
		VoyageKey:   v.GetString(keyVoyageKey),
		VoyageModel: v.GetString(keyVoyageModel),
		VoyageRPM:   v.GetInt(keyVoyageRPM),
		// Internal knobs: env-only, preserving the safe parse-fallback behaviour.
		Env:               getenv("MNEME_ENV", "development"),
		ReconcileEveryMin: getenvInt("MNEME_RECONCILE_INTERVAL_MIN", 15),
		SearchMaxDist:     getenvFloat("MNEME_SEARCH_MAX_DIST", 0.65),
	}
	if s, ok := v.Get(keyCORS).(string); ok {
		c.CORSOrigins = splitCSV(s)
	} else {
		c.CORSOrigins = v.GetStringSlice(keyCORS)
	}
	if c.DSN == "" {
		return nil, fmt.Errorf("data.dsn must not be empty")
	}
	return c, nil
}

// TLSEnabled reports whether the server should terminate TLS itself: both
// cert paths configured and both files present. Anything less falls back to
// plain HTTP so the test suite and first boot (before mkcert has run) work
// without ceremony.
func (c *Config) TLSEnabled() bool {
	if c.TLSCert == "" || c.TLSKey == "" {
		return false
	}
	for _, p := range []string{c.TLSCert, c.TLSKey} {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
