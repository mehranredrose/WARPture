#!/usr/bin/env python3
"""WARPture – process-monitor: scans processes, notifies tunnel-agent."""
import logging
import logging.handlers
import os
import platform
import signal
import sys
import time
from pathlib import Path
from typing import Dict, List

import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

from scanners.detector import AppDetector
from scanners.macos import MacOSScanner
from scanners.linux import LinuxScanner

LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
log_dir = Path.home() / ".local" / "share" / "warp-gui" / "logs"
log_dir.mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.handlers.RotatingFileHandler(
            log_dir / "process-monitor.log", maxBytes=5 * 1024 * 1024, backupCount=3
        ),
    ],
)
logger = logging.getLogger("process-monitor")

TUNNEL_AGENT_URL = os.environ.get("TUNNEL_AGENT_URL", "http://127.0.0.1:8080")
SCAN_INTERVAL = float(os.environ.get("SCAN_INTERVAL", "2.5"))
INSTALL_SCAN_INTERVAL = float(os.environ.get("INSTALL_SCAN_INTERVAL", "30"))


def build_session() -> requests.Session:
    s = requests.Session()
    retry = Retry(total=3, backoff_factor=0.5, status_forcelist=[500, 502, 503, 504])
    s.mount("http://", HTTPAdapter(max_retries=retry))
    return s


def get_scanner() -> AppDetector:
    system = platform.system()
    if system == "Darwin":
        logger.info("Platform: macOS")
        return MacOSScanner()
    elif system == "Linux":
        logger.info("Platform: Linux")
        return LinuxScanner()
    else:
        raise RuntimeError(f"Unsupported platform: {system}")


class ProcessMonitor:
    def __init__(self):
        self.session = build_session()
        self.scanner = get_scanner()
        self._running = True
        self._known_running: Dict[str, dict] = {}
        self._known_installed: Dict[str, dict] = {}
        self._last_install_scan = 0.0

    def run(self):
        logger.info("Starting (agent=%s, interval=%.1fs)", TUNNEL_AGENT_URL, SCAN_INTERVAL)
        self._scan_installed_apps()
        try:
            while self._running:
                start = time.monotonic()
                try:
                    self._tick()
                except Exception as exc:
                    logger.warning("tick error: %s", exc)
                elapsed = time.monotonic() - start
                time.sleep(max(0.0, SCAN_INTERVAL - elapsed))
        except KeyboardInterrupt:
            logger.info("Shutdown via KeyboardInterrupt")

    def _tick(self):
        if time.monotonic() - self._last_install_scan > INSTALL_SCAN_INTERVAL:
            self._scan_installed_apps()
        self._scan_running_processes()

    def _scan_installed_apps(self):
        self._last_install_scan = time.monotonic()
        try:
            apps = self.scanner.scan_installed()
            new_apps = [a for a in apps if a["id"] not in self._known_installed]
            for a in new_apps:
                self._known_installed[a["id"]] = a
            if new_apps:
                logger.info("Merging %d new apps", len(new_apps))
                self._push_merge(new_apps)
        except Exception as exc:
            logger.error("install scan failed: %s", exc)

    def _scan_running_processes(self):
        try:
            running_ids = set(self.scanner.get_running_app_ids())
        except Exception as exc:
            logger.warning("process scan failed: %s", exc)
            return
        prev_ids = set(self._known_running.keys())
        for app_id in running_ids - prev_ids:
            self._known_running[app_id] = {"id": app_id}
            self._push_running(app_id, True)
        for app_id in prev_ids - running_ids:
            del self._known_running[app_id]
            self._push_running(app_id, False)

    def _push_merge(self, apps: List[dict]):
        try:
            r = self.session.post(f"{TUNNEL_AGENT_URL}/api/v1/apps/merge", json=apps, timeout=5)
            r.raise_for_status()
        except requests.RequestException as exc:
            logger.warning("merge push failed: %s", exc)

    def _push_running(self, app_id: str, running: bool):
        try:
            r = self.session.post(
                f"{TUNNEL_AGENT_URL}/api/v1/apps/running",
                json={"appId": app_id, "running": running},
                timeout=3,
            )
            r.raise_for_status()
        except requests.RequestException as exc:
            logger.debug("running push failed: %s", exc)

    def stop(self):
        self._running = False


def main():
    monitor = ProcessMonitor()

    def _sig(_signum, _frame):
        monitor.stop()

    signal.signal(signal.SIGINT, _sig)
    signal.signal(signal.SIGTERM, _sig)
    monitor.run()
    logger.info("process-monitor exited")


if __name__ == "__main__":
    main()
