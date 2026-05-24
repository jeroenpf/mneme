package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DSN          string
	Port         string
	Env          string
	CORSOrigins  []string
}

func Load() (*Config, error) {
	c := &Config{
		DSN:         getenv("MNEME_DSN", "postgres://mneme:mneme@localhost:5432/mneme?sslmode=disable"),
		Port:        getenv("MNEME_PORT", "8080"),
		Env:         getenv("MNEME_ENV", "development"),
		CORSOrigins: splitCSV(getenv("MNEME_CORS_ORIGINS", "http://localhost:5173,http://mneme.local")),
	}
	if c.DSN == "" {
		return nil, fmt.Errorf("MNEME_DSN must not be empty")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
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
