#!/usr/bin/env bash
set -euo pipefail

# LinkStash Server Installer (Full Deployment)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/lupguo/linkstash/main/INSTALL.sh | sudo bash
#   curl -fsSL ... | sudo bash -s -- --version v0.2.1
#
# Installs LinkStash to /opt/linkstash with systemd service.
# After install, edit /opt/linkstash/.env and run: systemctl start linkstash

REPO="lupguo/linkstash"
INSTALL_DIR="/opt/linkstash"
SERVICE_USER="linkstash"
SERVICE_PORT="8085"
RELEASE_VERSION=""

# --- Helpers ---

info()  { echo -e "\033[32m==>\033[0m $1"; }
warn()  { echo -e "\033[33m==>\033[0m $1"; }
error() { echo -e "\033[31m==>\033[0m $1" >&2; exit 1; }

# --- Parse arguments ---

while [[ $# -gt 0 ]]; do
    case $1 in
        --version|-v) RELEASE_VERSION="$2"; shift 2 ;;
        *)            error "Unknown option: $1" ;;
    esac
done

# ============================================================
# [1/6] Environment detection
# ============================================================

info "[1/6] Detecting environment..."

if [[ $EUID -ne 0 ]]; then
    error "This script must be run as root. Use: curl ... | sudo bash"
fi

# Detect OS
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    info "  OS: ${PRETTY_NAME:-unknown}"
else
    warn "  Cannot detect OS, proceeding anyway."
fi

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             error "Unsupported architecture: $ARCH" ;;
esac
info "  Arch: $ARCH"

# Get version
if [[ -z "$RELEASE_VERSION" ]]; then
    info "  Querying latest version..."
    RELEASE_VERSION=$(curl --http1.1 -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$RELEASE_VERSION" ]]; then
        error "Could not determine latest version. Use --version to specify."
    fi
fi
info "  Version: $RELEASE_VERSION"

# Check if upgrading
UPGRADE=false
if [[ -f "${INSTALL_DIR}/bin/linkstash-server" ]]; then
    warn "  Existing installation detected. Upgrading to ${RELEASE_VERSION}."
    UPGRADE=true
    # Stop service before overwriting binaries (avoids "Failure writing" errors)
    if systemctl is-active --quiet linkstash 2>/dev/null; then
        info "  Stopping linkstash service for upgrade..."
        systemctl stop linkstash
    fi
fi

# ============================================================
# [2/6] Create user and directories
# ============================================================

info "[2/6] Creating user and directories..."

if id "$SERVICE_USER" &>/dev/null; then
    info "  User '${SERVICE_USER}' already exists, skipping."
else
    useradd -r -s /sbin/nologin "$SERVICE_USER"
    info "  Created user '${SERVICE_USER}'."
fi

mkdir -p "${INSTALL_DIR}"/{bin,conf,data,logs,web}
info "  Directories ready: ${INSTALL_DIR}/{bin,conf,data,logs,web}"

# ============================================================
# [3/6] Download binaries + web resources
# ============================================================

info "[3/6] Downloading LinkStash ${RELEASE_VERSION}..."

BASE_URL="https://github.com/${REPO}/releases/download/${RELEASE_VERSION}"

info "  Downloading linkstash-server..."
rm -f "${INSTALL_DIR}/bin/linkstash-server"
curl --http1.1 -fsSL "${BASE_URL}/linkstash-server-linux-${ARCH}" -o "${INSTALL_DIR}/bin/linkstash-server"
chmod +x "${INSTALL_DIR}/bin/linkstash-server"

info "  Downloading linkstash CLI..."
rm -f "${INSTALL_DIR}/bin/linkstash"
curl --http1.1 -fsSL "${BASE_URL}/linkstash-linux-${ARCH}" -o "${INSTALL_DIR}/bin/linkstash"
chmod +x "${INSTALL_DIR}/bin/linkstash"

info "  Downloading web resources..."
curl --http1.1 -fsSL "${BASE_URL}/web.tar.gz" -o /tmp/linkstash-web.tar.gz
tar -xzf /tmp/linkstash-web.tar.gz -C "${INSTALL_DIR}/web/"
rm -f /tmp/linkstash-web.tar.gz

info "  Binaries and web resources installed."

# ============================================================
# [4/6] Generate configuration
# ============================================================

info "[4/6] Generating configuration..."

SECRET_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)

if [[ -f "${INSTALL_DIR}/conf/app_prod.yaml" ]]; then
    warn "  Config already exists, skipping: ${INSTALL_DIR}/conf/app_prod.yaml"
