// Command mneme is the entrypoint for the Mneme CLI. It stays deliberately
// thin — signal wiring and process exit only; the command tree and the server
// lifecycle live in internal/cli.
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

	// Signal cancellation flows through the command context into the server
	// lifecycle so `mneme server` drains connections on Ctrl-C / SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := cli.Execute(ctx); err != nil {
		logger.Error("mneme exited with error", "err", err)
		os.Exit(1)
	}
}
