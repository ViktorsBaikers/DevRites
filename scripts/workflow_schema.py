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
    anchor = anchor_start = -1
    after_blank = True
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
            opened = _opening_fence(line, 3)
            listed = _opening_fence(line, anchor + 3) if anchor >= 0 else None
            if _blank(line):
                after_blank = True
            elif opened is not None:
                marker, width = opened
                _mask(masked, start, line_end)
                anchor, after_blank = -1, False
            elif listed is not None and _indent_columns(line) >= anchor:
                anchored, anchor, after_blank = anchor, -1, False
                close_end = _list_fence_end(
                    data, line_end, listed, _indent_columns(line), anchored
                )
                if close_end >= 0:
                    if not _lone_cr(data[anchor_start:close_end]):
                        _mask(masked, start, close_end)
                    if close_end == len(data):
                        break
                    start = close_end + 1
                    continue
            else:
                column = _top_level_item(line)
                if column is not None and (after_blank or anchor >= 0):
                    anchor, anchor_start = column, start
                else:
                    anchor = -1
                after_blank = False
        else:
            _mask(masked, start, line_end)
            if _closing_fence(line, marker, width, 3):
                marker = width = 0

        if line_end == len(data):
            break
        start = line_end + 1
    return masked.decode("utf-8")


def _leading_width(line: bytes) -> tuple[int, int]:
    columns = size = 0
    while size < len(line) and line[size] in b" \t":
        columns += 4 - columns % 4 if line[size] == ord("\t") else 1
        size += 1
    return columns, size


def _top_level_item(line: bytes) -> int | None:
    if len(line) < 2 or _is_thematic_break(line):
        return None
    marker_end = 1
    if line[0] in b"-*+":
        pass
    elif 49 <= line[0] <= 57 and line[1] == ord("."):
        marker_end = 2
    else:
        return None
    if marker_end >= len(line) or line[marker_end] not in b" \t":
        return None
    column = nxt = marker_end
    while nxt < len(line) and line[nxt] in b" \t":
        column += 4 - column % 4 if line[nxt] == ord("\t") else 1
        nxt += 1
    if nxt == len(line) or column - marker_end > 4 or _starts_block(line[nxt:]):
        return None
    return column


def _starts_block(content: bytes) -> bool:
    first = content[0]
    if first in b"-*+":
        end = 1
    elif 48 <= first <= 57:
        end = 1
        while end < len(content) and end < 9 and 48 <= content[end] <= 57:
            end += 1
        if end == len(content) or content[end] not in b".)":
            return False
        end += 1
    elif first == ord("#"):
        end = len(content) - len(content.lstrip(b"#"))
        if end > 6:
            return False
    elif first in b"`~":
        return content[:3] == bytes([first]) * 3
    else:
        return False
    return end == len(content) or content[end] in b" \t"


def _blank(line: bytes) -> bool:
    return not line.strip(b" \t")


def _indent_columns(line: bytes) -> int:
    column = 0
    for char in line:
        if char == ord(" "):
            column += 1
        elif char == ord("\t"):
            column += 4 - column % 4
        else:
            break
    return column


def _is_thematic_break(line: bytes) -> bool:
    trimmed = line.strip(b" \t")
    if len(trimmed) < 3 or trimmed[0] not in b"-*_":
        return False
    return all(char in (trimmed[0], ord(" "), ord("\t")) for char in trimmed) and (
        trimmed.count(bytes([trimmed[0]])) >= 3
    )


def _fence_start(line: bytes, limit: int) -> int | None:
    columns, start = _leading_width(line)
    if (
        columns <= limit
        and start < len(line)
        and line[start] in (ord("`"), ord("~"))
    ):
        return start
    return None


def _opening_fence(line: bytes, limit: int) -> tuple[int, int] | None:
    start = _fence_start(line, limit)
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


def _list_fence_end(
    data: bytes, open_end: int, opened: tuple[int, int], indent: int, container: int
) -> int:
    marker, width = opened
    start = open_end + 1
    while start < len(data):
        line_end = data.find(b"\n", start)
        if line_end < 0:
            line_end = len(data)
        content_end = line_end
        if content_end > start and data[content_end - 1] == ord("\r"):
            content_end -= 1
        line = data[start:content_end]
        if not _blank(line):
            columns = _indent_columns(line)
            if columns < indent:
                return -1
            if _closing_fence(line, marker, width, container + 3):
                return line_end if columns == indent else -1
        start = line_end + 1
    return -1


def _lone_cr(data: bytes) -> bool:
    """Report a carriage return that is not the final byte of its line."""
    return any(
        char == ord("\r") and index + 1 < len(data) and data[index + 1] != ord("\n")
        for index, char in enumerate(data)
    )


def _closing_fence(line: bytes, marker: int, width: int, limit: int) -> bool:
    start = _fence_start(line, limit)
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
