#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
VALIDATOR="$ROOT/scripts/validate-workspace-schema.py"
FIXTURES="$ROOT/tests/fixtures/workspace-schema"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

expect_msg() {
  grep -q -- "$2" "$1" || {
    echo "FAIL: $(basename "$1" .txt): expected '$2' in validator output"
    cat "$1"
    exit 1
  }
}
CANONICAL_SCHEMA="$ROOT/pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md"

python3 "$VALIDATOR" "$FIXTURES" >"$OUT/devrites-workspace-schema-ok.txt"

for phase in frame spec; do
  if python3 "$ROOT/scripts/workflow_schema.py" phase-property "$phase" blocksOpenQuestions; then
    echo "FAIL: $phase unexpectedly blocks open questions"
    exit 1
  fi
done
for phase in clarify temper define plan vet build converge prove polish review seal ship done; do
  if ! python3 "$ROOT/scripts/workflow_schema.py" phase-property "$phase" blocksOpenQuestions; then
    echo "FAIL: $phase does not block open questions"
    exit 1
  fi
done

PENDING_SLICE="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$PENDING_SLICE/fixtures"
perl -0pi -e 's/(\| SLICE-002 \| Pagination metadata \| AC-002 \| AFK \| advisory \| )built( \|)/${1}pending${2}/' \
  "$PENDING_SLICE/fixtures/.devrites/work/backend-api/tasks.md"
perl -0pi -e 's/(## SLICE-002 Pagination metadata.*?^Status: )built$/${1}pending/ms' \
  "$PENDING_SLICE/fixtures/.devrites/work/backend-api/tasks.md"
if python3 "$VALIDATOR" "$PENDING_SLICE/fixtures" >"$OUT/devrites-workspace-schema-pending-slice.txt" 2>&1; then
  echo "FAIL: proof-required workspace with a pending slice passed schema validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-pending-slice.txt" 'phase prove requires every slice built; incomplete: SLICE-002'

CANONICAL_WORKSPACE="$(mktemp -d "$OUT/case.XXXXXX")"
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
grep -q '^| phase | plan |' "$CANONICAL_WORKSPACE/fixtures/.devrites/work/backend-api/state.md" \
  || { echo "FAIL: canonical slice grammar case setup: state.md phase rewrite matched nothing"; exit 1; }
grep -q '^phase: plan$' "$CANONICAL_WORKSPACE/fixtures/.devrites/work/backend-api/README.md" \
  || { echo "FAIL: canonical slice grammar case setup: README.md phase rewrite matched nothing"; exit 1; }
if ! python3 "$VALIDATOR" "$CANONICAL_WORKSPACE/fixtures" >"$OUT/devrites-workspace-schema-canonical.txt" 2>&1; then
  echo "FAIL: canonical slice grammar case"
  cat "$OUT/devrites-workspace-schema-canonical.txt"
  exit 1
fi

BAD="$(mktemp -d "$OUT/case.XXXXXX")"
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

if python3 "$VALIDATOR" "$BAD" >"$OUT/devrites-workspace-schema-bad.txt" 2>&1; then
  echo "FAIL: invalid workspace passed schema validation"
  cat "$OUT/devrites-workspace-schema-bad.txt"
  exit 1
fi

expect_msg "$OUT/devrites-workspace-schema-bad.txt" 'legacy acceptance id AC1'

DATED_QUESTIONS="$(mktemp -d "$OUT/case.XXXXXX")"
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
  >"$OUT/devrites-workspace-schema-dated-questions.txt"

DUPLICATE_IDS="$(mktemp -d "$OUT/case.XXXXXX")"
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
  >"$OUT/devrites-workspace-schema-duplicate-ids.txt" 2>&1; then
  echo "FAIL: duplicate canonical IDs passed schema validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-duplicate-ids.txt" 'duplicate REQ-001 definition'
expect_msg "$OUT/devrites-workspace-schema-duplicate-ids.txt" 'duplicate SLICE-001 definition'
expect_msg "$OUT/devrites-workspace-schema-duplicate-ids.txt" 'duplicate EVID-001 definition'
expect_msg "$OUT/devrites-workspace-schema-duplicate-ids.txt" 'duplicate q-2026-08-01-001 definition'

