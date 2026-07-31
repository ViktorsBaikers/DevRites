#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
VALIDATOR="$ROOT/scripts/validate-workspace-schema.py"
FIXTURES="$ROOT/tests/fixtures/workspace-schema"
CANONICAL_SCHEMA="$ROOT/pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md"

python3 "$VALIDATOR" "$FIXTURES" >/tmp/devrites-workspace-schema-ok.txt

PENDING_SLICE="$(mktemp -d)"
cp -R "$FIXTURES" "$PENDING_SLICE/fixtures"
perl -0pi -e 's/(\| SLICE-002 \| Pagination metadata \| AC-002 \| AFK \| advisory \| )built( \|)/${1}pending${2}/' \
  "$PENDING_SLICE/fixtures/.devrites/work/backend-api/tasks.md"
perl -0pi -e 's/(## SLICE-002 Pagination metadata.*?^Status: )built$/${1}pending/ms' \
  "$PENDING_SLICE/fixtures/.devrites/work/backend-api/tasks.md"
if python3 "$VALIDATOR" "$PENDING_SLICE/fixtures" >/tmp/devrites-workspace-schema-pending-slice.txt 2>&1; then
  echo "FAIL: proof-required workspace with a pending slice passed schema validation"
  exit 1
fi
grep -q 'SLICE-002' /tmp/devrites-workspace-schema-pending-slice.txt

CANONICAL_WORKSPACE="$(mktemp -d)"
cp -R "$FIXTURES" "$CANONICAL_WORKSPACE/fixtures"
{
  printf '# Tasks\n\n## Slice index\n\n'
  awk '/<!-- canonical-slice:start -->/{on=1; next} /<!-- canonical-slice:end -->/{on=0} on' "$CANONICAL_SCHEMA" \
    | sed '/^```/d'
} > "$CANONICAL_WORKSPACE/fixtures/.devrites/work/backend-api/tasks.md"
# Put the workspace in plan phase because this case checks slice grammar, not
# later-phase proof artifacts.
perl -0pi -e 's/\| phase \| prove \|/| phase | plan |/' \
  "$CANONICAL_WORKSPACE/fixtures/.devrites/work/backend-api/state.md"
perl -0pi -e 's/^phase: prove$/phase: plan/m' \
  "$CANONICAL_WORKSPACE/fixtures/.devrites/work/backend-api/README.md"
python3 "$VALIDATOR" "$CANONICAL_WORKSPACE/fixtures" >/tmp/devrites-workspace-schema-canonical.txt

BAD="$(mktemp -d)"
mkdir -p "$BAD/.devrites/work/broken"
cat > "$BAD/.devrites/work/broken/README.md" <<'MD'
# Broken
phase: plan
MD
cat > "$BAD/.devrites/work/broken/state.md" <<'MD'
# State
phase: plan
MD
cat > "$BAD/.devrites/work/broken/spec.md" <<'MD'
# Spec

## Acceptance criteria
- [ ] [AC1] legacy id should fail.
MD

if python3 "$VALIDATOR" "$BAD" >/tmp/devrites-workspace-schema-bad.txt 2>&1; then
  echo "FAIL: invalid workspace passed schema validation"
  cat /tmp/devrites-workspace-schema-bad.txt
  exit 1
fi

grep -q 'legacy acceptance id AC1' /tmp/devrites-workspace-schema-bad.txt

DATED_QUESTIONS="$(mktemp -d)"
cp -R "$FIXTURES" "$DATED_QUESTIONS/fixtures"
cat > "$DATED_QUESTIONS/fixtures/.devrites/work/ui-settings-toggle/questions.md" <<'MD'
# Questions

## Question register

## q-2026-08-01-001
status: answered
slice: spec
gate: validating
question: Should copy say digest or summary?
answer: digest
impact: AC-001
MD
python3 "$VALIDATOR" "$DATED_QUESTIONS/fixtures" \
  >/tmp/devrites-workspace-schema-dated-questions.txt

