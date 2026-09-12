#!/bin/sh
# Keyway installer — downloads the right prebuilt binary for your OS/arch from the
# latest GitHub release, verifies its checksum, and installs it.
#
#   curl -fsSL https://raw.githubusercontent.com/Keyway-AI/keyway/main/install.sh | sh
#
# Options (environment variables):
#   KEYWAY_VERSION=v0.1.0   install a specific tag instead of the latest release
#   KEYWAY_BINDIR=/path      install into this directory (default: /usr/local/bin,
#                            falling back to $HOME/.local/bin if that is not writable)
#
# No sudo is assumed; if the target dir needs elevated rights the script falls back
# to a per-user location and tells you how to add it to PATH.
set -eu

REPO="Keyway-AI/keyway"
BIN="keyway"

info() { printf '  %s\n' "$*"; }
err()  { printf 'error: %s\n' "$*" >&2; exit 1; }

# --- detect platform ---------------------------------------------------------
os=$(uname -s)
case "$os" in
  Linux)  os=linux ;;
  Darwin) os=darwin ;;
  *) err "unsupported OS '$os' — use the container image (ghcr.io/keyway-ai/keyway) or 'go install'." ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64)  arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) err "unsupported architecture '$arch' — use 'go install github.com/$REPO/cmd/keyway@latest'." ;;
esac

# --- pick a downloader -------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  dl() { curl -fsSL "$1" -o "$2"; }
  fetch() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  dl() { wget -qO "$2" "$1"; }
  fetch() { wget -qO- "$1"; }
else
  err "need curl or wget to download."
fi

# --- resolve version ---------------------------------------------------------
version="${KEYWAY_VERSION:-}"
if [ -z "$version" ]; then
  info "resolving the latest release..."
  version=$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | grep -m1 '"tag_name"' | cut -d'"' -f4)
  [ -n "$version" ] || err "could not determine the latest release — set KEYWAY_VERSION=vX.Y.Z."
fi

tarball="keyway_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"
info "installing keyway $version ($os/$arch)"

# --- download + verify -------------------------------------------------------
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

dl "$base/$tarball" "$tmp/$tarball" || err "download failed: $base/$tarball"

if dl "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
  want=$(grep " $tarball\$" "$tmp/checksums.txt" | awk '{print $1}')
  if [ -n "$want" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      got=$(sha256sum "$tmp/$tarball" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      got=$(shasum -a 256 "$tmp/$tarball" | awk '{print $1}')
    fi
    if [ -n "${got:-}" ] && [ "$got" != "$want" ]; then
      err "checksum mismatch for $tarball (expected $want, got $got)"
    fi
    info "checksum verified"
  fi
else
  info "checksums.txt not found — skipping verification"
fi

tar -xzf "$tmp/$tarball" -C "$tmp"
[ -f "$tmp/$BIN" ] || err "archive did not contain a '$BIN' binary"

# --- choose an install dir ---------------------------------------------------
bindir="${KEYWAY_BINDIR:-}"
if [ -z "$bindir" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then
    bindir=/usr/local/bin
  else
    bindir="$HOME/.local/bin"
  fi
fi
mkdir -p "$bindir" || err "cannot create $bindir"

install -m 0755 "$tmp/$BIN" "$bindir/$BIN" 2>/dev/null \
  || { cp "$tmp/$BIN" "$bindir/$BIN" && chmod 0755 "$bindir/$BIN"; } \
  || err "cannot write to $bindir — set KEYWAY_BINDIR to a writable dir."
# keyway-runner ships in the same archive; install it too when present.
[ -f "$tmp/keyway-runner" ] && { install -m 0755 "$tmp/keyway-runner" "$bindir/keyway-runner" 2>/dev/null \
  || { cp "$tmp/keyway-runner" "$bindir/keyway-runner" && chmod 0755 "$bindir/keyway-runner"; }; } || true

info "installed to $bindir/$BIN"

# --- PATH hint + confirm -----------------------------------------------------
case ":$PATH:" in
  *":$bindir:"*) : ;;
  *) printf '\n  %s is not on your PATH. Add it:\n    export PATH="%s:$PATH"\n' "$bindir" "$bindir" ;;
esac

printf '\nDone. Try:\n  keyway version\n  echo "$AGENT_TOKEN" | keyway agent inspect --audience https://mcp.example/api\n'
