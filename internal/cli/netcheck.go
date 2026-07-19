package cli

import (
	"net"
	"strconv"
)

// portScanRange bounds how far above the preferred port availablePort probes
// before giving up and returning the preferred one (letting the server surface
// the bind error rather than wandering off to an unrelated port).
const portScanRange = 50

// availablePort returns preferred if it can be bound on host, otherwise the
// next free TCP port above it (scanning a small range). A non-numeric preferred
// is returned unchanged. Used by `mneme init` so setup never writes a config
// pointing at a port that is already in use.
func availablePort(host, preferred string) string {
	base, err := strconv.Atoi(preferred)
	if err != nil {
		return preferred
	}
	for p := base; p < base+portScanRange && p <= 65535; p++ {
		addr := net.JoinHostPort(host, strconv.Itoa(p))
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			_ = ln.Close()
			return strconv.Itoa(p)
		}
	}
	return preferred
}

// defaultPortFor is the built-in port for a network mode: 8443 for the HTTPS
// mneme.dev mode, 8765 for plain-HTTP localhost.
func defaultPortFor(mode string) string {
	if mode == "mneme.dev" {
		return "8443"
	}
	return "8765"
}

// resolvePort picks the port `mneme init` should write: the requested value (or
// the per-mode default when blank), bumped past any in-use port. The bool
// reports whether it had to move off the requested port so the caller can say so.
func resolvePort(host, mode, requested string) (port string, bumped bool) {
	want := orDefault(requested, defaultPortFor(mode))
	got := availablePort(host, want)
	return got, got != want
}
