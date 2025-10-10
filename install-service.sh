#!/bin/bash
# Ollama OpenAI Proxy - Service Installation Script
# This script installs the proxy as a systemd service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
INSTALL_DIR="/opt/ollama-openai-proxy"
SERVICE_USER="ollama-proxy"
CONFIG_FILE="production.yaml.example"

echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Ollama OpenAI Proxy Service Installer ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}Error: This script must be run as root${NC}"
   echo "Please run: sudo $0"
   exit 1
fi

# Check if systemd is available
if ! command -v systemctl &> /dev/null; then
    echo -e "${RED}Error: systemd is not available on this system${NC}"
    exit 1
fi

echo -e "${YELLOW}Installation Directory: ${INSTALL_DIR}${NC}"
echo -e "${YELLOW}Service User: ${SERVICE_USER}${NC}"
echo ""

# Ask for confirmation
read -p "Continue with installation? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Installation cancelled."
    exit 0
fi

# Step 1: Create service user
echo ""
echo -e "${GREEN}[1/7] Creating service user...${NC}"
if id "$SERVICE_USER" &>/dev/null; then
    echo "User $SERVICE_USER already exists"
else
    useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
    echo "User $SERVICE_USER created"
fi

# Step 2: Create installation directory
echo ""
echo -e "${GREEN}[2/7] Creating installation directory...${NC}"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/logs"
echo "Directories created"

# Step 3: Copy files
echo ""
echo -e "${GREEN}[3/7] Copying application files...${NC}"
cp -r bin "$INSTALL_DIR/"
cp -r configs "$INSTALL_DIR/"
cp -r web "$INSTALL_DIR/"
echo "Files copied"

# Step 4: Create configuration
echo ""
echo -e "${GREEN}[4/7] Setting up configuration...${NC}"
if [ ! -f "$INSTALL_DIR/configs/config.yaml" ]; then
    cp "$INSTALL_DIR/configs/$CONFIG_FILE" "$INSTALL_DIR/configs/config.yaml"
    echo "Configuration file created from template"
    echo -e "${YELLOW}⚠️  IMPORTANT: Edit $INSTALL_DIR/configs/config.yaml and change:${NC}"
    echo "   - auth.admin_key"
    echo "   - auth.jwt.secret"
    echo "   - database settings"
else
    echo "Configuration file already exists, skipping"
fi

# Step 5: Set permissions
echo ""
echo -e "${GREEN}[5/7] Setting permissions...${NC}"
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR/bin/server"
chmod 600 "$INSTALL_DIR/configs/config.yaml"
chmod 755 "$INSTALL_DIR/data"
chmod 755 "$INSTALL_DIR/logs"
echo "Permissions set"

# Step 6: Install systemd service
echo ""
echo -e "${GREEN}[6/7] Installing systemd service...${NC}"
cp ollama-openai-proxy.service /etc/systemd/system/
systemctl daemon-reload
echo "Service installed"

# Step 7: Enable and start service
echo ""
echo -e "${GREEN}[7/7] Enabling service...${NC}"
systemctl enable ollama-openai-proxy

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║      Installation Complete! ✓          ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}⚠️  IMPORTANT NEXT STEPS:${NC}"
echo ""
echo "1. Edit configuration:"
echo -e "   ${GREEN}nano $INSTALL_DIR/configs/config.yaml${NC}"
echo ""
echo "2. Set secure admin key and JWT secret"
echo ""
echo "3. Start the service:"
echo -e "   ${GREEN}systemctl start ollama-openai-proxy${NC}"
echo ""
echo "4. Check status:"
echo -e "   ${GREEN}systemctl status ollama-openai-proxy${NC}"
echo ""
echo "5. View logs:"
echo -e "   ${GREEN}journalctl -u ollama-openai-proxy -f${NC}"
echo ""
echo "6. Access WebUI:"
echo -e "   ${GREEN}http://localhost:8080${NC}"
echo ""
echo -e "${YELLOW}📚 For more information, see: SYSTEMD_INSTALL.md${NC}"
echo ""

