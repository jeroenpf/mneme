package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jeroenpfeil/mneme/internal/api"
	"github.com/jeroenpfeil/mneme/internal/config"
	"github.com/jeroenpfeil/mneme/internal/embed"
	"github.com/jeroenpfeil/mneme/internal/live"
	"github.com/jeroenpfeil/mneme/internal/mcp"
	"github.com/jeroenpfeil/mneme/internal/migrations"
	"github.com/jeroenpfeil/mneme/internal/store"
	"github.com/jeroenpfeil/mneme/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		logger.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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

	// Embedding is gated on a Voyage key: absent ⇒ nil client ⇒ no worker,
	// no reconcile, and search stays FTS-only. Present ⇒ start the async
	// worker (bound to the signal ctx so it drains on shutdown) and kick a
	// startup reconciliation on a background ctx so a fast shutdown can't
	// cancel the backfill mid-flight.
	client := embed.NewClient(*cfg)
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
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Router(cfg, st, mcpSrv.Handler(), web.Handler(), client, hub),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		useTLS := cfg.TLSEnabled()
		slog.Info("listening", "port", cfg.Port, "env", cfg.Env, "tls", useTLS)
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
