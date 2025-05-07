/**
 * WARPture – Electron Preload Script
 * Exposes a safe, typed API to the renderer process via contextBridge.
 */

const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("warpture", {
  // ── App management ────────────────────────────────────────────────────────
  getApps: () => ipcRenderer.invoke("getApps"),
  setAppPolicy: (appId, policy) => ipcRenderer.invoke("setAppPolicy", { appId, policy }),

  // ── Config ────────────────────────────────────────────────────────────────
  getConfig: () => ipcRenderer.invoke("getConfig"),
  setDefaultPolicy: (policy) => ipcRenderer.invoke("setDefaultPolicy", { policy }),

  // ── WARP control ─────────────────────────────────────────────────────────
  toggleWarp: () => ipcRenderer.invoke("toggleWarp"),
  applyPreset: (preset) => ipcRenderer.invoke("applyPreset", { preset }),

  // ── Stats ─────────────────────────────────────────────────────────────────
  getStats: () => ipcRenderer.invoke("getStats"),

  // ── Settings ──────────────────────────────────────────────────────────────
  setAutoLaunch: (enabled) => ipcRenderer.invoke("setAutoLaunch", { enabled }),
  openExternal: (url) => ipcRenderer.invoke("openExternal", { url }),

  // ── Event listeners ───────────────────────────────────────────────────────
  onWarpStatus: (cb) => {
    ipcRenderer.on("warpStatus", (_, status) => cb(status));
    return () => ipcRenderer.removeAllListeners("warpStatus");
  },
  onAgentError: (cb) => {
    ipcRenderer.on("agentError", (_, msg) => cb(msg));
    return () => ipcRenderer.removeAllListeners("agentError");
  },
  onPresetApplied: (cb) => {
    ipcRenderer.on("presetApplied", (_, preset) => cb(preset));
    return () => ipcRenderer.removeAllListeners("presetApplied");
  },
  onNavigate: (cb) => {
    ipcRenderer.on("navigate", (_, route) => cb(route));
    return () => ipcRenderer.removeAllListeners("navigate");
  },
});
