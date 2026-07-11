#!/usr/bin/env bash
# Non-destructive scorecard for the mneme.local host setup. One line per
# check, green or red; exits 1 if anything is red. No sudo needed.
set -uo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CERT="$REPO_DIR/.certs/mneme.local.pem"
# System curl trusts the macOS Keychain (where mkcert installs); a
# Homebrew curl ships its own CA bundle and would false-negative.
CURL=/usr/bin/curl

fail=0
check() {
  local label=$1
  shift
  if "$@" >/dev/null 2>&1; then
    printf '\033[32m✓\033[0m %s\n' "$label"
  else
    printf '\033[31m✗\033[0m %s\n' "$label"
    fail=1
  fi
}

check "mkcert on PATH"                 command -v mkcert
check "mkcert root CA present"         test -f "$(mkcert -CAROOT 2>/dev/null)/rootCA.pem"
check "cert exists (.certs/)"          test -f "$CERT"
check "cert not expired"               openssl x509 -checkend 0 -noout -in "$CERT"
check "cert covers mneme.local"        sh -c "openssl x509 -noout -ext subjectAltName -in '$CERT' | grep -q mneme.local"
check "LaunchDaemon plist installed"   test -f /Library/LaunchDaemons/local.mneme.loopback.plist
check "lo0 alias 127.0.0.2 up"         sh -c "ifconfig lo0 | grep -q 'inet 127.0.0.2 '"
check "/etc/hosts has mneme.local"     grep -q "mneme.local" /etc/hosts
check "https://mneme.local/health ok"  sh -c "$CURL -sf --max-time 5 https://mneme.local/health | grep -q '\"ok\":true'"

exit $fail
