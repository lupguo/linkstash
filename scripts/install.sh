#!/usr/bin/env bash
set -euo pipefail

# LinkStash Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/lupguo/linkstash/main/scripts/install.sh | bash
#   or:  curl -fsSL ... | bash -s -- --version v0.1.0 --dir /usr/local/bin

REPO="lupguo/linkstash"
INSTALL_DIR="${HOME}/.local/bin"
VERSION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version|-v) VERSION="$2"; shift 2 ;;
        --dir|-d)     INSTALL_DIR="$2"; shift 2 ;;
        *)            echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "==> Detected: ${OS}/${ARCH}"

# Get latest version if not specified
if [ -z "$VERSION" ]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version. Specify with --version"
        exit 1
    fi
fi

echo "==> Installing LinkStash ${VERSION}"

# Download URLs
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
SERVER_BIN="linkstash-server-${OS}-${ARCH}"
CLI_BIN="linkstash-${OS}-${ARCH}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download and install
echo "==> Downloading ${SERVER_BIN}..."
curl -fsSL "${BASE_URL}/${SERVER_BIN}" -o "${INSTALL_DIR}/linkstash-server"
chmod +x "${INSTALL_DIR}/linkstash-server"

echo "==> Downloading ${CLI_BIN}..."
curl -fsSL "${BASE_URL}/${CLI_BIN}" -o "${INSTALL_DIR}/linkstash"
chmod +x "${INSTALL_DIR}/linkstash"

# Download example config
echo "==> Downloading example config..."
mkdir -p "${HOME}/.linkstash"
curl -fsSL "https://raw.githubusercontent.com/${REPO}/${VERSION}/conf/app_dev.yaml" -o "${HOME}/.linkstash/config.yaml" 2>/dev/null || true

echo ""
echo "✅ LinkStash ${VERSION} installed successfully!"
echo ""
echo "   Server: ${INSTALL_DIR}/linkstash-server"
echo "   CLI:    ${INSTALL_DIR}/linkstash"
echo "   Config: ${HOME}/.linkstash/config.yaml"
echo ""

# Check if install dir is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo "⚠️  ${INSTALL_DIR} is not in your PATH. Add it:"
    echo ""
    echo "   export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
fi

echo "Quick start:"
echo "   1. Edit ${HOME}/.linkstash/config.yaml (set secret_key, LLM API key)"
echo "   2. linkstash-server -conf ${HOME}/.linkstash/config.yaml"
echo "   3. Visit http://localhost:8080"
