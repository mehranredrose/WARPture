#!/usr/bin/env bash
# WARPture macOS installer
set -euo pipefail

APP_NAME="WARPture"
VERSION="1.0.0"
INSTALL_DIR="/Applications/${APP_NAME}.app"
CONFIG_DIR="$HOME/.config/warp-gui"
LOG_DIR="$HOME/.local/share/warp-gui/logs"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"

echo "╔══════════════════════════════════╗"
echo "║  WARPture v${VERSION} – macOS Installer  ║"
echo "╚══════════════════════════════════╝"
echo ""

# Check for warp-cli
if ! command -v warp-cli &>/dev/null; then
    echo "⚠️  warp-cli not found. WARPture will run in simulation mode."
    echo "   Install Cloudflare WARP from: https://developers.cloudflare.com/warp-client/"
fi

# Create config and log directories
mkdir -p "$CONFIG_DIR" "$LOG_DIR"
echo "✓ Created config dir: $CONFIG_DIR"

# Write default config if none exists
if [[ ! -f "$CONFIG_DIR/split-tunnel.json" ]]; then
    cat > "$CONFIG_DIR/split-tunnel.json" << 'JSON'
{
  "version": 1,
  "defaultPolicy": "warp",
  "includedApps": [],
  "excludedApps": []
}
JSON
    echo "✓ Written default split-tunnel config"
fi

# Copy app bundle
if [[ -d "./WARPture.app" ]]; then
    cp -r "./WARPture.app" "$INSTALL_DIR"
    echo "✓ Installed $APP_NAME.app → $INSTALL_DIR"
fi

# Install LaunchAgent for tunnel-agent (background service)
LAUNCH_AGENT_PLIST="$LAUNCH_AGENTS_DIR/com.mehranredrose.warpture.tunnel-agent.plist"
mkdir -p "$LAUNCH_AGENTS_DIR"
cat > "$LAUNCH_AGENT_PLIST" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.mehranredrose.warpture.tunnel-agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>/Applications/WARPture.app/Contents/MacOS/tunnel-agent</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$LOG_DIR/tunnel-agent.log</string>
  <key>StandardErrorPath</key>
  <string>$LOG_DIR/tunnel-agent-error.log</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>CONFIG_PATH</key>
    <string>$CONFIG_DIR/split-tunnel.json</string>
    <key>HTTP_ADDR</key>
    <string>:8080</string>
    <key>WS_ADDR</key>
    <string>:8081</string>
  </dict>
</dict>
</plist>
PLIST

launchctl load "$LAUNCH_AGENT_PLIST" 2>/dev/null || true
echo "✓ Registered LaunchAgent: tunnel-agent"

echo ""
echo "✅ WARPture installed successfully!"
echo "   Launch from /Applications or Spotlight Search."
