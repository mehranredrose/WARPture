#!/usr/bin/env bash
# WARPture uninstaller
set -euo pipefail

echo "Uninstalling WARPture..."

# macOS
if [[ "$(uname)" == "Darwin" ]]; then
    launchctl unload ~/Library/LaunchAgents/com.mehranredrose.warpture.tunnel-agent.plist 2>/dev/null || true
    rm -f ~/Library/LaunchAgents/com.mehranredrose.warpture.tunnel-agent.plist
    rm -rf /Applications/WARPture.app
    echo "✓ Removed macOS components"
fi

# Linux
if [[ "$(uname)" == "Linux" ]]; then
    systemctl --user stop warpture-tunnel-agent.service 2>/dev/null || true
    systemctl --user disable warpture-tunnel-agent.service 2>/dev/null || true
    rm -f ~/.config/systemd/user/warpture-tunnel-agent.service
    rm -f ~/.config/autostart/warpture.desktop
    systemctl --user daemon-reload 2>/dev/null || true
    echo "✓ Removed Linux components"
fi

# Optional: remove config (ask user)
read -rp "Remove config files (~/.config/warp-gui)? [y/N] " answer
if [[ "${answer,,}" == "y" ]]; then
    rm -rf ~/.config/warp-gui
    echo "✓ Removed config"
fi

echo "✅ WARPture uninstalled."
