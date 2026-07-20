#!/usr/bin/env bash
# Outcome-eval harness — proves the deterministic feature grader BOTH passes a
# known-shippable workspace AND fails a known-blocked one (see-it-fail-first:
# a grader that never returns NO-GO proves nothing). Runs in CI; no API key.
#
# Usage: run-outcome-evals.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GRADER="$ROOT/scripts/grade-feature.sh"
good="$ROOT/evals/golden/shippable-feature"
bad="$ROOT/evals/golden/blocked-feature"
nearmiss="$ROOT/evals/golden/near-miss-unproven-ac"
fail=0
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "== grade golden/shippable-feature (expect GO) =="
if bash "$GRADER" "$good"; then
  echo "  PASS — graded GO"
else
  echo "  FAIL — a known-shippable workspace should grade GO"; fail=1
fi

echo
echo "== grade synthetic/canonical-cursor (expect GO) =="
canonical="$tmp/canonical-cursor"
mkdir -p "$canonical"
cp -R "$good/." "$canonical/"
cat > "$canonical/state.md" <<'MD'
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | done |
| status | done |
| next_action | archived |
MD
if bash "$GRADER" "$canonical"; then
  echo "  PASS — canonical cursor graded GO"
else
  echo "  FAIL — canonical cursor should grade GO"; fail=1
fi

echo
echo "== grade golden/blocked-feature (expect NO-GO) =="
if bash "$GRADER" "$bad"; then
  echo "  FAIL — a known-blocked workspace should NOT grade GO"; fail=1
else
  echo "  PASS — correctly graded NO-GO"
fi

echo
echo "== grade golden/near-miss-unproven-ac (expect NO-GO on ONE invariant) =="
# Isolates invariant 2 (every acceptance criterion proven): identical to the
# shippable fixture except a single AC is left unchecked. Proves that gate fails
# independently — not only when six invariants trip at once.
if bash "$GRADER" "$nearmiss"; then
  echo "  FAIL — an unproven acceptance criterion must grade NO-GO"; fail=1
else
  echo "  PASS — correctly graded NO-GO on the lone unchecked AC"
fi

echo
echo "== grade synthetic/checked-wrong-ac (expect NO-GO on id coverage) =="
wrongac="$tmp/checked-wrong-ac"
mkdir -p "$wrongac"
cp -R "$good/." "$wrongac/"
cat > "$wrongac/seal.md" <<'MD'
# Seal: checked-wrong-ac

Verdict: GO

## Acceptance Criteria
- [x] [AC999] An unrelated item is checked.

## Verification Evidence
Evidence exists.

## Blockers
none
MD
if bash "$GRADER" "$wrongac"; then
  echo "  FAIL — checked unrelated acceptance items must not prove spec [ACn] criteria"; fail=1
else
  echo "  PASS — correctly graded NO-GO when spec [ACn] ids are missing from seal.md"
fi

echo
if [ "$fail" -eq 0 ]; then
  echo "Outcome evals passed."
else
  echo "Outcome evals FAILED."
fi
exit "$fail"