CANONICAL_PHASE="$(mktemp -d "$OUT/case.XXXXXX")"
mkdir -p "$CANONICAL_PHASE/.devrites/work/converging"
cat > "$CANONICAL_PHASE/.devrites/work/converging/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | converge |
| status | running |
MD
if python3 "$VALIDATOR" "$CANONICAL_PHASE" >"$OUT/devrites-workspace-schema-canonical-phase.txt" 2>&1; then
  echo "FAIL: incomplete canonical converge workspace passed schema validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-canonical-phase.txt" 'phase converge requires architecture.md'

README_PHASE_ONLY="$(mktemp -d "$OUT/case.XXXXXX")"
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
  >"$OUT/devrites-workspace-schema-readme-phase-only.txt" 2>&1; then
  echo "FAIL: README phase replaced missing state.md authority"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-readme-phase-only.txt" 'no phase in state.md'

MISSING_FIELD="$(mktemp -d "$OUT/case.XXXXXX")"
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
if python3 "$VALIDATOR" "$MISSING_FIELD" >"$OUT/devrites-workspace-schema-missing-field.txt" 2>&1; then
  echo "FAIL: workspace with missing slice field passed schema validation"
  cat "$OUT/devrites-workspace-schema-missing-field.txt"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-missing-field.txt" "SLICE-001 missing field 'Files likely touched:'"

STALE_EVIDENCE="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$STALE_EVIDENCE/fixtures"
perl -0pi -e 's/, EVID-003//g' "$STALE_EVIDENCE/fixtures/.devrites/work/ui-settings-toggle/traceability.md"
if python3 "$VALIDATOR" "$STALE_EVIDENCE/fixtures" >"$OUT/devrites-workspace-schema-stale-evidence.txt" 2>&1; then
  echo "FAIL: workspace with unmapped browser evidence passed schema validation"
  cat "$OUT/devrites-workspace-schema-stale-evidence.txt"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-stale-evidence.txt" 'evidence ID EVID-003'

ALTERNATE_VERDICTS="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$ALTERNATE_VERDICTS/fixtures"
perl -0pi -e 's/Decision coverage: CLEAR/Decision coverage: NEEDS CLARIFICATION/' \
  "$ALTERNATE_VERDICTS/fixtures/.devrites/work/backend-api/decision-coverage.md"
perl -0pi -e 's/Implementation readiness: READY/Implementation readiness: NEEDS REPLAN/' \
  "$ALTERNATE_VERDICTS/fixtures/.devrites/work/backend-api/eng-review.md"
python3 "$VALIDATOR" "$ALTERNATE_VERDICTS/fixtures" \
  >"$OUT/devrites-workspace-schema-alternate-verdicts.txt"

EMPTY_VERDICT="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$EMPTY_VERDICT/fixtures"
perl -0pi -e 's/Decision coverage: CLEAR/Decision coverage:/' \
  "$EMPTY_VERDICT/fixtures/.devrites/work/backend-api/decision-coverage.md"
if python3 "$VALIDATOR" "$EMPTY_VERDICT/fixtures" \
  >"$OUT/devrites-workspace-schema-empty-verdict.txt" 2>&1; then
  echo "FAIL: empty decision-coverage verdict passed validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-empty-verdict.txt" 'must contain exactly one nonempty Decision coverage verdict'

MARKER_ONLY="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$MARKER_ONLY/fixtures"
printf 'Decision coverage: CLEAR\n' \
  > "$MARKER_ONLY/fixtures/.devrites/work/backend-api/decision-coverage.md"
if python3 "$VALIDATOR" "$MARKER_ONLY/fixtures" >"$OUT/devrites-workspace-schema-marker-only.txt" 2>&1; then
  echo "FAIL: marker-only readiness artifact passed validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-marker-only.txt" "missing heading 'Topology'"

EMPTY_TEST_PLAN="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$EMPTY_TEST_PLAN/fixtures"
: > "$EMPTY_TEST_PLAN/fixtures/.devrites/work/backend-api/test-plan.md"
if python3 "$VALIDATOR" "$EMPTY_TEST_PLAN/fixtures" >"$OUT/devrites-workspace-schema-empty-test-plan.txt" 2>&1; then
  echo "FAIL: empty test plan passed validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-empty-test-plan.txt" 'empty artifact'

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

