// Command server is the legacy entrypoint that boots the Mneme service
// directly. The command tree lives in internal/cli; this binary is kept only
// so existing tooling (make/air pointing at ./cmd/server) keeps working during
// the CLI restructure — it delegates straight to cli.RunServer. The shipped
// entrypoint is cmd/mneme.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jeroenpfeil/mneme/internal/cli"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cli.RunServer(ctx); err != nil {
		logger.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}
