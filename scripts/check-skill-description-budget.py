#!/usr/bin/env python3
"""Enforce short skill catalog descriptions.

Public/user-invocable skills: <=220 chars. Internal skills: <=260 chars.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

PUBLIC_LIMIT = 220
INTERNAL_LIMIT = 260


def frontmatter(text: str) -> dict[str, str]:
    if not text.startswith("---\n"):
        return {}
    end = text.find("\n---", 4)
    if end < 0:
        return {}
    out: dict[str, str] = {}
    for line in text[4:end].splitlines():
        if ":" not in line:
            continue
        k, v = line.split(":", 1)
        out[k.strip()] = v.strip().strip('"\'')
    return out


def main(argv: list[str]) -> int:
    roots = [Path(a) for a in argv] or [Path("pack/.claude/skills")]
    files: list[Path] = []
    for root in roots:
        if root.is_file():
            files.append(root)
        else:
            files.extend(root.glob("*/SKILL.md"))
    bad = 0
    for path in sorted(files):
        fm = frontmatter(path.read_text(encoding="utf-8"))
        desc = re.sub(r"\s+", " ", fm.get("description", "")).strip()
        if not desc:
            continue
        internal = fm.get("user-invocable") == "false"
        limit = INTERNAL_LIMIT if internal else PUBLIC_LIMIT
        if len(desc) > limit:
            print(f"FAIL: {path}: description {len(desc)} > {limit}")
            bad += 1
    if bad:
        print(f"{bad} skill description(s) over budget")
        return 1
    print(f"ok: {len(files)} skill descriptions within budget")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
