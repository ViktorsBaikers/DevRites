#!/usr/bin/env python3
"""Strictly validate every JSON file below the supplied pack roots."""

from __future__ import annotations

import json
import sys
from pathlib import Path


def reject_duplicate_keys(pairs: list[tuple[str, object]]) -> dict[str, object]:
    value: dict[str, object] = {}
    for key, item in pairs:
        if key in value:
            raise ValueError(f"duplicate key {key!r}")
        value[key] = item
    return value


def reject_constant(value: str) -> object:
    raise ValueError(f"non-finite number {value}")


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print("usage: validate-pack-json.py <root> [root ...]", file=sys.stderr)
        return 2

    files: list[Path] = []
    for raw_path in argv[1:]:
        path = Path(raw_path)
        if path.is_file() and path.suffix == ".json":
            files.append(path)
            continue
        if not path.is_dir():
            print(f"pack-json: missing JSON file or directory: {path}", file=sys.stderr)
            return 1
        files.extend(candidate for candidate in path.rglob("*.json") if candidate.is_file())

    unique_files = sorted(set(files))
    for path in unique_files:
        try:
            json.loads(
                path.read_text(encoding="utf-8"),
                object_pairs_hook=reject_duplicate_keys,
                parse_constant=reject_constant,
            )
        except (OSError, UnicodeError, json.JSONDecodeError, ValueError) as error:
            print(f"pack-json: invalid {path}: {error}", file=sys.stderr)
            return 1

    print(f"pack-json: {len(unique_files)} JSON files valid")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
