// Package web exposes the built Vue SPA as an http.Handler with SPA
// history-mode fallback. The handler is mounted last on the chi router
// in internal/api/routes.go so that /api/v1/*, /health, and /mcp keep
// precedence.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// distFS contains the built Vue app. `make build` copies web/dist into
// internal/web/dist (Go embed doesn't support parent paths). When the
// embedded dist is missing or has no index.html, Handler() returns a
// placeholder so the binary still serves *something* before the first
// build — useful for `go test ./...` runs.
//
//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the SPA. /assets/* falls
// back to 404 when missing (so bundler errors don't get masked).
// Anything else that isn't a real file is served as index.html so
// vue-router history mode works on deep links and reloads.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil || !hasIndex(sub) {
		return placeholderHandler()
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, statErr := fs.Stat(sub, path); statErr == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// Real /assets/ misses must 404 — masking them hides bundler
		// failures behind index.html, which is debugging hell.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		serveIndex(sub, w, r)
	})
}

func hasIndex(sub fs.FS) bool {
	_, err := fs.Stat(sub, "index.html")
	return err == nil
}

func serveIndex(sub fs.FS, w http.ResponseWriter, _ *http.Request) {
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "spa unreadable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func placeholderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/health" || r.URL.Path == "/mcp" {
			http.NotFound(w, r)
			return
		}
		// Mirror the real handler's /assets policy so `go test ./...`
		// behaves the same whether or not the SPA has been built.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body><div id="app"></div><p>web/dist not built. Run <code>make build</code>.</p></body></html>`))
	})
}
