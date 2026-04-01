#!/bin/bash
set -e

GITLAB_URL="https://gitlab.alexue4.dev"
PROJECT_ID="146"
PACKAGE_NAME="aigateway-agent"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           AIGateway Agent Installer                          ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Get latest version from GitLab Package Registry
echo "→ Finding latest version..."
LATEST_VERSION=$(curl -fsSL "${GITLAB_URL}/api/v4/projects/${PROJECT_ID}/packages?package_name=${PACKAGE_NAME}&order_by=created_at&sort=desc&per_page=10" | grep -o '"version":"[^"]*"' | cut -d'"' -f4 | grep -v "^latest$" | head -1)

if [ -z "$LATEST_VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

echo "   Latest version: ${LATEST_VERSION}"
echo ""

# Find RPM filename from package files
echo "→ Finding RPM package..."
PACKAGE_ID=$(curl -fsSL "${GITLAB_URL}/api/v4/projects/${PROJECT_ID}/packages?package_name=${PACKAGE_NAME}&package_version=${LATEST_VERSION}" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

if [ -n "$PACKAGE_ID" ]; then
    RPM_NAME=$(curl -fsSL "${GITLAB_URL}/api/v4/projects/${PROJECT_ID}/packages/${PACKAGE_ID}/package_files" | grep -o '"file_name":"[^"]*\.rpm"' | head -1 | cut -d'"' -f4)
fi

if [ -z "$RPM_NAME" ]; then
    # Fallback: try common patterns
    for suffix in "el9.x86_64.rpm" "x86_64.rpm"; do
        TEST_NAME="${PACKAGE_NAME}-${LATEST_VERSION}.${suffix}"
        if curl -fsSL --head "${GITLAB_URL}/api/v4/projects/${PROJECT_ID}/packages/generic/${PACKAGE_NAME}/${LATEST_VERSION}/${TEST_NAME}" 2>/dev/null | grep -q "200"; then
            RPM_NAME="$TEST_NAME"
            break
        fi
    done
fi

if [ -z "$RPM_NAME" ]; then
    echo "Error: Could not find RPM package"
    exit 1
fi

RPM_URL="${GITLAB_URL}/api/v4/projects/${PROJECT_ID}/packages/generic/${PACKAGE_NAME}/${LATEST_VERSION}/${RPM_NAME}"
echo "   Package: ${RPM_NAME}"
echo ""

# Backup existing config if upgrading
if [ -f /opt/aigateway-agent/configs/agent.yaml ]; then
    BACKUP="/opt/aigateway-agent/configs/agent.yaml.bak.$(date +%s)"
    echo "→ Backing up existing config to ${BACKUP}..."
    sudo cp /opt/aigateway-agent/configs/agent.yaml "${BACKUP}"
fi

# Download RPM
echo "→ Downloading..."
curl -fsSL "${RPM_URL}" -o "/tmp/${RPM_NAME}"

# Install (use rpm directly — -U handles both install and upgrade)
echo "→ Installing..."
sudo rpm -Uvh --nodeps "/tmp/${RPM_NAME}"

# Cleanup
rm -f "/tmp/${RPM_NAME}"

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    Installation complete!                    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Check if config exists
if [ ! -f /opt/aigateway-agent/configs/agent.yaml ]; then
    echo "→ Next steps:"
    echo ""
    echo "  1. Generate config in AIGateway Admin Panel:"
    echo "     Admin → Workers → Add Worker → generate config"
    echo ""
    echo "  2. Copy generated agent.yaml to this server:"
    echo "     sudo cp agent.yaml /opt/aigateway-agent/configs/"
    echo ""
    echo "  3. Start the agent:"
    echo "     sudo systemctl start aigateway-agent"
    echo ""
else
    echo "→ Agent status:"
    echo ""
    sudo systemctl status aigateway-agent --no-pager 2>/dev/null || echo "  Service not running"
fi

echo ""
echo "Commands:"
echo "  Start:   sudo systemctl enable --now aigateway-agent"
echo "  Status:  sudo systemctl status aigateway-agent"
echo "  Logs:    journalctl -u aigateway-agent -f"
echo "  Config:  /opt/aigateway-agent/configs/agent.yaml"
echo ""