DUPLICATE_IDS="$(mktemp -d)"
cp -R "$FIXTURES" "$DUPLICATE_IDS/fixtures"
cat >> "$DUPLICATE_IDS/fixtures/.devrites/work/ui-settings-toggle/spec.md" <<'MD'

- REQ-001: A second definition must fail.
MD
cat >> "$DUPLICATE_IDS/fixtures/.devrites/work/ui-settings-toggle/browser-evidence.md" <<'MD'
| EVID-001 | /settings | 375 | duplicate evidence identity | AC-001, SLICE-001 |
MD
cat >> "$DUPLICATE_IDS/fixtures/.devrites/work/ui-settings-toggle/tasks.md" <<'MD'

## SLICE-001 Duplicate slice identity
MD
cat >> "$DUPLICATE_IDS/fixtures/.devrites/work/ui-settings-toggle/questions.md" <<'MD'

## q-2026-08-01-001
status: answered
gate: advisory

## q-2026-08-01-001
status: answered
gate: advisory
MD
if python3 "$VALIDATOR" "$DUPLICATE_IDS/fixtures" \
  >/tmp/devrites-workspace-schema-duplicate-ids.txt 2>&1; then
  echo "FAIL: duplicate canonical IDs passed schema validation"
  exit 1
fi
grep -q 'duplicate REQ-001 definition' /tmp/devrites-workspace-schema-duplicate-ids.txt
grep -q 'duplicate SLICE-001 definition' /tmp/devrites-workspace-schema-duplicate-ids.txt
grep -q 'duplicate EVID-001 definition' /tmp/devrites-workspace-schema-duplicate-ids.txt
grep -q 'duplicate q-2026-08-01-001 definition' /tmp/devrites-workspace-schema-duplicate-ids.txt

CANONICAL_PHASE="$(mktemp -d)"
mkdir -p "$CANONICAL_PHASE/.devrites/work/converging"
cat > "$CANONICAL_PHASE/.devrites/work/converging/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | converge |
| status | running |
MD
if python3 "$VALIDATOR" "$CANONICAL_PHASE" >/tmp/devrites-workspace-schema-canonical-phase.txt 2>&1; then
  echo "FAIL: incomplete canonical converge workspace passed schema validation"
  exit 1
fi
grep -q 'phase converge requires architecture.md' /tmp/devrites-workspace-schema-canonical-phase.txt

README_PHASE_ONLY="$(mktemp -d)"
mkdir -p "$README_PHASE_ONLY/.devrites/work/readme-phase-only"
cat > "$README_PHASE_ONLY/.devrites/work/readme-phase-only/README.md" <<'MD'
# README Phase Only
phase: plan
MD
cat > "$README_PHASE_ONLY/.devrites/work/readme-phase-only/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| status | running |
MD
if python3 "$VALIDATOR" "$README_PHASE_ONLY" \
  >/tmp/devrites-workspace-schema-readme-phase-only.txt 2>&1; then
  echo "FAIL: README phase replaced missing state.md authority"
  exit 1
fi
grep -q 'no phase in state.md' /tmp/devrites-workspace-schema-readme-phase-only.txt

MISSING_FIELD="$(mktemp -d)"
mkdir -p "$MISSING_FIELD/.devrites/work/missing-slice-field"
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/README.md" <<'MD'
# Missing Field
phase: plan
status: running
next_action: /rite-vet
last_updated: 2026-07-07

## Artifact map
| File | Job |
| --- | --- |
| spec.md | Product contract |

## Read next
| Phase / role | Read |
| --- | --- |
| Builder | tasks.md |

## Blocking gates
None.
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/brief.md" <<'MD'
# Brief

## Objective
Do it.

## Non-goals
- None.

## Success definition
AC-001 passes.
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/spec.md" <<'MD'
# Spec

## Problem
Missing behavior.

## Goal
Add behavior.

## Non-goals
- None.

## Users / actors
| Actor | Need |
| --- | --- |
| User | Behavior. |

