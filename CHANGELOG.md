# Changelog

All notable changes to WARPture are documented in this file.

## [1.0.0] – 2026-01-05

### Added
- System tray / menu bar GUI (Electron + React + TypeScript)
- Per-app split tunneling with three policies: include, exclude, default
- Proxy mode (no root required) via `warp-cli set-mode proxy`
- IP rule mode (optional, requires root) via `pfctl` / `ip rule`
- macOS support: Intel (x86_64) + Apple Silicon (arm64), universal binary
- Linux support: Debian/Ubuntu (.deb), Fedora (.rpm), Arch (AUR), AppImage
- 50+ pre-configured app database (browsers, comms, dev tools, gaming, media)
- Quick presets: Work, Gaming, Streaming
- Real-time process monitoring (2.5s scan interval)
- Persistent JSON config at `~/.config/warp-gui/split-tunnel.json`
- WebSocket real-time status push to GUI
- Auto-reconnect on WARP disconnection (3 missed health checks)
- Launch at Login (macOS: loginItems, Linux: autostart .desktop)
- Keyboard shortcut: Cmd/Ctrl+Shift+W to toggle WARP
- Graceful degradation to simulation mode when `warp-cli` not installed
- Rotating log files at `~/.local/share/warp-gui/logs/`
- Docker Compose setup for local development
- Integration test suite (10 checks via `scripts/integration-test.sh`)
- Unit tests: Go (split package, 82% coverage), Python (detector, 78%)

### Services
- **warp-gui**: Electron 28 + React 18 + TypeScript + Zustand + Framer Motion
- **tunnel-agent**: Go 1.21 + Gin + gorilla/websocket + logrus
- **process-monitor**: Python 3.11 + requests + psutil

### Known Limitations
- IP rule mode on Linux requires `CAP_NET_ADMIN` or sudo
- macOS Gatekeeper may require manual allow in System Settings on first launch
- Snap app detection limited to `/snap/bin` symlink scanning

## [Unreleased] – 1.1.0-dev

### Planned
- Network statistics graph (bytes in/out over time)
- Per-app bandwidth usage tracking
- Import/export configuration profiles
- Dark/light theme toggle
- WireGuard mode support (when available in warp-cli)
- AppStore / Homebrew Cask distribution
