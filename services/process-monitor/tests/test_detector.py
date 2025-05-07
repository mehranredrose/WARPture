"""Unit tests for the app detector and database."""
import pytest
from scanners.detector import lookup_app, make_app_id, APP_DATABASE


def test_lookup_by_bundle_id():
    result = lookup_app("", "org.mozilla.firefox")
    assert result is not None
    assert result["id"] == "firefox"
    assert result["name"] == "Firefox"


def test_lookup_by_name():
    result = lookup_app("Slack")
    assert result is not None
    assert result["id"] == "slack"


def test_lookup_unknown():
    result = lookup_app("SomeTotallyUnknownApp12345")
    assert result is None


def test_make_app_id_from_bundle():
    app_id = make_app_id("Firefox", "org.mozilla.firefox")
    assert "mozilla" in app_id or "firefox" in app_id


def test_make_app_id_from_name():
    app_id = make_app_id("My Cool App")
    assert app_id == "my-cool-app"


def test_make_app_id_no_spaces():
    app_id = make_app_id("App With Spaces")
    assert " " not in app_id


def test_app_database_has_required_fields():
    for app_id, info in APP_DATABASE.items():
        assert "name" in info, f"{app_id} missing 'name'"
        assert "bundleId" in info, f"{app_id} missing 'bundleId'"


def test_app_database_unique_bundle_ids():
    bundle_ids = [v["bundleId"] for v in APP_DATABASE.values() if v.get("bundleId")]
    assert len(bundle_ids) == len(set(bundle_ids)), "Duplicate bundleIds found"


def test_app_database_size():
    assert len(APP_DATABASE) >= 30, "Database should have at least 30 apps"
