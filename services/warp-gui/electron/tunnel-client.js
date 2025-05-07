/**
 * WARPture – Tunnel Agent Client
 * Communicates with the tunnel-agent Go service via REST + WebSocket.
 */

const EventEmitter = require("events");
const WebSocket = require("ws");

class TunnelAgentClient extends EventEmitter {
  constructor(baseUrl, wsUrl) {
    super();
    this.baseUrl = baseUrl;
    this.wsUrl = wsUrl;
    this.ws = null;
    this.reconnectTimer = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 10;
    this.connected = false;
  }

  async init() {
    await this._connectWebSocket();
  }

  // ── REST API ────────────────────────────────────────────────────────────────
  async _fetch(method, path, body = null) {
    const opts = {
      method,
      headers: { "Content-Type": "application/json" },
    };
    if (body) opts.body = JSON.stringify(body);
    try {
      const res = await fetch(`${this.baseUrl}${path}`, opts);
      if (!res.ok) {
        const text = await res.text();
        throw new Error(`HTTP ${res.status}: ${text}`);
      }
      return res.json();
    } catch (err) {
      this.emit("error", err);
      throw err;
    }
  }

  getApps() {
    return this._fetch("GET", "/api/v1/apps");
  }

  getConfig() {
    return this._fetch("GET", "/api/v1/config");
  }

  setAppPolicy(appId, policy) {
    return this._fetch("POST", "/api/v1/apps/policy", { appId, policy });
  }

  setDefaultPolicy(policy) {
    return this._fetch("POST", "/api/v1/config/default-policy", { policy });
  }

  connect() {
    return this._fetch("POST", "/api/v1/warp/connect");
  }

  disconnect() {
    return this._fetch("POST", "/api/v1/warp/disconnect");
  }

  applyPreset(preset) {
    return this._fetch("POST", "/api/v1/presets/apply", { preset });
  }

  getStats() {
    return this._fetch("GET", "/api/v1/stats");
  }

  // ── WebSocket ──────────────────────────────────────────────────────────────
  _connectWebSocket() {
    return new Promise((resolve) => {
      if (this.ws) {
        this.ws.terminate();
      }

      this.ws = new WebSocket(this.wsUrl);

      this.ws.on("open", () => {
        console.log("[tunnel-client] WebSocket connected");
        this.connected = true;
        this.reconnectAttempts = 0;
        resolve();
      });

      this.ws.on("message", (data) => {
        try {
          const msg = JSON.parse(data.toString());
          this._handleMessage(msg);
        } catch (err) {
          console.error("[tunnel-client] WS parse error:", err);
        }
      });

      this.ws.on("error", (err) => {
        console.error("[tunnel-client] WS error:", err.message);
        this.emit("error", err);
      });

      this.ws.on("close", () => {
        console.log("[tunnel-client] WS disconnected");
        this.connected = false;
        this._scheduleReconnect();
      });

      // Resolve after timeout if connection fails (graceful degradation)
      setTimeout(resolve, 3000);
    });
  }

  _handleMessage(msg) {
    switch (msg.type) {
      case "status":
        this.emit("statusChange", msg.status);
        break;
      case "appUpdate":
        this.emit("appUpdate", msg.apps);
        break;
      case "error":
        this.emit("error", new Error(msg.message));
        break;
      default:
        console.log("[tunnel-client] Unknown WS message type:", msg.type);
    }
  }

  _scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("[tunnel-client] Max reconnect attempts reached");
      this.emit("error", new Error("Cannot connect to tunnel-agent"));
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;
    console.log(`[tunnel-client] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.reconnectTimer = setTimeout(() => {
      this._connectWebSocket();
    }, delay);
  }

  destroy() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this.ws.removeAllListeners();
      this.ws.terminate();
    }
  }
}

module.exports = { TunnelAgentClient };
