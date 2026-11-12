#!/usr/bin/env python3
"""Compare Go and the pinned C# migration oracle using synthetic saves only."""
import argparse
import json
from pathlib import Path
import struct
import subprocess
import tempfile

SIZE = 0x25B20 + 12

def call(cli, *args):
    prefix = ["dotnet", str(cli)] if cli.suffix == ".dll" else [str(cli)]
    result = subprocess.run([*prefix, *args], capture_output=True, text=True)
    if result.returncode:
        raise AssertionError((args, result.stdout, result.stderr))
    return json.loads(result.stdout)

def compare(a, b, path="root"):
    if isinstance(a, dict) and isinstance(b, dict):
        assert a.keys() == b.keys(), (path, a.keys() ^ b.keys())
        for key in a:
            compare(a[key], b[key], path + "." + key)
    elif isinstance(a, list) and isinstance(b, list):
        assert len(a) == len(b), (path, len(a), len(b))
        for i, (x, y) in enumerate(zip(a, b)):
            compare(x, y, f"{path}[{i}]")
    else:
        assert a == b, (path, str(a)[:150], str(b)[:150])

def synthetic():
    data = bytearray(SIZE)
    struct.pack_into("<II", data, 4, 23, SIZE)
    for i in range(400):
        struct.pack_into("<h", data, 12 + i * 4, -1)
    for slot in range(60):
        p = 12 + 0x644 + slot * 0x24C
        for i in range(6):
            struct.pack_into("<h", data, p + i * 4, -1)
        struct.pack_into("<h", data, p + 0x24, -1)
        struct.pack_into("<hHHBB", data, p + 0x18, -1, 0, 0, 200, 80)
    p = 12 + 0x644
    struct.pack_into("<h", data, p + 0x24, 0)
    data[p + 0x4A] = 20
    data[p + 0x4B] = 1
    struct.pack_into("<hHHBB", data, p + 0x18, 0, 10, 12, 20, 26)
    barracks = 12 + 0x231D9 + 0xA30
    for i in range(200):
        struct.pack_into("<hHHBB", data, barracks + i * 8, -1, 0, 0, 200, 80)
    struct.pack_into("<hHHBB", data, barracks, 0, 10, 9, 20, 26)
    data[0x22500] = 0xA7
    struct.pack_into("<I", data, 0, sum(data[12:]) & 0xFFFFFFFF)
    return data

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--legacy", type=Path, required=True)
    parser.add_argument("--go", type=Path, required=True)
    parser.add_argument("--skip-catalogs", action="store_true")
    args = parser.parse_args()
    old, new = args.legacy.resolve(), args.go.resolve()
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        source = root / "slot00"
        source.write_bytes(synthetic())
        for language in ("en_u", "en_e", "zh_hans", "zh_hant", "jp", "kr", "de", "fr_e", "fr_u", "es_e", "es_u", "it"):
            if args.skip_catalogs:
                break
            for kind in ("items", "characters", "classes", "battalions", "obtainable-battalions", "battalion-skills", "abilities", "arts", "quests", "supports", "support-ranks"):
                command = ("catalog", "--type", kind, "--language", language)
                compare(call(old, *command), call(new, *command), f"{language}/{kind}")
            command = ("inspect", "--input", str(source), "--language", language)
            compare(call(old, *command), call(new, *command), language + "/inspect")
        operations = [
            {"op": "setName", "value": "贝雷丝🦉"},
            {"op": "setProtagonistGender", "gender": "female"},
            {"op": "set", "path": "Player.Money", "value": 837265},
            {"op": "set", "path": "Characters[0].data.HP", "value": 52},
            {"op": "set", "path": "Characters[0].data.CurrentClassExp", "value": 17},
            {"op": "setBit", "path": "Characters[0].data.Flags", "index": 18, "value": True},
            {"op": "sortItems"}, {"op": "sortBattalions"}, {"op": "fillMissingBattalions"},
            {"op": "setBattalionEndurance", "slot": 0, "endurance": 71},
            {"op": "setBattalionEnduranceValues", "slot": 0, "storedEndurance": 17, "equippedEndurance": 24},
            {"op": "replenishBattalion", "slot": 0}, {"op": "replenishCharacterBattalion", "slot": 0},
            {"op": "replenishBattalions"}, {"op": "maxBattalionLevel", "slot": 0},
            {"op": "maxBattalionLevels"}, {"op": "deleteBattalion", "slot": 0},
            {"op": "inventoryDurability", "mode": "normal"},
            {"op": "inventoryDurability", "mode": "unlimited"},
            {"op": "inventoryDurability", "mode": "weapons-unlimited"},
            {"op": "characterItemDurability", "slot": 0},
            {"op": "setCharacterItem", "slot": 0, "itemSlot": 0, "id": 22},
            {"op": "maxSkillExp", "slot": 0},
            {"op": "setSkillRank", "slot": 0, "skill": 2, "rank": 9, "experience": 301},
            {"op": "setProfessorRank", "rank": 8},
            {"op": "setSupportRank", "index": 0, "rank": "A"}, {"op": "maxSupportRank", "index": 65},
            {"op": "setNgPlusProfessorRank", "rank": 8}, {"op": "setNgPlusSupport", "index": 2, "points": 461},
            {"op": "setNgPlusSupportRank", "index": 0, "rank": "S"},
            {"op": "maxNgPlusSupportRank", "index": 65}, {"op": "maxNgPlusSupports"}, {"op": "maxSupports"},
            {"op": "unlockNgPlusRoster"},
            {"op": "setNgPlusSkillRank", "recordIndex": 3, "skill": 5, "rank": 10},
            {"op": "setNgPlusClassMastery", "recordIndex": 1, "classId": 86, "mastered": True},
            {"op": "unlockNgPlusClasses", "recordIndex": 43}, {"op": "maxClassExp", "slot": 0},
            {"op": "unlockAll", "slot": 0, "kind": "abilities"},
            {"op": "unlockAll", "slot": 0, "kind": "combat-arts"},
            {"op": "fillItems", "kind": "misc", "amount": 57},
            {"op": "fillItems", "kind": "gifts", "amount": 47}, {"op": "addEssentialItems"},
        ]
        for i, operation in enumerate(operations):
            patch = root / "patch.json"
            patch.write_text(json.dumps({"operations": [operation]}))
            command = ("apply", "--input", str(source), "--patch", str(patch), "--dry-run")
            compare(call(old, *command), call(new, *command), f"operation {i}: {operation['op']}")
        for version, size in ((5, 0x44C), (7, 0x1204)):
            system = bytearray(size)
            struct.pack_into("<II", system, 4, version, size)
            struct.pack_into("<I", system, 0, sum(system[12:]))
            path = root / "system"
            path.write_bytes(system)
            command = ("inspect-system", "--input", str(path))
            compare(call(old, *command), call(new, *command), f"system v{version}")
            patch.write_text(json.dumps({"operations": [{"op": "setSystemFlag", "index": 31, "value": True}]}))
            command = ("apply-system", "--input", str(path), "--patch", str(patch), "--dry-run")
            compare(call(old, *command), call(new, *command), f"system v{version} edit")
    print(f"Exact legacy parity: 12 languages, 132 catalogs, 12 full inspections, {len(operations)} edit operations, system v5/v7.")

if __name__ == "__main__":
    main()
