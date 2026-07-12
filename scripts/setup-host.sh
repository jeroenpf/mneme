#!/usr/bin/env bash
# One-time host setup for https://mneme.dev:8443 — idempotent, safe to re-run.
# Run via: sudo make setup-host
#
# Root is required only for the /etc/hosts entry. mkcert steps are pinned to
# the *invoking user's* CAROOT (root's would be a second, unshared CA under
# /var/root) and results are chowned back.
#
# Why mneme.dev → 127.0.0.1:8443, and not mneme.local on a 127.0.0.2:443
# loopback alias (as earlier drafts did) — two macOS/Docker realities, both
# documented in the deployment spec's addendum:
#   1. *.local is resolved via mDNS on macOS, which adds a ~5s timeout before
#      falling back to /etc/hosts — mneme.dev resolves instantly instead.
#   2. Docker Desktop (gVisor) won't reliably forward to a secondary loopback
#      alias as a container's sole host IP; 127.0.0.1 is the reliable path,
#      and the high port 8443 keeps clear of anything already on :443.
set -euo pipefail

# Homebrew lives outside root's default PATH.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CERT_DIR="$REPO_DIR/.certs"

if [[ $EUID -ne 0 ]]; then
  echo "needs root — run: sudo make setup-host" >&2
  exit 1
fi
if [[ -z "${SUDO_USER:-}" || "$SUDO_USER" == "root" ]]; then
  echo "SUDO_USER is unset — run via sudo from your user account, not a root shell" >&2
  exit 1
fi
command -v mkcert >/dev/null 2>&1 || { echo "mkcert not found — brew install mkcert" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "docker not found — is Docker installed?" >&2; exit 1; }

echo "→ mkcert root CA (user's CAROOT, system trust store)"
CAROOT_DIR="$(sudo -u "$SUDO_USER" -H mkcert -CAROOT)"
CAROOT="$CAROOT_DIR" mkcert -install

echo "→ leaf cert for mneme.dev / localhost / 127.0.0.1"
mkdir -p "$CERT_DIR"
CAROOT="$CAROOT_DIR" mkcert \
  -cert-file "$CERT_DIR/mneme.dev.pem" \
  -key-file "$CERT_DIR/mneme.dev-key.pem" \
  mneme.dev localhost 127.0.0.1
chown -R "$SUDO_USER" "$CAROOT_DIR" "$CERT_DIR"

echo "→ /etc/hosts entry (mneme.dev → 127.0.0.1)"
if ! grep -qE '^\s*127\.0\.0\.1\s+.*\bmneme\.dev\b' /etc/hosts; then
  # Ensure the file ends with a newline before appending — otherwise a
  # last line lacking one (common) fuses with our entry into one broken row.
  [[ -s /etc/hosts && -n "$(tail -c1 /etc/hosts)" ]] && echo "" >> /etc/hosts
  printf "127.0.0.1\tmneme.dev\n" >> /etc/hosts
fi

echo "done — run 'make verify-host' for the scorecard"
