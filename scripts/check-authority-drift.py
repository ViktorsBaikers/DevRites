#!/usr/bin/env python3
"""Compare generated JSON authorities with their short documentation blocks."""

import argparse
import json
import re
import sys
from pathlib import Path


def table(headers, rows):
    return "\n".join(
        [
            "| " + " | ".join(headers) + " |",
            "| " + " | ".join("---" for _ in headers) + " |",
            *["| " + " | ".join(row) + " |" for row in rows],
        ]
    )


def replace_block(path, name, wanted, write):
    text = path.read_text()
    start = f"<!-- authority:{name}:start -->"
    end = f"<!-- authority:{name}:end -->"
    pattern = re.compile(re.escape(start) + r"\n.*?\n" + re.escape(end), re.DOTALL)
    replacement = f"{start}\n{wanted}\n{end}"
    if not pattern.search(text):
        raise ValueError(f"{path}: missing {name} authority markers")
    if pattern.search(text).group(0) == replacement:
        return
    if write:
        path.write_text(pattern.sub(replacement, text, count=1))
        return
    raise ValueError(f"{path}: {name} authority block is stale")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--write", action="store_true")
    args = parser.parse_args()
    root = args.root.resolve()

    manifest = json.loads((root / "engine/internal/state/workflow_manifest.json").read_text())
    phases = manifest["phases"]
    policy = manifest["authorityPolicy"]

    ids = [phase["id"] for phase in phases]
    rights = [phase["transitionRight"] for phase in phases]
    if len(ids) != len(set(ids)) or len(rights) != len(set(rights)):
        raise ValueError("workflow manifest has duplicate phase IDs or transition rights")
    resumes = {phase["id"]: phase.get("resumeVerb", "") for phase in phases}
    if resumes.get("define") != "define" or resumes.get("plan") != "vet":
        raise ValueError("workflow manifest violates ADR-0011 define/plan routing")
    lifecycle = "`" + " → ".join(phase.upper() for phase in ids) + "`"
    phase_rows = []
    for phase in phases:
        required = ", ".join(f"`{section}`" for section in phase.get("requiredSections", [])) or "*(none)*"
        resume = f"`/rite-{phase['resumeVerb']}`" if phase.get("resumeVerb") else "*(terminal)*"
        phase_rows.append([f"`{phase['id']}`", resume, required, phase["transitionRight"]])
    tracking = (
        "Git-tracked shared state: "
        + ", ".join(f"`{path}`" for path in policy["trackedState"])
        + ". Per-clone runtime state: "
        + ", ".join(f"`{path}`" for path in policy["localState"])
        + "."
    )
    blocks = [
        ("docs/quick-reference.md", "lifecycle", lifecycle),
        ("docs/engine/state-schema.md", "schema-version", f"`schemaVersion: {manifest['schemaVersion']}`."),
        (
            "docs/engine/state-schema.md",
            "phase-contract",
            table(["phase", "normal resume", "required sections", "transition right"], phase_rows),
        ),
        ("docs/engine/state-schema.md", "state-tracking", tracking),
        ("SECURITY.md", "principles-trust", policy["principlesTrust"]),
        (
            "pack/.claude/skills/devrites-lib/reference/standards/core.md",
            "principles-trust",
            policy["principlesTrust"],
        ),
    ]
    for relative, name, wanted in blocks:
        replace_block(root / relative, name, wanted, args.write)

    state_schema = (root / "docs/engine/state-schema.md").read_text()
    if re.search(r"schema(?:Version| version| v)[: ]+1\b", state_schema, re.IGNORECASE):
        raise ValueError("docs/engine/state-schema.md still claims schema v1")
    print("authority-drift: current docs match the workflow authority")


if __name__ == "__main__":
    try:
        main()
    except (KeyError, OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"authority-drift: {exc}", file=sys.stderr)
        sys.exit(1)
