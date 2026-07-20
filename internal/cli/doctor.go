package cli

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/store"
)

// newDoctorCmd builds `mneme doctor` — a one-shot diagnostics scorecard over
// config, database + migrations, networking (cert/hosts), search, and
// embeddings. Exits non-zero if any check fails.
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose config, database, migrations, certs/hosts, and search",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd)
		},
	}
}

// runDoctor collects every diagnostic, prints the scorecard, and returns an
// error (→ non-zero exit) when any check failed. Warnings never fail the run.
func runDoctor(cmd *cobra.Command) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	var results []check

	cfg, cfgErr := config.LoadWithFlags(cmd.Flags())
	results = append(results, checkConfig(cfg, cfgErr))

	if cfgErr == nil {
		if st, err := store.New(ctx, cfg.DSN); err != nil {
			results = append(results, fail("database", "cannot open store: "+err.Error()))
		} else {
			defer st.Close()
			// Idempotently bring the schema to head so doctor reflects (and, on
			// a fresh install, establishes) a healthy database.
			if err := migrations.Up(cfg.DSN); err != nil {
				results = append(results, fail("database", "migrations failed: "+err.Error()))
			} else {
				results = append(results, checkDatabase(ctx, st), checkSearch(ctx, st), checkEmbeddings(ctx, cfg, st), checkBackups(ctx, st))
			}
		}
		results = append(results, checkTLS(cfg))
	}

	renderScorecard(out, results)
	if n := countFailed(results); n > 0 {
		return fmt.Errorf("%d check(s) failed", n)
	}
	return nil
}

func countFailed(results []check) int {
	n := 0
	for _, c := range results {
		if c.level == levelFail {
			n++
		}
	}
	return n
}

// renderScorecard prints the checks as a lipgloss-styled scorecard. Colour is
// emitted only to a terminal (lipgloss/termenv strips it when redirected).
func renderScorecard(out io.Writer, results []check) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	name := lipgloss.NewStyle().Bold(true)

	fmt.Fprintln(out)
	fmt.Fprintln(out, lipgloss.NewStyle().Bold(true).Render("mneme doctor"))
	fmt.Fprintln(out)

	for _, c := range results {
		glyph, style := "✓", green
		switch c.level {
		case levelWarn:
			glyph, style = "!", yellow
		case levelFail:
			glyph, style = "✗", red
		}
		fmt.Fprintf(out, "  %s %s  %s\n", style.Render(glyph), name.Render(pad(c.name, 11)), c.detail)
	}

	fmt.Fprintln(out)
	if n := countFailed(results); n > 0 {
		fmt.Fprintln(out, red.Render(fmt.Sprintf("%d check(s) failed.", n)))
	} else {
		fmt.Fprintln(out, green.Render("All checks passed."))
	}
}

// pad right-pads s to width for column alignment (plain runes; styling is
// applied by the caller so it does not skew the width).
func pad(s string, width int) string {
	for len(s) < width {
		s += " "
	}
	return s
}
