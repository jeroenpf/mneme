package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/jeroenpfeil/mneme/internal/config"
)

// wizardAnswers is the raw set of choices the init form collects. It is a plain
// value with no IO so the mapping to Settings (buildSettings) can be unit-tested
// independently of the interactive huh layer.
type wizardAnswers struct {
	Backend     string // "sqlite" | "postgres"
	SQLitePath  string // sqlite backend; empty → the ~/.mneme default
	PostgresDSN string // postgres backend; required when Backend == "postgres"

	Embeddings bool   // true → Voyage; false → lexical-only (FTS) mode
	VoyageKey  string // required when Embeddings is true
	VoyageRPM  int    // optional proactive throttle; 0 = off

	NetMode string // "localhost" (plain HTTP) | "mneme.dev" (HTTPS)
	Port    string // empty → per-mode default (8080 localhost, 8443 mneme.dev)
}

// buildSettings maps validated wizard answers to a config.Settings. It is pure:
// no files touched, no mkcert run — that IO lives in the command layer. Returns
// an error for the two inconsistent combinations (postgres with no DSN,
// embeddings enabled with no key) so the form can re-prompt.
func buildSettings(a wizardAnswers) (config.Settings, error) {
	var s config.Settings

	switch a.Backend {
	case "postgres":
		if a.PostgresDSN == "" {
			return s, fmt.Errorf("postgres backend requires a DSN")
		}
		s.Data.DSN = a.PostgresDSN
	case "sqlite", "":
		path := a.SQLitePath
		if path == "" {
			path = config.DefaultSQLitePath()
		}
		s.Data.DSN = "sqlite://" + path
	default:
		return s, fmt.Errorf("unknown backend %q", a.Backend)
	}

	if a.Embeddings {
		if a.VoyageKey == "" {
			return s, fmt.Errorf("embeddings enabled but no Voyage API key provided")
		}
		s.Embeddings.VoyageAPIKey = a.VoyageKey
		s.Embeddings.VoyageModel = "voyage-4-large"
		s.Embeddings.VoyageRPM = a.VoyageRPM
	}

	switch a.NetMode {
	case "mneme.dev":
		s.Net.Port = orDefault(a.Port, "8443")
		s.Net.TLSCert, s.Net.TLSKey = config.CertPaths()
		s.Net.AllowedOrigins = []string{"https://mneme.dev:" + s.Net.Port}
	case "localhost", "":
		s.Net.Port = orDefault(a.Port, "8765")
		s.Net.AllowedOrigins = []string{"http://localhost:" + s.Net.Port}
	default:
		return s, fmt.Errorf("unknown network mode %q", a.NetMode)
	}

	return s, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// serverURL is the address a browser (or Claude Code) uses to reach the server
// for the chosen settings — the friendly line the summary prints.
func serverURL(s config.Settings) string {
	if s.Net.TLSCert != "" {
		return "https://mneme.dev:" + s.Net.Port
	}
	return "http://localhost:" + s.Net.Port
}

// initTheme is the wizard's lipgloss-based huh theme: the base theme with
// Mneme's accent applied so the form reads as part of the product.
func initTheme() *huh.Theme {
	t := huh.ThemeBase()
	accent := lipgloss.Color("#2f5d80") // the app's theme-color
	t.Focused.Title = t.Focused.Title.Foreground(accent).Bold(true)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.Base = t.Focused.Base.BorderForeground(accent)
	return t
}

// newInitForm builds the interactive setup form bound to the fields of a.
// Conditional groups keep irrelevant questions hidden (the SQLite path only
// when SQLite is chosen, the Voyage key only when embeddings are enabled). The
// RPM is collected as text and parsed in collectAnswers. The form is a thin
// shell over buildSettings — no logic lives here.
func newInitForm(a *wizardAnswers, rpmText *string) *huh.Form {
	// huh hides at the group level, so each conditional question is its own
	// group gated by a HideFunc reading the answers filled in so far.
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Storage backend").
				Description("SQLite is self-contained; Postgres is for an existing server.").
				Options(
					huh.NewOption("SQLite file (recommended)", "sqlite"),
					huh.NewOption("PostgreSQL", "postgres"),
				).
				Value(&a.Backend),
		).Title("Data"),

		huh.NewGroup(
			huh.NewInput().
				Title("SQLite database path").
				Placeholder(config.DefaultSQLitePath()).
				Description("Leave blank for the default ~/.mneme/mneme.db.").
				Value(&a.SQLitePath),
		).WithHideFunc(func() bool { return a.Backend != "sqlite" }),

		huh.NewGroup(
			huh.NewInput().
				Title("PostgreSQL DSN").
				Placeholder("postgres://user:pass@host:5432/mneme?sslmode=disable").
				Value(&a.PostgresDSN),
		).WithHideFunc(func() bool { return a.Backend != "postgres" }),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable semantic search (Voyage embeddings)?").
				Description("No keeps everything local (lexical/FTS only). Yes sends text to Voyage.").
				Value(&a.Embeddings),
		).Title("Embeddings"),

		huh.NewGroup(
			huh.NewInput().
				Title("Voyage API key").
				EchoMode(huh.EchoModePassword).
				Value(&a.VoyageKey),
			huh.NewInput().
				Title("Voyage requests/min (0 = no proactive throttle)").
				Placeholder("0").
				Value(rpmText),
		).WithHideFunc(func() bool { return !a.Embeddings }),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Networking").
				Description("localhost is zero-setup plain HTTP; mneme.dev adds trusted HTTPS.").
				Options(
					huh.NewOption("localhost + plain HTTP (recommended)", "localhost"),
					huh.NewOption("mneme.dev + HTTPS (runs mkcert + /etc/hosts)", "mneme.dev"),
				).
				Value(&a.NetMode),
			huh.NewInput().
				Title("Port").
				Description("Leave blank for the per-mode default (8080 / 8443).").
				Value(&a.Port),
		).Title("Networking"),
	)
}