## Requirements
- REQ-001: The system MUST do the behavior.

## Acceptance criteria
- [ ] AC-001: Given input, when run, then output appears. (REQ-001)

## Edge cases
- Empty input.

## Measurable success
- AC-001 is proven.

## Scope boundaries
- Feature only.
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/architecture.md" <<'MD'
# Architecture

## Owning module / layer
Module.

## Integration points
None.

## Data / API / events
None.

## Dependencies
None.

## Risks
None.

## Affected boundaries
Module boundary.
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/plan.md" <<'MD'
# Plan

## Approach
Implement directly.

## Slice strategy
One slice.

## Validation strategy
Focused test.

## Rollback
Revert.
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/tasks.md" <<'MD'
# Tasks

## Slice index
| Slice ID | Goal | AC IDs |
| --- | --- | --- |
| SLICE-001 | Do behavior | AC-001 |

## SLICE-001 Do behavior
Goal: Add behavior.
Satisfies: AC-001
Tests/proof: pending
Mode: AFK
Gate: advisory
Dependencies: none
Done condition: AC-001 passes.
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/traceability.md" <<'MD'
# Traceability

## Coverage matrix
| AC / REQ ID | Slice IDs | Test / proof | Evidence ID | Touched files | Status |
| --- | --- | --- | --- | --- | --- |
| AC-001 / REQ-001 | SLICE-001 | focused test | pending | src/example.ts | planned |
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | plan |
| status | running |
| next_action | /rite-vet |
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/decisions.md" <<'MD'
# Decisions

## Decision log
| Decision ID | Status | Context | Options | Decision | Consequences | Related IDs |
| --- | --- | --- | --- | --- | --- | --- |
| DEC-001 | accepted | Simple feature. | direct / indirect | direct | small diff | AC-001 |
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/assumptions.md" <<'MD'
# Assumptions

## Assumption register
| ID | Assumption | Confidence | Owner | Validation status |
| --- | --- | --- | --- | --- |
| ASM-001 | No migration. | high | agent | pending |
MD
cat > "$MISSING_FIELD/.devrites/work/missing-slice-field/questions.md" <<'MD'
# Questions

## Question register
| Question ID | Status | Gate | Question | Answer | Impact |
| --- | --- | --- | --- | --- | --- |
| Q-001 | answered | advisory | Any UI? | No. | AC-001 |
MD
if python3 "$VALIDATOR" "$MISSING_FIELD" >/tmp/devrites-workspace-schema-missing-field.txt 2>&1; then
  echo "FAIL: workspace with missing slice field passed schema validation"
  cat /tmp/devrites-workspace-schema-missing-field.txt
  exit 1
fi
grep -q "SLICE-001 missing field 'Files likely touched:'" /tmp/devrites-workspace-schema-missing-field.txt

STALE_EVIDENCE="$(mktemp -d)"
cp -R "$FIXTURES" "$STALE_EVIDENCE/fixtures"
perl -0pi -e 's/, EVID-003//g' "$STALE_EVIDENCE/fixtures/.devrites/work/ui-settings-toggle/traceability.md"
if python3 "$VALIDATOR" "$STALE_EVIDENCE/fixtures" >/tmp/devrites-workspace-schema-stale-evidence.txt 2>&1; then
  echo "FAIL: workspace with unmapped browser evidence passed schema validation"
  cat /tmp/devrites-workspace-schema-stale-evidence.txt
  exit 1
fi
grep -q 'evidence ID EVID-003' /tmp/devrites-workspace-schema-stale-evidence.txt

ALTERNATE_VERDICTS="$(mktemp -d)"
cp -R "$FIXTURES" "$ALTERNATE_VERDICTS/fixtures"
perl -0pi -e 's/Decision coverage: CLEAR/Decision coverage: NEEDS CLARIFICATION/' \
  "$ALTERNATE_VERDICTS/fixtures/.devrites/work/backend-api/decision-coverage.md"
