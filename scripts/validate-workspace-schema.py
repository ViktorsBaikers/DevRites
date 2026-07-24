#!/usr/bin/env python3
"""Validate generated DevRites workspace artifacts against the Markdown contract.

The validator uses only the standard library, so it does not add a runtime
parser dependency.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path

from workflow_schema import PHASES, cursor_field


READINESS_CONTRACT_PATH = (
    Path(__file__).resolve().parents[1]
    / "engine"
    / "internal"
    / "lib"
    / "readiness_contract.json"
)
READINESS_CONTRACT = json.loads(READINESS_CONTRACT_PATH.read_text(encoding="utf-8"))
READINESS_PLACEHOLDER_RE = re.compile(
    r"<[^>\n]+>|\b(?:TODO|TBD|FIXME|UNKNOWN)\b", re.IGNORECASE
)
READINESS_DIGEST_RE = re.compile(r"^[0-9a-f]{64}$")
READINESS_AC_RE = re.compile(r"\bAC-?\d+\b", re.IGNORECASE)


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
    "decision-coverage.md": 200,
    "ai-spec.md": 160,
    "plan.md": 220,
    "tasks.md": 280,
    "traceability.md": 220,
    "eng-review.md": 240,
    "test-plan.md": 260,
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
    "decision-coverage.md": [
        "Topology",
        "Coverage matrix",
        "Assumption audit",
        "Residual uncertainty",
        "Readiness verdict",
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
    "Forge",
    "Forge strategies",
    "Forge scorecard",
    "Files likely touched",
    "Tests/proof",
    "Mode",
    "Gate",
    "Dependencies",
    "Done condition",
)

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
        value = cursor_field(p, "phase")
        if value:
            return value.split()[0].lower()
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
    # Detect workspaces by DevRites artifacts, not README.md. Most repository
    # roots have a README, which would misclassify the root and stop the
    # .devrites/work scan before it starts.
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

        def field(name: str) -> str:
            match = re.search(rf"(?im)^{re.escape(name)}\s*:\s*(.+?)\s*$", body)
            return match.group(1).strip() if match else ""

        forge = field("Forge")
        strategies = field("Forge strategies")
        scorecard = field("Forge scorecard")
        if not forge:
            continue
        if re.fullmatch(r"(?i)no", forge):
            if strategies.lower() != "none" or scorecard.lower() != "none":
                errors.append(
                    f"{path}: {slice_id} Forge:no requires strategies and scorecard 'none'"
                )
            continue
        if not re.fullmatch(r"(?i)yes\s+[—-]\s+\S.*", forge):
            errors.append(f"{path}: {slice_id} Forge must be 'no' or 'yes — <reason>'")
            continue

        parsed: list[tuple[str, str]] = []
        for part in strategies.split("|"):
            match = re.fullmatch(r"\s*([ABC])\s*=\s*(\S.*?)\s*", part)
            if not match:
                parsed = []
                break
            parsed.append((match.group(1), " ".join(match.group(2).split())))
        ids = [item[0] for item in parsed]
        descriptions = [item[1].casefold() for item in parsed]
        if ids not in (["A", "B"], ["A", "B", "C"]) or len(set(descriptions)) != len(
            descriptions
        ):
            errors.append(
                f"{path}: {slice_id} Forge strategies must be 2-3 distinct contiguous A-C entries"
            )

        acceptance = re.search(r"(?i)\bacceptance\s*=\s*[^;\n]*", scorecard)
        satisfies_ids = set(ID_PATTERNS["AC"].findall(field("Satisfies")))
        scorecard_ids = set(ID_PATTERNS["AC"].findall(acceptance.group(0))) if acceptance else set()
        test_plan = re.search(
            r"(?i)\btest-plan\s*=\s*(?=\S)(?!none\b).*test-plan\.md", scorecard
        )
        if (
            not satisfies_ids
            or not satisfies_ids.issubset(scorecard_ids)
            or not test_plan
        ):
            errors.append(
                f"{path}: {slice_id} Forge scorecard must bind every Satisfies AC ID and exact test-plan.md rows or commands"
            )


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
        if status and status.group(1).lower() == "open" and gate and gate.group(1).lower() in {"blocking", "validating", "escalating"}:
            out.append(qid.group(0) if qid else "<unknown>")

    status_index = gate_index = qid_index = -1
    for raw in questions.splitlines():
        line = raw.strip()
        if not (line.startswith("|") and line.endswith("|")):
            status_index = gate_index = qid_index = -1
            continue
        cells = [cell.strip() for cell in line[1:-1].split("|")]
        lowered = [cell.lower() for cell in cells]
        if status_index < 0:
            status_index = lowered.index("status") if "status" in lowered else -1
            gate_index = lowered.index("gate") if "gate" in lowered else -1
            for label in ("question id", "question", "id"):
                if label in lowered:
                    qid_index = lowered.index(label)
                    break
            if status_index < 0 or gate_index < 0:
                status_index = gate_index = qid_index = -1
            continue
        if status_index >= len(cells) or gate_index >= len(cells):
            continue
        if cells[status_index].lower() != "open" or cells[gate_index].lower() not in {
            "blocking",
            "validating",
            "escalating",
        }:
            continue
        qid = cells[qid_index] if 0 <= qid_index < len(cells) else "<unknown>"
        out.append(qid or "<unknown>")
    return list(dict.fromkeys(out))


def readiness_digest(workspace: Path, names: list[str]) -> str:
    digest = hashlib.sha256()
    for name in names:
        path = workspace / name
        data = path.read_bytes()
        if not data.strip():
            raise ValueError(f"{name} is empty")
        digest.update(name.encode())
        digest.update(b"\0")
        digest.update(str(len(data)).encode())
        digest.update(b"\0")
        digest.update(data)
        digest.update(b"\0")
    return digest.hexdigest()


def cursor_values(text: str, key: str) -> list[str]:
    want = key.strip().lower().replace("_", " ")
    values: list[str] = []
    for raw in text.splitlines():
        line = raw.strip()
        if line.startswith("|") and line.endswith("|"):
            cells = [cell.strip() for cell in line[1:-1].split("|")]
            if len(cells) >= 2 and cells[0].lower().replace("_", " ") == want:
                values.append(cells[1])
            continue
        line = line.lstrip("-*+ \t")
        if ":" not in line:
            continue
        found, value = line.split(":", 1)
        if found.strip().lower().replace("_", " ") == want:
            values.append(value.strip())
    return values


def markdown_section(text: str, wanted: str) -> str | None:
    lines = text.splitlines()
    start = -1
    level = 0
    for index, line in enumerate(lines):
        match = re.match(r"^(#{1,6})\s+(.+?)\s*#*\s*$", line.strip())
        if not match:
            continue
        if start < 0:
            if match.group(2).strip().lower() == wanted.lower():
                start = index + 1
                level = len(match.group(1))
            continue
        if len(match.group(1)) <= level:
            return "\n".join(lines[start:index])
    return "\n".join(lines[start:]) if start >= 0 else None


def markdown_table(text: str, heading: str) -> list[list[str]]:
    section = markdown_section(text, heading)
    if section is None:
        return []
    rows: list[list[str]] = []
    header_seen = False
    for raw in section.splitlines():
        line = raw.strip()
        if not (line.startswith("|") and line.endswith("|")):
            continue
        cells = [cell.strip() for cell in line[1:-1].split("|")]
        if all(not cell.strip(" :-") for cell in cells):
            continue
        if not header_seen:
            header_seen = True
            continue
        rows.append(cells)
    return rows


def empty_or_na(value: str) -> bool:
    return value.strip().lower() in {"", "-", "n/a", "none"}


def substantive_cells(row: list[str], *indexes: int) -> bool:
    return all(index < len(row) and not empty_or_na(row[index]) for index in indexes)


def none_row(value: str) -> bool:
    return value.strip().lower() in {
        "none",
        "no material assumptions",
        "no residual uncertainty",
    }


def validate_readiness_base(
    workspace: Path, contract: dict[str, object], errors: list[str]
) -> str | None:
    path = workspace / str(contract["artifact"])
    if not path.exists():
        errors.append(f"{path}: missing readiness artifact")
        return None
    text = read(path)
    if not text.strip():
        errors.append(f"{path}: empty readiness artifact")
        return None
    if READINESS_PLACEHOLDER_RE.search(text):
        errors.append(f"{path}: contains an unresolved placeholder")
    for heading in contract.get("requiredHeadings", []):
        if markdown_section(text, str(heading)) is None:
            errors.append(f"{path}: missing heading '{heading}'")
    for heading in contract.get("requiredTables", []):
        if not markdown_table(text, str(heading)):
            errors.append(f"{path}: heading '{heading}' has no table data rows")

    verdicts = cursor_values(text, str(contract["verdictField"]))
    if len(verdicts) != 1 or verdicts[0].upper() != str(contract["readyValue"]).upper():
        errors.append(
            f"{path}: must contain exactly one "
            f"{contract['verdictField']}: {contract['readyValue']}"
        )
    digests = cursor_values(text, str(contract["digestField"]))
    if len(digests) != 1 or not READINESS_DIGEST_RE.fullmatch(digests[0].lower()):
        errors.append(f"{path}: must contain exactly one valid {contract['digestField']}")
    else:
        try:
            expected = readiness_digest(workspace, list(contract["inputs"]))
        except (OSError, ValueError) as exc:
            errors.append(f"{path}: cannot compute readiness digest: {exc}")
        else:
            if digests[0].lower() != expected:
                errors.append(f"{path}: input digest is stale")
    return text


def validate_decision_coverage(workspace: Path, errors: list[str]) -> None:
    contract = READINESS_CONTRACT["coverage"]
    text = validate_readiness_base(workspace, contract, errors)
    if text is None:
        return
    path = workspace / "decision-coverage.md"
    for index, row in enumerate(markdown_table(text, "Topology"), 1):
        if len(row) < 4 or not substantive_cells(row, 0, 1, 3):
            errors.append(f"{path}: topology row {index} is not evidence-backed")
    rows = markdown_table(text, "Coverage matrix")
    allowed = {"closed", "agent-owned", "not-applicable", "deferred-nonblocking"}
    for index, row in enumerate(rows, 1):
        if len(row) < 6:
            errors.append(f"{path}: coverage row {index} has fewer than 6 cells")
            continue
        status = row[2].strip().lower()
        if not substantive_cells(row, 0, 1, 5):
            errors.append(f"{path}: coverage row {index} is incomplete")
        if status not in allowed:
            errors.append(
                f"{path}: coverage row {index} has unresolved status {row[2]!r}"
            )
        if status == "closed" and not substantive_cells(row, 3):
            errors.append(f"{path}: coverage row {index} is closed without a canonical reference")
        if status in {"agent-owned", "deferred-nonblocking"} and empty_or_na(row[4]):
            errors.append(
                f"{path}: coverage row {index} has no owner/validation gate"
            )
    for index, row in enumerate(markdown_table(text, "Assumption audit"), 1):
        if len(row) < 6:
            errors.append(f"{path}: assumption row {index} has fewer than 6 cells")
        elif not none_row(row[0]) and not substantive_cells(row, 0, 1, 2, 3, 4, 5):
            errors.append(f"{path}: assumption row {index} is unowned or unverifiable")
    for index, row in enumerate(markdown_table(text, "Residual uncertainty"), 1):
        if len(row) < 4:
            errors.append(f"{path}: residual row {index} has fewer than 4 cells")
        elif not none_row(row[0]) and not substantive_cells(row, 0, 1, 2, 3):
            errors.append(f"{path}: residual row {index} is unowned or unverifiable")
    questions_path = workspace / "questions.md"
    if questions_path.exists():
        open_gates = blocking_questions(read(questions_path))
        if open_gates:
            errors.append(
                f"{questions_path}: open blocking/validating/escalating questions: "
                + ", ".join(open_gates)
            )


def validate_engineering_readiness(workspace: Path, errors: list[str]) -> None:
    contract = READINESS_CONTRACT["engineering"]
    text = validate_readiness_base(workspace, contract, errors)
    if text is None:
        return
    review_path = workspace / "eng-review.md"
    for index, row in enumerate(markdown_table(text, "2a. Build-entry preflight"), 1):
        if len(row) < 7:
            errors.append(f"{review_path}: preflight row {index} has fewer than 7 cells")
            continue
        verdict = row[6].strip().lower()
        if verdict not in {"pass", "n/a"}:
            errors.append(f"{review_path}: preflight row {index} is not passing")
        if not substantive_cells(row, 0) or (
            verdict == "pass" and not substantive_cells(row, 1, 2, 4, 5)
        ):
            errors.append(f"{review_path}: preflight row {index} lacks executable provenance")
    for index, row in enumerate(markdown_table(text, "2b. Implementation readiness"), 1):
        if (
            len(row) < 6
            or row[5].strip().lower() != "ready"
            or not substantive_cells(row, 0, 1, 2, 3, 4)
        ):
            errors.append(f"{review_path}: readiness row {index} is not ready")

    test_contract = READINESS_CONTRACT["testPlan"]
    test_path = workspace / str(test_contract["artifact"])
    if not test_path.exists():
        errors.append(f"{test_path}: missing test plan")
        return
    test_text = read(test_path)
    if not test_text.strip() or READINESS_PLACEHOLDER_RE.search(test_text):
        errors.append(f"{test_path}: empty or contains an unresolved placeholder")
        return
    for heading in test_contract.get("requiredHeadings", []):
        if markdown_section(test_text, str(heading)) is None:
            errors.append(f"{test_path}: missing heading '{heading}'")
    for heading in test_contract.get("requiredTables", []):
        rows = markdown_table(test_text, str(heading))
        if not rows:
            errors.append(f"{test_path}: heading '{heading}' has no table data rows")
            continue
        required_cells = (0, 1, 2, 3, 5)
        if heading == "Per-gap test requirements":
            required_cells = (0, 1, 2, 3, 4, 5, 6)
        for index, row in enumerate(rows, 1):
            if not substantive_cells(row, *required_cells):
                errors.append(f"{test_path}: heading '{heading}' row {index} is incomplete")
    mapping = markdown_section(test_text, "Acceptance → test map") or ""
    if "→" not in mapping and "->" not in mapping:
        errors.append(f"{test_path}: acceptance map has no mappings")
    spec_path = workspace / "spec.md"
    if spec_path.exists():
        for acceptance_id in sorted(set(READINESS_AC_RE.findall(read(spec_path)))):
            if acceptance_id.upper() not in mapping.upper():
                errors.append(f"{test_path}: acceptance map does not map {acceptance_id}")


def validate_workspace(workspace: Path) -> list[str]:
    errors: list[str] = []
    phase = phase_for(workspace)

    if not workspace_index_present(workspace):
        errors.append(f"{workspace}: missing README.md/index.md/feature.md workspace map")

    metadata = PHASES.get(phase, PHASES["spec"])
    required = list(metadata["workspaceRequired"])
    for name in required:
        if not existing_or_alias(workspace, name):
            errors.append(f"{workspace}: phase {phase} requires {name}")
    if "decision-coverage.md" in required:
        validate_decision_coverage(workspace, errors)
    if "eng-review.md" in required:
        validate_engineering_readiness(workspace, errors)

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
        if bool(metadata.get("proofRequired")):
            for eid in sorted(evid_ids):
                if eid not in trace:
                    errors.append(f"{workspace / 'traceability.md'}: evidence ID {eid} from evidence/browser proof is not mapped")

    for slice_id in sorted(completed_slices(tasks)):
        if not evidence:
            errors.append(f"{workspace}: completed {slice_id} has no evidence.md/proof.md")
        elif slice_id not in evidence:
            errors.append(f"{evidence_path}: completed {slice_id} is not referenced by evidence")

    if bool(metadata.get("blocksOpenQuestions")):
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
    parser.add_argument(
        "paths",
        nargs="*",
        default=["."],
        help="workspace directories or roots containing .devrites/work|features|archive",
    )
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

    print(f"workspace-schema: OK: {len(set(workspaces))} workspace(s) validated")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