// collectAnswers runs the init form over the given IO and returns the resolved
// answers. Passing a non-TTY reader (e.g. in tests) runs the form in accessible
// mode, which reads plain lines — one per visible field.
func collectAnswers(in io.Reader, out io.Writer, accessible bool) (wizardAnswers, error) {
	var a wizardAnswers
	var rpmText string
	form := newInitForm(&a, &rpmText).
		WithTheme(initTheme()).
		WithInput(in).
		WithOutput(out).
		WithAccessible(accessible)
	if err := form.Run(); err != nil {
		return a, err
	}
	if rpmText != "" {
		n, err := strconv.Atoi(rpmText)
		if err != nil {
			return a, fmt.Errorf("invalid requests/min %q: %w", rpmText, err)
		}
		a.VoyageRPM = n
	}
	return a, nil
}

// writeInit is the deterministic tail of the wizard: validate answers into
// Settings, write settings.toml (0600), and print a friendly summary. It is the
// flow's testable core — no TTY, only the file write.
func writeInit(a wizardAnswers, path string, out io.Writer) (config.Settings, error) {
	s, err := buildSettings(a)
	if err != nil {
		return s, err
	}
	if err := config.WriteSettings(path, &s); err != nil {
		return s, err
	}
	fmt.Fprint(out, renderSummary(s, path))
	return s, nil
}

// renderSummary is the post-write recap: where the config went, the effective
// storage/search/network choices, and the URL to reach the server. The Voyage
// key is never echoed.
func renderSummary(s config.Settings, path string) string {
	search := "lexical only (FTS, fully local)"
	if s.Embeddings.VoyageAPIKey != "" {
		search = "semantic (Voyage " + s.Embeddings.VoyageModel + ")"
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2f5d80"))
	return fmt.Sprintf(`
%s

  config    %s
  storage   %s
  search    %s
  address   %s

Start it with:  mneme server
`, title.Render("Mneme is configured."), path, s.Data.DSN, search, serverURL(s))
}
