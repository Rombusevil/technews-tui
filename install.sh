#!/bin/sh
set -e

REPO="joeyism/technews-tui"
BINARY="technews-tui"

# Determine OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS="Linux";;
    Darwin*)    OS="Darwin";;
    *)          echo "Unsupported OS: ${OS}" && exit 1;;
esac

# Determine architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) ARCH="x86_64";;
    arm64|aarch64) ARCH="arm64";;
    *)             echo "Unsupported architecture: ${ARCH}" && exit 1;;
esac

echo "Detected ${OS} ${ARCH}"

# Get the latest release download URL
# Use grep/cut to avoid dependency on jq
LATEST_RELEASE_JSON=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest")
LATEST_URL=$(echo "$LATEST_RELEASE_JSON" | grep "browser_download_url.*${OS}_${ARCH}\.tar\.gz" | head -n 1 | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
    echo "Could not find a release for ${OS} ${ARCH}"
    exit 1
fi

echo "Downloading from $LATEST_URL..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download and extract
curl -sL "$LATEST_URL" | tar -xz -C "$TMP_DIR"

# Install location
INSTALL_DIR="/usr/local/bin"

# If /usr/local/bin is not writable, try ~/.local/bin
if [ ! -w "$INSTALL_DIR" ]; then
    # check if ~/.local/bin exists and is in PATH
    LOCAL_BIN="$HOME/.local/bin"
    if [ -d "$LOCAL_BIN" ] && echo "$PATH" | grep -q "$LOCAL_BIN"; then
        INSTALL_DIR="$LOCAL_BIN"
    fi
fi

if [ ! -w "$INSTALL_DIR" ]; then
    echo "Installing to $INSTALL_DIR requires sudo privileges"
    sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
else
    mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "Successfully installed $BINARY to $INSTALL_DIR"
echo "Run '$BINARY' to start."
