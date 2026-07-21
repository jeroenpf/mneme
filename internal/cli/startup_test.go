package cli

import (
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/appinfo"
)

func TestRenderStartupLocalhost(t *testing.T) {
	info := appinfo.Info{
		Version:     "0.1.1",
		Mode:        "localhost",
		URL:         "http://localhost:8901",
		MCPEndpoint: "http://localhost:8901/mcp",
		DB:          appinfo.DBInfo{Driver: "sqlite", Path: "/home/u/.mneme/mneme.db"},
		Embeddings:  appinfo.EmbInfo{Enabled: false},
	}
	out := renderStartup(info)

	for _, want := range []string{
		"http://localhost:8901",
		"http://localhost:8901/mcp",
		"http://localhost:8901/help",
		"sqlite",
		"lexical",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("startup box missing %q:\n%s", want, out)
		}
	}
	// The friendly URL must never be the 127.0.0.1 bind address.
	if strings.Contains(out, "127.0.0.1") {
		t.Errorf("startup box must not show the bind address:\n%s", out)
	}
}

func TestRenderStartupDrawsOutlinedBox(t *testing.T) {
	info := appinfo.Info{
		URL:         "http://localhost:8901",
		MCPEndpoint: "http://localhost:8901/mcp",
		DB:          appinfo.DBInfo{Driver: "sqlite", Path: "/x/mneme.db"},
	}
	out := renderStartup(info)

	for _, want := range []string{"╭", "╰", "│", "Mneme is running.", "connect Claude Code", "Stop with Ctrl-C."} {
		if !strings.Contains(out, want) {
			t.Errorf("startup box missing %q:\n%s", want, out)
		}
	}
	// The Ctrl-C hint lives outside the box; the last border row must close it.
	if strings.Index(out, "Stop with Ctrl-C.") < strings.Index(out, "╰") {
		t.Errorf("Ctrl-C hint should follow the box, not sit inside it:\n%s", out)
	}
}

func TestRenderStartupSemanticNamesModel(t *testing.T) {
	info := appinfo.Info{
		URL:         "http://localhost:8901",
		MCPEndpoint: "http://localhost:8901/mcp",
		DB:          appinfo.DBInfo{Driver: "sqlite", Path: "/x/mneme.db"},
		Embeddings:  appinfo.EmbInfo{Enabled: true, Model: "voyage-4-large"},
	}
	out := renderStartup(info)

	if !strings.Contains(out, "voyage-4-large") {
		t.Errorf("semantic mode should name the model:\n%s", out)
	}
	if strings.Contains(out, "lexical") {
		t.Errorf("semantic mode should not say lexical:\n%s", out)
	}
}
