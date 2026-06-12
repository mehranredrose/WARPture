"""Linux application scanner using .desktop files and /proc."""
from __future__ import annotations

import base64
import configparser
import glob
import logging
import os
import re
from pathlib import Path
from typing import List, Optional

from .detector import AppDetector, lookup_app, make_app_id

logger = logging.getLogger(__name__)

DESKTOP_DIRS = [
    "/usr/share/applications",
    "/usr/local/share/applications",
    str(Path.home() / ".local" / "share" / "applications"),
    "/var/lib/snapd/desktop/applications",
    "/var/lib/flatpak/exports/share/applications",
    str(Path.home() / ".local" / "share" / "flatpak" / "exports" / "share" / "applications"),
]

ICON_DIRS = [
    "/usr/share/pixmaps",
    "/usr/share/icons/hicolor/128x128/apps",
    "/usr/share/icons/hicolor/64x64/apps",
    "/usr/share/icons/hicolor/48x48/apps",
    "/usr/share/icons/hicolor/32x32/apps",
    "/usr/share/icons/Adwaita/32x32/apps",
    str(Path.home() / ".local" / "share" / "icons"),
    str(Path.home() / ".icons"),
]


class LinuxScanner(AppDetector):

    def scan_installed(self) -> List[dict]:
        apps = []
        seen_ids = set()
        for desktop_dir in DESKTOP_DIRS:
            if not os.path.isdir(desktop_dir):
                continue
            for desktop_file in glob.glob(os.path.join(desktop_dir, "*.desktop")):
                try:
                    entry = self._parse_desktop_file(desktop_file)
                    if entry and entry["id"] not in seen_ids:
                        seen_ids.add(entry["id"])
                        apps.append(entry)
                except Exception as exc:
                    logger.debug("Failed to parse %s: %s", desktop_file, exc)
        logger.debug("Linux scan: found %d apps", len(apps))
        return apps

    def get_running_app_ids(self) -> List[str]:
        app_ids = []
        try:
            for comm_file in glob.glob("/proc/[0-9]*/comm"):
                try:
                    with open(comm_file) as f:
                        comm = f.read().strip().lower()
                    db = lookup_app(comm)
                    if db and db["id"] not in app_ids:
                        app_ids.append(db["id"])
                except (IOError, PermissionError):
                    continue
        except Exception as exc:
            logger.warning("proc scan failed: %s", exc)
        return app_ids

    def _parse_desktop_file(self, path: str) -> Optional[dict]:
        cp = configparser.RawConfigParser(strict=False)
        cp.read(path, encoding="utf-8")

        if not cp.has_section("Desktop Entry"):
            return None

        entry = cp["Desktop Entry"]

        # Skip non-application and hidden entries
        if entry.get("Type", "Application") != "Application":
            return None
        if entry.get("NoDisplay", "false").lower() == "true":
            return None
        if entry.get("Hidden", "false").lower() == "true":
            return None

        name = entry.get("Name", Path(path).stem)
        icon = entry.get("Icon", "")

        # Resolve icon to base64
        icon_b64 = self._icon_to_b64(self._resolve_icon(icon))

        # Try to get bundle ID from known app database
        db = lookup_app(name)
        bundle_id = db.get("bundleId", "") if db else ""

        return self._make_entry(name, path, bundle_id, icon_b64)

    def _resolve_icon(self, icon: str) -> str:
        """Resolve icon name to absolute file path."""
        if not icon:
            return ""

        # Already an absolute path
        if os.path.isabs(icon) and os.path.isfile(icon):
            return icon

        # Search icon dirs with common extensions
        for ext in ("", ".png", ".svg", ".xpm"):
            for icon_dir in ICON_DIRS:
                candidate = os.path.join(icon_dir, icon + ext)
                if os.path.isfile(candidate):
                    return candidate

        # Recursive search in hicolor theme
        hicolor = "/usr/share/icons/hicolor"
        if os.path.isdir(hicolor):
            for root, _, files in os.walk(hicolor):
                for ext in (".png", ".svg"):
                    candidate = os.path.join(root, icon + ext)
                    if os.path.isfile(candidate):
                        return candidate

        return ""

    def _icon_to_b64(self, path: str) -> str:
        """Convert icon file to base64 data URI."""
        if not path or not os.path.isfile(path):
            return ""
        try:
            ext = path.rsplit(".", 1)[-1].lower()
            mime_map = {
                "png": "image/png",
                "svg": "image/svg+xml",
                "xpm": "image/x-xpixmap",
                "jpg": "image/jpeg",
                "jpeg": "image/jpeg",
            }
            mime = mime_map.get(ext, "image/png")

            with open(path, "rb") as f:
                data = f.read()

            return f"data:{mime};base64," + base64.b64encode(data).decode()

        except Exception as exc:
            logger.debug("icon_to_b64 failed for %s: %s", path, exc)
            return ""