package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jeroenpf/mneme/internal/appinfo"
)

// renderStartup is the human-facing summary printed when `mneme server` boots:
// where to open it, where data lives, the search mode, and where to wire up an
// agent. It mirrors the init wizard's summary style and uses the friendly URL
// from appinfo — never the 127.0.0.1 bind address.
func renderStartup(info appinfo.Info) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2f5d80"))

	search := "lexical only (FTS, local)"
	if info.Embeddings.Enabled {
		search = "semantic (Voyage " + info.Embeddings.Model + ")"
	}

	storage := info.DB.Driver
	if info.DB.Path != "" {
		storage = abbreviateHome(info.DB.Path) + "  (" + info.DB.Driver + ")"
	}

	return fmt.Sprintf(`
%s

  open       %s
  storage    %s
  search     %s
  MCP        %s
  help       %s   -> connect Claude Code, Codex, ...

  Stop with Ctrl-C.
`,
		title.Render("Mneme is running."),
		info.URL,
		storage,
		search,
		info.MCPEndpoint,
		info.URL+"/help",
	)
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
