#!/bin/sh
# PickMem installer — downloads the right prebuilt binary from the latest
# GitHub release and puts it on your PATH.
#
#   curl -fsSL https://raw.githubusercontent.com/kaiagaoo/PickMem/main/install.sh | sh
#
# This installs everything: the `pickmem` binary (TUI + MCP server + the
# built-in web app, which is embedded in the binary and served by `pickmem
# web`) AND the Chrome extension, unpacked into a folder you can "Load
# unpacked" directly.
#
# Options (env vars):
#   PICKMEM_VERSION        install a specific tag, e.g. v0.1.1 (default: latest)
#   PICKMEM_INSTALL_DIR    binary target dir (default: /usr/local/bin if
#                          writable, else ~/.local/bin)
#   PICKMEM_EXTENSION_DIR  where to unpack the Chrome extension
#                          (default: ~/pickmem-extension). Kept visible on
#                          purpose: you have to browse to it in Chrome's
#                          "Load unpacked" picker, so a hidden dir just hides it.
#   PICKMEM_NO_EXTENSION   set to 1 to skip the Chrome extension entirely
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

# ---------- Chrome extension ----------
#
# The release ships the extension pre-built and zipped. Unpack it into a
# stable folder so the user can point Chrome's "Load unpacked" at it. The
# path never changes across upgrades, so re-running this script just refreshes
# the files in place — reload the extension in chrome://extensions afterward.

EXT_DIR="${PICKMEM_EXTENSION_DIR:-$HOME/pickmem-extension}"

if [ "${PICKMEM_NO_EXTENSION:-}" = "1" ]; then
  say ""
  say "Skipping Chrome extension (PICKMEM_NO_EXTENSION=1)."
elif ! command -v unzip >/dev/null 2>&1; then
  say ""
  say "Note: 'unzip' not found; skipping the Chrome extension."
  say "  Install it, or download the zip manually from:"
  say "  https://github.com/$REPO/releases/download/$TAG/pickmem-extension_${TAG}.zip"
else
  EXT_ASSET="pickmem-extension_${TAG}.zip"
  say ""
  # Brace the var: a bare "$EXT_ASSET…" lets some shells/locales fold the
  # multibyte ellipsis bytes into the variable name -> unbound-variable abort.
  say "Downloading ${EXT_ASSET}…"
  if curl -fsSL "$BASE/$EXT_ASSET" -o "$TMP/$EXT_ASSET" 2>/dev/null; then
    # Fresh dir each time so an old build can't leave stale files behind.
    rm -rf "$EXT_DIR"
    mkdir -p "$EXT_DIR"
    unzip -qo "$TMP/$EXT_ASSET" -d "$EXT_DIR"
    say "Chrome extension unpacked to $EXT_DIR"
  else
    say "Note: $TAG has no extension asset; skipping the Chrome extension."
    EXT_DIR=""
  fi
fi

say ""
say "Get started:"
say "  $BINARY init ~/PickMemVault    # create your vault"
say "  $BINARY web ~/PickMemVault     # open the web app (embedded, no build)"
say "  $BINARY --help"

if [ -n "${EXT_DIR:-}" ] && [ -d "$EXT_DIR" ]; then
  say ""
  say "Load the Chrome extension (one-time):"
  say "  1. Open chrome://extensions"
  say "  2. Enable \"Developer mode\" (top-right toggle)"
  say "  3. Click \"Load unpacked\" and select:"
  say "       $EXT_DIR"
fi
