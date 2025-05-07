"""macOS application scanner using /Applications and Launch Services."""
from __future__ import annotations
import glob
import logging
import os
import plistlib
import subprocess
from pathlib import Path
from typing import List, Optional

from .detector import AppDetector, lookup_app, make_app_id

logger = logging.getLogger(__name__)

APP_DIRS = [
    "/Applications",
    str(Path.home() / "Applications"),
    "/System/Applications",
]


class MacOSScanner(AppDetector):
    def scan_installed(self) -> List[dict]:
        apps = []
        seen_ids = set()
        for app_dir in APP_DIRS:
            if not os.path.isdir(app_dir):
                continue
            for app_path in glob.glob(os.path.join(app_dir, "*.app")):
                try:
                    entry = self._parse_app_bundle(app_path)
                    if entry and entry["id"] not in seen_ids:
                        seen_ids.add(entry["id"])
                        apps.append(entry)
                except Exception as exc:
                    logger.debug("Failed to parse %s: %s", app_path, exc)
        logger.debug("macOS scan: found %d apps", len(apps))
        return apps

    def get_running_app_ids(self) -> List[str]:
        try:
            result = subprocess.run(
                ["osascript", "-e",
                 'tell application "System Events" to get bundle identifier of every process whose background only is false'],
                capture_output=True, text=True, timeout=5
            )
            if result.returncode == 0:
                raw = result.stdout.strip()
                bundle_ids = [b.strip() for b in raw.split(",") if b.strip()]
                app_ids = []
                for bid in bundle_ids:
                    db = lookup_app("", bid)
                    if db:
                        app_ids.append(db["id"])
                    else:
                        app_ids.append(make_app_id("", bid))
                return app_ids
        except Exception as exc:
            logger.debug("osascript failed: %s", exc)
        return self._fallback_ps_scan()

    def _fallback_ps_scan(self) -> List[str]:
        try:
            result = subprocess.run(
                ["ps", "-eo", "comm"],
                capture_output=True, text=True, timeout=5
            )
            procs = result.stdout.strip().splitlines()
            app_ids = []
            for proc in procs:
                proc = os.path.basename(proc).lower()
                db = lookup_app(proc)
                if db:
                    app_ids.append(db["id"])
            return app_ids
        except Exception as exc:
            logger.debug("ps fallback failed: %s", exc)
            return []

    def _parse_app_bundle(self, app_path: str) -> Optional[dict]:
        plist_path = os.path.join(app_path, "Contents", "Info.plist")
        if not os.path.isfile(plist_path):
            return None
        with open(plist_path, "rb") as f:
            plist = plistlib.load(f)
        name = plist.get("CFBundleDisplayName") or plist.get("CFBundleName") or Path(app_path).stem
        bundle_id = plist.get("CFBundleIdentifier", "")
        icon_name = plist.get("CFBundleIconFile", "")
        icon_path = ""
        if icon_name:
            if not icon_name.endswith(".icns"):
                icon_name += ".icns"
            icon_candidate = os.path.join(app_path, "Contents", "Resources", icon_name)
            if os.path.isfile(icon_candidate):
                icon_path = icon_candidate
        return self._make_entry(name, app_path, bundle_id, icon_path)
