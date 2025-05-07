# WARPture 🛡️
### Cloudflare WARP GUI Manager — App Split Tunneling Edition

> A production-grade GUI manager for Cloudflare WARP with per-application split tunneling support on macOS and Linux.

---

## ✨ Features

- **System Tray / Menu Bar Integration** — Quick toggle with status indicator
- **Per-App Split Tunneling** — Route individual apps through or around WARP
- **Auto App Detection** — Scans installed apps and .desktop files automatically
- **Two Tunneling Modes** — Proxy-based (no root) and IP rule-based (with root)
- **Real-time Process Monitoring** — Detects newly launched apps instantly
- **Cross-Platform** — macOS (Intel + Apple Silicon) + Linux (Debian/Ubuntu/Fedora/Arch)
- **Persistent Config** — Settings survive reboots via `~/.config/warp-gui/`

---

## 📸 Screenshots

```
┌─────────────────────────────────┐
│  🛡️ WARPture            [●] ON  │
├─────────────────────────────────┤
│  Search apps...         🔍      │
├─────────────────────────────────┤
│  Chrome        [INCLUDE ●]      │
│  Firefox       [INCLUDE ●]      │
│  Zoom          [EXCLUDE ○]      │
│  Slack         [DEFAULT ~]      │
│  Discord       [INCLUDE ●]      │
│  Steam         [EXCLUDE ○]      │
├─────────────────────────────────┤
│  Preset: [Work] [Gaming] [Stream]│
├─────────────────────────────────┤
│  ⚙️ Settings    📊 Stats        │
└─────────────────────────────────┘
```

---

## 🏗️ Architecture

```
┌──────────────┐     IPC/WS      ┌──────────────────┐
│   warp-gui   │◄───────────────►│  tunnel-agent    │
│  (Electron+  │                 │  (Go REST+WS)    │
│   React)     │                 │                  │
└──────────────┘                 └────────┬─────────┘
                                          │ gRPC/REST
                                 ┌────────▼─────────┐
                                 │ process-monitor  │
                                 │  (Python)        │
                                 └──────────────────┘
                                          │
                                 ┌────────▼─────────┐
                                 │   warp-cli        │
                                 │  (Cloudflare)     │
                                 └──────────────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for full details.

---

## 🚀 Quick Start

### macOS

```bash
# Install via DMG
open WARPture-v1.0.0.dmg

# Or install via Homebrew (coming soon)
brew install --cask warpture

# Manual build
make build-macos
```

### Linux (Debian/Ubuntu)

```bash
sudo dpkg -i warpture_1.0.0_amd64.deb
# or
sudo apt install ./warpture_1.0.0_amd64.deb
```

### Linux (Fedora/RHEL)

```bash
sudo rpm -i warpture-1.0.0.x86_64.rpm
```

### From Source

```bash
git clone https://github.com/mehranredrose/WARPture.git
cd WARPture
make install-deps
make dev
```

---

## 📋 Prerequisites

- **Cloudflare WARP CLI** — Install from [cloudflareclient.com](https://developers.cloudflare.com/warp-client/get-started/)
- **Node.js** ≥ 18 (for warp-gui)
- **Go** ≥ 1.21 (for tunnel-agent)
- **Python** ≥ 3.11 (for process-monitor)

---

## ⚙️ Configuration

Config is stored at `~/.config/warp-gui/split-tunnel.json`:

```json
{
  "version": 1,
  "defaultPolicy": "warp",
  "includedApps": [
    {
      "name": "Firefox",
      "bundleId": "org.mozilla.firefox",
      "path": "/Applications/Firefox.app"
    }
  ],
  "excludedApps": [
    {
      "name": "Zoom",
      "bundleId": "us.zoom.xos",
      "path": "/Applications/zoom.us.app"
    }
  ]
}
```

### Tunneling Modes

| Mode | Root Required | How it Works |
|------|--------------|--------------|
| **Proxy Mode** (default) | ❌ No | WARP local proxy on `127.0.0.1:40000`, env injection |
| **IP Rule Mode** | ✅ Yes | `ip rule` / `pfctl` routing rules per-app |

---

## 🧩 Split Tunneling Logic

### Include Mode
App traffic is **forced through WARP** regardless of default policy.

### Exclude Mode
App traffic **bypasses WARP** entirely — direct internet connection.

### Default Policy
Apps not in either list follow the global default: `warp`, `bypass`, or `system`.

---

## 🛠️ Development

```bash
# Start all services (dev mode)
docker-compose up

# Or run individually:
cd services/warp-gui && npm run dev
cd services/tunnel-agent && go run cmd/server/main.go
cd services/process-monitor && python monitor.py

# Run tests
make test

# Lint
make lint
```

---

## 🧪 Testing

```bash
# Unit tests
make test-unit

# Integration tests
./scripts/integration-test.sh

# Test split tunnel logic only
cd services/tunnel-agent && go test ./internal/split/...
```

---

## 📁 Project Structure

```
WARPture/
├── services/
│   ├── warp-gui/          # Electron + React frontend
│   ├── tunnel-agent/      # Go REST+WebSocket middleware
│   └── process-monitor/   # Python process scanner
├── scripts/               # Install/uninstall scripts
├── packaging/             # Platform packaging configs
├── docs/                  # Architecture & API docs
└── Makefile               # Build automation
```

---

## 🔒 Security

- No hardcoded credentials
- Proxy mode requires no elevated privileges
- IP rule mode uses a minimal privileged helper (`warp-helper`) via SMJobBless (macOS) or polkit (Linux)
- All IPC communication is local-only

---

## 📜 License

MIT — see [LICENSE](LICENSE)

---

## 👤 Author

**mehranredrose** — [github.com/mehranredrose](https://github.com/mehranredrose)

---

## 🤝 Contributing

1. Fork the repo
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Follow [Git-Flow](docs/ARCHITECTURE.md#git-flow)
4. Submit a PR to `develop`
