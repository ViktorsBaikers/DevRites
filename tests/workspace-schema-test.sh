#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
VALIDATOR="$ROOT/scripts/validate-workspace-schema.py"
FIXTURES="$ROOT/tests/fixtures/workspace-schema"
CANONICAL_SCHEMA="$ROOT/pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md"

python3 "$VALIDATOR" "$FIXTURES" >/tmp/devrites-workspace-schema-ok.txt

CANONICAL_WORKSPACE="$(mktemp -d)"
cp -R "$FIXTURES" "$CANONICAL_WORKSPACE/fixtures"
{
  printf '# Tasks\n\n## Slice index\n\n'
  awk '/<!-- canonical-slice:start -->/{on=1; next} /<!-- canonical-slice:end -->/{on=0} on' "$CANONICAL_SCHEMA" \
    | sed '/^```/d'
} > "$CANONICAL_WORKSPACE/fixtures/.devrites/work/backend-api/tasks.md"
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

echo "ok: workspace schema validator accepts the canonical slice grammar and rejects legacy, underspecified, and stale-evidence workspaces"
