"""macOS application scanner using /Applications and Launch Services."""
from __future__ import annotations

import base64
import glob
import logging
import os
import plistlib
import subprocess
import tempfile
from pathlib import Path
from typing import List, Optional

from .detector import AppDetector, lookup_app, make_app_id

logger = logging.getLogger(__name__)

APP_DIRS = [
    "/Applications",
    str(Path.home() / "Applications"),
    "/System/Applications",
    "/System/Applications/Utilities",
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
                [
                    "osascript", "-e",
                    'tell application "System Events" to get bundle identifier'
                    ' of every process whose background only is false',
                ],
                capture_output=True,
                text=True,
                timeout=5,
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
                capture_output=True,
                text=True,
                timeout=5,
            )
            app_ids = []
            for proc in result.stdout.strip().splitlines():
                proc = os.path.basename(proc).lower()
                db = lookup_app(proc)
                if db and db["id"] not in app_ids:
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

        name = (
            plist.get("CFBundleDisplayName")
            or plist.get("CFBundleName")
            or Path(app_path).stem
        )
        bundle_id = plist.get("CFBundleIdentifier", "")
        icon_name = plist.get("CFBundleIconFile", "")
        icon_b64 = self._extract_icon_b64(app_path, icon_name)

        return self._make_entry(name, app_path, bundle_id, icon_b64)

    def _extract_icon_b64(self, app_path: str, icon_name: str) -> str:
        """Convert .icns app icon to base64 PNG data URI using sips."""
        if not icon_name:
            return ""

        # Ensure .icns extension
        if not icon_name.endswith(".icns"):
            icon_name += ".icns"

        icon_path = os.path.join(app_path, "Contents", "Resources", icon_name)
        if not os.path.isfile(icon_path):
            # Try without extension (some bundles omit it)
            icon_path_no_ext = os.path.join(
                app_path, "Contents", "Resources",
                icon_name.replace(".icns", "")
            )
            if os.path.isfile(icon_path_no_ext):
                icon_path = icon_path_no_ext
            else:
                return ""

        try:
            with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as tmp:
                tmp_path = tmp.name

            result = subprocess.run(
                [
                    "sips",
                    "-s", "format", "png",
                    "--resampleWidth", "32",
                    icon_path,
                    "--out", tmp_path,
                ],
                capture_output=True,
                timeout=5,
            )

            if result.returncode != 0:
                return ""

            with open(tmp_path, "rb") as f:
                data = f.read()

            return "data:image/png;base64," + base64.b64encode(data).decode()

        except Exception as exc:
            logger.debug("icon extraction failed for %s: %s", app_path, exc)
            return ""
        finally:
            try:
                os.unlink(tmp_path)
            except Exception:
                pass