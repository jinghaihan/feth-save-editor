#!/usr/bin/env python3
"""Exercise the CLI against a synthetic v23 save without publishing user data."""

from __future__ import annotations

import argparse
import json
import struct
import subprocess
import tempfile
from pathlib import Path


SAVE_SIZE = 0x25B20 + 12


def checksum(data: bytes) -> int:
    return sum(data[12:]) & 0xFFFFFFFF


def run(cli: Path, *args: str, success: bool = True) -> dict:
    command = [str(cli)]
    result = subprocess.run([*command, *args], capture_output=True, text=True)
    if (result.returncode == 0) != success:
        raise AssertionError((result.returncode, result.stdout, result.stderr))
    return json.loads(result.stdout if success else result.stderr)


def verify_jeritza_classes(cli: Path, directory: Path, raw: bytes) -> None:
    roster = 12 + 0x644
    data = bytearray(raw)
    struct.pack_into("<h", data, roster + 0x24, 1045)
    data[roster + 0x4A] = 30
    data[roster + 0x4B] = 91
    struct.pack_into("<H", data, roster + 0x112 + 2, 7)
    struct.pack_into("<I", data, 0, checksum(data))
    source = directory / "jeritza-source"
    source.write_bytes(data)
    for name, operations in [
        ("single", [{"op": "unlockNgPlusClasses", "recordIndex": 43}]),
        ("all", [{"op": "unlockNgPlusRoster"}]),
        ("preserved", [
            {"op": "setNgPlusClassMastery", "recordIndex": 43, "classId": 1, "mastered": True},
            {"op": "setNgPlusClassMastery", "recordIndex": 43, "classId": 60, "mastered": True},
            {"op": "unlockNgPlusClasses", "recordIndex": 43},
            {"op": "unlockNgPlusRoster"},
        ]),
        ("experience", [{"op": "maxClassExp", "slot": 0}]),
    ]:
        patch = directory / f"jeritza-{name}.json"
        patch.write_text(json.dumps({"operations": operations}), encoding="utf-8")
        target = directory / f"jeritza-{name}"
        run(cli, "apply", "--input", str(source), "--patch", str(patch), "--output", str(target))
        result = target.read_bytes()
        assert len(result) == len(data) and checksum(result) == struct.unpack_from("<I", result)[0]
        if name == "experience":
            assert struct.unpack_from("<H", result, roster + 0x112)[0] == 20
            assert struct.unpack_from("<H", result, roster + 0x112 + 2)[0] == 7
            assert struct.unpack_from("<H", result, roster + 0x112 + 91 * 2)[0] == 200
            assert struct.unpack_from("<H", result, roster + 0x48)[0] == 200
            allowed = set(range(4)) | set(range(roster + 0x112, roster + 0x112 + 100 * 2))
            allowed.update((roster + 0x48, roster + 0x49))
            assert all(before == after or index in allowed
                       for index, (before, after) in enumerate(zip(data, result)))
        else:
            history = run(cli, "inspect", "--input", str(target), "--section", "inheritance")["inheritance"]
            classes = history["characters"][43]["masteredClassIds"]
            assert 0 in classes and 91 in classes
            assert (1 in classes) == (name == "preserved")
            assert (60 in classes) == (name == "preserved")
        assert run(cli, "apply", "--input", str(target), "--patch", str(patch), "--dry-run")["changedBytes"] == 0
    assert source.read_bytes() == data