perl -0pi -e 's/Implementation readiness: READY/Implementation readiness: NEEDS REPLAN/' \
  "$ALTERNATE_VERDICTS/fixtures/.devrites/work/backend-api/eng-review.md"
python3 "$VALIDATOR" "$ALTERNATE_VERDICTS/fixtures" \
  >/tmp/devrites-workspace-schema-alternate-verdicts.txt

EMPTY_VERDICT="$(mktemp -d)"
cp -R "$FIXTURES" "$EMPTY_VERDICT/fixtures"
perl -0pi -e 's/Decision coverage: CLEAR/Decision coverage:/' \
  "$EMPTY_VERDICT/fixtures/.devrites/work/backend-api/decision-coverage.md"
if python3 "$VALIDATOR" "$EMPTY_VERDICT/fixtures" \
  >/tmp/devrites-workspace-schema-empty-verdict.txt 2>&1; then
  echo "FAIL: empty decision-coverage verdict passed validation"
  exit 1
fi
grep -q 'must contain exactly one nonempty Decision coverage verdict' \
  /tmp/devrites-workspace-schema-empty-verdict.txt

MARKER_ONLY="$(mktemp -d)"
cp -R "$FIXTURES" "$MARKER_ONLY/fixtures"
printf 'Decision coverage: CLEAR\n' \
  > "$MARKER_ONLY/fixtures/.devrites/work/backend-api/decision-coverage.md"
if python3 "$VALIDATOR" "$MARKER_ONLY/fixtures" >/tmp/devrites-workspace-schema-marker-only.txt 2>&1; then
  echo "FAIL: marker-only readiness artifact passed validation"
  exit 1
fi
grep -q "missing heading 'Topology'" /tmp/devrites-workspace-schema-marker-only.txt

EMPTY_TEST_PLAN="$(mktemp -d)"
cp -R "$FIXTURES" "$EMPTY_TEST_PLAN/fixtures"
: > "$EMPTY_TEST_PLAN/fixtures/.devrites/work/backend-api/test-plan.md"
if python3 "$VALIDATOR" "$EMPTY_TEST_PLAN/fixtures" >/tmp/devrites-workspace-schema-empty-test-plan.txt 2>&1; then
  echo "FAIL: empty test plan passed validation"
  exit 1
fi
grep -q 'empty artifact' /tmp/devrites-workspace-schema-empty-test-plan.txt

PYTHONPATH="$ROOT/scripts" python3 - "$ROOT/engine/internal/markdowntext/testdata/structural.json" <<'PY'
import json
import sys
from pathlib import Path

from workflow_schema import cursor_field_text, decode_markdown, structural_markdown

cases = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert cases
for case in cases:
    view = structural_markdown(case["input"])
    assert view == case["output"], case["name"]
    assert len(view.encode()) == len(case["input"].encode()), case["name"]
assert cursor_field_text("~~~md\nphase: hidden\n~~~\nphase: build\n", "phase") == "build"
for data, expected in ((b"bad\x00", "NUL"), (b"bad\xff", "UTF-8")):
    try:
        decode_markdown(data, "test.md")
    except ValueError as exc:
        assert expected in str(exc)
    else:
        raise AssertionError(f"{expected} input was accepted")
PY

CURSOR_FILE="$(mktemp)"
cat > "$CURSOR_FILE" <<'MD'
~~~md
| phase | hidden |
~~~
| phase | build |
MD
test "$(python3 "$ROOT/scripts/workflow_schema.py" field "$CURSOR_FILE" phase)" = "build"
printf '\0' >> "$CURSOR_FILE"
if python3 "$ROOT/scripts/workflow_schema.py" field "$CURSOR_FILE" phase \
  >/tmp/devrites-workflow-schema-corrupt.txt 2>&1; then
  echo "FAIL: corrupt cursor Markdown was accepted"
  exit 1
