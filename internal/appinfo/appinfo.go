// Package appinfo derives the install facts that both the `mneme server`
// startup box and the GET /api/v1/install endpoint present, so the two surfaces
// can never drift. Info is safe to serialize to the local UI: it never carries
// the Voyage API key or database credentials.
package appinfo

import (
	"strings"

	"github.com/jeroenpf/mneme/internal/config"
)

// Info is the single source of truth for install facts.
type Info struct {
	Version     string  `json:"version"`
	Mode        string  `json:"mode"` // "localhost" | "mnemedev"
	URL         string  `json:"url"`
	MCPEndpoint string  `json:"mcp_endpoint"`
	DB          DBInfo  `json:"db"`
	Embeddings  EmbInfo `json:"embeddings"`
}

// DBInfo describes the store without leaking credentials.
type DBInfo struct {
	Driver string `json:"driver"`
	Path   string `json:"path"`
}

// EmbInfo reports whether semantic search is on and, if so, the model. The API
// key is deliberately absent.
type EmbInfo struct {
	Enabled bool   `json:"enabled"`
	Model   string `json:"model"`
}

// Summarize derives Info from the effective config. embeddingsEnabled mirrors
// the server's own check (an embedding client was constructed). The URL uses a
// friendly host (localhost / mneme.dev), never the loopback bind address.
func Summarize(cfg *config.Config, version string, embeddingsEnabled bool) Info {
	scheme, host, mode := "http", "localhost", "localhost"
	if cfg.TLSEnabled() {
		scheme, host, mode = "https", "mneme.dev", "mnemedev"
	}
	url := scheme + "://" + host + ":" + cfg.Port

	info := Info{
		Version:     version,
		Mode:        mode,
		URL:         url,
		MCPEndpoint: url + "/mcp",
		DB:          dbInfo(cfg.DSN),
	}
	if embeddingsEnabled {
		info.Embeddings = EmbInfo{Enabled: true, Model: cfg.VoyageModel}
	}
	return info
}

// dbInfo names the store driver and, for SQLite only, its file path. A Postgres
// DSN can embed a password, so only the driver is reported for it.
func dbInfo(dsn string) DBInfo {
	switch {
	case strings.HasPrefix(dsn, "sqlite://"):
		return DBInfo{Driver: "sqlite", Path: strings.TrimPrefix(dsn, "sqlite://")}
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return DBInfo{Driver: "postgres"}
	default:
		driver := dsn
		if i := strings.Index(dsn, "://"); i > 0 {
			driver = dsn[:i]
		}
		return DBInfo{Driver: driver}
	}
}
