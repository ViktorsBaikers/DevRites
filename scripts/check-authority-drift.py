#!/usr/bin/env python3
"""Compare generated JSON authorities with their short documentation blocks."""

import argparse
import json
import re
import sys
from pathlib import Path


# Current-guidance documents that assert the workspace schema contract in prose.
# Every listed file must exist. Historical records (docs/adr/**, dated audits
# under docs/research/**) are deliberately not scanned: an accepted ADR keeps
# its body and carries an amendment instead of a silent rewrite.
SCHEMA_CLAIM_FILES = (
    "CONTEXT.md",
    "docs/architecture.md",
    "docs/engine/state-schema.md",
    "docs/engine/workspace-schema.md",
    "docs/flow.md",
)

# Present-tense claim forms only: `schemaVersion: 4`, `schema version: 4`,
# `state schema is v4`, `state schema (v4)`, `schema v4`. Historical references
# ("resolve to schema 2", "the v2→v3 normalization", the v3 cursor-encoding
# table form) and bare version tokens are not claims about the current contract
# and must not match.
SCHEMA_CLAIM_PATTERNS = (
    re.compile(r"\bschemaVersion\b[^0-9]{0,8}(\d+)", re.IGNORECASE),
    re.compile(r"\bschema[\s-]+version\b[^0-9]{0,8}(\d+)", re.IGNORECASE),
    re.compile(r"\bschema\s+is\s+v?(\d+)\b", re.IGNORECASE),
    re.compile(r"\bschema\s*\(\s*v?(\d+)\s*\)", re.IGNORECASE),
    re.compile(r"\bschema\s+v\s*[:=]?\s*(\d+)\b", re.IGNORECASE),
)

# The afk-hitl irreversible-risk list is the canonical owner; these documents
# restate it inline and must carry exactly the same items.
RISK_LIST_OWNER = "pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md"
RISK_LIST_COPIES = {
    "pack/.claude/skills/rite-vet/reference/depth.md": "prose",
    "pack/.claude/skills/rite-temper/reference/significance.md": "prose",
    "pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md": "bullets",
}


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
    raise ValueError(f"{path}: {name} authority block is stale; run python3 scripts/check-authority-drift.py --write")


def check_schema_claims(root, current):
    for relative in SCHEMA_CLAIM_FILES:
        path = root / relative
        if not path.is_file():
            raise ValueError(f"{relative}: schema-claim document is missing")
        text = path.read_text()
        for pattern in SCHEMA_CLAIM_PATTERNS:
            for match in pattern.finditer(text):
                claimed = int(match.group(1))
                if claimed == current:
                    continue
                line = text.count("\n", 0, match.start()) + 1
                raise ValueError(
                    f"{relative}:{line} claims schema v{claimed}; "
                    f"the workflow authority is schemaVersion {current}"
                )


def risk_item(text):
    text = re.sub(r"\s*\(.*?\)", "", text.lower()).replace("-", " ")
    text = re.sub(r"\s*/\s*", "/", re.sub(r"\s+", " ", text)).strip(" .;")
    return re.sub(r"[\s;]+or$", "", text).removesuffix(" change").strip(" .;")


def bullet_items(text, heading):
    match = re.search(re.escape(heading) + r"\n(?:.*\n)*?((?:- .*\n(?:[ \t]+\S.*\n)*)+)", text)
    if not match:
        raise ValueError(f"missing risk-list bullets under {heading}")
    return {risk_item(" ".join(item.split())) for item in re.split(r"^- ", match.group(1), flags=re.M)[1:]}


def check_risk_list(root):
    wanted = bullet_items((root / RISK_LIST_OWNER).read_text(), "## Irreversible-risk list (always pause)")
    problems = []
    for relative, form in RISK_LIST_COPIES.items():
        text = (root / relative).read_text()
        if form == "bullets":
            found = bullet_items(text, "## Always stop")
        else:
            match = re.search(r"irreversible-risk\s+list:\s*(.*?)\.\s", text, re.DOTALL)
            if not match:
                raise ValueError(f"{relative}: irreversible-risk list not found")
            found = {risk_item(item) for item in match.group(1).split(",")}
        if found != wanted:
            problems.append(
                f"{relative}: irreversible-risk list differs from {RISK_LIST_OWNER} "
                f"(missing: {sorted(wanted - found)}; extra: {sorted(found - wanted)}); edit the copy by hand to match"
            )
    if problems:
        raise ValueError("\n".join(problems))


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

    check_schema_claims(root, manifest["schemaVersion"])
    check_risk_list(root)
    print("authority-drift: current docs match the workflow authority")


if __name__ == "__main__":
    try:
        main()
    except (KeyError, OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"authority-drift: {exc}", file=sys.stderr)
        sys.exit(1)
