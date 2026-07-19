package cli

import (
	"github.com/spf13/cobra"
)

// newDoctorCmd builds `mneme doctor`. A stub for now (the command surface is
// locked in cli-p1-t3); the real diagnostics scorecard lands in a later phase.
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose config, database, migrations, certs/hosts, and search",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("mneme doctor: diagnostics are not yet implemented.")
			return nil
		},
	}
}
