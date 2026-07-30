package cli

import (
	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/mcp"
	"github.com/jeroenpf/mneme/internal/store"
)

// newLintCmd builds `mneme lint`: the read-only sweep of every stored document
// for inline-only violations and structural problems that predate write-path
// validation. An operator action, so it lives here rather than on the MCP
// surface; each hit's doc_id + block_id feed update_section/update_task fixes.
func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Report structural and prose violations across every stored document (read-only)",
		Long: "Scan every stored document — all projects and statuses, archived included —\n" +
			"for inline-only violations (lists/headings/fences in prose fields) and\n" +
			"structural problems (unknown types or fields) that predate write-path\n" +
			"validation. Reports doc id, block id, path, and what was found; changes nothing.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadWithFlags(cmd.Flags())
			if err != nil {
				return err
			}
			return runLint(cmd, cfg)
		},
	}
	cmd.Flags().String("dsn", "", "storage DSN (sqlite:// file or postgres:// URL); overrides config/env")
	return cmd
}

func runLint(cmd *cobra.Command, cfg *config.Config) error {
	ctx := cmd.Context()
	st, err := store.New(ctx, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	report, err := mcp.LintStore(ctx, st)
	if err != nil {
		return err
	}
	cmd.Printf("scanned %d document(s)\n", report.DocsScanned)
	if len(report.Hits) == 0 {
		cmd.Println("no violations — every document passes the current write-path rules")
		return nil
	}
	cmd.Printf("%d violation(s) in %d document(s):\n", len(report.Hits), report.DocsWithHits)
	for _, h := range report.Hits {
		cmd.Printf("  %s (%s) %s.%s: %s", h.DocID, h.DocStatus, h.Path, h.Field, h.Found)
		if h.BlockID != "" {
			cmd.Printf(" [block %s]", h.BlockID)
		}
		if h.Excerpt != "" {
			cmd.Printf(" — %s", h.Excerpt)
		}
		cmd.Println()
	}
	return nil
}
