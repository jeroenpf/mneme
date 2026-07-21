package cli

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jeroenpf/mneme/internal/appinfo"
)

// renderStartup is the human-facing summary printed when `mneme server` boots:
// where to open it, where data lives, the search mode, and where to wire up an
// agent, drawn as an outlined box in the app accent. It uses the friendly URL
// from appinfo — never the 127.0.0.1 bind address.
func renderStartup(info appinfo.Info) string {
	accent := lipgloss.Color("#2f5d80")
	dim := lipgloss.Color("#626262")

	title := lipgloss.NewStyle().Bold(true).Foreground(accent)
	label := lipgloss.NewStyle().Bold(true).Foreground(accent).Width(10)
	hint := lipgloss.NewStyle().Foreground(dim)

	search := "lexical only (FTS, local)"
	if info.Embeddings.Enabled {
		search = "semantic (Voyage " + info.Embeddings.Model + ")"
	}

	storage := info.DB.Driver
	if info.DB.Path != "" {
		storage = abbreviateHome(info.DB.Path) + "  (" + info.DB.Driver + ")"
	}

	rows := []string{
		title.Render("Mneme is running."),
		"",
		label.Render("open") + info.URL,
		label.Render("storage") + storage,
		label.Render("search") + search,
		label.Render("MCP") + info.MCPEndpoint,
		label.Render("help") + info.URL + "/help",
		label.Render("") + hint.Render("└─ connect Claude Code, Codex, ..."),
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1).
		Render(strings.Join(rows, "\n"))

	return "\n" + box + "\n  " + hint.Render("Stop with Ctrl-C.") + "\n"
}

// abbreviateHome shortens the user's home directory to ~ for display. A path
// outside home (or an unknown home) is returned unchanged.
func abbreviateHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(path, home) {
		return path
	}
	return "~" + strings.TrimPrefix(path, home)
}
