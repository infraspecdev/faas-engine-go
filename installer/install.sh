#!/bin/bash

set -e

echo "Installing Nimbus CLI..."

REPO="infraspecdev/faas-engine-go"

VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f 4)

OS="$(uname -s)"

if [ "$OS" = "Linux" ]; then
  FILE="nimbus-linux-amd64-$VERSION"
elif [ "$OS" = "Darwin" ]; then
  FILE="nimbus-darwin-amd64-$VERSION"
else
  echo "Unsupported OS"
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$VERSION/$FILE"

echo "⬇️ Downloading $FILE..."
curl -L -o nimbus $URL

chmod +x nimbus
sudo mv nimbus /usr/local/bin/nimbus

echo "✅ Installed! Run: nimbus"