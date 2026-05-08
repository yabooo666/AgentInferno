#!/bin/bash
# AgentInferno Automated Setup Script for Ubuntu/Debian

set -e

# ==========================================
# CONFIGURATION
# Replace this with the URL where you are hosting the files!
HOST_URL="place url here where is hosted agentinferno and agentinfro.service"
# ==========================================

echo "[*] Starting AgentInferno Setup..."

# 1. Check for root
if [ "$EUID" -ne 0 ]; then
  echo "[-] Error: Please run this script as root (e.g. sudo bash setup.sh)"
  exit 1
fi

# 2. Stop existing service if this is an update
if systemctl is-active --quiet agentinferno.service 2>/dev/null; then
    echo "[*] Stopping existing AgentInferno service for update..."
    systemctl stop agentinferno.service
fi

# 3. Download Binary
echo "[*] Downloading AgentInferno binary..."
curl -sSLf -o /tmp/agentinferno "${HOST_URL}/agentinferno" || { echo "[-] Failed to download binary. Please check the URL."; exit 1; }
mv -f /tmp/agentinferno /usr/bin/agentinferno
chmod +x /usr/bin/agentinferno

# 4. Download Systemd Service
echo "[*] Downloading systemd service file..."
curl -sSLf -o /tmp/agentinferno.service "${HOST_URL}/agentinferno.service" || { echo "[-] Failed to download service file."; exit 1; }
mv -f /tmp/agentinferno.service /etc/systemd/system/agentinferno.service

# 4. Create dedicated restricted user
echo "[*] Creating restricted 'agentinferno' user..."
if ! id "agentinferno" &>/dev/null; then
    useradd -r -s /bin/false agentinferno
fi

# 5. Add to docker group if Docker is installed
if getent group docker > /dev/null 2>&1; then
    echo "[*] Docker detected, adding agentinferno to docker group..."
    usermod -aG docker agentinferno
fi

# 6. Setup Data Directory
echo "[*] Setting up data directories..."
mkdir -p /var/lib/agentinferno
chown agentinferno:agentinferno /var/lib/agentinferno
chmod 700 /var/lib/agentinferno

# 7. Configure Sudoers for Reboot
echo "[*] Configuring sudoers to allow reboot..."
echo "agentinferno ALL=(root) NOPASSWD: /sbin/reboot" > /etc/sudoers.d/agentinferno
chmod 440 /etc/sudoers.d/agentinferno

# 8. Start and Enable Service
echo "[*] Starting AgentInferno service..."
systemctl daemon-reload
systemctl enable agentinferno.service
systemctl restart agentinferno.service

# 9. Verify Status
sleep 2
if systemctl is-active --quiet agentinferno.service; then
    echo "[+] Setup Complete! AgentInferno is running and secured."
    echo "[i] View logs with: sudo journalctl -u agentinferno -f"
else
    echo "[-] Error: Service failed to start. Check logs with: sudo journalctl -u agentinferno -e"
    exit 1
fi
