#!/usr/bin/env bash
# WARPture Linux installer (Debian/Ubuntu)
set -euo pipefail

VERSION="1.0.0"
CONFIG_DIR="$HOME/.config/warp-gui"
LOG_DIR="$HOME/.local/share/warp-gui/logs"
AUTOSTART_DIR="$HOME/.config/autostart"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"

echo "╔═══════════════════════════════════╗"
echo "║  WARPture v${VERSION} – Linux Installer  ║"
echo "╚═══════════════════════════════════╝"
echo ""

if ! command -v warp-cli &>/dev/null; then
    echo "⚠️  warp-cli not found. Simulation mode will be used."
    echo "   See: https://developers.cloudflare.com/warp-client/get-started/linux/"
fi

mkdir -p "$CONFIG_DIR" "$LOG_DIR" "$AUTOSTART_DIR" "$SYSTEMD_USER_DIR"

if [[ ! -f "$CONFIG_DIR/split-tunnel.json" ]]; then
    cat > "$CONFIG_DIR/split-tunnel.json" << 'JSON'
{
  "version": 1,
  "defaultPolicy": "warp",
  "includedApps": [],
  "excludedApps": []
}
JSON
    echo "✓ Created default config"
fi

# Write systemd user service for tunnel-agent
cat > "$SYSTEMD_USER_DIR/warpture-tunnel-agent.service" << SVCEOF
[Unit]
Description=WARPture Tunnel Agent
After=network.target

[Service]
Type=simple
ExecStart=%h/.local/bin/warpture-tunnel-agent
Restart=on-failure
RestartSec=5
Environment=CONFIG_PATH=%h/.config/warp-gui/split-tunnel.json
Environment=HTTP_ADDR=:8080
Environment=WS_ADDR=:8081
StandardOutput=append:%h/.local/share/warp-gui/logs/tunnel-agent.log
StandardError=append:%h/.local/share/warp-gui/logs/tunnel-agent-error.log

[Install]
WantedBy=default.target
SVCEOF

systemctl --user daemon-reload 2>/dev/null || true
systemctl --user enable warpture-tunnel-agent.service 2>/dev/null || true
systemctl --user start warpture-tunnel-agent.service 2>/dev/null || true
echo "✓ Registered systemd user service: warpture-tunnel-agent"

# Write autostart .desktop for GUI
cat > "$AUTOSTART_DIR/warpture.desktop" << DESKEOF
[Desktop Entry]
Type=Application
Name=WARPture
Exec=warpture --hidden
Icon=warpture
Comment=Cloudflare WARP GUI Manager
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
DESKEOF

echo "✓ Registered autostart entry"
echo ""
echo "✅ WARPture installed! Run 'warpture' or find it in your app launcher."
