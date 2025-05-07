# WARPture tunnel-agent API Reference

Base URL: `http://127.0.0.1:8080`
WebSocket: `ws://127.0.0.1:8081/ws`

---

## REST Endpoints

### Health

#### `GET /health`
Returns service health and WARP connection status.

**Response:**
```json
{
  "status": "ok",
  "warp": "connected",
  "mock": false,
  "version": "1.0.0",
  "timestamp": "2025-10-15T14:30:00Z"
}
```

---

### WARP Control

#### `GET /api/v1/warp/status`
```json
{ "status": "connected", "mock": false }
```

#### `POST /api/v1/warp/connect`
Connect WARP tunnel.
```json
{ "ok": true }
```

#### `POST /api/v1/warp/disconnect`
Disconnect WARP tunnel.
```json
{ "ok": true }
```

---

### Applications

#### `GET /api/v1/apps`
Returns all tracked apps with their current policies.
```json
[
  {
    "id": "firefox",
    "name": "Firefox",
    "bundleId": "org.mozilla.firefox",
    "path": "/Applications/Firefox.app",
    "policy": "include",
    "running": true
  }
]
```

#### `POST /api/v1/apps/policy`
Update an app's split-tunnel policy.

**Body:**
```json
{ "appId": "firefox", "policy": "include" }
```
Policy values: `"include"` | `"exclude"` | `"default"`

#### `POST /api/v1/apps/running`
Update an app's running state (called by process-monitor).

**Body:**
```json
{ "appId": "firefox", "running": true }
```

#### `POST /api/v1/apps/merge`
Merge newly detected apps (called by process-monitor).

**Body:**
```json
[
  {
    "id": "firefox",
    "name": "Firefox",
    "bundleId": "org.mozilla.firefox",
    "path": "/Applications/Firefox.app",
    "policy": "default",
    "running": false
  }
]
```

---

### Configuration

#### `GET /api/v1/config`
```json
{
  "version": 1,
  "defaultPolicy": "warp",
  "includedApps": [...],
  "excludedApps": [...],
  "updatedAt": "2025-12-01T10:00:00Z"
}
```

#### `POST /api/v1/config/default-policy`
```json
{ "policy": "warp" }
```
Values: `"warp"` | `"bypass"` | `"system"`

---

### Presets

#### `POST /api/v1/presets/apply`
```json
{ "preset": "work" }
```
Values: `"work"` | `"gaming"` | `"streaming"`

---

### Statistics

#### `GET /api/v1/stats`
```json
{
  "bytesIn": 104857600,
  "bytesOut": 20971520,
  "latencyMs": 14,
  "connectedSince": "2025-10-15T09:00:00Z",
  "account": "Team",
  "mode": "proxy"
}
```

---

## WebSocket Events

Connect to `ws://127.0.0.1:8081/ws`.

On connect, the server immediately sends current state.

### Incoming (server → client)

#### `status` — WARP connection changed
```json
{
  "type": "status",
  "status": "connected",
  "payload": {
    "status": "connected",
    "stats": { "latencyMs": 12, "account": "Team" }
  }
}
```

#### `appUpdate` — App list updated (every 3s)
```json
{
  "type": "appUpdate",
  "payload": [
    { "id": "firefox", "name": "Firefox", "policy": "include", "running": true },
    { "id": "zoom", "name": "Zoom", "policy": "exclude", "running": false }
  ]
}
```

---

## Error Responses

All errors follow:
```json
{ "error": "human-readable error message" }
```

| Status | Meaning |
|--------|---------|
| 400 | Bad request / invalid body |
| 404 | App not found |
| 500 | Internal error (warp-cli failed, etc.) |
