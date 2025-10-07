#!/bin/bash
set -e

# bwenv installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/saiteja-madha/bwenv/main/install.sh | bash

REPO="saiteja-madha/bwenv"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Map OS names  
case $OS in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

BINARY_NAME="bwenv-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BINARY_NAME="${BINARY_NAME}.exe"
fi

echo "Installing bwenv for ${OS}-${ARCH}..."

# Get latest release info
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
DOWNLOAD_URL=$(curl -s "$LATEST_URL" | grep "browser_download_url.*${BINARY_NAME}" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Could not find binary for ${OS}-${ARCH}" >&2
  exit 1
fi

# Download binary
INSTALL_PATH="${INSTALL_DIR}/bwenv"
echo "Downloading from: $DOWNLOAD_URL"
echo "Installing to: $INSTALL_PATH"

curl -L -o "$INSTALL_PATH" "$DOWNLOAD_URL"
chmod +x "$INSTALL_PATH"

echo "✅ bwenv installed successfully!"
echo "Run 'bwenv --help' to get started."