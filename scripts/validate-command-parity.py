#!/usr/bin/env python3
"""Validate Claude/Codex command parity for public DevRites skills."""

from __future__ import annotations
import argparse, re, sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def fm(text):
    if not text.startswith("---\n"):
        return {}
    end = text.find("\n---", 4)
    out = {}
    if end == -1:
        return out
    for line in text[4:end].splitlines():
        if not line.strip() or ":" not in line or line.startswith((" ", "\t")):
            continue
        k, v = line.split(":", 1)
        out[k.strip()] = v.strip().strip("\"'")
    return out


def public_rites(skills_dir: Path):
    names = []
    for f in sorted(skills_dir.glob("*/SKILL.md")):
        meta = fm(f.read_text(encoding="utf-8"))
        name = meta.get("name", f.parent.name)
        if (
            meta.get("user-invocable") == "true"
            and name.startswith("rite-")
            and name != "rite"
        ):
            names.append(name)
    return names


COMMAND_ROW = re.compile(r"\[\s*`?(?P<name>/[A-Za-z0-9_.-]+)[^\]`]*`?\s*\]\((?P<target>[^)]+)\)")


def public_command_rows(docs_map: str):
    """Rows of the `## Public commands` table, keyed by the command each row names.

    Only the leading `/name` token keys a row, so `/rite use <slug>` counts as `/rite`.

    Keyed on the command cell, not on the link target: a row renamed to a command that
    does not exist still contains the old name inside its own link path, so a presence
    needle over the whole document survives that falsification.
    """
    rows: dict[str, list[str]] = {}
    lines = docs_map.splitlines()
    start = next(
        (i for i, line in enumerate(lines) if line.startswith("## Public commands")), None
    )
    if start is None:
        return rows
    end = next(
        (i for i in range(start + 1, len(lines)) if lines[i].startswith("## ")), len(lines)
    )
    for line in lines[start + 1:end]:
        if not line.startswith("|") or line.count("|") < 2:
            continue
        cell = line.split("|")[1]
        match = COMMAND_ROW.search(cell)
        if match:
            rows.setdefault(match.group("name"), []).append(match.group("target"))
    return rows


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--skills-dir", type=Path, default=ROOT / "pack/.claude/skills")
    p.add_argument("--docs-skills", type=Path, default=ROOT / "docs/skills.md")
    p.add_argument(
        "--docs-command-map", type=Path, default=ROOT / "docs/command-map.md"
    )
    p.add_argument("--readme", type=Path, default=ROOT / "README.md")
    p.add_argument(
        "--router", type=Path, default=ROOT / "pack/.claude/skills/rite/SKILL.md"
    )
    p.add_argument("--generated-root", type=Path, default=ROOT / "pack/generated")
    p.add_argument("--quiet", action="store_true")
    args = p.parse_args()
    errors = []
    skills = public_rites(args.skills_dir)
    docs_skills = (
        args.docs_skills.read_text(encoding="utf-8")
        if args.docs_skills.exists()
        else ""
    )
    docs_map = (
        args.docs_command_map.read_text(encoding="utf-8")
        if args.docs_command_map.exists()
        else ""
    )
    readme = args.readme.read_text(encoding="utf-8") if args.readme.exists() else ""
    router = args.router.read_text(encoding="utf-8") if args.router.exists() else ""
    all_docs = "\n".join([docs_skills, docs_map, readme])
    if "npx devrites" not in all_docs:
        errors.append("docs missing npx devrites distribution contract")
    for line in all_docs.splitlines():
        if (
            re.search(
                r"\b(install|installed|installing)\s+via\s+.*\b(plugin|marketplace)\b",
                line,
                re.I,
            )
            and "not" not in line.lower()
        ):
            errors.append("docs imply plugin distribution instead of npx install")
            break
    rows = public_command_rows(docs_map)
    if not args.generated_root.exists():
        errors.append(f"generated root absent: {args.generated_root}")
    for name in skills:
        verb = name.removeprefix("rite-")
        needle = f"/rite-{verb}"
        if needle not in router:
            errors.append(f"router missing {needle} for {name}")
        if needle not in docs_map:
            errors.append(
                f"docs/command-map Claude direct: missing Claude command {needle} for {name}"
            )
        targets = rows.get(needle, [])
        if len(targets) != 1:
            errors.append(
                f"docs/command-map Claude direct: expected exactly one command row naming "
                f"{needle} for {name}, found {len(targets)}"
            )
        if args.generated_root.exists():
            for rel in [
                f"claude/skills/{name}/SKILL.md",
                f"codex/skills/{name}/SKILL.md",
                f"pi/skills/{name}/SKILL.md",
                f"omp/skills/{name}/SKILL.md",
                f"omp/commands/{name}.md",
                f"pi/prompts/{name}.md",
                f"devin/skills/{name}/SKILL.md",
            ]:
                if not (args.generated_root / rel).exists():
                    errors.append(f"generated artifact missing {rel}")
    for command, targets in rows.items():
        for target in targets:
            linked = re.search(r"skills/([^/]+)/SKILL\.md$", target)
            if linked is None or linked.group(1) != command[1:]:
                errors.append(
                    f"docs/command-map Claude direct: command row for {command} links to "
                    f"{target}, which is not skills/{command[1:]}/SKILL.md"
                )
            elif not (args.skills_dir / linked.group(1) / "SKILL.md").is_file():
                errors.append(
                    f"docs/command-map Claude direct: command row for {command} links to "
                    f"a skill directory that does not exist: {linked.group(1)}"
                )
    if errors:
        for e in errors:
            print(f"FAIL: {e}")
        print(f"validate-command-parity: {len(errors)} failure(s)")
        return 1
    if not args.quiet:
        print("validate-command-parity: PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