else
    # Download example config from repo as base template
    RAW_URL="https://raw.githubusercontent.com/${REPO}/${RELEASE_VERSION}/conf/app_example.yaml"
    info "  Downloading config template..."
    if curl --http1.1 -fsSL "${RAW_URL}" -o "${INSTALL_DIR}/conf/app_prod.yaml"; then
        # Patch: bind to localhost only, use prod port, inject random secrets, set log level
        sed -i \
            -e "s|host: 0.0.0.0|host: 127.0.0.1|" \
            -e "s|port: 8080|port: ${SERVICE_PORT}|" \
            -e "s|secret_key: \"passwd\"|secret_key: \"${SECRET_KEY}\"|" \
            -e "s|jwt_secret: \"linkstash-jwt-secret-key-change-in-production\"|jwt_secret: \"${JWT_SECRET}\"|" \
            -e "s|level: \"debug\"|level: \"info\"|" \
            "${INSTALL_DIR}/conf/app_prod.yaml"
        info "  Created config from template with auto-generated secrets."
    else
        warn "  Failed to download config template. Creating minimal config."
        cat > "${INSTALL_DIR}/conf/app_prod.yaml" <<YAML
server:
  host: 127.0.0.1
  port: ${SERVICE_PORT}
auth:
  secret_key: "${SECRET_KEY}"
  jwt_secret: "${JWT_SECRET}"
  jwt_expire_hours: 72
database:
  path: "./data/linkstash.db"
log:
  level: "info"
  file: "./logs/app.log"
  format: "text"
YAML
        info "  Created minimal config. See conf/app_example.yaml for full options."
    fi
fi

if [[ -f "${INSTALL_DIR}/.env" ]]; then
    warn "  Env file already exists, skipping: ${INSTALL_DIR}/.env"
else
    cat > "${INSTALL_DIR}/.env" <<'ENV'
# LinkStash Environment Variables
# Edit this file, then run: systemctl start linkstash

# LLM API Key (required for URL auto-analysis)
OPENROUTER_API_KEY=sk-or-REPLACE-WITH-YOUR-KEY

# Alternative: OpenAI (uncomment and set, also update conf/app_prod.yaml)
# OPENAI_API_KEY=sk-REPLACE-WITH-YOUR-KEY
ENV
    info "  Created env file: ${INSTALL_DIR}/.env"
fi

chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"
chmod 600 "${INSTALL_DIR}/.env"

# ============================================================
# [5/6] Install Chromium
# ============================================================

info "[5/6] Installing Chromium (headless browser for URL analysis)..."

if command -v chromium-browser &>/dev/null || command -v chromium &>/dev/null; then
    CHROMIUM_PATH=$(command -v chromium-browser 2>/dev/null || command -v chromium 2>/dev/null)
    info "  Chromium already installed: ${CHROMIUM_PATH}"
else
    if command -v dnf &>/dev/null; then
        dnf install -y chromium &>/dev/null && info "  Chromium installed via dnf." || {
            dnf install -y epel-release &>/dev/null
            dnf install -y chromium &>/dev/null && info "  Chromium installed via EPEL." || {
                warn "  Could not install Chromium. URL analysis will fall back to HTTP fetching."
                warn "  Install manually: sudo dnf install chromium"
            }
        }
    elif command -v apt-get &>/dev/null; then
        apt-get update -qq &>/dev/null
        apt-get install -y -qq chromium-browser &>/dev/null && info "  Chromium installed via apt." || {
            warn "  Could not install Chromium. Install manually: sudo apt install chromium-browser"
        }
    else
        warn "  No supported package manager found. Install Chromium manually."
    fi
fi

# ============================================================
# [6/6] Configure Systemd
# ============================================================

info "[6/6] Configuring systemd service..."

cat > /etc/systemd/system/linkstash.service <<SERVICE
[Unit]
Description=LinkStash - Bookmark Management Service
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/linkstash-server -conf ${INSTALL_DIR}/conf/app_prod.yaml
EnvironmentFile=${INSTALL_DIR}/.env
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}/data ${INSTALL_DIR}/logs

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable linkstash &>/dev/null
info "  Systemd service configured and enabled."

# ============================================================
# Done
# ============================================================

echo ""
echo "=============================================="
echo "  ✅ LinkStash ${RELEASE_VERSION} installed!"
echo "=============================================="
echo ""
echo "  Install dir:  ${INSTALL_DIR}"
echo "  Config:       ${INSTALL_DIR}/conf/app_prod.yaml"
echo "  Env file:     ${INSTALL_DIR}/.env  ← edit this!"
echo "  Service port: 127.0.0.1:${SERVICE_PORT}"
echo ""
echo "  Next steps:"
echo "    1. vim ${INSTALL_DIR}/.env                    # fill in OPENROUTER_API_KEY"
echo "    2. systemctl start linkstash                  # start the service"
echo "    3. curl -s http://127.0.0.1:${SERVICE_PORT}/health   # verify"
echo ""
if [[ "$UPGRADE" == "true" ]]; then
    echo "  Upgrade note: config and env files were preserved."
    echo "  Restarting service..."
    systemctl restart linkstash
    echo "  Service restarted. Run: systemctl status linkstash"
    echo ""
fi
