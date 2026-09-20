#!/usr/bin/env python3
"""Expand <!-- include:REL/PATH.md --> markers in-place under a directory.

Paths resolve relative to the file containing the marker. Used by
build-host-artifacts.sh on a scratch copy of pack/.claude so every host
receives identical expanded text while canonical sources deduplicate
shared blocks into pack/.claude/**/_shared/.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

MARKER = re.compile(r"<!--\s*include:([^\s>]+)\s*-->")


def expand_file(path: Path, errors: list[str]) -> str:
    text = path.read_text(encoding="utf-8")

    def repl(m: re.Match) -> str:
        target = (path.parent / m.group(1)).resolve()
        if not target.is_file():
            errors.append(f"{path}: include not found: {m.group(1)}")
            return m.group(0)
        return target.read_text(encoding="utf-8").rstrip("\n")

    return MARKER.sub(repl, text)


def main() -> int:
    root = Path(sys.argv[1])
    errors: list[str] = []
    for f in sorted(root.rglob("*.md")):
        if "include:" not in f.read_text(encoding="utf-8"):
            continue
        new = expand_file(f, errors)
        if new != f.read_text(encoding="utf-8"):
            f.write_text(new, encoding="utf-8")
    for e in errors:
        print(e, file=sys.stderr)
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
