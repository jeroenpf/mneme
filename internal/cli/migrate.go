package cli

import (
	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/mcp"
	"github.com/jeroenpf/mneme/internal/store"
)

// newMigrateCmd is the parent for data migrations over existing knowledge.
func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Data migrations for existing knowledge",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newMigrateIDsCmd())
	return cmd
}

// newMigrateIDsCmd builds `mneme migrate ids`: backfill missing block/task ids
// so every node is addressable by a mneme:// reference. It reports by default
// and only writes with --apply, matching the plan's "report before changing".
func newMigrateIDsCmd() *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "ids",
		Short: "Backfill missing block/task ids across every document (reports by default)",
		Long: "Scan every stored document and mint ids for the blocks and tasks that lack\n" +
			"one, so all are addressable by a mneme:// reference. Existing ids — generated\n" +
			"or legacy semantic — are preserved. Reports what would change; pass --apply to\n" +
			"persist. Documents with duplicate ids are reported for manual repair, never\n" +
			"changed automatically.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadWithFlags(cmd.Flags())
			if err != nil {
				return err
			}
			return runMigrateIDs(cmd, cfg, apply)
		},
	}
	cmd.Flags().String("dsn", "", "storage DSN (sqlite:// file or postgres:// URL); overrides config/env")
	cmd.Flags().BoolVar(&apply, "apply", false, "persist the backfill (default: report only)")
	return cmd
}

func runMigrateIDs(cmd *cobra.Command, cfg *config.Config, apply bool) error {
	ctx := cmd.Context()
	st, err := store.New(ctx, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	sum, err := mcp.MigrateDocIDs(ctx, st, apply)
	if err != nil {
		return err
	}

	cmd.Printf("scanned %d document(s)\n", sum.Scanned)
	if sum.Changed() == 0 && len(sum.Problems) == 0 {
		cmd.Println("all documents already have complete, unique ids — nothing to do")
		return nil
	}
	verb := "would mint"
	if apply {
		verb = "minted"
	}
	for id, n := range sum.Backfilled {
		cmd.Printf("  %s: %s %d id(s)\n", id, verb, n)
	}
	for id, problem := range sum.Problems {
		cmd.Printf("  %s: PROBLEM — %s (manual repair needed)\n", id, problem)
	}
	if !apply && sum.Changed() > 0 {
		cmd.Println("re-run with --apply to persist")
	}
	return nil
}
