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
// /health, /api/v1, /mcp, and falls back to the embedded SPA for
// any remaining path. mcpHandler and webHandler may be nil — useful
// in test setups that don't need them.
func Router(cfg *config.Config, st store.Store, mcpHandler, webHandler http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	health := &Health{Store: st}
	r.Get("/health", health.Handler)

	docs := &DocumentsHandler{Store: st}
	projects := &ProjectsHandler{Store: st}
	memory := &MemoryHandler{Store: st}
	decisions := &DecisionsHandler{Store: st}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/documents", docs.List)
		r.Post("/documents", docs.Create)
		r.Get("/documents/{id}", docs.Get)
		r.Patch("/documents/{id}", docs.Update)
		r.Post("/documents/{id}/archive", docs.Archive)
		r.Get("/projects", projects.List)
		r.Post("/projects", projects.Create)
		r.Get("/memory", memory.List)
		r.Put("/memory/{scope}/{key}", memory.Upsert)
		r.Delete("/memory/{scope}/{key}", memory.Delete)
		r.Get("/decisions", decisions.List)
	})

	if mcpHandler != nil {
		// The streamable HTTP transport uses both GET (open stream) and
		// POST (send JSON-RPC) under the same path, so Handle (not
		// Mount, which strips the prefix) is the right primitive.
		r.Handle("/mcp", mcpHandler)
	}
	if webHandler != nil {
		// SPA fallback. chi's NotFound catches anything that didn't
		// match a route — exactly the right hook for vue-router's
		// history mode (so deep links and reloads serve index.html).
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			webHandler.ServeHTTP(w, r)
		})
	}
	return r
}
