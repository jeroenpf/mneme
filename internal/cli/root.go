// Package cli is the mneme command tree (cobra). The bare `mneme` prints help;
// `mneme server` runs the service (today's behaviour, the operational default).
// main stays a thin entrypoint — command wiring and the server lifecycle live
// here, per the "main.go is wiring & lifecycle only" guideline.
package cli

import (
	"context"

	"github.com/spf13/cobra"
)

// newRootCmd builds the `mneme` root command with every subcommand attached.
// It is a constructor (not a package global) so tests can build a fresh tree
// with isolated flags/output.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mneme",
		Short: "Mneme — a local AI dev knowledge service",
		Long: "Mneme is a local, single-user knowledge service for AI-assisted development.\n" +
			"It stores plans, decisions, snippets, and journals, and exposes them to\n" +
			"Claude Code over MCP. Run `mneme init` to set up, then `mneme server`.",
		// No Run: a bare invocation prints help and exits 0 rather than
		// silently starting the server.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newServerCmd())
	return root
}

// Execute runs the mneme CLI, returning the resolved command's error. The
// caller (main) is responsible for translating a non-nil error into a non-zero
// exit code. ctx carries signal cancellation down to the server lifecycle.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}
