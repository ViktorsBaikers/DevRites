#!/usr/bin/env python3
"""Read workflow metadata and validated structural Markdown cursor fields."""

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


def decode_markdown(data: bytes, source: str | Path = "markdown text") -> str:
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError:
        raise ValueError(f"{source}: malformed UTF-8") from None
    if "\0" in text:
        raise ValueError(f"{source}: contains NUL byte")
    return text


def read_markdown(path: Path) -> str:
    return decode_markdown(path.read_bytes(), path)


def structural_markdown(text: str, source: str | Path = "markdown text") -> str:
    if "\0" in text:
        raise ValueError(f"{source}: contains NUL byte")
    try:
        data = text.encode("utf-8")
    except UnicodeEncodeError:
        raise ValueError(f"{source}: malformed UTF-8") from None

    masked = bytearray(data)
    marker = 0
    width = 0
    start = 0
    while start < len(data):
        line_end = data.find(b"\n", start)
        if line_end < 0:
            line_end = len(data)
        content_end = line_end
        if content_end > start and data[content_end - 1] == ord("\r"):
            content_end -= 1
        line = data[start:content_end]

        if marker == 0:
            opened = _opening_fence(line)
            if opened is not None:
                marker, width = opened
                _mask(masked, start, line_end)
        else:
            _mask(masked, start, line_end)
            if _closing_fence(line, marker, width):
                marker = width = 0

        if line_end == len(data):
            break
        start = line_end + 1
    return masked.decode("utf-8")


def _fence_start(line: bytes) -> int | None:
    start = 0
    while start < len(line) and line[start] == ord(" "):
        start += 1
    if (
        start <= 3
        and start < len(line)
        and line[start] in (ord("`"), ord("~"))
    ):
        return start
    return None


def _opening_fence(line: bytes) -> tuple[int, int] | None:
    start = _fence_start(line)
    if start is None:
        return None
    marker = line[start]
    end = start
    while end < len(line) and line[end] == marker:
        end += 1
    width = end - start
    if width < 3 or (marker == ord("`") and ord("`") in line[end:]):
        return None
    return marker, width


def _closing_fence(line: bytes, marker: int, width: int) -> bool:
    start = _fence_start(line)
    if start is None or line[start] != marker:
        return False
    end = start
    while end < len(line) and line[end] == marker:
        end += 1
    return end - start >= width and all(
        char in (ord(" "), ord("\t")) for char in line[end:]
    )


def _mask(data: bytearray, start: int, end: int) -> None:
    for index in range(start, end):
        if data[index] not in (ord("\r"), ord("\n")):
            data[index] = ord(" ")


def normalize_cursor_key(key: str) -> str:
    normalized = "".join(char for char in key.lower() if char.isalnum())
    canonical = CURSOR_KEY_ALIASES.get(normalized, normalized)
    return "".join(char for char in canonical.lower() if char.isalnum())


def cursor_field_text(text: str, key: str) -> str | None:
    text = structural_markdown(text)
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
    return cursor_field_text(read_markdown(path), key)


def phase_property(phase: str, name: str) -> bool:
    return bool(PHASES.get(phase.lower(), {}).get(name, False))


def main(argv: list[str]) -> int:
    try:
        if len(argv) == 4 and argv[1] == "field":
            value = cursor_field(Path(argv[2]), argv[3])
            if value is None:
                return 1
            print(value)
            return 0
        if len(argv) == 4 and argv[1] == "phase-property":
            return 0 if phase_property(argv[2], argv[3]) else 1
    except (OSError, ValueError) as exc:
        print(f"workflow_schema.py: {exc}", file=sys.stderr)
        return 2
    print("usage: workflow_schema.py field <state.md> <key> | phase-property <phase> <property>", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
