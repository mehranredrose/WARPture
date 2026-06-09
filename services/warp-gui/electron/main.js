/**
 * WARPture – Electron Main Process
 * Handles: tray icon, system menu, window management, IPC bridge, auto-launch
 * Services: tunnel-agent and process-monitor are launched as child processes
 */

const {
  app,
  BrowserWindow,
  Tray,
  Menu,
  nativeImage,
  ipcMain,
  globalShortcut,
  shell,
  Notification,
  screen,
} = require("electron");
const path = require("path");
const url = require("url");
const { spawn } = require("child_process");
const { TunnelAgentClient } = require("./tunnel-client");
const { AutoLauncher } = require("./auto-launch");
const { AppUpdater } = require("./updater");

// ─── Constants ─────────────────────────────────────────────────────────────────
const DEV = process.env.NODE_ENV === "development";
const TUNNEL_AGENT_URL = process.env.TUNNEL_AGENT_URL || "http://127.0.0.1:8080";
const WS_URL = process.env.WS_URL || "ws://127.0.0.1:8081/ws";
const ASSETS = path.join(__dirname, "../assets");

// ─── Globals ───────────────────────────────────────────────────────────────────
let mainWindow = null;
let tray = null;
let tunnelClient = null;
let autoLauncher = null;
let warpStatus = "disconnected";
let tunnelAgentProc = null;
let processMonitorProc = null;

// ─── Service Launcher ──────────────────────────────────────────────────────────
function getResourcePath(name) {
  if (app.isPackaged) {
    return path.join(process.resourcesPath, name);
  }
  return path.join(__dirname, "../resources", name);
}

function startServices() {
  const agentBin = getResourcePath("tunnel-agent");
  const monitorBin = getResourcePath("process-monitor");

  try {
    tunnelAgentProc = spawn(agentBin, [], {
      env: {
        ...process.env,
        HTTP_ADDR: ":8080",
        WS_ADDR: ":8081",
        LOG_LEVEL: "info",
      },
      stdio: "ignore",
      detached: false,
    });

    tunnelAgentProc.on("error", (err) => {
      console.warn("[tunnel-agent] failed to start:", err.message);
    });

    tunnelAgentProc.on("exit", (code) => {
      console.log("[tunnel-agent] exited with code:", code);
    });

    console.log("[main] tunnel-agent started, pid:", tunnelAgentProc.pid);
  } catch (err) {
    console.warn("[main] could not start tunnel-agent:", err.message);
  }

  // Give tunnel-agent 1.5s to boot before starting process-monitor
  setTimeout(() => {
    try {
      processMonitorProc = spawn(monitorBin, [], {
        env: {
          ...process.env,
          TUNNEL_AGENT_URL: "http://127.0.0.1:8080",
          SCAN_INTERVAL: "2.5",
          LOG_LEVEL: "info",
        },
        stdio: "ignore",
        detached: false,
      });

      processMonitorProc.on("error", (err) => {
        console.warn("[process-monitor] failed to start:", err.message);
      });

      processMonitorProc.on("exit", (code) => {
        console.log("[process-monitor] exited with code:", code);
      });

      console.log("[main] process-monitor started, pid:", processMonitorProc.pid);
    } catch (err) {
      console.warn("[main] could not start process-monitor:", err.message);
    }
  }, 1500);
}

function stopServices() {
  if (tunnelAgentProc) {
    tunnelAgentProc.kill();
    tunnelAgentProc = null;
  }
  if (processMonitorProc) {
    processMonitorProc.kill();
    processMonitorProc = null;
  }
}

