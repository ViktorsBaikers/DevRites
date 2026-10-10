#!/usr/bin/env bash
# Contract: the "Required by phase" table in workspace-artifact-schema.md must equal the
# engine's per-phase RequiredArtifacts (engine/internal/state/schema.go artifacts* lists).
# DOC / SCHEMA_GO env vars override the inputs (used for mutation checks).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export DOC="${DOC:-$ROOT/pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md}"
export SCHEMA_GO="${SCHEMA_GO:-$ROOT/engine/internal/state/schema.go}"
python3 -I - <<'PY'
import os, re, sys

# Engine side: evaluate the artifacts* lists, then map each phase Target to its list.
src = open(os.environ["SCHEMA_GO"]).read()
lists = {}
for name, body in re.findall(r"^\s*(artifacts\w+)\s*=\s*(.+)$", src, re.M):
    parent = re.search(r"\.\.\.\)", body) and re.search(r"(artifacts\w+)\.\.\.", body)
    items = re.findall(r'"([^"]+)"', body)
    lists[name] = (lists[parent.group(1)] if parent else []) + items
engine = {}
for target, lst in re.findall(r"Target:\s*Phase(\w+).*?RequiredArtifacts:\s*(artifacts\w+)", src):
    engine[target.lower()] = set(lists[lst])
assert engine, "no phase rows parsed from schema.go"

# Doc side: expand cumulative "<x> artifacts plus ..." rows; drop ';' notes and 'when' conditionals.
doc = open(os.environ["DOC"]).read()
table = doc.split("## Required by phase", 1)[1].split("\n## ", 1)[0]
prev, rows = set(), {}
for line in table.splitlines():
    cells = [c.strip() for c in line.strip().strip("|").split("|")]
    if len(cells) != 2 or cells[0] in ("Phase", "---", "conditional"):
        continue
    cell = cells[1].split(";")[0]
    cur = set(prev) if "artifacts plus" in cell else set()
    for piece in cell.split(" plus ")[-1].split(", ") if "artifacts plus" in cell else cell.split(", "):
        if " when " not in piece:
            cur |= set(re.findall(r"`([^`]+)`", piece))
    for phase in cells[0].split("/"):
        rows[phase] = cur
    prev = cur

bad = []
for phase, want in rows.items():
    got = engine.get(phase)
    if got is None:
        bad.append(f"{phase}: no engine row")
    elif got != want:
        bad.append(f"{phase}: doc-only={sorted(want - got)} engine-only={sorted(got - want)}")
for phase in ("frame", "spec", "clarify", "temper", "define", "plan", "vet", "build", "converge", "prove", "polish", "review", "seal", "ship", "done"):
    if phase not in rows:
        bad.append(f"{phase}: missing from doc table")
if bad:
    print("FAIL workspace-artifact-schema contract:\n  " + "\n  ".join(bad))
    sys.exit(1)
print(f"PASS workspace-artifact-schema contract ({len(rows)} phases compared)")
PY
