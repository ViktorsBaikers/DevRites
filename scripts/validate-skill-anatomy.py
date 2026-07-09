#!/usr/bin/env python3
"""Validate DevRites-native skill anatomy contracts."""
from __future__ import annotations
import argparse, re, sys
from datetime import date
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SKILLS_DIR = ROOT / "pack" / ".claude" / "skills"

PUBLIC_REQUIRED = {
    "operating contract": [r"## Operating rules", r"## Rules", r"## Contract", r"## Core operating rules", r"## Two modes", r"## Safety", r"## Mode", r"## Protocol"],
    "rules consulted / standards read": [r"## Rules consulted", r"governing rules", r"standards/core\.md", r"devrites-lib/reference/standards"],
    "workflow or phase-contract pointer": [r"## Workflow", r"phase-contract\.md", r"## Steps", r"## Protocol", r"## Flow", r"## Run", r"## Menu", r"## Process", r"## Modes"],
    "output/reply contract": [r"## Output", r"## What to output", r"reply-contract\.md", r"Default success shape", r"Completion"],
    "phase boundary / stop condition": [r"stop", r"Stop when", r"Done when", r"NO-GO", r"GO", r"one slice"],
}
INTERNAL_REQUIRED = {
    "trigger boundary": [r"Use when", r"## Trigger", r"## When to use", r"## Boundaries", r"Triggered by"],
    "hard rules": [r"## Hard rules", r"## Rules", r"## NEVER", r"## Operating rules", r"## Boundaries", r"Do not", r"Never", r"Must", r"Scope"],
    "output contract": [r"## Output", r"## Return", r"Output format", r"Default success shape", r"Visual Verdict", r"writes? `", r"writes? ", r"Return"],
    "when not to use": [r"Not for", r"## When NOT to use", r"## Boundaries", r"Do not", r"Never"],
}
GOTCHA_PATTERNS = [r"## Gotchas", r"reference/anti-patterns\.md", r"Mid-flight discipline", r"## Hard rules", r"## NEVER", r"## Boundaries", r"## Rules", r"## Scope reminders", r"## When NOT to use", r"## What NOT to include", r"Anti-AI-slop", r"Do not", r"Never"]
EXEMPT = {"devrites-lib"}
# Legacy exceptions waive only the named missing section. They are owned and
# time-bounded so a compact skill cannot silently bypass unrelated contracts.
def exemption(owner: str, reason: str) -> dict[str, str]:
    return {"owner": owner, "reason": reason, "expires": "2026-10-01"}


SECTION_EXEMPTIONS = {
    "devrites-api-interface": {
        "hard rules": exemption("guidance", "compact specialist encodes rules in prose"),
        "output contract": exemption("guidance", "output is owned by the caller contract"),
        "gotchas/anti-patterns pointer": exemption("guidance", "compact specialist has no standalone reference payload"),
    },
    "devrites-frontend-craft": {"output contract": exemption("guidance", "reference-driven craft output")},
    "rite-doctor": {
        "operating contract": exemption("guidance", "read-only diagnostic utility"),
        "rules consulted / standards read": exemption("guidance", "engine owns the diagnostic checks"),
    },
    "rite-frame": {
        "rules consulted / standards read": exemption("guidance", "ad-hoc lens utility"),
        "workflow or phase-contract pointer": exemption("guidance", "FRAME/AUDIT flow is encoded in prose"),
    },
    "rite-handoff": {
        "operating contract": exemption("guidance", "single-purpose artifact writer"),
        "workflow or phase-contract pointer": exemption("guidance", "handoff artifact is the workflow")},
    "rite-polish": {"workflow or phase-contract pointer": exemption("guidance", "workflow is split across code and UI references")},
    "rite-pressure-test": {
        "operating contract": exemption("guidance", "compact ideation utility"),
        "workflow or phase-contract pointer": exemption("guidance", "pressure-test loop is encoded in prose")},
    "rite-prototype": {
        "operating contract": exemption("guidance", "throwaway prototype lifecycle"),
        "workflow or phase-contract pointer": exemption("guidance", "prototype loop is encoded in prose"),
        "phase boundary / stop condition": exemption("guidance", "prototype is intentionally disposable"),
        "gotchas/anti-patterns pointer": exemption("guidance", "prototype warning is encoded in prose")},
    "rite-status": {
        "operating contract": exemption("guidance", "read-only engine status utility"),
        "workflow or phase-contract pointer": exemption("guidance", "engine progress command owns flow")},
    "rite-zoom-out": {
        "operating contract": exemption("guidance", "read-only mapping utility"),
        "workflow or phase-contract pointer": exemption("guidance", "structural map is the workflow")},
}


