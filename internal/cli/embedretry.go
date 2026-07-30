package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/embed"
	"github.com/jeroenpf/mneme/internal/store"
)

// newEmbedRetryCmd builds `mneme embed-retry`: re-embed every source whose
// last attempt failed terminally (the 'failed' count in search status). An
// operator action, so it lives here rather than on the MCP surface. Unlike
// the server's async queue, this processes synchronously — the command exits
// when every retry has been attempted.
func newEmbedRetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "embed-retry",
		Short: "Re-embed every source whose last embedding attempt failed terminally",
		Long: "List every source in the terminal-failure bucket and retry each one now,\n" +
			"synchronously. Use after fixing a transient outage (Voyage downtime, rate\n" +
			"limits). Requires embeddings to be configured (MNEME_VOYAGE_API_KEY).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadWithFlags(cmd.Flags())
			if err != nil {
				return err
			}
			return runEmbedRetry(cmd, cfg)
		},
	}
	cmd.Flags().String("dsn", "", "storage DSN (sqlite:// file or postgres:// URL); overrides config/env")
	return cmd
}

func runEmbedRetry(cmd *cobra.Command, cfg *config.Config) error {
	ctx := cmd.Context()
	client := embed.NewClient(*cfg)
	if client == nil {
		return errors.New("embeddings are disabled (no MNEME_VOYAGE_API_KEY) — nothing to retry")
	}
	st, err := store.New(ctx, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	n, err := embed.NewWorker(st, client, 256, cfg.VoyageRPM).RetryFailedNow(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		cmd.Println("no failed embedding sources — nothing to retry")
		return nil
	}
	cmd.Printf("retried %d failed embedding source(s); check `search status` for remaining failures\n", n)
	return nil
}
