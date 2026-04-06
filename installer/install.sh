#!/bin/bash

set -e

echo "Installing Nimbus CLI..."

REPO="infraspecdev/faas-engine-go"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Check dependencies
for cmd in curl; do
  if ! command -v $cmd >/dev/null 2>&1; then
    echo "Error: $cmd is required"
    exit 1
  fi
done

# Get version (allow override via VERSION env var)
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "Fetching latest version..."
  VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep -o '"tag_name":"[^"]*' | cut -d'"' -f4)
fi

if [ -z "$VERSION" ]; then
  echo "Error: Failed to fetch version"
  exit 1
fi

echo "Version: $VERSION"

# Detect OS and Architecture
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

# Download with cleanup on error
trap "rm -f $TMPFILE" EXIT

echo "Downloading binary from $URL..."
if ! curl -fL --progress-bar -o "$TMPFILE" "$URL"; then
  echo "Error: Failed to download"
  exit 1
fi

chmod +x "$TMPFILE"

# Verify directory and permissions
if [ ! -d "$INSTALL_DIR" ]; then
  echo "Error: Install directory $INSTALL_DIR does not exist"
  exit 1
fi

# Install
echo "Installing to $INSTALL_DIR..."

if ! sudo mv "$TMPFILE" "$INSTALL_DIR/nimbus" 2>/dev/null; then
  if [ "$EUID" -ne 0 ]; then
    echo "Error: Insufficient permissions. Try running with sudo or as root."
    exit 1
  fi
  echo "Error: Failed to move binary"
  exit 1
fi

echo "✓ Installation complete"
echo "Run: nimbus --help"