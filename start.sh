#!/bin/sh
# Build every PickMem surface from the current checkout, initialize a vault
# when needed, and launch the embedded web app.
#
#   ./start.sh [vault-path]
#
# Environment overrides:
#   PICKMEM_VAULT       vault to use when no path argument is supplied
#   PICKMEM_PORT        web server port (default: 4577)
#   PICKMEM_NO_OPEN     set to 1 to avoid opening the browser

set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
VAULT=${1:-${PICKMEM_VAULT:-$HOME/PickMemVault}}
PORT=${PICKMEM_PORT:-4577}

say()  { printf '%s\n' "$*"; }
fail() { printf 'start.sh: %s\n' "$*" >&2; exit 1; }

command -v go >/dev/null 2>&1 || fail "Go is required (see https://go.dev/dl/)"
command -v node >/dev/null 2>&1 || fail "Node.js is required (see https://nodejs.org/)"
command -v npm >/dev/null 2>&1 || fail "npm is required"
[ "$#" -le 1 ] || fail "usage: ./start.sh [vault-path]"

say "Building the web app…"
npm --prefix "$ROOT/webapp" ci
npm --prefix "$ROOT/webapp" run build

say "Building the Chrome extension…"
npm --prefix "$ROOT/extension" ci
npm --prefix "$ROOT/extension" run test
npm --prefix "$ROOT/extension" run build

say "Building PickMem…"
(cd "$ROOT" && go build -o "$ROOT/pickmem" ./cmd/pickmem)

if [ ! -f "$VAULT/pickmem/config.json" ]; then
  say "Initializing vault at ${VAULT}…"
  "$ROOT/pickmem" init "$VAULT"
fi

set -- --vault "$VAULT" web --port "$PORT"
if [ "${PICKMEM_NO_OPEN:-}" = "1" ]; then
  set -- "$@" --no-open
fi

say "Starting PickMem…"
exec "$ROOT/pickmem" "$@"
