package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/jeroenpf/mneme/internal/command"
	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/store"
)

// Router builds the top-level HTTP handler. Composes middleware,
// /api/events (SSE), /health, /api/v1, /mcp, and falls back to the embedded
// SPA for any remaining path. mcpHandler, webHandler, and hub may be nil —
// useful in test setups that don't need them. client (may be nil) embeds the
// /search query for hybrid ranking and flips /search/status enabled; nil ⇒
// FTS-only, enabled=false.
func Router(cfg *config.Config, st store.Store, mcpHandler, webHandler http.Handler, client embed.Client, hub *live.Hub, enq embed.Enqueuer) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	// Origin validation (the MCP MUST for local HTTP transport): reject any
	// browser request whose Origin is not allow-listed, on /mcp and every other
	// route. No-Origin requests (native MCP clients, curl, healthchecks, page
	// navigations) pass. This is stronger than CORS, which only withholds
	// response headers rather than refusing the request.
	r.Use(OriginGuard(cfg.CORSOrigins))

	// SSE lives OUTSIDE the request-timeout group: the stream is
	// deliberately long-lived and a 30s Timeout would sever it.
	if hub != nil {
		events := &EventsHandler{Hub: hub}
		r.Get("/api/events", events.Stream)
	}

	// Everything else keeps the 30s request timeout.
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))

		health := &Health{Store: st}
		r.Get("/health", health.Handler)

		// hub is a concrete *live.Hub; pass it as a Broadcaster only when non-nil
		// so a nil hub becomes a true nil interface (→ NopBroadcaster), not a
		// typed-nil that panics on Broadcast.
		var bc live.Broadcaster
		if hub != nil {
			bc = hub
		}
		docs := &DocumentsHandler{Store: st, Writer: command.NewDocuments(st, enq, bc)}
		projects := &ProjectsHandler{Store: st}
		memory := &MemoryHandler{Store: st}
		env := &EnvHandler{Store: st}
		decisions := &DecisionsHandler{Store: st}
		snippets := &SnippetsHandler{Store: st}
		journal := &JournalHandler{Store: st}
		solutions := &SolutionsHandler{Store: st}
		bundleH := &BundleHandler{Store: st}
		searchH := &SearchHandler{Store: st, Client: client}
		statusH := &SearchStatusHandler{Store: st, Enabled: client != nil}
		if client != nil {
			statusH.Model = client.Model()
			statusH.Provider = "voyage"
		}
		// The enqueuer is the concrete embedding worker when enabled; expose its
		// live queue/reconcile signals and manual retry to the status endpoint.
		// A NopEnqueuer (disabled) does not satisfy EmbedRuntime → Runtime stays nil.
		if rt, ok := enq.(EmbedRuntime); ok {
			statusH.Runtime = rt
		}

		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/documents", docs.List)
			r.Post("/documents", docs.Create)
			r.Get("/documents/{id}", docs.Get)
			r.Patch("/documents/{id}", docs.Update)
			r.Post("/documents/{id}/archive", docs.Archive)
			r.Get("/documents/{id}/revisions", docs.Revisions)
			r.Post("/documents/{id}/restore", docs.Restore)
			r.Get("/projects", projects.List)
			r.Post("/projects", projects.Create)
			r.Get("/memory", memory.List)
			r.Put("/memory/{scope}/{key}", memory.Upsert)
			r.Delete("/memory/{scope}/{key}", memory.Delete)
			r.Get("/env", env.List)
			r.Put("/env/{key}", env.Upsert)
			r.Delete("/env/{key}", env.Delete)
			r.Get("/decisions", decisions.List)
			r.Get("/snippets", snippets.List)
			r.Get("/journal", journal.List)
			r.Get("/solutions", solutions.List)
			r.Get("/bundle", bundleH.Get)
			r.Get("/search", searchH.Get)
			r.Get("/search/status", statusH.Get)
			r.Post("/search/reindex-failed", statusH.Retry)
		})

		if mcpHandler != nil {
			// The streamable HTTP transport uses both GET (open stream) and
			// POST (send JSON-RPC) under the same path, so Handle (not
			// Mount, which strips the prefix) is the right primitive.
			r.Handle("/mcp", mcpHandler)
		}
	})

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