fi
grep -q 'NUL' /tmp/devrites-workflow-schema-corrupt.txt
if grep -q 'Traceback' /tmp/devrites-workflow-schema-corrupt.txt; then
  echo "FAIL: corrupt cursor error included a traceback"
  exit 1
fi

FENCED="$(mktemp -d)"
cp -R "$FIXTURES" "$FENCED/fixtures"
python3 - "$VALIDATOR" "$FENCED/fixtures/.devrites/work/backend-api" <<'PY'
import runpy
import sys
from pathlib import Path

validator = Path(sys.argv[1])
workspace = Path(sys.argv[2])
sys.path.insert(0, str(validator.parent))
schema = runpy.run_path(str(validator))

def prepend(name: str, body: str) -> None:
    path = workspace / name
    path.write_text(body + path.read_text(encoding="utf-8"), encoding="utf-8")

prepend("questions.md", "```md\n## Q-999\nstatus: open\ngate: blocking\n```\n")
prepend("spec.md", "```md\n## Acceptance criteria\n- AC-999 example\nTODO\n[stale](missing.md)\n```\n")
prepend("tasks.md", "```md\n## SLICE-999 Example\nGoal: fake\nSlice 99\n```\n")
prepend(
    "decision-coverage.md",
    "```md\nDecision coverage: BLOCKED\nTODO\n## Topology\nnot a table\n```\n",
)
prepend(
    "test-plan.md",
    "```md\nTODO\n## Build-entry preflight\nnot a table\n"
    "## Acceptance → test map\n- AC-999 -> fake\n```\n",
)
prepend(
    "eng-review.md",
    "```md\nImplementation readiness: BLOCKED\nTODO\n"
    "## 2a. Build-entry preflight\nnot a table\n```\n",
)
PY
python3 "$VALIDATOR" "$FENCED/fixtures" >/tmp/devrites-workspace-schema-fenced.txt

for kind in nul utf8; do
  CORRUPT="$(mktemp -d)"
  cp -R "$FIXTURES" "$CORRUPT/fixtures"
  if [ "$kind" = nul ]; then
    printf '\0' >> "$CORRUPT/fixtures/.devrites/work/backend-api/spec.md"
    expected='NUL'
  else
    printf '\377' >> "$CORRUPT/fixtures/.devrites/work/backend-api/spec.md"
    expected='UTF-8'
  fi
  if python3 "$VALIDATOR" "$CORRUPT/fixtures" \
    >"/tmp/devrites-workspace-schema-corrupt-$kind.txt" 2>&1; then
    echo "FAIL: corrupt $kind Markdown passed workspace validation"
    exit 1
  fi
  grep -q "$expected" "/tmp/devrites-workspace-schema-corrupt-$kind.txt"
  if grep -q 'Traceback' "/tmp/devrites-workspace-schema-corrupt-$kind.txt"; then
    echo "FAIL: corrupt $kind validator error included a traceback"
    exit 1
  fi
done

FENCED_BUDGET="$(mktemp -d)"
cp -R "$FIXTURES" "$FENCED_BUDGET/fixtures"
{
  printf '\n```md\nBudget override: fake\n```\n'
  for _ in $(seq 1 300); do
    echo
  done
} >> "$FENCED_BUDGET/fixtures/.devrites/work/backend-api/spec.md"
if python3 "$VALIDATOR" "$FENCED_BUDGET/fixtures" \
  >/tmp/devrites-workspace-schema-fenced-budget.txt 2>&1; then
  echo "FAIL: fenced budget override bypassed the raw line-count budget"
  exit 1
fi
grep -q 'lines exceeds budget' /tmp/devrites-workspace-schema-fenced-budget.txt

BAD_MERMAID="$(mktemp -d)"
cp -R "$FIXTURES" "$BAD_MERMAID/fixtures"
perl -0pi -e 's/^sequenceDiagram$/unsupportedDiagram/m' \
  "$BAD_MERMAID/fixtures/.devrites/work/backend-api/architecture.md"
