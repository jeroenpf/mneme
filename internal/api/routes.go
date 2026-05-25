package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/jeroenpfeil/mneme/internal/config"
	"github.com/jeroenpfeil/mneme/internal/store"
)

// Router builds the top-level HTTP handler. Composes middleware,
// /health, and the /api/v1 group. If mcpHandler is non-nil it's
// mounted at /mcp — separated this way so test setups that only
// exercise the REST surface don't have to construct an MCP server.
func Router(cfg *config.Config, st store.Store, mcpHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	health := &Health{Store: st}
	r.Get("/health", health.Handler)

	docs := &DocumentsHandler{Store: st}
	projects := &ProjectsHandler{Store: st}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/documents", docs.List)
		r.Post("/documents", docs.Create)
		r.Get("/documents/{id}", docs.Get)
		r.Patch("/documents/{id}", docs.Update)
		r.Post("/documents/{id}/archive", docs.Archive)
		r.Get("/projects", projects.List)
	})

	if mcpHandler != nil {
		// The streamable HTTP transport uses both GET (open stream) and
		// POST (send JSON-RPC) under the same path, so Handle (not
		// Mount, which strips the prefix) is the right primitive.
		r.Handle("/mcp", mcpHandler)
	}
	return r
}
