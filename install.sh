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

INSTALL_PATH="${INSTALL_DIR}/bwenv"

echo "Installing bwenv for ${OS}-${ARCH}..."

# Ensure dependencies are available
for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: $cmd is required but not installed" >&2
    exit 1
  fi
done

# Get latest release info
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
RELEASE_DATA=$(curl -sf "$LATEST_URL")
if [ -z "$RELEASE_DATA" ]; then
  echo "Error: Failed to fetch latest release info" >&2
  exit 1
fi

# Extract download URL and tag for checksum lookup
DOWNLOAD_URL=$(echo "$RELEASE_DATA" | jq -r ".assets[] | select(.name == \"$BINARY_NAME\") | .browser_download_url")
TAG=$(echo "$RELEASE_DATA" | jq -r ".tag_name")

if [ -z "$DOWNLOAD_URL" ] || [ "$DOWNLOAD_URL" = "null" ]; then
  echo "Error: Could not find binary for ${OS}-${ARCH}" >&2
  exit 1
fi

# Determine checksums URL
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

# Create a temp directory for downloads
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

BINARY_PATH="${TMP_DIR}/${BINARY_NAME}"
CHECKSUMS_PATH="${TMP_DIR}/checksums.txt"

# Download binary and checksums
echo "Downloading from: $DOWNLOAD_URL"
curl -sfL -o "$BINARY_PATH" "$DOWNLOAD_URL"

echo "Verifying checksum..."
if curl -sfL -o "$CHECKSUMS_PATH" "$CHECKSUMS_URL"; then
  EXPECTED_HASH=$(grep "$BINARY_NAME" "$CHECKSUMS_PATH" | awk '{print $1}')
  if [ -n "$EXPECTED_HASH" ]; then
    COMPUTED_HASH=$(sha256sum "$BINARY_PATH" | awk '{print $1}')
    if [ "$EXPECTED_HASH" != "$COMPUTED_HASH" ]; then
      echo "Error: Checksum mismatch" >&2
      echo "  expected: $EXPECTED_HASH" >&2
      echo "  got:      $COMPUTED_HASH" >&2
      exit 1
    fi
    echo "Checksum verified: $EXPECTED_HASH"
  else
    echo "Warning: No checksum found for $BINARY_NAME, skipping verification" >&2
  fi
else
  echo "Warning: Could not fetch checksums, skipping verification" >&2
fi

# Check write permission and install
if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo >/dev/null 2>&1; then
    echo "Escalating to install to $INSTALL_DIR..."
    sudo cp "$BINARY_PATH" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
  else
    echo "Error: No write permission to $INSTALL_DIR. Try: sudo $0" >&2
    exit 1
  fi
else
  cp "$BINARY_PATH" "$INSTALL_PATH"
  chmod +x "$INSTALL_PATH"
fi

echo "Installed to: $INSTALL_PATH"
echo "Run 'bwenv --help' to get started."
