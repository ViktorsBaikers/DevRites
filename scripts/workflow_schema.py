#!/usr/bin/env python3
"""Read generated workflow metadata and canonical/legacy cursor fields."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path


MANIFEST = Path(__file__).resolve().parents[1] / "engine" / "internal" / "state" / "workflow_manifest.json"


DOCUMENT = json.loads(MANIFEST.read_text(encoding="utf-8"))


PHASES = {str(phase["id"]): phase for phase in DOCUMENT["phases"]}
CURSOR_KEY_ALIASES = {
    str(entry["alias"]): str(entry["canonical"])
    for entry in DOCUMENT["cursorKeyAliases"]
}


def normalize_cursor_key(key: str) -> str:
    normalized = "".join(char for char in key.lower() if char.isalnum())
    canonical = CURSOR_KEY_ALIASES.get(normalized, normalized)
    return "".join(char for char in canonical.lower() if char.isalnum())


def cursor_field_text(text: str, key: str) -> str | None:
    wanted = normalize_cursor_key(key)
    for line in text.splitlines():
        stripped = line.strip()
        if stripped.startswith("|") and stripped.endswith("|"):
            cells = [cell.strip() for cell in stripped[1:-1].split("|")]
            if len(cells) >= 2 and normalize_cursor_key(cells[0]) == wanted:
                return cells[1]
        legacy = re.match(r"^\s*[-*+]?\s*([^:]+):\s*(.*?)\s*$", line)
        if legacy and normalize_cursor_key(legacy.group(1)) == wanted:
            return re.sub(r"\s*(?:#|\|).*?$", "", legacy.group(2)).rstrip()
    return None


def cursor_field(path: Path, key: str) -> str | None:
    if not path.is_file():
        return None
    return cursor_field_text(path.read_text(encoding="utf-8"), key)


def phase_property(phase: str, name: str) -> bool:
    return bool(PHASES.get(phase.lower(), {}).get(name, False))


def main(argv: list[str]) -> int:
    if len(argv) == 4 and argv[1] == "field":
        value = cursor_field(Path(argv[2]), argv[3])
        if value is None:
            return 1
        print(value)
        return 0
    if len(argv) == 4 and argv[1] == "phase-property":
        return 0 if phase_property(argv[2], argv[3]) else 1
    print("usage: workflow_schema.py field <state.md> <key> | phase-property <phase> <property>", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
