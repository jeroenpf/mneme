package cli

import (
	"github.com/spf13/cobra"
)

// newExportCmd and newImportCmd stub the Postgres⇄SQLite ETL commands from the
// spec's command surface. The real data migration is deferred to a later
// effort; the commands exist now so the surface is stable and discoverable, and
// they say plainly that they do nothing yet rather than exiting silently.
func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export knowledge to a portable SQLite file (deferred)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("mneme export: not yet implemented (the Postgres → SQLite ETL is deferred).")
			return nil
		},
	}
}

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import knowledge from a SQLite file (deferred)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("mneme import: not yet implemented (the SQLite → Postgres ETL is deferred).")
			return nil
		},
	}
}
