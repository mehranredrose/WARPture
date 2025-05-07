# WARPture – Architecture

## Overview

WARPture is a three-service microarchitecture:

```
┌───────────────────────────────────────────────────────────┐
│                        User Machine                        │
│                                                           │
│  ┌──────────────┐  IPC/REST+WS  ┌───────────────────────┐│
│  │  warp-gui    │◄─────────────►│   tunnel-agent (Go)   ││
│  │ (Electron +  │               │                       ││
│  │  React/TS)   │               │  ┌─────────────────┐  ││
│  │              │               │  │ warp/ (CLI wrap)│  ││
│  │  • Tray icon │               │  ├─────────────────┤  ││
│  │  • App list  │               │  │ split/ (rules)  │  ││
│  │  • Settings  │               │  ├─────────────────┤  ││
│  └──────────────┘               │  │ api/ (REST+WS)  │  ││
│                                 │  └─────────────────┘  ││
│                                 └──────────┬────────────┘│
│                                            │ REST         │
│                          ┌─────────────────▼────────────┐│
│                          │  process-monitor (Python)    ││
│                          │                              ││
│                          │  • scan_installed()          ││
│                          │  • get_running_app_ids()     ││
│                          │  • MacOSScanner / Linux      ││
│                          └──────────────────────────────┘│
│                                            │              │
│                          ┌─────────────────▼────────────┐│
│                          │       warp-cli (Cloudflare)  ││
│                          └──────────────────────────────┘│
└───────────────────────────────────────────────────────────┘
```

## Services

### 1. warp-gui (Electron + React/TypeScript)

| Responsibility | Implementation |
|---|---|
| System tray icon | Electron `Tray` + `nativeImage` |
| Popup window | `BrowserWindow` (frameless, transparent) |
| App list + policy toggles | React + Zustand + `react-window` |
| IPC to tunnel-agent | `contextBridge` + `ipcMain/ipcRenderer` |
| Status push | WebSocket listener → IPC → React state |
| Auto-launch | Electron `setLoginItemSettings` / Linux autostart |
| Keyboard shortcut | `globalShortcut` (Cmd/Ctrl+Shift+W) |

### 2. tunnel-agent (Go)

| Responsibility | Implementation |
|---|---|
| WARP CLI wrapper | `internal/warp/client.go` – `exec.Command` |
| Mock/sim mode | Auto-detected when `warp-cli` not in PATH |
| Split-tunnel config | `internal/split/manager.go` – JSON persistence |
| REST API | Gin v1.9 – `/api/v1/*` |
| WebSocket server | `gorilla/websocket` hub+client pattern |
| Health monitor | Background goroutine, 5s polling, auto-reconnect |

### 3. process-monitor (Python)

| Responsibility | Implementation |
|---|---|
| macOS app scan | `MacOSScanner` – `/Applications` + `plistlib` |
| macOS proc scan | `osascript` → bundle IDs; fallback `ps` |
| Linux app scan | `LinuxScanner` – `.desktop` files via `configparser` |
| Linux proc scan | `/proc/[pid]/comm` scanning |
| Push to agent | `requests` with retry – `/api/v1/apps/merge` |
| Running state | `/api/v1/apps/running` delta updates |

## Split Tunneling Modes

### Mode 1: Proxy Mode (Default, No Root Required)

```
warp-cli set-mode proxy
# WARP listens on 127.0.0.1:40000

Included apps:
  └─ Set HTTP_PROXY=127.0.0.1:40000 in app environment

Excluded apps:
  └─ Unset HTTP_PROXY (or set NO_PROXY=*)

Default policy:
  └─ Follow global defaultPolicy setting
```

### Mode 2: IP Rule Mode (Root Required)

**Linux:**
```bash
# Include: force app traffic through WARP interface
ip rule add fwmark 0x100 table 100
ip route add default dev warp0 table 100
iptables -t mangle -A OUTPUT -m owner --uid-owner <uid> -j MARK --set-mark 0x100

# Exclude: bypass WARP
ip rule add from <app-src-ip> table main
```

**macOS:**
```bash
# Include: route app traffic through utun interface
pfctl -a warpture/include -f - << RULES
pass out route-to (utun3 192.168.100.1) from <app-src> to any
RULES

# Exclude: bypass
pfctl -a warpture/exclude -f - << RULES
pass out from <app-src> to any no state
RULES
```

## Data Flow

```
User toggles app → include
        │
        ▼
warp-gui (React)
  useWarpStore.setAppPolicy("firefox", "include")
        │
        ▼ IPC (ipcRenderer.invoke)
Electron main
  tunnelClient.setAppPolicy("firefox", "include")
        │
        ▼ HTTP POST /api/v1/apps/policy
tunnel-agent (Go)
  split.Manager.SetAppPolicy("firefox", "include")
  → writes ~/.config/warp-gui/split-tunnel.json
  → enforces proxy/routing rule
        │
        ▼ WebSocket broadcast
warp-gui receives "appUpdate" event
  → React state updates optimistically
```

## Git Flow

```
main ──────────────────────────────────────────► (production)
  │
  └─► develop ──────────────────────────────────► (integration)
          │
          ├─► feature/gui-app-list     (merged → develop)
          ├─► feature/tunnel-controller (merged → develop)
          ├─► feature/process-monitor   (merged → develop)
          ├─► feature/split-tunnel-logic (merged → develop)
          ├─► feature/tray-toggle        (merged → develop)
          ├─► feature/macos-daemon       (merged → develop)
          ├─► feature/linux-service      (merged → develop)
          └─► release/v1.0.0 ──────────► main (tag v1.0.0)
```

## Configuration File

`~/.config/warp-gui/split-tunnel.json`:

```json
{
  "version": 1,
  "defaultPolicy": "warp",
  "includedApps": [
    {
      "id": "firefox",
      "name": "Firefox",
      "bundleId": "org.mozilla.firefox",
      "path": "/Applications/Firefox.app",
      "policy": "include",
      "running": false
    }
  ],
  "excludedApps": [
    {
      "id": "zoom",
      "name": "Zoom",
      "bundleId": "us.zoom.xos",
      "path": "/Applications/zoom.us.app",
      "policy": "exclude",
      "running": true
    }
  ],
  "updatedAt": "2025-12-01T10:00:00Z"
}
```

## Resource Usage

| Component | RAM (idle) | CPU (idle) | RAM (active) |
|---|---|---|---|
| warp-gui (Electron) | ~35 MB | <0.5% | ~60 MB |
| tunnel-agent (Go) | ~8 MB | <0.1% | ~12 MB |
| process-monitor (Python) | ~18 MB | <1% | ~22 MB |
| **Total** | **~61 MB** | **<1.6%** | **~94 MB** |