CURSOR_FILE="$(mktemp "$OUT/cursor.XXXXXX")"
cat > "$CURSOR_FILE" <<'MD'
~~~md
| phase | hidden |
~~~
| phase | build |
MD
test "$(python3 "$ROOT/scripts/workflow_schema.py" field "$CURSOR_FILE" phase)" = "build"
printf '\0' >> "$CURSOR_FILE"
if python3 "$ROOT/scripts/workflow_schema.py" field "$CURSOR_FILE" phase \
  >"$OUT/devrites-workflow-schema-corrupt.txt" 2>&1; then
  echo "FAIL: corrupt cursor Markdown was accepted"
  exit 1
fi
expect_msg "$OUT/devrites-workflow-schema-corrupt.txt" 'NUL'
if grep -q 'Traceback' "$OUT/devrites-workflow-schema-corrupt.txt"; then
  echo "FAIL: corrupt cursor error included a traceback"
  exit 1
fi

FENCED="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$FENCED/fixtures"
python3 - "$FENCED/fixtures/.devrites/work/backend-api" <<'PY'
import sys
from pathlib import Path

workspace = Path(sys.argv[1])

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
python3 "$VALIDATOR" "$FENCED/fixtures" >"$OUT/devrites-workspace-schema-fenced.txt"

# The FENCED case above asserts that a stale local link, an open blocking
# question and an unreferenced AC are all IGNORED inside a fence. Each rule is
# therefore also exercised OUTSIDE a fence, on a workspace that is otherwise
# valid, so deleting the enforcement site turns the suite red.
STALE_LOCAL_LINK="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$STALE_LOCAL_LINK/fixtures"
printf '\nSee [the removed design brief](design-brief-archive.md).\n' \
  >> "$STALE_LOCAL_LINK/fixtures/.devrites/work/backend-api/spec.md"
if python3 "$VALIDATOR" "$STALE_LOCAL_LINK/fixtures" \
  >"$OUT/devrites-workspace-schema-stale-local-link.txt" 2>&1; then
  echo "FAIL: workspace with a stale local link passed schema validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-stale-local-link.txt" 'stale local link to design-brief-archive.md'

OPEN_BLOCKING_QUESTION="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$OPEN_BLOCKING_QUESTION/fixtures"
cat >> "$OPEN_BLOCKING_QUESTION/fixtures/.devrites/work/backend-api/questions.md" <<'MD'

## q-2026-08-02-001
status: open
slice: prove
gate: blocking
question: Does the cursor contract change on reset?
impact: AC-001
MD
if python3 "$VALIDATOR" "$OPEN_BLOCKING_QUESTION/fixtures" \
  >"$OUT/devrites-workspace-schema-open-blocking-question.txt" 2>&1; then
  echo "FAIL: phase-gated workspace with an open blocking question passed schema validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-open-blocking-question.txt" 'unresolved blocking/escalating question q-2026-08-02-001 blocks phase prove'

UNREFERENCED_AC="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$UNREFERENCED_AC/fixtures"
perl -0pi -e 's/(- \[ \] AC-002:.*\n)/$1- [ ] AC-003: Given a cursor reset, the response returns to the first page.\n/' \
  "$UNREFERENCED_AC/fixtures/.devrites/work/backend-api/spec.md"
perl -0pi -e 's/(\| AC-002 \/ REQ-002 [^\n]*\n)/$1| AC-003 | SLICE-001 | API contract cursor-reset test | pending | app\/serializers\/audit_event_page_serializer.rb | planned |\n/' \
  "$UNREFERENCED_AC/fixtures/.devrites/work/backend-api/traceability.md"
perl -0pi -e 's/(- AC-002 → T1\n)/$1- AC-003 → T1\n/' \
  "$UNREFERENCED_AC/fixtures/.devrites/work/backend-api/test-plan.md"
if python3 "$VALIDATOR" "$UNREFERENCED_AC/fixtures" \
  >"$OUT/devrites-workspace-schema-unreferenced-ac.txt" 2>&1; then
  echo "FAIL: spec acceptance criterion that no slice references passed schema validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-unreferenced-ac.txt" 'AC-003 is not referenced by any slice'

