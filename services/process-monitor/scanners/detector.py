"""Base AppDetector class and shared app database."""
from __future__ import annotations
import hashlib
import re
from abc import ABC, abstractmethod
from typing import Dict, List, Optional

APP_DATABASE: Dict[str, dict] = {
    "chrome": {"name": "Google Chrome", "bundleId": "com.google.Chrome", "linuxExec": "google-chrome"},
    "firefox": {"name": "Firefox", "bundleId": "org.mozilla.firefox", "linuxExec": "firefox"},
    "safari": {"name": "Safari", "bundleId": "com.apple.Safari", "macosOnly": True},
    "edge": {"name": "Microsoft Edge", "bundleId": "com.microsoft.edgemac", "linuxExec": "microsoft-edge"},
    "brave": {"name": "Brave Browser", "bundleId": "com.brave.Browser", "linuxExec": "brave-browser"},
    "slack": {"name": "Slack", "bundleId": "com.tinyspeck.slackmacgap", "linuxExec": "slack"},
    "discord": {"name": "Discord", "bundleId": "com.hnc.Discord", "linuxExec": "discord"},
    "zoom": {"name": "Zoom", "bundleId": "us.zoom.xos", "linuxExec": "zoom"},
    "teams": {"name": "Microsoft Teams", "bundleId": "com.microsoft.teams", "linuxExec": "teams"},
    "telegram": {"name": "Telegram", "bundleId": "ru.keepcoder.Telegram", "linuxExec": "telegram-desktop"},
    "signal": {"name": "Signal", "bundleId": "org.whispersystems.signal-desktop", "linuxExec": "signal-desktop"},
    "whatsapp": {"name": "WhatsApp", "bundleId": "net.whatsapp.WhatsApp", "linuxExec": "whatsapp-desktop"},
    "skype": {"name": "Skype", "bundleId": "com.skype.skype", "linuxExec": "skype"},
    "vscode": {"name": "Visual Studio Code", "bundleId": "com.microsoft.VSCode", "linuxExec": "code"},
    "iterm": {"name": "iTerm2", "bundleId": "com.googlecode.iterm2", "macosOnly": True},
    "xcode": {"name": "Xcode", "bundleId": "com.apple.dt.Xcode", "macosOnly": True},
    "pycharm": {"name": "PyCharm", "bundleId": "com.jetbrains.pycharm", "linuxExec": "pycharm"},
    "postman": {"name": "Postman", "bundleId": "com.postmanlabs.mac", "linuxExec": "postman"},
    "steam": {"name": "Steam", "bundleId": "com.valvesoftware.steam", "linuxExec": "steam"},
    "epic-games": {"name": "Epic Games Launcher", "bundleId": "com.epicgames.launcher"},
    "battle-net": {"name": "Battle.net", "bundleId": "net.battle.app"},
    "spotify": {"name": "Spotify", "bundleId": "com.spotify.client", "linuxExec": "spotify"},
    "vlc": {"name": "VLC", "bundleId": "org.videolan.vlc", "linuxExec": "vlc"},
    "plex": {"name": "Plex", "bundleId": "tv.plex.plex-media-player", "linuxExec": "plex"},
    "notion": {"name": "Notion", "bundleId": "notion.id", "linuxExec": "notion-app"},
    "obsidian": {"name": "Obsidian", "bundleId": "md.obsidian", "linuxExec": "obsidian"},
    "1password": {"name": "1Password", "bundleId": "com.1password.1password", "linuxExec": "1password"},
    "dropbox": {"name": "Dropbox", "bundleId": "com.getdropbox.dropbox", "linuxExec": "dropbox"},
    "bitwarden": {"name": "Bitwarden", "bundleId": "com.bitwarden.desktop", "linuxExec": "bitwarden"},
    "figma": {"name": "Figma", "bundleId": "com.figma.Desktop", "linuxExec": "figma-linux"},
    "linear": {"name": "Linear", "bundleId": "com.linear", "linuxExec": "linear"},
    "nordvpn": {"name": "NordVPN", "bundleId": "com.nordvpn.macos", "linuxExec": "nordvpn"},
    "alfred": {"name": "Alfred", "bundleId": "com.runningwithcrayons.Alfred", "macosOnly": True},
    "rectangle": {"name": "Rectangle", "bundleId": "com.knollsoft.Rectangle", "macosOnly": True},
    "docker-desktop": {"name": "Docker Desktop", "bundleId": "com.docker.docker", "linuxExec": "docker-desktop"},
    "intellij": {"name": "IntelliJ IDEA", "bundleId": "com.jetbrains.intellij", "linuxExec": "idea"},
    "gimp": {"name": "GIMP", "bundleId": "org.gimp.gimp", "linuxExec": "gimp"},
    "inkscape": {"name": "Inkscape", "bundleId": "org.inkscape.Inkscape", "linuxExec": "inkscape"},
    "libreoffice": {"name": "LibreOffice", "bundleId": "org.libreoffice.script", "linuxExec": "libreoffice"},
    "thunderbird": {"name": "Thunderbird", "bundleId": "org.mozilla.thunderbird", "linuxExec": "thunderbird"},
    "tor-browser": {"name": "Tor Browser", "bundleId": "org.torproject.torbrowser", "linuxExec": "torbrowser-launcher"},
    "virtualbox": {"name": "VirtualBox", "bundleId": "org.virtualbox.app.VirtualBox", "linuxExec": "virtualbox"},
    "wireshark": {"name": "Wireshark", "bundleId": "org.wireshark.Wireshark", "linuxExec": "wireshark"},
    "proxyman": {"name": "Proxyman", "bundleId": "com.proxyman.NSProxy", "macosOnly": True},
    "charles": {"name": "Charles Proxy", "bundleId": "com.xk72.charles", "linuxExec": "charles"},
    "transmit": {"name": "Transmit", "bundleId": "com.panic.Transmit", "macosOnly": True},
    "cyberduck": {"name": "Cyberduck", "bundleId": "ch.sudo.cyberduck", "linuxExec": "cyberduck"},
}


def make_app_id(name: str, bundle_id: Optional[str] = None, path: Optional[str] = None) -> str:
    if bundle_id:
        return re.sub(r"[^a-z0-9\-]", "-", bundle_id.lower()).strip("-")
    if path:
        return hashlib.md5(path.encode()).hexdigest()[:12]
    return re.sub(r"[^a-z0-9\-]", "-", name.lower()).strip("-")


def lookup_app(name: str, bundle_id: Optional[str] = None) -> Optional[dict]:
    if bundle_id:
        for app_id, info in APP_DATABASE.items():
            if info.get("bundleId") == bundle_id:
                return {**info, "id": app_id}
    name_lower = name.lower()
    for app_id, info in APP_DATABASE.items():
        if info["name"].lower() == name_lower or app_id in name_lower:
            return {**info, "id": app_id}
    return None


class AppDetector(ABC):
    @abstractmethod
    def scan_installed(self) -> List[dict]:
        """Return list of installed application entries."""

    @abstractmethod
    def get_running_app_ids(self) -> List[str]:
        """Return list of currently running app IDs."""

    def _make_entry(self, name: str, path: str, bundle_id: Optional[str] = None, icon: Optional[str] = None) -> dict:
        db = lookup_app(name, bundle_id)
        app_id = db["id"] if db else make_app_id(name, bundle_id, path)
        return {
            "id": app_id,
            "name": db["name"] if db else name,
            "bundleId": bundle_id or (db.get("bundleId", "") if db else ""),
            "path": path,
            "icon": icon or "",
            "policy": "default",
            "running": False,
        }
