#!/usr/bin/env python3
"""Package the standalone Go GUI and CLI for each desktop platform."""

import argparse
import filecmp
import os
import plistlib
import shutil
import subprocess
import tempfile
import zipfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
APP_NAME = "FETH Save Editor.app"
WINDOWS_GUI_NAME = "FETH Save Editor.exe"


def merge_publishes(gui: Path, cli: Path, destination: Path, rid: str) -> None:
    """Collect GUI and CLI binaries, keeping matching files only once."""
    for published in (gui, cli):
        for source in published.rglob("*"):
            if not source.is_file():
                continue
            relative = source.relative_to(published)
            if relative.suffix.lower() == ".pdb":
                continue
            target = destination / relative
            if target.exists():
                if not filecmp.cmp(source, target, shallow=False):
                    raise ValueError(f"Conflicting GUI/CLI publish file: {relative}")
                continue
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)

    if rid != "win-x64":
        for path in destination.rglob("*"):
            if path.is_file() and (path.name in ("FethEditor.Gui", "FethEditor.Cli")
                                   or path.suffix == ".dylib" or ".so" in path.name):
                os.chmod(path, path.stat().st_mode | 0o111)


def make_macos_app(destination: Path, version: str) -> None:
    contents = destination / "Contents"
    executable_dir = contents / "MacOS"
    resources = contents / "Resources"
    resources.mkdir()

    iconset = contents / "Sothis.iconset"
    iconset.mkdir()
    portrait = ROOT / "assets/sothis-portrait.png"
    for name, pixels in (
        ("icon_16x16", 16),
        ("icon_16x16@2x", 32),
        ("icon_32x32", 32),
        ("icon_32x32@2x", 64),
        ("icon_128x128", 128),
        ("icon_128x128@2x", 256),
        ("icon_256x256", 256),
        ("icon_256x256@2x", 512),
        ("icon_512x512", 512),
        ("icon_512x512@2x", 1024),
    ):
        subprocess.run(
            ["sips", "-z", str(pixels), str(pixels), str(portrait),
             "--out", str(iconset / f"{name}.png")],
            check=True,
            capture_output=True,
        )
    subprocess.run(
        ["iconutil", "-c", "icns", str(iconset),
         "-o", str(resources / "Sothis.icns")],
        check=True,
    )
    shutil.rmtree(iconset)

    with (contents / "Info.plist").open("wb") as output:
        plistlib.dump(
            {
                "CFBundleDevelopmentRegion": "en",
                "CFBundleDisplayName": "FETH Save Editor",
                "CFBundleExecutable": "FethEditor.Gui",
                "CFBundleIconFile": "Sothis.icns",
                "CFBundleIdentifier": "com.jinghaihan.feth-save-editor",
                "CFBundleName": "FETH Save Editor",
                "CFBundlePackageType": "APPL",
                "CFBundleShortVersionString": version,
                "CFBundleVersion": version,
                "NSHighResolutionCapable": True,
            },
            output,
        )
    subprocess.run(
        ["codesign", "--force", "--deep", "--sign", "-", "--timestamp=none", str(destination)],
        check=True,
    )
    subprocess.run(
        ["codesign", "--verify", "--deep", "--strict", str(destination)],
        check=True,
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("rid", choices=("win-x64", "osx-arm64", "osx-x64", "linux-x64"))
    parser.add_argument("--gui", type=Path)
    parser.add_argument("--cli", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()

    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    gui = args.gui or ROOT / "dist/gui" / args.rid
    cli = args.cli or gui
    output = args.output or ROOT / "dist" / f"feth-save-editor-v{version}-{args.rid}.zip"
    suffix = ".exe" if args.rid == "win-x64" else ""
    for executable in (gui / f"FethEditor.Gui{suffix}", cli / f"FethEditor.Cli{suffix}"):
        if not executable.is_file():
            parser.error(f"Missing published executable: {executable}")

    with tempfile.TemporaryDirectory(prefix="feth-release-") as temporary:
        staging = Path(temporary)
        if args.rid.startswith("osx-"):
            app = staging / APP_NAME
            merge_publishes(gui, cli, app / "Contents/MacOS", args.rid)
            make_macos_app(app, version)
            gui_binary = staging / APP_NAME / "Contents/MacOS/FethEditor.Gui"
            (staging / "Cli").mkdir()
            cli_binary = staging / "Cli/FethEditor.Cli"
            cli_binary.write_text(
                '#!/bin/sh\n'
                'app_dir=$(CDPATH= cd "$(dirname "$0")/.." && pwd) || exit 1\n'
                'exec "$app_dir/FETH Save Editor.app/Contents/MacOS/FethEditor.Cli" "$@"\n',
                encoding="utf-8",
            )
            os.chmod(cli_binary, 0o755)
        else:
            merge_publishes(gui, cli, staging, args.rid)
            if args.rid == "win-x64":
                gui_binary = staging / WINDOWS_GUI_NAME
                (staging / "FethEditor.Gui.exe").rename(gui_binary)
            else:
                gui_binary = staging / "FethEditor.Gui"
            cli_binary = staging / f"FethEditor.Cli{suffix}"

        shutil.copy2(ROOT / "LICENSE", staging / "LICENSE")
        shutil.copytree(ROOT / "licenses", staging / "licenses")
        output.parent.mkdir(parents=True, exist_ok=True)
        if args.rid.startswith("osx-"):
            # Keep macOS bundle metadata, signatures and executable permissions.
            subprocess.run(
                ["ditto", "-c", "-k", "--sequesterRsrc", str(staging), str(output)],
                check=True,
            )
        else:
            with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
                for path in sorted(staging.rglob("*")):
                    archive.write(path, path.relative_to(staging))
        with zipfile.ZipFile(output) as archive:
            expected = [
                gui_binary.relative_to(staging).as_posix(),
                cli_binary.relative_to(staging).as_posix(),
            ]
            for name in expected:
                info = archive.getinfo(name)
                if args.rid != "win-x64" and not (info.external_attr >> 16) & 0o111:
                    raise RuntimeError(f"Executable permission missing from archive: {name}")
        print(output)


if __name__ == "__main__":
    main()