// ─── App Init ─────────────────────────────────────────────────────────────────
app.whenReady().then(async () => {
  // Single instance lock
  const gotLock = app.requestSingleInstanceLock();
  if (!gotLock) {
    app.quit();
    return;
  }

  app.on("second-instance", () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.show();
      mainWindow.focus();
    }
  });

  // macOS: don't show in dock (tray-only app)
  if (process.platform === "darwin") {
    app.dock.hide();
  }

  // Start bundled services
  startServices();

  // Wait for tunnel-agent to be ready before connecting
  await new Promise((resolve) => setTimeout(resolve, 2000));

  // Init tunnel client
  tunnelClient = new TunnelAgentClient(TUNNEL_AGENT_URL, WS_URL);
  autoLauncher = new AutoLauncher(app.getName());

  try {
    await tunnelClient.init();
  } catch (err) {
    console.warn("[main] tunnel-agent not reachable, running in offline mode:", err.message);
  }

  tunnelClient.on("statusChange", handleStatusChange);
  tunnelClient.on("error", handleAgentError);

  createTray();
  createWindow();
  registerShortcuts();

  // Updater (production only)
  if (!DEV) {
    const updater = new AppUpdater();
    updater.checkForUpdates();
  }
});

// ─── Window ────────────────────────────────────────────────────────────────────
function createWindow() {
  const { width, height } = screen.getPrimaryDisplay().workAreaSize;

  mainWindow = new BrowserWindow({
    width: 480,
    height: 680,
    x: width - 500,
    y: 60,
    resizable: false,
    frame: false,
    transparent: true,
    alwaysOnTop: false,
    skipTaskbar: true,
    show: false,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, "preload.js"),
      devTools: DEV,
    },
  });

  const startURL = DEV
    ? "http://localhost:3000"
    : url.format({
        pathname: path.join(__dirname, "../build/index.html"),
        protocol: "file:",
        slashes: true,
      });

  mainWindow.loadURL(startURL);

  if (DEV) mainWindow.webContents.openDevTools({ mode: "detach" });

  mainWindow.on("blur", () => {
    if (!mainWindow.webContents.isDevToolsOpened()) {
      mainWindow.hide();
    }
  });

  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}

// ─── Tray ──────────────────────────────────────────────────────────────────────
function createTray() {
  const icon = getTrayIcon("disconnected");
  tray = new Tray(icon);
  tray.setToolTip("WARPture – WARP Manager");

  tray.on("click", toggleWindow);
  tray.on("right-click", () => {
    const contextMenu = buildContextMenu();
    tray.popUpContextMenu(contextMenu);
  });

  updateTray();
}

function getTrayIcon(status) {
  const iconMap = {
    connected: "tray-connected.png",
    disconnected: "tray-disconnected.png",
    connecting: "tray-connecting.png",
  };
  const iconPath = path.join(ASSETS, "tray", iconMap[status] || iconMap.disconnected);
  const img = nativeImage.createFromPath(iconPath);
  if (process.platform === "darwin") {
    img.setTemplateImage(true);
  }
  return img.resize({ width: 18, height: 18 });
}

function buildContextMenu() {
  const isConnected = warpStatus === "connected";
  return Menu.buildFromTemplate([
    {
      label: `WARPture v${app.getVersion()}`,
      enabled: false,
    },
    { type: "separator" },
    {
      label: isConnected ? "● Connected" : "○ Disconnected",
      enabled: false,
    },
    { type: "separator" },
    {
      label: isConnected ? "Disconnect WARP" : "Connect WARP",
      click: toggleWarp,
      accelerator: "CmdOrCtrl+Shift+W",
    },
    { type: "separator" },
    {
      label: "Quick Presets",
      submenu: [
        { label: "🏢 Work Mode", click: () => applyPreset("work") },
        { label: "🎮 Gaming Mode", click: () => applyPreset("gaming") },
        { label: "📺 Streaming Mode", click: () => applyPreset("streaming") },
      ],
    },
    { type: "separator" },
    {
      label: "Open WARPture",
      click: showWindow,
    },
    {
      label: "Settings",
      click: () => {
        showWindow();
        mainWindow?.webContents.send("navigate", "/settings");
      },
    },
    { type: "separator" },
    {
      label: "Launch at Login",
      type: "checkbox",
      checked: autoLauncher?.isEnabled() ?? false,
      click: (item) => autoLauncher?.setEnabled(item.checked),
    },
    {
      label: "Quit WARPture",
      click: () => {
        stopServices();
        app.quit();
      },
    },
  ]);
}

function updateTray() {
  if (!tray) return;
  const icon = getTrayIcon(warpStatus);
  tray.setImage(icon);

  const labels = {
    connected: "WARPture: Connected",
    disconnected: "WARPture: Disconnected",
    connecting: "WARPture: Connecting…",
  };
  tray.setToolTip(labels[warpStatus] || "WARPture");
}

