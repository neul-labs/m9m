"""Binary download and caching for the m9m Python SDK."""

import os
import platform
import shutil
import stat
import subprocess
import sys
import urllib.request
from pathlib import Path


_GITHUB_REPO = "neul-labs/m9m"
_CACHE_DIR = Path.home() / ".m9m" / "bin"


def _get_platform() -> tuple[str, str]:
    """Return (os, arch) identifiers matching GitHub release assets."""
    system = platform.system().lower()
    machine = platform.machine().lower()

    os_name = {"darwin": "darwin", "linux": "linux", "windows": "windows"}.get(
        system, system
    )

    arch_name = {
        "x86_64": "amd64",
        "amd64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64",
    }.get(machine, machine)

    return os_name, arch_name


def _latest_release_version() -> str:
    """Fetch the latest release tag from GitHub (e.g. 'v0.2.0')."""
    try:
        import urllib.request
        import json

        url = f"https://api.github.com/repos/{_GITHUB_REPO}/releases/latest"
        req = urllib.request.Request(url, headers={"Accept": "application/vnd.github.v3+json"})
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read().decode())
            return data.get("tag_name", "v0.2.0")
    except Exception:
        return "v0.2.0"


def get_binary_path() -> str:
    """Return the path to the cached m9m binary, downloading if necessary."""
    os_name, arch_name = _get_platform()
    binary_name = "m9m.exe" if os_name == "windows" else "m9m"
    cached = _CACHE_DIR / binary_name

    if cached.exists():
        return str(cached)

    download_binary()
    return str(cached)


def download_binary(version: str | None = None) -> str:
    """Download the m9m binary for this platform and cache it."""
    os_name, arch_name = _get_platform()
    if version is None:
        version = _latest_release_version()
    if not version.startswith("v"):
        version = f"v{version}"

    binary_name = "m9m.exe" if os_name == "windows" else "m9m"
    artifact_name = f"m9m-{os_name}-{arch_name}"
    url = f"https://github.com/{_GITHUB_REPO}/releases/download/{version}/{artifact_name}"
    cached = _CACHE_DIR / binary_name

    _CACHE_DIR.mkdir(parents=True, exist_ok=True)

    # Download
    req = urllib.request.Request(url, headers={"User-Agent": "m9m-python-sdk/1.0"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        cached.write_bytes(resp.read())

    # Make executable (Unix)
    if os_name != "windows":
        st = os.stat(cached)
        os.chmod(cached, st.st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

    return str(cached)
