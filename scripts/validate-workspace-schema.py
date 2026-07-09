#!/usr/bin/env python3
"""Validate generated DevRites workspace artifacts.

The validator is intentionally dependency-free: it checks the Markdown contract
well enough to catch drift without introducing a runtime parser dependency.
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path


ID_PATTERNS = {
    "AC": re.compile(r"\bAC-\d{3}\b"),
    "REQ": re.compile(r"\bREQ-\d{3}\b"),
    "SLICE": re.compile(r"\bSLICE-\d{3}\b"),
    "DEC": re.compile(r"\bDEC-\d{3}\b"),
    "Q": re.compile(r"\bQ-\d{3}\b"),
    "DRIFT": re.compile(r"\bDRIFT-\d{3}\b"),
    "EVID": re.compile(r"\bEVID-\d{3}\b"),
}

LEGACY_AC_RE = re.compile(r"\bAC\d+\b")
LEGACY_SLICE_RE = re.compile(r"\bSlice\s+\d+\b", re.IGNORECASE)
MD_LINK_RE = re.compile(r"(?<!!)\[[^\]]+\]\(([^)]+)\)")
MERMAID_STARTS = (
    "graph ",
    "flowchart ",
    "sequenceDiagram",
    "stateDiagram",
    "stateDiagram-v2",
    "classDiagram",
    "erDiagram",
    "journey",
    "gantt",
    "timeline",
    "mindmap",
    "quadrantChart",
)

HIGH_TRAFFIC_BUDGETS = {
    "README.md": 120,
    "index.md": 120,
    "feature.md": 120,
    "brief.md": 80,
    "spec.md": 260,
    "ai-spec.md": 160,
    "plan.md": 220,
    "tasks.md": 280,
    "traceability.md": 220,
    "state.md": 120,
    "handoff.md": 120,
    "evidence.md": 280,
    "proof.md": 280,
    "browser-evidence.md": 220,
}

REQUIRED_HEADINGS = {
    "README.md": ["Artifact map", "Read next", "Blocking gates"],
    "index.md": ["Artifact map", "Read next", "Blocking gates"],
    "brief.md": ["Objective", "Non-goals", "Success definition"],
    "spec.md": [
        "Problem",
        "Goal",
        "Non-goals",
        "Users / actors",
        "Requirements",
        "Acceptance criteria",
        "Edge Coverage",
        "Prohibitions (must-NOT)",
        "Edge cases",
        "Measurable success",
        "Scope boundaries",
    ],
    "architecture.md": [
        "Owning module / layer",
        "Integration points",
        "Data / API / events",
        "Dependencies",
        "Risks",
        "Affected boundaries",
    ],
    "plan.md": ["Approach", "Slice strategy", "Validation strategy", "Rollback"],
    "tasks.md": ["Slice index"],
    "traceability.md": ["Coverage matrix"],
    "state.md": ["Cursor"],
    "decisions.md": ["Decision log"],
    "assumptions.md": ["Assumption register"],
    "questions.md": ["Question register"],
    "evidence.md": ["Evidence log"],
    "proof.md": ["Evidence log"],
    "drift.md": ["Drift register"],
    "touched-files.md": ["Touched files"],
    "design-brief.md": ["Design direction", "States", "Interaction model"],
    "browser-evidence.md": ["Browser evidence", "Visual Verdict"],
    "handoff.md": ["Resume", "Read next", "Next action"],
    "references.md": ["Reference index"],
}

WORKSPACE_MAP_FIELDS = ("phase", "status", "next_action", "last_updated")
SLICE_REQUIRED_FIELDS = (
    "Goal",
    "Satisfies",
    "Files likely touched",
    "Tests/proof",
    "Mode",
    "Gate",
    "Dependencies",
    "Done condition",
)

PHASE_REQUIRED = {
    "frame": ["state.md"],
    "spec": ["brief.md", "spec.md", "state.md", "decisions.md", "assumptions.md", "questions.md"],
    "temper": ["brief.md", "spec.md", "state.md", "decisions.md", "assumptions.md", "questions.md"],
    "plan": [
        "brief.md",
        "spec.md",
        "architecture.md",
        "plan.md",
        "tasks.md",
        "traceability.md",
        "state.md",
        "decisions.md",
        "assumptions.md",
        "questions.md",
    ],
    "vet": [
        "brief.md",
        "spec.md",
        "architecture.md",
        "plan.md",
        "tasks.md",
        "traceability.md",
        "state.md",
        "decisions.md",
        "assumptions.md",
        "questions.md",
    ],
    "build": [
        "brief.md",
        "spec.md",
        "architecture.md",
        "plan.md",
        "tasks.md",
        "traceability.md",
        "state.md",
        "decisions.md",
        "assumptions.md",
        "questions.md",
    ],
    "prove": [
        "brief.md",
        "spec.md",
        "architecture.md",
        "plan.md",
        "tasks.md",
        "traceability.md",
        "state.md",
        "decisions.md",
        "assumptions.md",
        "questions.md",
    ],
    "polish": [],
    "review": [],
    "seal": [],
    "ship": [],
    "done": [],
}
for phase in ("polish", "review", "seal", "ship", "done"):
    PHASE_REQUIRED[phase] = PHASE_REQUIRED["prove"]


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def headings(text: str) -> set[str]:
    found = set()
    for line in text.splitlines():
        m = re.match(r"^#{1,6}\s+(.+?)\s*#*\s*$", line)
        if m:
            found.add(m.group(1).strip())
    return found


def line_count(text: str) -> int:
    return len(text.splitlines())


def has_budget_override(text: str) -> bool:
    return re.search(r"(?im)^Budget override:\s*\S", text) is not None


def phase_for(workspace: Path) -> str:
    for name in ("state.md", "status.md", "README.md", "index.md", "feature.md"):
        p = workspace / name
        if not p.exists():
            continue
        text = read(p)
        m = re.search(r"(?im)^\s*-?\s*phase\s*:\s*([a-z-]+)", text)
        if m:
            return m.group(1).lower()
    return "spec"


def workspace_index_present(workspace: Path) -> bool:
    return any((workspace / name).is_file() for name in ("README.md", "index.md", "feature.md"))


def evidence_file(workspace: Path) -> Path | None:
    for name in ("evidence.md", "proof.md"):
        p = workspace / name
        if p.is_file():
            return p
    return None


def existing_or_alias(workspace: Path, name: str) -> bool:
    if (workspace / name).is_file():
        return True
    if name == "evidence.md" and (workspace / "proof.md").is_file():
        return True
    if name == "proof.md" and (workspace / "evidence.md").is_file():
        return True
    if name == "README.md" and workspace_index_present(workspace):
        return True
    return False


def find_local_workspaces(root: Path) -> list[Path]:
    # A devrites workspace is identified by its devrites-specific artifacts, NOT by
    # README.md — every repo root has a README.md, and matching on it here both
    # misclassifies the repo root as a workspace and short-circuits the .devrites/work
    # scan below (return [root] before the loop ever runs).
    if root.is_dir() and any((root / name).exists() for name in ("spec.md", "state.md", "feature.md")):
        return [root]
    out: list[Path] = []
    for base in (
        root / ".devrites" / "work",
        root / ".devrites" / "features",
        root / ".devrites" / "archive",
        root / "work",
        root / "features",
        root / "archive",
    ):
        if not base.is_dir():
            continue
        for child in sorted(base.iterdir()):
            if child.is_dir() and any((child / n).exists() for n in ("spec.md", "state.md", "feature.md")):
                out.append(child)
    return out


def local_link_target(workspace: Path, raw: str) -> Path | None:
    target = raw.split("#", 1)[0].strip()
    if not target or re.match(r"^[a-z][a-z0-9+.-]*:", target, re.I):
        return None
    if target.startswith("#"):
        return None
    return (workspace / target).resolve()


def validate_mermaid(path: Path, text: str, errors: list[str]) -> None:
    in_block = False
    first = ""
    start_line = 0
    for i, line in enumerate(text.splitlines(), 1):
        if line.strip() == "```mermaid" and not in_block:
            in_block = True
            first = ""
            start_line = i
            continue
        if in_block and line.startswith("```"):
            if not first:
                errors.append(f"{path}: mermaid block at line {start_line} is empty")
            elif not first.startswith(MERMAID_STARTS):
                errors.append(f"{path}: mermaid block at line {start_line} starts with unsupported syntax: {first!r}")
            in_block = False
            continue
        if in_block and not first and line.strip():
            first = line.strip()
    if in_block:
        errors.append(f"{path}: mermaid block at line {start_line} is not closed")


def mermaid_block_starts(text: str) -> list[int]:
    return [i for i, line in enumerate(text.splitlines(), 1) if line.strip() == "```mermaid"]


def table_rows(text: str) -> list[str]:
    return [line for line in text.splitlines() if line.strip().startswith("|") and not re.match(r"^\s*\|?\s*:?-{2,}", line)]


def validate_workspace_map(path: Path, text: str, errors: list[str]) -> None:
    for field in WORKSPACE_MAP_FIELDS:
        if not re.search(rf"(?im)^\s*{re.escape(field)}\s*:\s*\S+", text):
            errors.append(f"{path}: missing workspace map field '{field}:'")


def validate_flows(path: Path, text: str, errors: list[str]) -> None:
    for line_no in mermaid_block_starts(text):
        before = "\n".join(text.splitlines()[max(0, line_no - 6):line_no - 1])
        if not re.search(r"(?i)why this matters\s*:", before):
            errors.append(f"{path}: mermaid block at line {line_no} needs a preceding 'Why this matters:' sentence")
        if not re.search(r"\b(AC|REQ|SLICE|DEC|DRIFT)-\d{3}\b", before):
            errors.append(f"{path}: mermaid block at line {line_no} needs related stable IDs near the why-it-matters text")


def validate_tasks(path: Path, text: str, errors: list[str]) -> None:
    chunks = re.split(r"(?m)^##\s+(SLICE-\d{3}[^\n]*)\n", text)
    for i in range(1, len(chunks), 2):
        heading = chunks[i]
        body = chunks[i + 1]
        slice_id = ID_PATTERNS["SLICE"].search(heading).group(0)
        for field in SLICE_REQUIRED_FIELDS:
            if not re.search(rf"(?im)^{re.escape(field)}\s*:\s*\S+", body):
                errors.append(f"{path}: {slice_id} missing field '{field}:'")


def validate_id_register(path: Path, text: str, kind: str, errors: list[str]) -> None:
    if text and not ID_PATTERNS[kind].search(text):
        errors.append(f"{path}: no {kind}-### IDs found")


def validate_evidence_entries(path: Path, text: str, errors: list[str]) -> None:
    for line in text.splitlines():
        evids = ID_PATTERNS["EVID"].findall(line)
        if not evids:
            continue
        if not ID_PATTERNS["AC"].search(line):
            errors.append(f"{path}: {evids[0]} missing related AC-### on the same entry")
        if not ID_PATTERNS["SLICE"].search(line):
            errors.append(f"{path}: {evids[0]} missing related SLICE-### on the same entry")


def completed_slices(tasks: str) -> set[str]:
    done = set()
    current = ""
    for line in tasks.splitlines():
        ids = ID_PATTERNS["SLICE"].findall(line)
        if ids and re.match(r"^#{2,6}\s+", line):
            current = ids[0]
        if current and re.search(r"(?i)\b(status|done)\s*:\s*(built|done|complete|completed)\b", line):
            done.add(current)
        if ids and re.search(r"(?i)\b(built|done|complete|completed)\b", line):
            done.update(ids)
        if re.match(r"^\s*-\s*\[[xX]\]", line) and ids:
            done.update(ids)
    return done


def blocking_questions(questions: str) -> list[str]:
    out = []
    chunks = re.split(r"(?m)^##\s+", questions)
    for chunk in chunks:
        if not chunk.strip():
            continue
        qid = (ID_PATTERNS["Q"].search(chunk) or re.search(r"\bq-\d{4}-\d{2}-\d{2}-\d{3}\b", chunk))
        status = re.search(r"(?im)^status:\s*(\w+)", chunk)
        gate = re.search(r"(?im)^gate:\s*(\w+)", chunk)
        if status and status.group(1).lower() == "open" and gate and gate.group(1).lower() in {"blocking", "escalating"}:
            out.append(qid.group(0) if qid else "<unknown>")
    return out


def validate_workspace(workspace: Path) -> list[str]:
    errors: list[str] = []
    phase = phase_for(workspace)

    if not workspace_index_present(workspace):
        errors.append(f"{workspace}: missing README.md/index.md/feature.md workspace map")

    required = PHASE_REQUIRED.get(phase, PHASE_REQUIRED["spec"])
    if phase in {"prove", "polish", "review", "seal", "ship", "done"}:
        required = [*required, "evidence.md", "touched-files.md"]
    for name in required:
        if not existing_or_alias(workspace, name):
            errors.append(f"{workspace}: phase {phase} requires {name}")

    for path in sorted(workspace.glob("*.md")):
        text = read(path)
        rel = path.name
        if rel in HIGH_TRAFFIC_BUDGETS and line_count(text) > HIGH_TRAFFIC_BUDGETS[rel] and not has_budget_override(text):
            errors.append(f"{path}: {line_count(text)} lines exceeds budget {HIGH_TRAFFIC_BUDGETS[rel]} without Budget override")
        validate_mermaid(path, text, errors)

        if rel in REQUIRED_HEADINGS:
            present = headings(text)
            for heading in REQUIRED_HEADINGS[rel]:
                if heading not in present:
                    errors.append(f"{path}: missing heading '{heading}'")
        if rel in {"README.md", "index.md"}:
            validate_workspace_map(path, text, errors)
        if rel == "flows.md":
            validate_flows(path, text, errors)
        if rel == "tasks.md":
            validate_tasks(path, text, errors)
        if rel == "decisions.md":
            validate_id_register(path, text, "DEC", errors)
        if rel == "questions.md":
            validate_id_register(path, text, "Q", errors)
        if rel == "drift.md":
            validate_id_register(path, text, "DRIFT", errors)
        if rel in {"evidence.md", "proof.md", "browser-evidence.md"}:
            validate_evidence_entries(path, text, errors)

        for link in MD_LINK_RE.findall(text):
            target = local_link_target(workspace, link)
            if target is None:
                continue
            try:
                target.relative_to(workspace.resolve())
            except ValueError:
                continue
            if not target.exists():
                errors.append(f"{path}: stale local link to {link}")

    spec = read(workspace / "spec.md") if (workspace / "spec.md").exists() else ""
    tasks = read(workspace / "tasks.md") if (workspace / "tasks.md").exists() else ""
    trace = read(workspace / "traceability.md") if (workspace / "traceability.md").exists() else ""
    evidence_path = evidence_file(workspace)
    evidence = read(evidence_path) if evidence_path else ""
    browser_evidence = read(workspace / "browser-evidence.md") if (workspace / "browser-evidence.md").exists() else ""

    ac_ids = set(ID_PATTERNS["AC"].findall(spec))
    req_ids = set(ID_PATTERNS["REQ"].findall(spec))
    slice_ids = set(ID_PATTERNS["SLICE"].findall(tasks))
    evid_ids = set(ID_PATTERNS["EVID"].findall(evidence + "\n" + browser_evidence))

    if spec and not ac_ids:
        errors.append(f"{workspace / 'spec.md'}: no AC-### acceptance criteria found")
    for old in sorted(set(LEGACY_AC_RE.findall(spec + tasks + trace))):
        errors.append(f"{workspace}: legacy acceptance id {old}; use AC-###")
    for old in sorted(set(LEGACY_SLICE_RE.findall(tasks))):
        errors.append(f"{workspace}: legacy slice label {old}; use SLICE-###")

    for ac in sorted(ac_ids):
        if tasks and ac not in tasks:
            errors.append(f"{workspace / 'tasks.md'}: {ac} is not referenced by any slice")
        if trace and ac not in trace:
            errors.append(f"{workspace / 'traceability.md'}: missing {ac}")

    if tasks:
        for slice_id in sorted(slice_ids):
            if not re.search(rf"{slice_id}.*AC-\d{{3}}|AC-\d{{3}}.*{slice_id}", tasks, re.S):
                errors.append(f"{workspace / 'tasks.md'}: {slice_id} has no AC-### reference")
            if trace and slice_id not in trace:
                errors.append(f"{workspace / 'traceability.md'}: missing {slice_id}")

    if trace:
        rows = "\n".join(table_rows(trace))
        for ac in sorted(ac_ids):
            if ac not in rows:
                errors.append(f"{workspace / 'traceability.md'}: {ac} absent from coverage matrix")
        if phase in {"prove", "polish", "review", "seal", "ship", "done"}:
            for eid in sorted(evid_ids):
                if eid not in trace:
                    errors.append(f"{workspace / 'traceability.md'}: evidence ID {eid} from evidence/browser proof is not mapped")

    for slice_id in sorted(completed_slices(tasks)):
        if not evidence:
            errors.append(f"{workspace}: completed {slice_id} has no evidence.md/proof.md")
        elif slice_id not in evidence:
            errors.append(f"{evidence_path}: completed {slice_id} is not referenced by evidence")

    if phase in {"plan", "vet", "build", "prove", "polish", "review", "seal", "ship", "done"}:
        qfile = workspace / "questions.md"
        if qfile.exists():
            for qid in blocking_questions(read(qfile)):
                errors.append(f"{qfile}: unresolved blocking/escalating question {qid} blocks phase {phase}")

    if req_ids and trace:
        for req in sorted(req_ids):
            if req not in trace:
                errors.append(f"{workspace / 'traceability.md'}: missing {req}")

    return errors


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="*", default=["."], help="workspace dirs or roots containing .devrites/work|features|archive")
    args = parser.parse_args(argv)

    workspaces: list[Path] = []
    for raw in args.paths:
        workspaces.extend(find_local_workspaces(Path(raw)))

    if not workspaces:
        print("workspace-schema: no workspaces found", file=sys.stderr)
        return 2

    errors: list[str] = []
    for workspace in sorted(set(p.resolve() for p in workspaces)):
        errors.extend(validate_workspace(workspace))

    if errors:
        for err in errors:
            print(f"FAIL: {err}", file=sys.stderr)
        print(f"workspace-schema: {len(errors)} error(s) across {len(set(workspaces))} workspace(s)", file=sys.stderr)
        return 1

    print(f"workspace-schema: OK — {len(set(workspaces))} workspace(s) validated")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