for kind in nul utf8; do
  CORRUPT="$(mktemp -d "$OUT/case.XXXXXX")"
  cp -R "$FIXTURES" "$CORRUPT/fixtures"
  if [ "$kind" = nul ]; then
    printf '\0' >> "$CORRUPT/fixtures/.devrites/work/backend-api/spec.md"
    expected='NUL'
  else
    printf '\377' >> "$CORRUPT/fixtures/.devrites/work/backend-api/spec.md"
    expected='UTF-8'
  fi
  if python3 "$VALIDATOR" "$CORRUPT/fixtures" \
    >"$OUT/devrites-workspace-schema-corrupt-$kind.txt" 2>&1; then
    echo "FAIL: corrupt $kind Markdown passed workspace validation"
    exit 1
  fi
  expect_msg "$OUT/devrites-workspace-schema-corrupt-$kind.txt" "$expected"
  if grep -q 'Traceback' "$OUT/devrites-workspace-schema-corrupt-$kind.txt"; then
    echo "FAIL: corrupt $kind validator error included a traceback"
    exit 1
  fi
done

FENCED_BUDGET="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$FENCED_BUDGET/fixtures"
{
  printf '\n```md\nBudget override: fake\n```\n'
  for _ in $(seq 1 300); do
    echo
  done
} >> "$FENCED_BUDGET/fixtures/.devrites/work/backend-api/spec.md"
if python3 "$VALIDATOR" "$FENCED_BUDGET/fixtures" \
  >"$OUT/devrites-workspace-schema-fenced-budget.txt" 2>&1; then
  echo "FAIL: fenced budget override bypassed the raw line-count budget"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-fenced-budget.txt" 'lines exceeds budget'

for kind in reasonless lowercase; do
  case "$kind" in
    reasonless) override=$'Budget override:\nprose on the next line' ;;
    lowercase) override='budget override: reason' ;;
  esac
  BAD_BUDGET="$(mktemp -d "$OUT/case.XXXXXX")"
  cp -R "$FIXTURES" "$BAD_BUDGET/fixtures"
  {
    printf '\n%s\n' "$override"
    for _ in $(seq 1 300); do
      echo
    done
  } >> "$BAD_BUDGET/fixtures/.devrites/work/backend-api/spec.md"
  if python3 "$VALIDATOR" "$BAD_BUDGET/fixtures" \
    >"$OUT/devrites-workspace-schema-bad-budget-$kind.txt" 2>&1; then
    echo "FAIL: $kind budget override bypassed the raw line-count budget"
    exit 1
  fi
  expect_msg "$OUT/devrites-workspace-schema-bad-budget-$kind.txt" 'lines exceeds budget'
done

INDENTED_BUDGET="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$INDENTED_BUDGET/fixtures"
{
  printf '\n  Budget override: reviewed reason\n'
  for _ in $(seq 1 300); do
    echo
  done
} >> "$INDENTED_BUDGET/fixtures/.devrites/work/backend-api/spec.md"
if ! python3 "$VALIDATOR" "$INDENTED_BUDGET/fixtures" \
  >"$OUT/devrites-workspace-schema-indented-budget.txt" 2>&1; then
  echo "FAIL: indented budget override with a reason was rejected"
  cat "$OUT/devrites-workspace-schema-indented-budget.txt"
  exit 1
fi

for kind in nbsp ideographic cr-separator cr-eol fs; do
  case "$kind" in
    nbsp) override=$'Budget override:\xc2\xa0reviewed reason' ;;
    ideographic) override=$'Budget override:\xe3\x80\x80reviewed reason' ;;
    cr-separator) override=$'Budget override:\rreviewed reason' ;;
    cr-eol) override=$'Budget override: reviewed reason\r' ;;
    fs) override=$'Budget override: \x1c' ;;
  esac
  UNICODE_BUDGET="$(mktemp -d "$OUT/case.XXXXXX")"
  cp -R "$FIXTURES" "$UNICODE_BUDGET/fixtures"
  {
    printf '\n%s\n' "$override"
    for _ in $(seq 1 300); do
      echo
    done
  } >> "$UNICODE_BUDGET/fixtures/.devrites/work/backend-api/spec.md"
  if ! python3 "$VALIDATOR" "$UNICODE_BUDGET/fixtures" \
    >"$OUT/devrites-workspace-schema-unicode-budget-$kind.txt" 2>&1; then
    echo "FAIL: $kind budget override with a reason was rejected"
    cat "$OUT/devrites-workspace-schema-unicode-budget-$kind.txt"
    exit 1
  fi
done

BAD_MERMAID="$(mktemp -d "$OUT/case.XXXXXX")"
cp -R "$FIXTURES" "$BAD_MERMAID/fixtures"
perl -0pi -e 's/^sequenceDiagram$/unsupportedDiagram/m' \
  "$BAD_MERMAID/fixtures/.devrites/work/backend-api/architecture.md"
