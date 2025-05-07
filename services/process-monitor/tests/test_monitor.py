"""Tests for WARPture process-monitor."""
import sys
import types
import pytest
from unittest.mock import MagicMock, patch
from pathlib import Path


# ── Fixtures ──────────────────────────────────────────────────────────────────

@pytest.fixture
def sample_db():
    return [
        {"id": "firefox", "name": "Firefox", "bundleId": "org.mozilla.firefox",
         "macPath": "/Applications/Firefox.app", "linuxExec": "firefox", "category": "browser"},
        {"id": "slack", "name": "Slack", "bundleId": "com.tinyspeck.slackmacgap",
         "macPath": "/Applications/Slack.app", "linuxExec": "slack", "category": "communication"},
        {"id": "steam", "name": "Steam", "bundleId": "com.valvesoftware.steam",
         "macPath": "/Applications/Steam.app", "linuxExec": "steam", "category": "gaming"},
    ]


# ── App Database Tests ─────────────────────────────────────────────────────────

def test_app_database_not_empty():
    from scanners.app_database import APP_DATABASE
    assert len(APP_DATABASE) >= 50, "App database should contain at least 50 entries"


def test_app_database_required_fields():
    from scanners.app_database import APP_DATABASE
    required = {"id", "name", "bundleId", "macPath", "linuxExec", "category"}
    for app in APP_DATABASE:
        for field in required:
            assert field in app, f"App {app.get('id', '?')} missing field '{field}'"


def test_app_database_unique_ids():
    from scanners.app_database import APP_DATABASE
    ids = [app["id"] for app in APP_DATABASE]
    assert len(ids) == len(set(ids)), "App database contains duplicate IDs"


def test_app_database_categories():
    from scanners.app_database import APP_DATABASE
    valid_categories = {"browser", "communication", "development", "gaming",
                        "media", "productivity", "cloud", "design", "security", "utility"}
    for app in APP_DATABASE:
        assert app["category"] in valid_categories, \
            f"App {app['id']} has invalid category '{app['category']}'"


def test_app_database_has_browsers():
    from scanners.app_database import APP_DATABASE
    browser_ids = {a["id"] for a in APP_DATABASE if a["category"] == "browser"}
    assert "firefox" in browser_ids
    assert "google-chrome" in browser_ids


def test_app_database_has_communication():
    from scanners.app_database import APP_DATABASE
    ids = {a["id"] for a in APP_DATABASE}
    assert "slack" in ids
    assert "discord" in ids
    assert "zoom" in ids


# ── AppInfo Tests ─────────────────────────────────────────────────────────────

def test_app_info_to_dict():
    from scanners.detector import AppInfo
    app = AppInfo(
        id="test-app",
        name="Test App",
        path="/Applications/Test.app",
        bundle_id="com.test.app",
        icon="/path/to/icon.png",
        policy="default",
    )
    d = app.to_dict()
    assert d["id"] == "test-app"
    assert d["name"] == "Test App"
    assert d["path"] == "/Applications/Test.app"
    assert d["bundleId"] == "com.test.app"
    assert d["icon"] == "/path/to/icon.png"
    assert d["policy"] == "default"


# ── Linux Scanner Tests ───────────────────────────────────────────────────────

def test_linux_make_id():
    """Test that _make_id produces stable kebab-case IDs."""
    from scanners.linux import _make_id
    assert _make_id("Google Chrome") == "google-chrome"
    assert _make_id("visual-studio-code") == "visual-studio-code"
    assert _make_id("org.mozilla.Firefox") == "org-mozilla-firefox"
    assert _make_id("  leading-trailing  ") == "leading-trailing"


def test_linux_extract_binary():
    from scanners.linux import _extract_binary
    assert _extract_binary("google-chrome --no-sandbox %U") == "google-chrome"
    assert _extract_binary("/usr/bin/firefox %u") == "firefox"
    assert _extract_binary("") == ""
    assert _extract_binary("/usr/bin/slack") == "slack"


def test_linux_scanner_discover_skips_non_application(tmp_path):
    """Scanner should skip non-Application .desktop entries."""
    from scanners.linux import LinuxScanner

    desktop_content = """[Desktop Entry]
Type=Link
Name=Not an App
Exec=something
"""
    desktop_file = tmp_path / "link.desktop"
    desktop_file.write_text(desktop_content)

    scanner = LinuxScanner()
    # Monkey-patch DESKTOP_DIRS
    import scanners.linux as linux_mod
    original = linux_mod.DESKTOP_DIRS
    linux_mod.DESKTOP_DIRS = [tmp_path]
    try:
        apps = scanner.discover_installed()
        assert len(apps) == 0
    finally:
        linux_mod.DESKTOP_DIRS = original


def test_linux_scanner_discover_valid_app(tmp_path):
    from scanners.linux import LinuxScanner
    import scanners.linux as linux_mod

    desktop_content = """[Desktop Entry]
Type=Application
Name=Test Browser
Exec=/usr/bin/testbrowser %U
Icon=testbrowser
Categories=Network;WebBrowser;
"""
    desktop_file = tmp_path / "testbrowser.desktop"
    desktop_file.write_text(desktop_content)

    scanner = LinuxScanner()
    original = linux_mod.DESKTOP_DIRS
    linux_mod.DESKTOP_DIRS = [tmp_path]
    try:
        apps = scanner.discover_installed()
        assert len(apps) == 1
        assert apps[0].name == "Test Browser"
        assert apps[0].id == "testbrowser"
    finally:
        linux_mod.DESKTOP_DIRS = original


def test_linux_scanner_skips_nodisplay(tmp_path):
    from scanners.linux import LinuxScanner
    import scanners.linux as linux_mod

    desktop_content = """[Desktop Entry]
Type=Application
Name=Hidden App
Exec=/usr/bin/hidden
NoDisplay=true
"""
    (tmp_path / "hidden.desktop").write_text(desktop_content)

    scanner = LinuxScanner()
    original = linux_mod.DESKTOP_DIRS
    linux_mod.DESKTOP_DIRS = [tmp_path]
    try:
        apps = scanner.discover_installed()
        assert len(apps) == 0
    finally:
        linux_mod.DESKTOP_DIRS = original


# ── macOS Scanner Tests ───────────────────────────────────────────────────────

def test_macos_make_id():
    from scanners.macos import _make_id
    assert _make_id("com.google.Chrome") == "com-google-chrome"
    assert _make_id("Firefox") == "firefox"


def test_macos_scanner_parse_bundle(tmp_path):
    """Test parsing a synthetic .app bundle."""
    import plistlib
    from scanners.macos import MacOSScanner

    # Create synthetic .app bundle
    bundle = tmp_path / "TestApp.app"
    contents = bundle / "Contents"
    (contents / "MacOS").mkdir(parents=True)
    (contents / "Resources").mkdir()

    plist_data = {
        "CFBundleName": "TestApp",
        "CFBundleDisplayName": "Test Application",
        "CFBundleIdentifier": "com.test.testapp",
        "CFBundleIconFile": "",
    }
    with open(contents / "Info.plist", "wb") as f:
        plistlib.dump(plist_data, f)

    scanner = MacOSScanner()
    import scanners.macos as macos_mod
    original = macos_mod.APP_DIRS
    macos_mod.APP_DIRS = [tmp_path]
    try:
        apps = scanner.discover_installed()
        assert len(apps) == 1
        assert apps[0].name == "Test Application"
        assert apps[0].bundle_id == "com.test.testapp"
    finally:
        macos_mod.APP_DIRS = original
