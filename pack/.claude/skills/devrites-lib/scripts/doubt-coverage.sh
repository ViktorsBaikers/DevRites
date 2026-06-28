#!/usr/bin/env bash
# doubt-coverage.sh — did the build doubt the decisions it stood?
# Deterministic, zero-token. Two checks, both sourced from records /rite-build writes:
#   1. decisions.md `## Decisions stood` — each stood decision ends `— doubt: <verdict>`; a
#      `doubt: MISSING` token means the decision was recorded but never doubted (the unambiguous
#      skip — caught even when only some slices skipped doubt).
#   2. footprint.log — `doubt` lines vs `wright` lines; zero doubt across any wright run is the
#      coarse "no doubt ran at all" signal the seal then verifies against decisions.md.
# It cannot prove a 1:1 verdict-per-decision match beyond what the build recorded — the seal's
# prose gate (step 4a) + the reviewer fan-out judge the rest.
#
#   doubt-coverage.sh <slug>
#
# Exit: 0 = doubt covered, or not assessable (no log / no wright / inline build) → pass.
#       1 = footprint shows wright dispatch(es) but ZERO doubt → no doubt ran at all; a prompt to
#           VERIFY against decisions.md (every-slice-trivial passes; a skipped decision is a finding).
#       2 = usage.
#       3 = decisions.md records a stood decision with `doubt: MISSING` → doubt was definitively
#           skipped for a decision on record. The seal treats it as a finding.
set -u

slug="${1:-}"
[ -n "$slug" ] || { echo "usage: doubt-coverage.sh <slug>" >&2; exit 2; }

dir=".devrites/work/$slug"
log="$dir/footprint.log"
dec="$dir/decisions.md"

# A recorded stood decision that was never doubted is the unambiguous failure — check it first,
# and on every path (it survives the inline-build case below, where the footprint heuristic can't).
if [ -f "$dec" ] && grep -q 'doubt: MISSING' "$dec"; then
  echo "doubt-coverage: SKIPPED — a '## Decisions stood' entry in decisions.md has doubt: MISSING."
  echo "A decision was stood and recorded but never doubted. Re-dispatch doubt or escalate."
  exit 3
fi

# Inline build (Task tool unavailable): the wright + doubt ran in-context, dispatching nothing,
# so the footprint heuristic does not apply — the decisions.md check above is the assessment.
if [ -f "$dir/.reconcile-inline" ]; then
  echo "doubt-coverage: inline build (.reconcile-inline) — footprint heuristic n/a; no MISSING"
  echo "verdict recorded. Verify by hand that each stood decision carries a devrites-doubt verdict."
  exit 0
fi

if [ ! -f "$log" ]; then
  echo "doubt-coverage: no footprint log — nothing to assess (pass)"
  exit 0
fi

w=$(awk '$2=="wright"{n++} END{print n+0}' "$log")
d=$(awk '$2=="doubt"{n++}  END{print n+0}' "$log")
echo "doubt-coverage: $w wright · $d doubt"

if [ "$w" -gt 0 ] && [ "$d" -eq 0 ]; then
  echo "doubt-coverage: no doubt dispatched across $w wright run(s). Either every slice's"
  echo "'Decisions stood' was genuinely empty, or doubt was skipped — verify against decisions.md."
  exit 1
fi
exit 0
