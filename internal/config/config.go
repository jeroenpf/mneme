package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
}

func Load() (*Config, error) {
	c := &Config{
		DSN:         getenv("MNEME_DSN", "postgres://mneme:mneme@localhost:5432/mneme?sslmode=disable"),
		Port:        getenv("MNEME_PORT", "8080"),
		Env:         getenv("MNEME_ENV", "development"),
		CORSOrigins: splitCSV(getenv("MNEME_CORS_ORIGINS", "http://localhost:5273,https://mneme.dev:8443")),
		TLSCert:     getenv("MNEME_TLS_CERT", ""),
		TLSKey:      getenv("MNEME_TLS_KEY", ""),
		VoyageKey:   getenv("MNEME_VOYAGE_API_KEY", ""),
		VoyageModel: getenv("MNEME_VOYAGE_MODEL", "voyage-4-large"),
		VoyageRPM:   getenvInt("MNEME_VOYAGE_RPM", 0),
	}
	if c.DSN == "" {
		return nil, fmt.Errorf("MNEME_DSN must not be empty")
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
