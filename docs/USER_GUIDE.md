# WARPture User Guide

## Getting Started

### 1. Install Cloudflare WARP
Download from [cloudflareclient.com](https://developers.cloudflare.com/warp-client/) and register:
```bash
warp-cli register
warp-cli connect
```

### 2. Install WARPture
- **macOS**: Open `WARPture-1.0.0.dmg`, drag to Applications
- **Linux DEB**: `sudo dpkg -i warpture_1.0.0_amd64.deb`
- **Linux RPM**: `sudo rpm -i warpture-1.0.0.x86_64.rpm`

### 3. Launch
WARPture appears in your menu bar (macOS) or system tray (Linux) as a shield icon 🛡️.

---

## The Interface

Click the tray icon to open the WARPture panel:

```
┌─────────────────────────────────┐
│ 🛡️ WARPture              [Apps]│
├─────────────────────────────────┤
│ ● Connected           [Disconnect]│
├─────────────────────────────────┤
│ Presets: [Work] [Gaming] [Stream]│
├─────────────────────────────────┤
│ 🔍 Search apps...               │
├─────────────────────────────────┤
│ [icon] Chrome     [● INCLUDE]   │
│ [icon] Firefox    [● INCLUDE]   │
│ [icon] Zoom       [○ EXCLUDE]   │
│ [icon] Slack      [~ DEFAULT]   │
│ [icon] Discord    [○ EXCLUDE]   │
│ [icon] Steam      [○ EXCLUDE]   │
└─────────────────────────────────┘
```

---

## Per-App Policies

Each app has three possible states (click to cycle):

| State | Symbol | Meaning |
|-------|--------|---------|
| **INCLUDE** | `●` | App traffic always routes through WARP |
| **EXCLUDE** | `○` | App traffic bypasses WARP entirely |
| **DEFAULT** | `~` | Follows global default policy |

### How to change a policy
1. Find the app in the list (or search for it)
2. Click the policy badge on the right to cycle: `DEFAULT → INCLUDE → EXCLUDE → DEFAULT`
3. The change is applied immediately and saved automatically

---

## Quick Presets

Presets apply recommended policies for a group of apps with one click:

| Preset | Included | Excluded |
|--------|----------|----------|
| 🏢 **Work** | Slack, Zoom, Teams, Notion | Steam, Discord, Spotify |
| 🎮 **Gaming** | Slack | Steam, Epic, Battle.net |
| 📺 **Streaming** | Slack, Zoom | Netflix, Spotify, Twitch, Plex |

---

## Default Policy

Found in **Settings → Default Policy**. Controls apps not explicitly listed:

- **Route through WARP** – All unnamed apps use WARP (most secure)
- **Bypass WARP** – All unnamed apps go direct (fastest)
- **System Default** – Follow whatever the OS routing table decides

---

## Keyboard Shortcut

| Shortcut | Action |
|----------|--------|
| `⌘⇧W` (macOS) | Toggle WARP on/off |
| `Ctrl+Shift+W` (Linux) | Toggle WARP on/off |

---

## Tunneling Modes

### Proxy Mode (Default)
- **No root required**
- WARP creates a local proxy at `127.0.0.1:40000`
- Included apps have `HTTP_PROXY` set; excluded apps bypass it
- Enable: `warp-cli set-mode proxy`

### IP Rule Mode
- **Root required**
- Uses `pfctl` (macOS) or `ip rule`/`nftables` (Linux)
- More reliable for apps that ignore proxy settings
- Enable in Settings → Tunneling Mode → IP Rules

---

## Configuration File

Your settings are stored at:
- **macOS/Linux**: `~/.config/warp-gui/split-tunnel.json`

You can edit this file directly while WARPture is stopped.

---

## Auto-Start

Enable **Settings → Launch at Login** to start WARPture automatically on login.

- macOS: registered as `LSLoginItem`
- Linux: creates `~/.config/autostart/warpture.desktop`

---

## Troubleshooting

### "warp-cli not found"
WARPture runs in **simulation mode**. Install Cloudflare WARP CLI from:
https://developers.cloudflare.com/warp-client/

### Apps not appearing
- macOS: check `/Applications` and `~/Applications`
- Linux: check `/usr/share/applications` and `~/.local/share/applications`
- Click the refresh button or wait 30 seconds for the next scan

### WARP toggle not working
1. Check `warp-cli status` in terminal
2. Ensure the tunnel-agent service is running:
   - macOS: `launchctl list | grep warpture`
   - Linux: `systemctl --user status warpture-tunnel-agent`

### Logs
Logs are at `~/.local/share/warp-gui/logs/`:
- `tunnel-agent.log` — WARP connection and API activity
- `process-monitor.log` — App detection activity
