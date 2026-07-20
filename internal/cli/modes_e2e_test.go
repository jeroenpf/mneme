package cli

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenpf/mneme/internal/config"
)

// Both network modes selectable via the wizard produce a coherent settings.toml
// that loads into a serveable Config: loopback-bound in both, plain HTTP on the
// localhost default, TLS-configured on the mneme.dev default.
func TestModesProduceLoadableConfig(t *testing.T) {
	cases := []struct {
		mode        string
		wantPort    string
		wantTLS     bool
		originStart string
	}{
		{"localhost", "8765", false, "http://localhost:"},
		{"mneme.dev", "8443", true, "https://mneme.dev:"},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.toml")
			s, err := writeInit(wizardAnswers{Backend: "sqlite", NetMode: tc.mode}, path, io.Discard)
			if err != nil {
				t.Fatalf("writeInit: %v", err)
			}
			if len(s.Net.AllowedOrigins) == 0 || !strings.HasPrefix(s.Net.AllowedOrigins[0], tc.originStart) {
				t.Errorf("origins: got %v, want prefix %q", s.Net.AllowedOrigins, tc.originStart)
			}

			// Isolate the loader onto this file, then verify the resolved Config.
			for _, k := range []string{"MNEME_DSN", "MNEME_HOST", "MNEME_PORT", "MNEME_TLS_CERT", "MNEME_TLS_KEY"} {
				t.Setenv(k, "")
			}
			t.Setenv("MNEME_CONFIG", path)
			c, err := config.Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.Host != "127.0.0.1" {
				t.Errorf("%s mode should bind loopback, got Host %q", tc.mode, c.Host)
			}
			if c.Port != tc.wantPort {
				t.Errorf("%s port: got %q, want %q", tc.mode, c.Port, tc.wantPort)
			}
			if (c.TLSCert != "") != tc.wantTLS {
				t.Errorf("%s TLS cert set = %v, want %v (cert=%q)", tc.mode, c.TLSCert != "", tc.wantTLS, c.TLSCert)
			}
		})
	}
}
