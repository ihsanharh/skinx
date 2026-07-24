"""Minecraft Bedrock premium_cache discovery and pack listing."""

import os
from pathlib import Path


def find_premium_cache(minecraft_dir=None) -> Path:
    """Locate the premium_cache/skin_packs directory.

    Searches in order:
    1. *minecraft_dir* if provided (``<dir>/premium_cache/skin_packs``)
    2. Native Windows (%APPDATA%)
    3. BedrockOnLinux Wine prefix (https://github.com/Wyze3306/BedrockOnLinux)
    """
    if minecraft_dir:
        cache = Path(minecraft_dir) / "premium_cache/skin_packs"
        if cache.exists():
            return cache
        raise FileNotFoundError(f"Not found: {cache}")

    appdata = os.environ.get("APPDATA")
    if appdata:
        cache = Path(appdata) / "Minecraft Bedrock/premium_cache/skin_packs"
        if cache.exists():
            return cache
    bedrock_wine = Path.home() / ".local/share/bedrock-on-linux/compatdata/pfx/drive_c/users"
    for user_dir in bedrock_wine.glob("*"):
        cache = user_dir / "AppData/Roaming/Minecraft Bedrock/premium_cache/skin_packs"
        if cache.exists():
            return cache
    raise FileNotFoundError("Could not find premium_cache directory")


def list_packs(cache: Path):
    """List all valid skin packs in the cache directory.

    Returns list of (pack_dir, display_name) sorted alphabetically.
    """
    packs = []
    for pack_dir in cache.iterdir():
        if pack_dir.is_dir() and (pack_dir / "manifest.json").exists() and not pack_dir.name.endswith(".backup"):
            try:
                with open(pack_dir / "manifest.json") as f:
                    import json
                    manifest = json.load(f)
                name = manifest.get("header", {}).get("name", pack_dir.name)
            except Exception:
                name = pack_dir.name
            packs.append((pack_dir, name))
    packs.sort(key=lambda x: x[1].lower())
    return packs
