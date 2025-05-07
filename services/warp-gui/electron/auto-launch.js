/**
 * WARPture – Auto Launch Helper
 * Manages launch-at-login via Electron's built-in loginItems API.
 */

const { app } = require("electron");

class AutoLauncher {
  constructor(appName) {
    this.appName = appName;
  }

  isEnabled() {
    if (process.platform === "win32" || process.platform === "darwin") {
      const settings = app.getLoginItemSettings();
      return settings.openAtLogin;
    }
    // Linux: check ~/.config/autostart/<appName>.desktop
    const { existsSync } = require("fs");
    const desktopPath = this._linuxDesktopPath();
    return existsSync(desktopPath);
  }

  setEnabled(enabled) {
    if (process.platform === "win32" || process.platform === "darwin") {
      app.setLoginItemSettings({
        openAtLogin: enabled,
        openAsHidden: true,
      });
      return;
    }
    // Linux
    if (enabled) {
      this._writeLinuxAutostart();
    } else {
      this._removeLinuxAutostart();
    }
  }

  _linuxDesktopPath() {
    const os = require("os");
    const path = require("path");
    return path.join(os.homedir(), ".config", "autostart", "warpture.desktop");
  }

  _writeLinuxAutostart() {
    const fs = require("fs");
    const path = require("path");
    const desktopPath = this._linuxDesktopPath();
    const dir = path.dirname(desktopPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
    const content = `[Desktop Entry]
Type=Application
Name=WARPture
Exec=${process.execPath} --hidden
Icon=warpture
Comment=Cloudflare WARP GUI Manager
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
`;
    fs.writeFileSync(desktopPath, content, "utf8");
  }

  _removeLinuxAutostart() {
    const fs = require("fs");
    const p = this._linuxDesktopPath();
    if (fs.existsSync(p)) fs.unlinkSync(p);
  }
}

module.exports = { AutoLauncher };