if python3 "$VALIDATOR" "$BAD_MERMAID/fixtures" \
  >/tmp/devrites-workspace-schema-bad-mermaid.txt 2>&1; then
  echo "FAIL: invalid raw Mermaid input passed validation"
  exit 1
fi
grep -q 'starts with unsupported syntax' /tmp/devrites-workspace-schema-bad-mermaid.txt

LEDGER_ONLY="$(mktemp -d)"
mkdir -p "$LEDGER_ONLY/.devrites/work/ledger-only"
cat > "$LEDGER_ONLY/.devrites/work/ledger-only/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | frame |
MD
python3 "$VALIDATOR" "$LEDGER_ONLY" >/tmp/devrites-workspace-schema-ledger-only.txt

UNKNOWN_PHASE="$(mktemp -d)"
mkdir -p "$UNKNOWN_PHASE/.devrites/work/unknown"
cat > "$UNKNOWN_PHASE/.devrites/work/unknown/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | invented |
MD
if python3 "$VALIDATOR" "$UNKNOWN_PHASE" >/tmp/devrites-workspace-schema-unknown.txt 2>&1; then
  echo "FAIL: unknown phase passed workspace validation"
  exit 1
fi
grep -q "unknown phase 'invented'" /tmp/devrites-workspace-schema-unknown.txt

REMNANTS="$(mktemp -d)"
for name in "native-engine-cleanup" "native-engine-cleanup-s1" "native-engine-cleanup-s10" "native-engine-cleanup-s11" "native-engine-cleanup-s12" "native-engine-cleanup-s13" "native-engine-cleanup-s14" "native-engine-cleanup-s15" "native-engine-cleanup-s16" "native-engine-cleanup-s16b" "native-engine-cleanup-s17" "native-engine-cleanup-s18" "native-engine-cleanup-s19" "native-engine-cleanup-s2" "native-engine-cleanup-s20" "native-engine-cleanup-s21" "native-engine-cleanup-s22" "native-engine-cleanup-s23" "native-engine-cleanup-s24" "native-engine-cleanup-s3" "native-engine-cleanup-s3b" "native-engine-cleanup-s4" "native-engine-cleanup-s5a" "native-engine-cleanup-s5b" "native-engine-cleanup-s6a" "native-engine-cleanup-s6b" "native-engine-cleanup-s7" "native-engine-cleanup-s8" "native-engine-cleanup-s9"; do
  mkdir -p "$REMNANTS/.devrites/work/$name"
done
printf 'bounded paths\n' > "$REMNANTS/.devrites/work/native-engine-cleanup/.wright-allowlist"
printf '{}\n' > "$REMNANTS/.devrites/work/native-engine-cleanup-s20/recovery-attempts.jsonl"
if python3 "$VALIDATOR" "$REMNANTS" >/tmp/devrites-workspace-schema-remnants.txt 2>&1; then
  echo "FAIL: operational remnants were treated as workspaces"
  exit 1
fi
grep -q "workspace-schema: no workspaces found" /tmp/devrites-workspace-schema-remnants.txt

SYMLINKS="$(mktemp -d)"
mkdir -p "$SYMLINKS/.devrites/work/file-link" "$SYMLINKS/target"
printf '| phase | invented |\n' > "$SYMLINKS/target/state.md"
ln -s "$SYMLINKS/target" "$SYMLINKS/.devrites/work/directory-link"
ln -s "$SYMLINKS/target/state.md" "$SYMLINKS/.devrites/work/file-link/state.md"
if python3 "$VALIDATOR" "$SYMLINKS" >/tmp/devrites-workspace-schema-symlinks.txt 2>&1; then
  echo "FAIL: symlinked workspace authority passed validation"
  exit 1
fi
grep -q "workspace-schema: no workspaces found" /tmp/devrites-workspace-schema-symlinks.txt

echo "ok: workspace schema validator enforces structural Markdown trust boundaries, canonical layouts, complete mappings and verdict shapes without semantic verdict policing"
