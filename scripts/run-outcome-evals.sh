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
fail=0

echo "== grade golden/shippable-feature (expect GO) =="
if bash "$GRADER" "$good"; then
  echo "  PASS — graded GO"
else
  echo "  FAIL — a known-shippable workspace should grade GO"; fail=1
fi

echo
echo "== grade golden/blocked-feature (expect NO-GO) =="
if bash "$GRADER" "$bad"; then
  echo "  FAIL — a known-blocked workspace should NOT grade GO"; fail=1
else
  echo "  PASS — correctly graded NO-GO"
fi

echo
if [ "$fail" -eq 0 ]; then
  echo "Outcome evals passed."
else
  echo "Outcome evals FAILED."
fi
exit "$fail"
