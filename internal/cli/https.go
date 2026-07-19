package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/jeroenpfeil/mneme/internal/config"
)

// hostsPath is the system hosts file; a var so tests could point elsewhere.
var hostsPath = "/etc/hosts"

// mnemeHostsLine matches a live (uncommented) 127.0.0.1 → mneme.dev mapping,
// allowing other hosts to share the line. \b after mneme\.dev prevents matching
// e.g. "mneme.development".
var mnemeHostsLine = regexp.MustCompile(`(?m)^\s*127\.0\.0\.1\s+(\S+\s+)*mneme\.dev\b`)

// hostsHasMnemeEntry reports whether /etc/hosts already maps mneme.dev to
// loopback, so setup is idempotent and skips a redundant (root-requiring) edit.
func hostsHasMnemeEntry(content string) bool {
	return mnemeHostsLine.MatchString(content)
}

// mkcertLeafArgs is the argument vector for generating the leaf certificate that
// covers mneme.dev plus the loopback names.
func mkcertLeafArgs(certPath, keyPath string) []string {
	return []string{"-cert-file", certPath, "-key-file", keyPath, "mneme.dev", "localhost", "127.0.0.1"}
}

// mkcertAvailable reports whether the mkcert binary is on PATH.
func mkcertAvailable() bool {
	_, err := exec.LookPath("mkcert")
	return err == nil
}

// setupHTTPS runs the mneme.dev + HTTPS opt-in automation: install the mkcert
// local CA, generate the leaf cert into ~/.mneme/certs, ensure the /etc/hosts
// entry, and verify the cert files landed. Long steps show a spinner. It is
// only invoked from `mneme init` when the user picks the mneme.dev network mode.
func setupHTTPS(ctx context.Context, out io.Writer) error {
	if !mkcertAvailable() {
		return fmt.Errorf("mkcert not found — install it first (macOS: `brew install mkcert`), then re-run `mneme init`")
	}

	cert, key := config.CertPaths()
	if err := os.MkdirAll(filepath.Dir(cert), 0o700); err != nil {
		return fmt.Errorf("create cert dir: %w", err)
	}

	if err := runStep(out, "Installing a locally-trusted certificate (mkcert)", func() error {
		return runMkcert(ctx, cert, key)
	}); err != nil {
		return err
	}

	if err := ensureHostsEntry(out); err != nil {
		return err
	}

	for _, p := range []string{cert, key} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("expected cert file %s missing after mkcert: %w", p, err)
		}
	}
	fmt.Fprintln(out, "HTTPS ready — certificate trusted, reachable at https://mneme.dev:8443")
	return nil
}

// runMkcert installs the CA (idempotent) and writes the leaf cert/key.
func runMkcert(ctx context.Context, certPath, keyPath string) error {
	if out, err := exec.CommandContext(ctx, "mkcert", "-install").CombinedOutput(); err != nil {
		return fmt.Errorf("mkcert -install: %w: %s", err, out)
	}
	if out, err := exec.CommandContext(ctx, "mkcert", mkcertLeafArgs(certPath, keyPath)...).CombinedOutput(); err != nil {
		return fmt.Errorf("mkcert leaf cert: %w: %s", err, out)
	}
	return nil
}

// ensureHostsEntry adds the mneme.dev → 127.0.0.1 mapping to /etc/hosts if it is
// missing. Editing /etc/hosts needs root, so when the file is not writable this
// prints the exact sudo command for the user rather than failing setup — the
// cert is already generated and the rest of setup can complete.
func ensureHostsEntry(out io.Writer) error {
	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", hostsPath, err)
	}
	if hostsHasMnemeEntry(string(content)) {
		return nil
	}

	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintf(out, "\nOne manual step — add mneme.dev to your hosts file (needs sudo):\n\n"+
			"  echo '127.0.0.1 mneme.dev' | sudo tee -a %s\n\n", hostsPath)
		return nil
	}
	defer f.Close()
	if _, err := f.WriteString("127.0.0.1\tmneme.dev\n"); err != nil {
		return fmt.Errorf("append %s: %w", hostsPath, err)
	}
	fmt.Fprintf(out, "Added mneme.dev → 127.0.0.1 to %s\n", hostsPath)
	return nil
}