// ─── Window Toggle ─────────────────────────────────────────────────────────────
function toggleWindow() {
  if (!mainWindow) {
    createWindow();
    return;
  }
  if (mainWindow.isVisible()) {
    mainWindow.hide();
  } else {
    showWindow();
  }
}

function showWindow() {
  if (!mainWindow) createWindow();
  const { x, y } = tray.getBounds();
  const [w, h] = mainWindow.getSize();
  const { width: sw } = screen.getPrimaryDisplay().workAreaSize;

  let posX = Math.min(x - w / 2, sw - w - 10);
  let posY = process.platform === "darwin" ? y + 24 : y - h - 10;

  mainWindow.setPosition(Math.round(posX), Math.round(posY));
  mainWindow.show();
  mainWindow.focus();
}

// ─── WARP Control ──────────────────────────────────────────────────────────────
async function toggleWarp() {
  try {
    if (warpStatus === "connected") {
      await tunnelClient.disconnect();
    } else {
      await tunnelClient.connect();
    }
  } catch (err) {
    showNotification("WARPture Error", `Failed to toggle WARP: ${err.message}`);
  }
}

async function applyPreset(preset) {
  try {
    await tunnelClient.applyPreset(preset);
    showNotification("WARPture", `Applied ${preset} preset`);
    mainWindow?.webContents.send("presetApplied", preset);
  } catch (err) {
    showNotification("WARPture Error", `Failed to apply preset: ${err.message}`);
  }
}

// ─── Event Handlers ────────────────────────────────────────────────────────────
function handleStatusChange(newStatus) {
  const prev = warpStatus;
  warpStatus = newStatus;
  updateTray();
  mainWindow?.webContents.send("warpStatus", newStatus);

  if (prev !== newStatus) {
    const msgs = {
      connected: "WARP connected",
      disconnected: "WARP disconnected",
      connecting: null,
    };
    if (msgs[newStatus]) showNotification("WARPture", msgs[newStatus]);
  }
}

function handleAgentError(err) {
  console.error("[tunnel-agent error]", err);
  mainWindow?.webContents.send("agentError", err.message);
}

// ─── Keyboard Shortcuts ───────────────────────────────────────────────────────
function registerShortcuts() {
  globalShortcut.register("CmdOrCtrl+Shift+W", toggleWarp);
}

// ─── IPC Handlers ─────────────────────────────────────────────────────────────
ipcMain.handle("getApps", async () => {
  return tunnelClient.getApps();
});

ipcMain.handle("getConfig", async () => {
  return tunnelClient.getConfig();
});

ipcMain.handle("setAppPolicy", async (event, { appId, policy }) => {
  return tunnelClient.setAppPolicy(appId, policy);
});

ipcMain.handle("setDefaultPolicy", async (event, { policy }) => {
  return tunnelClient.setDefaultPolicy(policy);
});

ipcMain.handle("toggleWarp", async () => {
  return toggleWarp();
});

ipcMain.handle("applyPreset", async (event, { preset }) => {
  return applyPreset(preset);
});

ipcMain.handle("getStats", async () => {
  return tunnelClient.getStats();
});

ipcMain.handle("setAutoLaunch", async (event, { enabled }) => {
  return autoLauncher?.setEnabled(enabled);
});

ipcMain.handle("openExternal", async (event, { url: targetUrl }) => {
  shell.openExternal(targetUrl);
});

// ─── Notifications ─────────────────────────────────────────────────────────────
function showNotification(title, body) {
  if (Notification.isSupported()) {
    new Notification({
      title,
      body,
      icon: path.join(ASSETS, "icon.png"),
      silent: false,
    }).show();
  }
}

// ─── Cleanup ──────────────────────────────────────────────────────────────────
app.on("will-quit", () => {
  globalShortcut.unregisterAll();
  tunnelClient?.destroy();
  stopServices();
});

app.on("window-all-closed", () => {
  // Keep running in tray on all platforms
});

app.on("activate", () => {
  if (!mainWindow) createWindow();
});