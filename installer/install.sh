#!/bin/bash

set -e

echo "Installing Nimbus CLI..."

REPO="infraspecdev/faas-engine-go"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Check dependency
if ! command -v curl >/dev/null 2>&1; then
  echo "Error: curl is required"
  exit 1
fi

# Get version (allow override)
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "Fetching latest version..."
  VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
fi

if [ -z "$VERSION" ]; then
  echo "Error: Failed to fetch version"
  exit 1
fi

echo "Version: $VERSION"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)
    case "$ARCH" in
      x86_64) FILE="nimbus-linux-amd64-$VERSION" ;;
      aarch64) FILE="nimbus-linux-arm64-$VERSION" ;;
      *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    ;;
  Darwin)
    case "$ARCH" in
      x86_64) FILE="nimbus-darwin-amd64-$VERSION" ;;
      arm64) FILE="nimbus-darwin-arm64-$VERSION" ;;
      *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    ;;
  *)
    echo "Error: Unsupported OS: $OS"
    exit 1
    ;;
esac

URL="https://github.com/$REPO/releases/download/$VERSION/$FILE"
TMPFILE=$(mktemp)

trap "rm -f $TMPFILE" EXIT

echo "Downloading binary..."
curl -fL --progress-bar -o "$TMPFILE" "$URL"

chmod +x "$TMPFILE"

if [ ! -d "$INSTALL_DIR" ]; then
  echo "Error: Install directory $INSTALL_DIR does not exist"
  exit 1
fi

echo "Installing to $INSTALL_DIR..."

if [ "$EUID" -ne 0 ]; then
  sudo mv "$TMPFILE" "$INSTALL_DIR/nimbus"
else
  mv "$TMPFILE" "$INSTALL_DIR/nimbus"
fi

echo "Installation complete"
echo "Run: nimbus --help"

# PATH warning
if ! command -v nimbus >/dev/null 2>&1; then
  echo "Warning: $INSTALL_DIR is not in your PATH"
  echo "Add it using:"
  echo "  export PATH=\$PATH:$INSTALL_DIR"
fi