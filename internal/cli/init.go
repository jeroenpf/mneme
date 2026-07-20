package cli

import (
	"errors"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/config"
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

	// Piped/scripted stdin (or a screen reader) → accessible line-based prompts
	// instead of the full-screen TUI.
	accessible := !isInputTerminal(in)

	answers, err := collectAnswers(in, out, accessible)
	if errors.Is(err, huh.ErrUserAborted) {
		cmd.Println("\nSetup cancelled — no changes made.")
		return nil
	}
	if err != nil {
		return err
	}

	// Never write a config pointing at an in-use port: default per mode, then
	// bump past anything already listening on loopback.
	port, bumped := resolvePort("127.0.0.1", answers.NetMode, answers.Port)
	if bumped {
		cmd.Printf("Port %s is in use — using %s instead.\n", orDefault(answers.Port, defaultPortFor(answers.NetMode)), port)
	}
	answers.Port = port

	// HTTPS opt-in: generate the trusted cert + hosts entry the settings will
	// point at, before writing the config.
	if answers.NetMode == "mneme.dev" {
		if err := setupHTTPS(cmd.Context(), out); err != nil {
			return err
		}
	}

	if _, err := writeInit(answers, path, out); err != nil {
		return err
	}

	launch, err := confirmLaunch(in, out, accessible)
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
func confirmLaunch(in io.Reader, out io.Writer, accessible bool) (bool, error) {
	var launch bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start the Mneme server now?").
				Value(&launch),
		),
	).WithTheme(initTheme()).WithInput(in).WithOutput(out).WithAccessible(accessible)
	if err := form.Run(); err != nil {
		return false, err
	}
	return launch, nil
}
