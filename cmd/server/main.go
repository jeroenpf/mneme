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
		go func() {
			if err := worker.ReconcileAll(context.Background()); err != nil {
				slog.Error("startup embed reconcile failed", "err", err)
			}
		}()
		enq = worker
		slog.Info("embeddings enabled", "model", client.Model())
	} else {
		slog.Info("embeddings disabled (no MNEME_VOYAGE_API_KEY) — FTS-only search")
	}

	mcpSrv := mcp.New(st, enq, client)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Router(cfg, st, mcpSrv.Handler(), web.Handler(), client),
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