if python3 "$VALIDATOR" "$BAD_MERMAID/fixtures" \
  >"$OUT/devrites-workspace-schema-bad-mermaid.txt" 2>&1; then
  echo "FAIL: invalid raw Mermaid input passed validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-bad-mermaid.txt" 'starts with unsupported syntax'

LEDGER_ONLY="$(mktemp -d "$OUT/case.XXXXXX")"
mkdir -p "$LEDGER_ONLY/.devrites/work/ledger-only"
cat > "$LEDGER_ONLY/.devrites/work/ledger-only/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | frame |
MD
python3 "$VALIDATOR" "$LEDGER_ONLY" >"$OUT/devrites-workspace-schema-ledger-only.txt"

UNKNOWN_PHASE="$(mktemp -d "$OUT/case.XXXXXX")"
mkdir -p "$UNKNOWN_PHASE/.devrites/work/unknown"
cat > "$UNKNOWN_PHASE/.devrites/work/unknown/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | invented |
MD
if python3 "$VALIDATOR" "$UNKNOWN_PHASE" >"$OUT/devrites-workspace-schema-unknown.txt" 2>&1; then
  echo "FAIL: unknown phase passed workspace validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-unknown.txt" "unknown phase 'invented'"

REMNANTS="$(mktemp -d "$OUT/case.XXXXXX")"
for name in "native-engine-cleanup" "native-engine-cleanup-s1" "native-engine-cleanup-s10" "native-engine-cleanup-s11" "native-engine-cleanup-s12" "native-engine-cleanup-s13" "native-engine-cleanup-s14" "native-engine-cleanup-s15" "native-engine-cleanup-s16" "native-engine-cleanup-s16b" "native-engine-cleanup-s17" "native-engine-cleanup-s18" "native-engine-cleanup-s19" "native-engine-cleanup-s2" "native-engine-cleanup-s20" "native-engine-cleanup-s21" "native-engine-cleanup-s22" "native-engine-cleanup-s23" "native-engine-cleanup-s24" "native-engine-cleanup-s3" "native-engine-cleanup-s3b" "native-engine-cleanup-s4" "native-engine-cleanup-s5a" "native-engine-cleanup-s5b" "native-engine-cleanup-s6a" "native-engine-cleanup-s6b" "native-engine-cleanup-s7" "native-engine-cleanup-s8" "native-engine-cleanup-s9"; do
  mkdir -p "$REMNANTS/.devrites/work/$name"
done
printf 'bounded paths\n' > "$REMNANTS/.devrites/work/native-engine-cleanup/.wright-allowlist"
printf '{}\n' > "$REMNANTS/.devrites/work/native-engine-cleanup-s20/recovery-attempts.jsonl"
if python3 "$VALIDATOR" "$REMNANTS" >"$OUT/devrites-workspace-schema-remnants.txt" 2>&1; then
  echo "FAIL: operational remnants were treated as workspaces"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-remnants.txt" "workspace-schema: no workspaces found"

SYMLINKS="$(mktemp -d "$OUT/case.XXXXXX")"
mkdir -p "$SYMLINKS/.devrites/work/file-link" "$SYMLINKS/target"
printf '| phase | invented |\n' > "$SYMLINKS/target/state.md"
ln -s "$SYMLINKS/target" "$SYMLINKS/.devrites/work/directory-link"
ln -s "$SYMLINKS/target/state.md" "$SYMLINKS/.devrites/work/file-link/state.md"
if python3 "$VALIDATOR" "$SYMLINKS" >"$OUT/devrites-workspace-schema-symlinks.txt" 2>&1; then
  echo "FAIL: symlinked workspace authority passed validation"
  exit 1
fi
expect_msg "$OUT/devrites-workspace-schema-symlinks.txt" "workspace-schema: no workspaces found"

echo "ok: workspace schema suite pins failing cases for: blocksOpenQuestions phase flags (workflow_schema.py), pending slice in a proof phase, legacy acceptance ID, duplicate canonical ID, missing phase artifact, README phase without state.md authority, missing slice field, unmapped browser evidence, empty verdict, marker-only readiness artifact, empty artifact, corrupt cursor Markdown (workflow_schema.py), stale local link, open blocking question, unreferenced acceptance criterion, NUL or invalid UTF-8 Markdown, line-count budget, unsupported Mermaid, unknown phase, operational remnants as workspaces, and symlinked workspace authority; other rules are not pinned by this suite"
