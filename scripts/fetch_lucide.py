#!/usr/bin/env python3
"""Vendor the editor's Lucide SVGs from Iconify; never fetch icons at runtime."""
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import urllib.request

ROOT = Path(__file__).resolve().parents[1] / "internal/app/icons"
NAMES = ("house", "package", "users", "swords", "flag", "heart", "database", "graduation-cap", "settings", "folder-open", "save", "info")

def fetch(name):
    request = urllib.request.Request(f"https://api.iconify.design/lucide/{name}.svg", headers={"User-Agent": "feth-save-editor"})
    with urllib.request.urlopen(request, timeout=30) as response:
        content = response.read()
    if b"<svg" not in content:
        raise ValueError(f"Invalid SVG: {name}")
    (ROOT / f"{name}.svg").write_bytes(content)

if __name__ == "__main__":
    ROOT.mkdir(parents=True, exist_ok=True)
    with ThreadPoolExecutor(max_workers=4) as pool:
        list(pool.map(fetch, NAMES))
    print(f"Vendored {len(NAMES)} Lucide icons via Iconify.")
