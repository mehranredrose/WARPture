/**
 * WARPture – App Updater
 * Uses electron-updater for auto-updates via GitHub Releases.
 */

class AppUpdater {
  checkForUpdates() {
    try {
      const { autoUpdater } = require("electron-updater");
      autoUpdater.logger = require("electron-log");
      autoUpdater.logger.transports.file.level = "info";
      autoUpdater.checkForUpdatesAndNotify();
    } catch (err) {
      // electron-updater may not be available in all builds
      console.log("[updater] Auto-update not available:", err.message);
    }
  }
}

module.exports = { AppUpdater };
