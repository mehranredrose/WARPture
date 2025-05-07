"""Linux application scanner using .desktop files and /proc."""
from __future__ import annotations
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
            for pid_dir in glob.glob("/proc/[0-9]*/comm"):
                try:
                    with open(pid_dir) as f:
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
        if entry.get("NoDisplay", "false").lower() == "true":
            return None
        if entry.get("Type", "Application") != "Application":
            return None
        name = entry.get("Name", Path(path).stem)
        exec_cmd = entry.get("Exec", "")
        icon = entry.get("Icon", "")
        # Resolve icon path
        icon_path = self._resolve_icon(icon)
        # Try to determine bundle ID from well-known map
        exec_base = re.sub(r"\s+.*", "", exec_cmd).strip().lower()
        db = lookup_app(name, None)
        bundle_id = db.get("bundleId", "") if db else ""
        return self._make_entry(name, path, bundle_id, icon_path)

    def _resolve_icon(self, icon: str) -> str:
        if not icon:
            return ""
        if os.path.isabs(icon) and os.path.isfile(icon):
            return icon
        # Search common icon directories
        search_dirs = [
            "/usr/share/pixmaps",
            "/usr/share/icons/hicolor/128x128/apps",
            "/usr/share/icons/hicolor/64x64/apps",
            str(Path.home() / ".local/share/icons"),
        ]
        for ext in ("", ".png", ".svg", ".xpm"):
            for d in search_dirs:
                candidate = os.path.join(d, icon + ext)
                if os.path.isfile(candidate):
                    return candidate
        return ""
