package cli

import (
	"errors"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/jeroenpfeil/mneme/internal/config"
)

// newInitCmd builds `mneme init` — the interactive setup wizard. It collects a
// few answers, writes ~/.mneme/settings.toml (0600), and offers to launch the
// server. The wizard is a thin shell over the pure buildSettings/writeInit core.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard (writes ~/.mneme/settings.toml)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd)
		},
	}
}

// runInit drives the interactive flow: collect answers → write settings + show
// a summary → offer to start the server. A user abort (Ctrl-C in the form) is a
// clean cancel, not an error.
func runInit(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	path := config.SettingsPath()
	if f := cmd.Flags().Lookup("config"); f != nil && f.Changed {
		path = f.Value.String()
	}

	answers, err := collectAnswers(in, out, false)
	if errors.Is(err, huh.ErrUserAborted) {
		cmd.Println("\nSetup cancelled — no changes made.")
		return nil
	}
	if err != nil {
		return err
	}

	// NOTE: cli-p4 inserts the mkcert + /etc/hosts automation for mneme.dev
	// mode here, before the file is written.

	if _, err := writeInit(answers, path, out); err != nil {
		return err
	}

	launch, err := confirmLaunch(in, out)
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	if err != nil {
		return err
	}
	if !launch {
		return nil
	}

	cfg, err := config.LoadWithFlags(cmd.Flags())
	if err != nil {
		return err
	}
	return RunServer(cmd.Context(), cfg)
}

// confirmLaunch asks whether to start the server immediately after setup.
func confirmLaunch(in io.Reader, out io.Writer) (bool, error) {
	var launch bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start the Mneme server now?").
				Value(&launch),
		),
	).WithTheme(initTheme()).WithInput(in).WithOutput(out)
	if err := form.Run(); err != nil {
		return false, err
	}
	return launch, nil
}
