#!/bin/sh
# ntzh installer — downloads the latest release archive for your OS/arch.
# Usage: curl -fsSL https://raw.githubusercontent.com/Nortezh/cli/main/install.sh | sh
#   Override version: VERSION=v0.5.0 sh install.sh
#   Override target dir: BINDIR=$HOME/.local/bin sh install.sh
set -e

REPO="Nortezh/cli"
BINDIR="${BINDIR:-/usr/local/bin}"

os=$(uname -s)
arch=$(uname -m)

case "$os" in
  Linux) OS="Linux" ;;
  Darwin) OS="Darwin" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64 | amd64) ARCH="x86_64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
    grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
fi
if [ -z "$VERSION" ]; then
  echo "could not determine latest version" >&2; exit 1
fi

# Strip leading 'v' for the archive name (matches .goreleaser name_template).
ver_no_v="${VERSION#v}"
archive="ntzh_${ver_no_v}_${OS}_${ARCH}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"

echo "Downloading ntzh ${VERSION} (${OS}/${ARCH})..."
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$url" -o "$tmp/$archive"
tar -xzf "$tmp/$archive" -C "$tmp"

if [ -w "$BINDIR" ]; then
  mv "$tmp/ntzh" "$BINDIR/ntzh"
else
  echo "Installing to ${BINDIR} (requires sudo)..."
  sudo mv "$tmp/ntzh" "$BINDIR/ntzh"
fi
chmod +x "$BINDIR/ntzh"

echo "Installed ntzh to ${BINDIR}/ntzh"
"$BINDIR/ntzh" --version
