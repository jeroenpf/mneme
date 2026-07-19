package cli

import (
	"github.com/spf13/cobra"
)

// newInitCmd builds `mneme init`. A stub for now (the command surface is locked
// in cli-p1-t3); the interactive setup wizard lands in a later phase.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard (writes ~/.mneme/settings.toml)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("mneme init: interactive setup is not yet implemented.")
			return nil
		},
	}
}