def parse_frontmatter(text: str) -> dict[str, str]:
    if not text.startswith("---\n"): return {}
    end = text.find("\n---", 4)
    fields = {}
    if end == -1: return fields
    for line in text[4:end].splitlines():
        if not line.strip() or line.startswith((" ", "\t", "#")) or ":" not in line: continue
        k,v = line.split(":",1)
        fields[k.strip()] = v.strip().strip('"\'')
    return fields


def has_any(text: str, patterns: list[str]) -> bool:
    return any(re.search(p, text, re.I | re.M) for p in patterns)


def validate(skills_dir: Path) -> list[str]:
    errors = []
    for f in sorted(skills_dir.glob("*/SKILL.md")):
        text = f.read_text(encoding="utf-8")
        fm = parse_frontmatter(text)
        name = fm.get("name", f.parent.name)
        if name in EXEMPT:
            continue
        exemptions = SECTION_EXEMPTIONS.get(name, {})
        invocable = fm.get("user-invocable") == "true"
        rules = PUBLIC_REQUIRED if invocable else INTERNAL_REQUIRED
        for label, patterns in rules.items():
            if not has_any(text, patterns) and label not in exemptions:
                errors.append(f"{f}: missing {label}")
        if not has_any(text, GOTCHA_PATTERNS) and "gotchas/anti-patterns pointer" not in exemptions:
            errors.append(f"{f}: missing gotchas/anti-patterns pointer")
        for label, record in exemptions.items():
            if not record.get("owner") or not record.get("reason") or not re.fullmatch(r"\d{4}-\d{2}-\d{2}", record.get("expires", "")):
                errors.append(f"{f}: anatomy exemption {label!r} requires owner, reason, and YYYY-MM-DD expires")
            elif record["expires"] < date.today().isoformat():
                errors.append(f"{f}: anatomy exemption {label!r} expired {record['expires']}")
        desc = fm.get("description", "")
        explicit_only = fm.get("disable-model-invocation") == "true"
        if not explicit_only and not re.search(r"\bUse when\b|\bUse to\b|\bTrigger|\bwhen the user\b|\bfor ", desc, re.I):
            errors.append(f"{f}: model-invoked description must include trigger language")
        # Cross-skill references: obvious backtick references should exist.
        for ref in re.findall(r"`((?:rite|devrites)-[a-z0-9-]+)`", text):
            if ref in {"rite-use", "devrites-source-cache"}:
                continue
            if not (skills_dir / ref / "SKILL.md").exists() and not (ROOT / "pack" / ".claude" / "agents" / f"{ref}.md").exists():
                errors.append(f"{f}: unknown cross-skill/agent reference `{ref}`")
    return errors


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--skills-dir", type=Path, default=SKILLS_DIR)
    p.add_argument("--quiet", action="store_true")
    args = p.parse_args()
    errors = validate(args.skills_dir)
    if errors:
        for e in errors: print(f"FAIL: {e}")
        print(f"validate-skill-anatomy: {len(errors)} failure(s)")
        return 1
    if not args.quiet:
        print("validate-skill-anatomy: PASS")
    return 0

if __name__ == "__main__": sys.exit(main())
