#!/bin/sh
# PickMem installer — downloads the right prebuilt binary from the latest
# GitHub release and puts it on your PATH.
#
#   curl -fsSL https://raw.githubusercontent.com/kaiagaoo/PickMem/main/install.sh | sh
#
# Options (env vars):
#   PICKMEM_VERSION      install a specific tag, e.g. v0.1.1 (default: latest)
#   PICKMEM_INSTALL_DIR  target dir (default: /usr/local/bin if writable,
#                        else ~/.local/bin)
#
# POSIX sh on purpose — works under dash/ash so `curl | sh` behaves the
# same on minimal Linux images as on macOS.

set -eu

REPO="kaiagaoo/PickMem"
BINARY="pickmem"

say()  { printf '%s\n' "$*"; }
fail() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar  >/dev/null 2>&1 || fail "tar is required"

# ---------- platform ----------

OS=$(uname -s)
case "$OS" in
  Darwin) OS=darwin ;;
  Linux)  OS=linux ;;
  *) fail "unsupported OS: $OS — on Windows, download the .zip from https://github.com/$REPO/releases/latest" ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64|amd64)  ARCH=amd64 ;;
  *) fail "unsupported architecture: $ARCH" ;;
esac

# ---------- version ----------

if [ "${PICKMEM_VERSION:-}" != "" ]; then
  TAG="$PICKMEM_VERSION"
else
  # The releases/latest redirect carries the tag in its Location header —
  # no API quota, no JSON parsing.
  TAG=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/$REPO/releases/latest" | sed 's|.*/tag/||')
  [ -n "$TAG" ] || fail "could not determine the latest release tag"
fi
VERSION=${TAG#v}

# ---------- download + verify ----------

ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$TAG"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

say "Downloading $ASSET ($TAG)…"
curl -fsSL "$BASE/$ASSET" -o "$TMP/$ASSET" \
  || fail "download failed — does $TAG have an asset for ${OS}/${ARCH}?"

if curl -fsSL "$BASE/checksums.txt" -o "$TMP/checksums.txt" 2>/dev/null; then
  (
    cd "$TMP"
    SUM=$(grep " $ASSET\$" checksums.txt | awk '{print $1}')
    [ -n "$SUM" ] || fail "asset missing from checksums.txt"
    if command -v sha256sum >/dev/null 2>&1; then
      GOT=$(sha256sum "$ASSET" | awk '{print $1}')
    else
      GOT=$(shasum -a 256 "$ASSET" | awk '{print $1}')
    fi
    [ "$SUM" = "$GOT" ] || fail "checksum mismatch for $ASSET"
  )
  say "Checksum verified."
else
  say "Warning: checksums.txt not found; skipping verification."
fi

tar -xzf "$TMP/$ASSET" -C "$TMP" "$BINARY"

# ---------- install ----------

if [ "${PICKMEM_INSTALL_DIR:-}" != "" ]; then
  DIR="$PICKMEM_INSTALL_DIR"
elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
  DIR=/usr/local/bin
else
  DIR="$HOME/.local/bin"
fi
mkdir -p "$DIR"
install -m 0755 "$TMP/$BINARY" "$DIR/$BINARY"

say ""
say "Installed $BINARY $VERSION to $DIR/$BINARY"

case ":$PATH:" in
  *":$DIR:"*) ;;
  *)
    say ""
    say "Note: $DIR is not on your PATH. Add this to your shell profile:"
    say "  export PATH=\"$DIR:\$PATH\""
    ;;
esac

say ""
say "Get started:"
say "  $BINARY init ~/PickMemVault    # create your vault"
say "  $BINARY --help"
