#!/usr/bin/env bash
# One-time host setup for https://mneme.local — idempotent, safe to re-run.
# Run via: sudo make setup-host
#
# Root is required for the LaunchDaemon, ifconfig, and /etc/hosts. mkcert
# steps are pinned to the *invoking user's* CAROOT (root's would be a
# second, unshared CA under /var/root) and results are chowned back.
set -euo pipefail

# Homebrew lives outside root's default PATH.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLIST_SRC="$REPO_DIR/scripts/local.mneme.loopback.plist"
PLIST_DST="/Library/LaunchDaemons/local.mneme.loopback.plist"
CERT_DIR="$REPO_DIR/.certs"
LABEL="local.mneme.loopback"

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

echo "→ leaf cert for mneme.local / 127.0.0.2 / localhost"
mkdir -p "$CERT_DIR"
CAROOT="$CAROOT_DIR" mkcert \
  -cert-file "$CERT_DIR/mneme.local.pem" \
  -key-file "$CERT_DIR/mneme.local-key.pem" \
  mneme.local 127.0.0.2 localhost
chown -R "$SUDO_USER" "$CAROOT_DIR" "$CERT_DIR"

echo "→ LaunchDaemon (alias survives reboots)"
install -o root -g wheel -m 0644 "$PLIST_SRC" "$PLIST_DST"
# bootstrap fails EEXIST when already loaded; print-probe first. After
# editing the plist, unload with: sudo launchctl bootout system/$LABEL
launchctl print "system/$LABEL" >/dev/null 2>&1 || launchctl bootstrap system "$PLIST_DST"

echo "→ lo0 alias now (no reboot needed)"
ifconfig lo0 | grep -q "inet 127.0.0.2 " || ifconfig lo0 alias 127.0.0.2 up

echo "→ /etc/hosts entry"
grep -q "mneme.local" /etc/hosts || printf "127.0.0.2\tmneme.local\n" >> /etc/hosts

echo "done — run 'make verify-host' for the scorecard"