def verify_battalion_endurance(cli: Path, directory: Path, raw: bytes) -> None:
    data = bytearray(raw)
    barracks = 12 + 0x231D9 + 0xA30
    roster = 12 + 0x644
    for slot in range(200):
        struct.pack_into("<hHHBB", data, barracks + slot * 8, -1, 0, 99, 200, 80)
    entries = [
        (1, 400, 60, 115, 26),  # Equipped Essar: its live endurance is 44.
        (-1, 123, 10, 115, 26),  # A second Essar, wounded and unequipped.
        (-1, 88, 0, 1, 0),  # Depleted, unequipped battalion.
        (-1, 55, 13, 150, 5),  # Unknown type: never guess its maximum.
        (2, 35, 60, 115, 26),  # Same type, equipped by a different character.
        (-1, 600, 30, 0, 4),  # Already full; preserve above-target experience.
        (4, 22, 11, 115, 26),  # Stale owner: the matching character is inactive.
    ]
    for slot, entry in enumerate(entries):
        struct.pack_into("<hHHBB", data, barracks + slot * 8, *entry)
    for slot, (unit, level, endurance, kind) in enumerate([
        (1, 5, 44, 115), (2, 1, 12, 115), (3, 1, 7, 5),
        (4, 0, 9, 115), (5, 1, 7, 150),
    ]):
        start = roster + slot * 0x24C
        struct.pack_into("<hHHBB", data, start + 0x18, unit, 77, endurance, kind, 26)
        struct.pack_into("<h", data, start + 0x24, unit)
        data[start + 0x4A] = level
    struct.pack_into("<I", data, 0, checksum(data))
    source = directory / "endurance-source"
    source.write_bytes(data)
    snapshot = run(cli, "inspect", "--input", str(source), "--section", "battalions")["battalions"]
    assert snapshot[0]["Stamina"] == 44 and snapshot[0]["storedStamina"] == 60
    assert snapshot[0]["maximumEndurance"] == 60
    assert snapshot[1]["Stamina"] == 10 and snapshot[3]["maximumEndurance"] is None

    def apply(name: str, operations: list[dict], input_path: Path = source) -> bytes:
        patch = directory / f"{name}.json"
        patch.write_text(json.dumps({"operations": operations}), encoding="utf-8")
        target = directory / name
        run(cli, "apply", "--input", str(input_path), "--patch", str(patch), "--output", str(target))
        return target.read_bytes()

    def unchanged_except(result: bytes, offsets: list[int]) -> None:
        allowed = set(range(4)) | {byte for offset in offsets for byte in (offset, offset + 1)}
        assert all(a == b or index in allowed for index, (a, b) in enumerate(zip(data, result)))
        assert len(result) == len(data)
        assert struct.unpack_from("<I", result)[0] == checksum(result)

    single = apply("endurance-single", [{"op": "replenishBattalion", "slot": 0}])
    assert struct.unpack_from("<H", single, roster + 0x1C)[0] == 60
    unchanged_except(single, [barracks + 4, roster + 0x1C])
    roster_single = apply("endurance-roster", [{"op": "replenishCharacterBattalion", "slot": 0}])
    assert roster_single == single
    orphan = apply("endurance-orphan", [{"op": "replenishCharacterBattalion", "slot": 2}])
    assert struct.unpack_from("<H", orphan, roster + 2 * 0x24C + 0x1C)[0] == 60
    unchanged_except(orphan, [roster + 2 * 0x24C + 0x1C])
    unequipped = apply("endurance-unequipped", [{"op": "replenishBattalion", "slot": 1}])
    assert struct.unpack_from("<H", unequipped, barracks + 8 + 4)[0] == 60
    unchanged_except(unequipped, [barracks + 8 + 4])
    manual = apply("endurance-manual", [{"op": "setBattalionEndurance", "slot": 0, "endurance": 20}])
    assert struct.unpack_from("<H", manual, roster + 0x1C)[0] == 20
    assert struct.unpack_from("<H", manual, barracks + 4)[0] == 20
    unchanged_except(manual, [barracks + 4, roster + 0x1C])
    independent = apply("endurance-independent", [{"op": "setBattalionEnduranceValues", "slot": 0,
                                                  "storedEndurance": 100, "equippedEndurance": 37}])
    assert struct.unpack_from("<H", independent, barracks + 4)[0] == 100
    assert struct.unpack_from("<H", independent, roster + 0x1C)[0] == 37
    unchanged_except(independent, [barracks + 4, roster + 0x1C])
    stored_only = apply("endurance-stored-only", [{"op": "setBattalionEnduranceValues", "slot": 0,
                                                  "storedEndurance": 80}])
    assert struct.unpack_from("<H", stored_only, barracks + 4)[0] == 80
    unchanged_except(stored_only, [barracks + 4])
    wounded = apply("endurance-wounded-edit", [{"op": "setBattalionEnduranceValues", "slot": 1,
                                               "storedEndurance": 6}])
    assert struct.unpack_from("<H", wounded, barracks + 8 + 4)[0] == 6
    unchanged_except(wounded, [barracks + 8 + 4])

    bulk = apply("endurance-bulk", [{"op": "replenishBattalions"}])
    expected = [60, 60, 30, 13, 60, 30, 60]
    assert [struct.unpack_from("<H", bulk, barracks + slot * 8 + 4)[0]
            for slot in range(len(entries))] == expected
    assert [struct.unpack_from("<H", bulk, roster + slot * 0x24C + 0x1C)[0]
            for slot in range(5)] == [60, 60, 60, 9, 7]
    unchanged_except(bulk, [barracks + slot * 8 + 4 for slot in (0, 1, 2, 4, 6)]
                     + [roster + slot * 0x24C + 0x1C for slot in (0, 1, 2)])
    assert run(cli, "apply", "--input", str(directory / "endurance-bulk"),
               "--patch", str(directory / "endurance-bulk.json"), "--dry-run")["changedBytes"] == 0
    assert source.read_bytes() == data

    maximum = apply("level-single", [{"op": "maxBattalionLevel", "slot": 0}])
    assert struct.unpack_from("<H", maximum, roster + 0x1A)[0] == 400
    unchanged_except(maximum, [barracks + 2, roster + 0x1A])
    bench_maximum = apply("level-unequipped", [{"op": "maxBattalionLevel", "slot": 1}])
    assert struct.unpack_from("<H", bench_maximum, barracks + 8 + 2)[0] == 400
    unchanged_except(bench_maximum, [barracks + 8 + 2])
    all_maximum = apply("level-bulk", [{"op": "maxBattalionLevels"}])
    assert [struct.unpack_from("<H", all_maximum, barracks + slot * 8 + 2)[0]
            for slot in range(len(entries))] == [400, 400, 400, 400, 400, 600, 400]
    assert [struct.unpack_from("<H", all_maximum, roster + slot * 0x24C + 0x1A)[0]
            for slot in range(5)] == [400, 400, 400, 77, 400]
    unchanged_except(all_maximum, [barracks + slot * 8 + 2 for slot in range(len(entries))]
                     + [roster + slot * 0x24C + 0x1A for slot in (0, 1, 2, 4)])
    assert run(cli, "apply", "--input", str(directory / "level-bulk"),
               "--patch", str(directory / "level-bulk.json"), "--dry-run")["changedBytes"] == 0

    empty = (-1, 0, 0, 200, 80)
    deleted = apply("delete-equipped", [{"op": "deleteBattalion", "slot": 0}])
    assert struct.unpack_from("<hHHBB", deleted, barracks) == empty
    assert struct.unpack_from("<hHHBB", deleted, roster + 0x18) == empty
    unchanged_except(deleted, [barracks + field for field in range(0, 8, 2)]
                     + [roster + 0x18 + field for field in range(0, 8, 2)])
    bench_deleted = apply("delete-unequipped", [{"op": "deleteBattalion", "slot": 1}])
    assert struct.unpack_from("<hHHBB", bench_deleted, barracks + 8) == empty
    unchanged_except(bench_deleted, [barracks + 8 + field for field in range(0, 8, 2)])
    stale_deleted = apply("delete-inactive", [{"op": "deleteBattalion", "slot": 6}])
    assert struct.unpack_from("<hHHBB", stale_deleted, barracks + 6 * 8) == empty
    unchanged_except(stale_deleted, [barracks + 6 * 8 + field for field in range(0, 8, 2)])

    for name, operation in [
        ("unknown", {"op": "replenishBattalion", "slot": 3}),
        ("empty", {"op": "replenishBattalion", "slot": 100}),
        ("out-of-range", {"op": "replenishBattalion", "slot": 200}),
        ("negative", {"op": "setBattalionEndurance", "slot": 0, "endurance": -1}),
        ("overflow", {"op": "setBattalionEndurance", "slot": 0, "endurance": 65536}),
        ("roster-inactive", {"op": "replenishCharacterBattalion", "slot": 3}),
        ("roster-unknown", {"op": "replenishCharacterBattalion", "slot": 4}),
        ("roster-out-of-range", {"op": "replenishCharacterBattalion", "slot": 500}),
        ("not-equipped", {"op": "setBattalionEnduranceValues", "slot": 1,
                          "storedEndurance": 100, "equippedEndurance": 60}),
        ("stored-overflow", {"op": "setBattalionEnduranceValues", "slot": 0,
                             "storedEndurance": 65536}),
        ("level-empty", {"op": "maxBattalionLevel", "slot": 100}),
        ("level-out-of-range", {"op": "maxBattalionLevel", "slot": 200}),
        ("delete-empty", {"op": "deleteBattalion", "slot": 100}),
        ("delete-negative", {"op": "deleteBattalion", "slot": -1}),
        ("delete-out-of-range", {"op": "deleteBattalion", "slot": 200}),
    ]:
        patch = directory / f"endurance-{name}.json"
        patch.write_text(json.dumps({"operations": [operation]}), encoding="utf-8")
        output = directory / f"endurance-{name}-output"
        run(cli, "apply", "--input", str(source), "--patch", str(patch), "--output", str(output), success=False)
        assert not output.exists() and source.read_bytes() == data

    ambiguous = bytearray(data)
    struct.pack_into("<h", ambiguous, barracks + 8, 1)
    struct.pack_into("<I", ambiguous, 0, checksum(ambiguous))
    ambiguous_source = directory / "endurance-ambiguous"
    ambiguous_source.write_bytes(ambiguous)
    for patch_name in ("endurance-single.json", "endurance-bulk.json", "endurance-roster.json",
                       "level-single.json", "level-bulk.json", "delete-equipped.json"):
        output = directory / f"ambiguous-{patch_name}"
        error = run(cli, "apply", "--input", str(ambiguous_source), "--patch", str(directory / patch_name),
                    "--output", str(output), success=False)
        assert "uniquely matched" in error["error"] and not output.exists()
    assert ambiguous_source.read_bytes() == ambiguous


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cli", type=Path, required=True)
    cli = parser.parse_args().cli.resolve()

    with tempfile.TemporaryDirectory() as temporary:
        directory = Path(temporary)
        source = directory / "slot00"
        target = directory / "edited-slot00"
        patch_file = directory / "patch.json"
        raw = bytearray(SAVE_SIZE)
        struct.pack_into("<II", raw, 4, 23, SAVE_SIZE)
        for slot in range(400):
            struct.pack_into("<h", raw, 12 + slot * 4, -1)
        for item_slot in range(6):
            struct.pack_into("<h", raw, 12 + 0x644 + item_slot * 4, -1)
        sentinel = 12 + 0x644 + 0x2A
        raw[sentinel] = 0xA5
        player = 12 + 0x231D9
        struct.pack_into("<H", raw, player + 0x1576, 1001)
        raw[player + 0x17CE] = 9
        raw[player + 0x17D8] = 11
        class_start = player + 0x17D8 + 45 * 11
        raw[class_start + 42 // 8] |= 1 << (42 % 8)
        raw[class_start + 38 * 13 + 42 // 8] |= 1 << (42 % 8)
        struct.pack_into("<I", raw, 0, checksum(raw))
        source.write_bytes(raw)

        summary = run(cli, "inspect", "--input", str(source), "--section", "summary")
        assert summary["summary"]["Money"] == 0
        history = run(cli, "inspect", "--input", str(source), "--section", "inheritance")["inheritance"]
        assert history["professorRank"] == 9
        assert history["supports"][0]["maxPoints"] == 1001
        assert history["characters"][0]["skillRanks"][0] == 11
        assert 42 in history["characters"][0]["masteredClassIds"]
        assert 42 in history["characters"][38]["masteredClassIds"]
        assert history["characters"][30]["name"] == "Gilbert"
        assert [history["characters"][index]["name"] for index in range(38, 45)] == [
            "Yuri", "Balthus", "Constance", "Hapi", "Aelfric", "Jeritza", "Anna"]
        assert history["supports"][0]["name"]
        assert history["characters"][0]["name"]
        assert run(cli, "get", "--input", str(source), "--path", "Items[0].Id")["value"] == -1
        catalog = run(cli, "catalog", "--type", "classes")
        assert len(catalog) == 101  # 100 classes plus the GUI's "none" sentinel.
        assert run(cli, "catalog", "--type", "classes", "--id", "0")["details"]
        assert run(cli, "catalog", "--type", "items", "--id", "22")["details"]
        assert run(cli, "catalog", "--type", "battalion-skills")
        battalion_catalog = run(cli, "catalog", "--type", "obtainable-battalions")
        assert len(battalion_catalog) == 128
        assert len({entry["id"] for entry in battalion_catalog}) == 128
        assert battalion_catalog[0] == {
            "id": 0, "name": "Church of Seiros Soldiers",
            "experience": 400, "stamina": 30, "skill": 4,
        }
        assert run(cli, "catalog", "--type", "support-ranks")[-1] == {"rank": "S", "points": 1001}
        support_catalog = run(cli, "catalog", "--type", "supports")
        assert support_catalog[1]["maximumRank"] == "S"
        assert support_catalog[65]["ranks"] == ["None", "C", "B"]

        battalion_raw = bytearray(raw)
        battalion_start = player + 0xA30
        for slot in range(200):
            struct.pack_into("<hHHBB", battalion_raw, battalion_start + slot * 8,
                             -1, 0, 0, 200, 0)
        existing_battalion = (-1, 77, 43, 20, 1)
        struct.pack_into("<hHHBB", battalion_raw, battalion_start, *existing_battalion)
        character_start = 12 + 0x644
        battalion_raw[character_start + 0x4A] = 1
        struct.pack_into("<hHHBB", battalion_raw, character_start + 0x18,
                         -1, 11, 55, 10, 3)
        struct.pack_into("<I", battalion_raw, 0, checksum(battalion_raw))
        battalion_source = directory / "battalion-source"
        battalion_source.write_bytes(battalion_raw)
        battalion_patch = directory / "battalion-patch.json"
        battalion_patch.write_text(json.dumps({"operations": [
            {"op": "fillMissingBattalions"},
        ]}), encoding="utf-8")
        battalion_target = directory / "all-battalions"
        run(cli, "apply", "--input", str(battalion_source), "--patch", str(battalion_patch),
            "--output", str(battalion_target))
        battalion_data = battalion_target.read_bytes()
        entries = [struct.unpack_from("<hHHBB", battalion_data, battalion_start + slot * 8)
                   for slot in range(200)]
        assert next(entry for entry in entries if entry[3] == 20) == existing_battalion
        owned_types = {entry[3] for entry in entries if entry[3] < 200}
        assert len(owned_types) == 127
        assert 10 not in owned_types  # Equipped by a character, not duplicated in barracks.
        assert owned_types | {10} == {entry["id"] for entry in battalion_catalog}
        assert next(entry for entry in entries if entry[3] == 0) == (-1, 400, 30, 0, 4)
        assert battalion_source.read_bytes() == battalion_raw
        assert run(cli, "apply", "--input", str(battalion_target),
                   "--patch", str(battalion_patch), "--dry-run")["changedBytes"] == 0

        full_battalion_raw = bytearray(battalion_raw)
        for slot in range(200):
            struct.pack_into("<hHHBB", full_battalion_raw, battalion_start + slot * 8,
                             *existing_battalion)
        struct.pack_into("<I", full_battalion_raw, 0, checksum(full_battalion_raw))
        full_battalion_source = directory / "full-battalion-source"
        full_battalion_source.write_bytes(full_battalion_raw)
        full_battalion_target = directory / "full-battalion-target"
        error = run(cli, "apply", "--input", str(full_battalion_source),
                    "--patch", str(battalion_patch), "--output", str(full_battalion_target),
                    success=False)
        assert "free battalion slots" in error["error"]
        assert not full_battalion_target.exists()
        assert full_battalion_source.read_bytes() == full_battalion_raw

        verify_battalion_endurance(cli, directory, raw)
        verify_jeritza_classes(cli, directory, raw)

        rank_patch = directory / "rank.json"
        rank_patch.write_text(json.dumps({"operations": [
            {"op": "setProfessorRank", "rank": 9},
            {"op": "setSupportRank", "index": 1, "rank": "A"},
            {"op": "setNgPlusSupportRank", "index": 2, "rank": "S"},
        ]}), encoding="utf-8")
        rank_target = directory / "rank-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(rank_patch),
            "--output", str(rank_target))
        assert run(cli, "get", "--input", str(rank_target),
                   "--path", "Activities.InstructExp")["value"] == 44500
        assert run(cli, "inspect", "--input", str(rank_target),
                   "--section", "supports")["supports"][1]["rank"] == "A"
        assert run(cli, "inspect", "--input", str(rank_target),
                   "--section", "inheritance")["inheritance"]["supports"][2]["maxPoints"] == 1001

        invalid_rank_patch = directory / "invalid-support-rank.json"
        invalid_rank_patch.write_text(json.dumps({"operations": [
            {"op": "setSupportRank", "index": 1, "rank": "A+"},
        ]}), encoding="utf-8")
        invalid_rank_target = directory / "invalid-support-rank-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(invalid_rank_patch),
            "--output", str(invalid_rank_target), success=False)
        assert not invalid_rank_target.exists()

        max_rank_patch = directory / "max-support-ranks.json"
        max_rank_patch.write_text(json.dumps({"operations": [
            {"op": "maxSupportRank", "index": 65},
            {"op": "maxNgPlusSupportRank", "index": 65},
            {"op": "maxNgPlusSupports"},
        ]}), encoding="utf-8")
        max_rank_target = directory / "max-support-ranks-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(max_rank_patch),
            "--output", str(max_rank_target))
        assert run(cli, "inspect", "--input", str(max_rank_target),
                   "--section", "supports")["supports"][65]["rank"] == "B"
        max_history = run(cli, "inspect", "--input", str(max_rank_target),
                          "--section", "inheritance")["inheritance"]["supports"]
        assert max_history[0]["maxPoints"] == 1001
        assert max_history[35]["maxPoints"] == 0
        assert max_history[62]["maxPoints"] == 801
        assert max_history[65]["maxPoints"] == 301

        bulk_patch = directory / "bulk-unlocks.json"
        bulk_patch.write_text(json.dumps({"operations": [
            {"op": "set", "path": "Player.CharacterSupportValues[0]", "value": 1200},
            {"op": "maxSupports"},
            {"op": "maxNgPlusSupports"},
            {"op": "unlockNgPlusRoster"},
        ]}), encoding="utf-8")
        bulk_target = directory / "bulk-unlocked-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(bulk_patch),
            "--output", str(bulk_target))
        current_supports = run(cli, "inspect", "--input", str(bulk_target),
                               "--section", "supports")["supports"]
        assert current_supports[0]["rank"] == "S"
        assert current_supports[0]["value"] == 1200
        assert current_supports[65]["rank"] == "B"
        assert current_supports[35]["value"] == 0
        bulk_history = run(cli, "inspect", "--input", str(bulk_target),
                           "--section", "inheritance")["inheritance"]
        assert bulk_history["supports"][0]["maxPoints"] == 1001
        assert bulk_history["supports"][65]["maxPoints"] == 301
        assert bulk_history["supports"][35]["maxPoints"] == 0
        for record in [*range(35), *range(38, 42), 43, 44]:
            assert bulk_history["characters"][record]["skillRanks"] == [11] * 11
        for record in [35, 36, 37, 42]:
            assert bulk_history["characters"][record]["skillRanks"] == [0] * 11
        assert 42 in bulk_history["characters"][0]["masteredClassIds"]
        assert 54 not in bulk_history["characters"][0]["masteredClassIds"]
        assert 54 in bulk_history["characters"][1]["masteredClassIds"]
        assert 40 in bulk_history["characters"][2]["masteredClassIds"]
        assert 44 in bulk_history["characters"][3]["masteredClassIds"]
        assert 17 in bulk_history["characters"][4]["masteredClassIds"]
        assert 91 in bulk_history["characters"][43]["masteredClassIds"]
        assert 23 not in bulk_history["characters"][1]["masteredClassIds"]
        assert run(cli, "apply", "--input", str(bulk_target),
                   "--patch", str(bulk_patch), "--dry-run")["changedBytes"] == 0

        item_patch = directory / "character-item.json"
        item_patch.write_text(json.dumps({"operations": [
            {"op": "setCharacterItem", "slot": 0, "itemSlot": 0, "id": 22},
        ]}), encoding="utf-8")
        item_target = directory / "character-item-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(item_patch),
            "--output", str(item_target))
        item_bytes = item_target.read_bytes()
        assert struct.unpack_from("<h", item_bytes, 12 + 0x644)[0] == 22
        assert item_bytes[12 + 0x644 + 0x87] == 1
        assert item_bytes[12 + 0x644 + 2] > 0
        assert run(cli, "inspect", "--input", str(item_target),
                   "--section", "characters")["characters"][0]["ItemCount"] == 1
        invalid_item = directory / "invalid-character-item.json"
        invalid_item.write_text(json.dumps({"operations": [
            {"op": "setCharacterItem", "slot": 0, "itemSlot": 2, "id": 22},
        ]}), encoding="utf-8")
        run(cli, "apply", "--input", str(source), "--patch", str(invalid_item),
            "--output", str(directory / "invalid-character-item"), success=False)
        invalid_stat = directory / "invalid-stat.json"
        invalid_stat.write_text(json.dumps({"operations": [
            {"op": "set", "path": "Characters[0].data.Movement", "value": 250},
        ]}), encoding="utf-8")
        stat_error = run(cli, "apply", "--input", str(source), "--patch", str(invalid_stat),
                         "--output", str(directory / "invalid-stat"), success=False)
        assert "maximum" in stat_error["error"]

        system_raw = bytearray(0x1204)
        struct.pack_into("<II", system_raw, 4, 7, len(system_raw))
        struct.pack_into("<I", system_raw, 0, checksum(system_raw))
        system_source = directory / "system"
        system_source.write_bytes(system_raw)
        system_info = run(cli, "inspect-system", "--input", str(system_source),
                          "--section", "flags")
        assert system_info["checksumValid"] is True
        assert len(system_info["flags"]) == 2464
        assert system_info["flags"][8]["enabled"] is False
        system_patch = directory / "system-patch.json"
        system_patch.write_text(json.dumps({
            "expectedSha256": system_info["sha256"],
            "operations": [{"op": "setSystemFlag", "index": 8, "value": True}],
        }), encoding="utf-8")
        system_target = directory / "edited-system"
        system_dry = run(cli, "apply-system", "--input", str(system_source),
                         "--patch", str(system_patch), "--dry-run")
        assert system_dry["changedFlags"] == [8]
        assert not system_target.exists()
        run(cli, "apply-system", "--input", str(system_source), "--patch", str(system_patch),
            "--output", str(system_target))
        assert system_source.read_bytes() == system_raw
        assert run(cli, "inspect-system", "--input", str(system_target),
                   "--section", "flags")["flags"][8]["enabled"] is True
        assert struct.unpack_from("<I", system_target.read_bytes(), 0)[0] == checksum(system_target.read_bytes())
        inplace_system = directory / "inplace-system"
        inplace_system.write_bytes(system_raw)
        written = run(cli, "apply-system", "--input", str(inplace_system),
                      "--patch", str(system_patch), "--in-place")
        assert Path(written["backup"]).read_bytes() == system_raw

        patch_file.write_text(json.dumps({
            "expectedSha256": summary["sha256"],
            "operations": [
                {"op": "set", "path": "Player.Money", "value": 12345},
                {"op": "set", "path": "Activities.Reputation", "value": 321},
                {"op": "set", "path": "Characters[0].data.SkillExp[0]", "value": 39},
                {"op": "setSkillRank", "slot": 0, "skill": 1, "rank": 11, "experience": 0},
                {"op": "setBit", "path": "Characters[0].data.Abilities", "index": 3, "value": True},
                {"op": "set", "path": "Items[0].Id", "value": 65},
            ],
        }), encoding="utf-8")
        dry = run(cli, "apply", "--input", str(source), "--patch", str(patch_file), "--dry-run")
        assert dry["changedBytes"] > 0
        assert not target.exists()

        applied = run(cli, "apply", "--input", str(source), "--patch", str(patch_file), "--output", str(target))
        assert applied["changedBytes"] > 0
        edited = target.read_bytes()
        assert source.read_bytes() == raw
        assert edited[sentinel] == 0xA5
        assert struct.unpack_from("<I", edited, 0)[0] == checksum(edited)
        assert struct.unpack_from("<I", edited, 12 + 0x640)[0] == 1
        assert struct.unpack_from("<h", edited, 12)[0] == 65
        assert struct.unpack_from("<H", edited, 12 + 0x644 + 0x32)[0] == 39
        assert struct.unpack_from("<H", edited, 12 + 0x644 + 0xFC)[0] == 39
        assert edited[12 + 0x644 + 0x88 + 1] == 11
        assert edited[12 + 0x644 + 0x1DC + 1] == 11
        assert edited[12 + 0x644 + 0x61] & 8
        assert struct.unpack_from("<I", edited, 12 + 0x231D9 + 0x1074)[0] == 12345
        assert struct.unpack_from("<I", edited, 12 + 0x250A1 + 0xC)[0] == 321

        class_flags_patch = directory / "class-flags.json"
        class_flags_patch.write_text(json.dumps({"operations": [
            {"op": "setBit", "path": "Characters[0].data.ClassUnlockFlags", "index": 17, "value": True},
            {"op": "setBit", "path": "Characters[0].data.ClassFlags", "index": 1, "value": True},
        ]}), encoding="utf-8")
        class_flags_target = directory / "class-flags-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(class_flags_patch),
            "--output", str(class_flags_target))
        class_flags_bytes = class_flags_target.read_bytes()
        unlock_offset = 12 + 0x644 + 0xD3 + 17 // 8
        class_flag_offset = 12 + 0x644 + 0xDF
        assert class_flags_bytes[unlock_offset] == raw[unlock_offset] | (1 << (17 % 8))
        assert class_flags_bytes[class_flag_offset] == raw[class_flag_offset] | (1 << 1)
        assert all(before == after for index, (before, after) in enumerate(zip(raw, class_flags_bytes))
                   if index not in {0, 1, 2, 3, unlock_offset, class_flag_offset})
        assert struct.unpack_from("<I", class_flags_bytes, 0)[0] == checksum(class_flags_bytes)

        invalid_checksum = bytearray(raw)
        invalid_checksum[12 + 0x231D9 + 0x1074] ^= 1
        invalid_source = directory / "checksum-mismatch-slot00"
        invalid_source.write_bytes(invalid_checksum)
        invalid_summary = run(cli, "inspect", "--input", str(invalid_source), "--section", "summary")
        assert invalid_summary["checksumValid"] is False
        repair_patch = directory / "repair-checksum.json"
        repair_patch.write_text(json.dumps({"operations": [
            {"op": "set", "path": "Player.Money", "value": 54321},
        ]}), encoding="utf-8")
        repaired_path = directory / "repaired-slot00"
        run(cli, "apply", "--input", str(invalid_source), "--patch", str(repair_patch),
            "--output", str(repaired_path))
        repaired = repaired_path.read_bytes()
        assert checksum(repaired) == struct.unpack_from("<I", repaired, 0)[0]
        assert invalid_source.read_bytes() == invalid_checksum

        ng_patch = directory / "ng-plus.json"
        ng_patch.write_text(json.dumps({"expectedSha256": summary["sha256"], "operations": [
            {"op": "setNgPlusProfessorRank", "rank": 8},
            {"op": "setNgPlusSupport", "index": 0, "points": 1200},
            {"op": "setNgPlusSkillRank", "recordIndex": 0, "skill": 0, "rank": 10},
            {"op": "setNgPlusClassMastery", "recordIndex": 0, "classId": 42, "mastered": False},
            {"op": "setNgPlusClassMastery", "recordIndex": 34, "classId": 99, "mastered": True},
            {"op": "setNgPlusClassMastery", "recordIndex": 38, "classId": 42, "mastered": False},
            {"op": "setNgPlusClassMastery", "recordIndex": 44, "classId": 99, "mastered": True},
        ]}), encoding="utf-8")
        ng_target = directory / "ng-plus-slot00"
        run(cli, "apply", "--input", str(source), "--patch", str(ng_patch),
            "--output", str(ng_target))
        ng_edited = ng_target.read_bytes()
        assert source.read_bytes() == raw
        assert ng_edited[player + 0x17CE] == 8
        assert struct.unpack_from("<H", ng_edited, player + 0x1576)[0] == 1200
        assert ng_edited[player + 0x17D8] == 10
        assert not ng_edited[class_start + 42 // 8] & (1 << (42 % 8))
        assert ng_edited[class_start + 34 * 13 + 99 // 8] & (1 << (99 % 8))
        assert not ng_edited[class_start + 38 * 13 + 42 // 8] & (1 << (42 % 8))
        assert ng_edited[class_start + 44 * 13 + 99 // 8] & 0x08
        changed_offsets = {0, 1, 2, 3, player + 0x17CE, player + 0x1576,
                           player + 0x1577, player + 0x17D8,
                           class_start + 42 // 8, class_start + 34 * 13 + 99 // 8,
                           class_start + 38 * 13 + 42 // 8, class_start + 44 * 13 + 99 // 8}
        assert all(before == after for index, (before, after) in enumerate(zip(raw, ng_edited))
                   if index not in changed_offsets)
        assert struct.unpack_from("<I", ng_edited, 0)[0] == checksum(ng_edited)
        ng_history = run(cli, "inspect", "--input", str(ng_target),
                         "--section", "inheritance")["inheritance"]
        assert ng_history["professorRank"] == 8
        assert ng_history["supports"][0]["maxPoints"] == 1200
        assert ng_history["characters"][0]["skillRanks"][0] == 10
        assert 42 not in ng_history["characters"][0]["masteredClassIds"]
        assert 99 in ng_history["characters"][34]["masteredClassIds"]
        assert 42 not in ng_history["characters"][38]["masteredClassIds"]
        assert 99 in ng_history["characters"][44]["masteredClassIds"]

        invalid_ng_patch = directory / "invalid-ng-plus.json"
        invalid_ng_patch.write_text(json.dumps({"operations": [
            {"op": "setNgPlusClassMastery", "recordIndex": 45, "classId": 42, "mastered": True},
        ]}), encoding="utf-8")
        run(cli, "apply", "--input", str(source), "--patch", str(invalid_ng_patch),
            "--output", str(directory / "invalid-ng-plus"), success=False)
        assert not (directory / "invalid-ng-plus").exists()

        # Agent edits must not bypass the GUI's rank-specific skill limit.
        out_of_range = directory / "out-of-range.json"
        out_of_range.write_text(json.dumps({"operations": [
            {"op": "set", "path": "Characters[0].data.SkillExp[0]", "value": 40},
        ]}), encoding="utf-8")
        error = run(cli, "apply", "--input", str(source), "--patch", str(out_of_range),
                    "--output", str(directory / "invalid-skill"), success=False)
        assert "rank limit" in error["error"]

        # In-place mode must retain the exact original save as a backup.
        inplace = directory / "inplace-slot00"
        inplace.write_bytes(raw)
        applied_inplace = run(cli, "apply", "--input", str(inplace), "--patch", str(patch_file), "--in-place")
        assert Path(applied_inplace["backup"]).read_bytes() == raw
        assert inplace.read_bytes() == edited

        # Raw character exchange preserves the record, without rewriting the rest of the save.
        character = directory / "character.bin"
        run(cli, "export-character", "--input", str(target), "--slot", "0", "--output", str(character))
        imported = directory / "imported-slot00"
        run(cli, "import-character", "--input", str(source), "--slot", "0",
            "--character", str(character), "--output", str(imported))
        assert imported.read_bytes()[12 + 0x644:12 + 0x644 + len(character.read_bytes())] == character.read_bytes()
        assert imported.read_bytes()[sentinel] == 0xA5

        bad_patch = directory / "bad.json"
        bad_patch.write_text(json.dumps({"operations": [
            {"op": "set", "path": "Player.field_17D8[0]", "value": 99},
        ]}), encoding="utf-8")
        error = run(cli, "apply", "--input", str(source), "--patch", str(bad_patch),
                    "--output", str(directory / "should-not-exist"), success=False)
        assert "not writable" in error["error"]
        assert not (directory / "should-not-exist").exists()

        suspend = directory / "suspend"
        suspend.write_bytes(raw + b"battle-state")
        error = run(cli, "inspect", "--input", str(suspend), success=False)
        assert "Suspend" in error["error"]

        print("CLI slot/system inspection and editing, NG+ history, character items, battalion endurance, backups, and rejection checks passed.")


if __name__ == "__main__":
    main()
