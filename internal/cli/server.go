package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/api"
	"github.com/jeroenpf/mneme/internal/appinfo"
	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/live"
	"github.com/jeroenpf/mneme/internal/mcp"
	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/relations"
	"github.com/jeroenpf/mneme/internal/store"
	"github.com/jeroenpf/mneme/internal/web"
)

// newServerCmd builds `mneme server` — the long-running service. It is the
// operational default: everything the old `cmd/server` binary did lives here.
// Flags (--dsn, --port) sit at the top of the config precedence chain, ahead of
// MNEME_* env vars, settings.toml, and the built-in defaults.
func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the Mneme service (HTTP API, MCP endpoint, embedded UI)",
		Long: "Start the Mneme server: it applies migrations, serves the REST API,\n" +
			"the MCP endpoint at /mcp, and the embedded web UI, then blocks until\n" +
			"interrupted (Ctrl-C / SIGTERM), draining connections on shutdown.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadWithFlags(cmd.Flags())
			if err != nil {
				return err
			}
			return RunServer(cmd.Context(), cfg)
		},
	}
	cmd.Flags().String("dsn", "", "storage DSN (sqlite:// file or postgres:// URL); overrides config/env")
	cmd.Flags().String("port", "", "TCP port to listen on; overrides config/env")
	return cmd
}

// RunServer boots the full service from a resolved config and blocks until ctx
// is cancelled (an interrupt signal) or the listener fails. It owns store open,
// migrations, the embedding worker, and graceful HTTP shutdown — the lifecycle
// formerly in cmd/server/main.go's run(). Config resolution (env/file/flags)
// happens in the caller so this stays a pure lifecycle function.
func RunServer(ctx context.Context, cfg *config.Config) error {
	st, err := store.New(ctx, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()
	// Semantic relevance floor for hybrid search — vector candidates beyond
	// this cosine distance are dropped so a vague query returns nothing
	// rather than the whole corpus (keyword matches always pass).
	st.SetSearchMaxDist(cfg.SearchMaxDist)

	if err := migrations.Up(cfg.DSN); err != nil {
		return err
	}
	slog.Info("migrations applied")

	// One-time relations backfill: populates mention edges for documents
	// written before the relations table existed; no-op on every later start.
	if scanned, err := relations.Backfill(ctx, st); err != nil {
		return err
	} else if scanned > 0 {
		slog.Info("relations backfilled", "documents", scanned)
	}

	// Embedding is gated on a Voyage key: absent ⇒ nil client ⇒ no worker,
	// no reconcile, and search stays FTS-only. Present ⇒ start the async
	// worker (bound to the signal ctx so it drains on shutdown) and kick a
	// startup reconciliation on a background ctx so a fast shutdown can't
	// cancel the backfill mid-flight.
	client := embed.NewClient(*cfg)
	if client != nil {
		// Cache repeated search-query embeds so an identical query does not
		// re-hit Voyage; document embeds pass through uncached.
		client = embed.NewCachingClient(client, 256)
	}
	var enq embed.Enqueuer = embed.NopEnqueuer{}
	if client != nil {
		worker := embed.NewWorker(st, client, 256, cfg.VoyageRPM)
		go worker.Run(ctx)
		// Bounded startup reconcile + periodic passes so missed enqueue
		// events (dropped signals, crashes, restarts) self-heal.
		go worker.Reconcile(ctx, time.Duration(cfg.ReconcileEveryMin)*time.Minute)
		enq = worker
		slog.Info("embeddings enabled", "model", client.Model())
	} else {
		slog.Info("embeddings disabled (no MNEME_VOYAGE_API_KEY) — FTS-only search")
	}

	// One live hub shared by the MCP write path (Broadcast after a
	// successful write) and the /api/events SSE handler (Subscribe per
	// browser), so agent writes push straight to connected viewers.
	hub := live.NewHub()
	mcpSrv := mcp.New(st, enq, hub, client)
	info := appinfo.Summarize(cfg, version, client != nil)
	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.Router(cfg, st, mcpSrv.Handler(), web.Handler(), client, hub, enq, info),
		ReadHeaderTimeout: 5 * time.Second,
		// Parent every request context on the lifecycle ctx so long-lived
		// streams (the /api/events SSE loop, MCP hanging GETs) terminate on
		// shutdown. Shutdown alone never cancels in-flight requests — with an
		// open stream it would wait out its full deadline and surface
		// "context deadline exceeded" on every Ctrl-C.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Human-friendly boot summary to stdout — where to open it, where data
	// lives, and how to connect an agent. The slog "listening" line below stays
	// for machine logs.
	fmt.Fprint(os.Stdout, renderStartup(info))

	errCh := make(chan error, 1)
	go func() {
		useTLS := cfg.TLSEnabled()
		slog.Info("listening", "addr", cfg.ListenAddr(), "env", cfg.Env, "tls", useTLS)
		var err error
		if useTLS {
			err = srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdownCtx)
}
