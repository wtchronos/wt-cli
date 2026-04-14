#!/bin/sh
# wt-cli installer — detects OS/arch, downloads from GitHub Releases, installs to /usr/local/bin
set -e

REPO="wtchronos/wt-cli"
INSTALL_DIR="${WT_INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)              echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi

FILENAME="wt_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

echo "Installing wt v${VERSION} (${OS}/${ARCH})..."

# Download and extract
TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

curl -fsSL "$URL" -o "${TMP}/${FILENAME}"
tar -xzf "${TMP}/${FILENAME}" -C "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/wt" "${INSTALL_DIR}/wt"
else
  sudo mv "${TMP}/wt" "${INSTALL_DIR}/wt"
fi

chmod +x "${INSTALL_DIR}/wt"

echo "Installed wt v${VERSION} to ${INSTALL_DIR}/wt"
echo ""
echo "Add shell integration to your rc file:"
echo '  eval "$(wt shell init zsh)"   # for zsh'
echo '  eval "$(wt shell init bash)"  # for bash'
