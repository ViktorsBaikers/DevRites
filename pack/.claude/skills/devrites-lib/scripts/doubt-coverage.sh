#!/usr/bin/env bash
# doubt-coverage.sh — did the build doubt the decisions it stood?
# Deterministic, zero-token. Reads the append-only footprint log the orchestrator writes at
# /rite-build step 4 (one `doubt` line per devrites-doubt dispatch) and reports doubt coverage
# against wright dispatches. It CANNOT prove a 1:1 verdict-per-decision match — `Decisions stood`
# is not a structured ledger — so it proves the cheap, unambiguous half: a feature that ran the
# wright but never once doubted a decision. The seal's prose gate (step 4a) + the reviewer fan-out
# judge severity against decisions.md.
#
#   doubt-coverage.sh <slug>
#
# Exit: 0 = doubt dispatches present, or nothing to assess (no log / no wright) → pass.
#       1 = wright dispatch(es) logged but ZERO doubt dispatches → coverage gap. The seal treats
#           it as a finding (Important; NO-GO when decisions.md records an irreversible-risk
#           decision — auth / public-API / migration — with no verdict).
#       2 = usage.
set -u

slug="${1:-}"
[ -n "$slug" ] || { echo "usage: doubt-coverage.sh <slug>" >&2; exit 2; }

log=".devrites/work/$slug/footprint.log"
if [ ! -f "$log" ]; then
  echo "doubt-coverage: no footprint log — nothing to assess (pass)"
  exit 0
fi

w=$(awk '$2=="wright"{n++} END{print n+0}' "$log")
d=$(awk '$2=="doubt"{n++}  END{print n+0}' "$log")

echo "doubt-coverage: $w wright · $d doubt"

if [ "$w" -gt 0 ] && [ "$d" -eq 0 ]; then
  echo "doubt-coverage: GAP — $w wright dispatch(es), 0 doubt dispatches. Either every slice's"
  echo "'Decisions stood' was genuinely empty, or step-4 doubt was skipped. Confirm against"
  echo "decisions.md before GO."
  exit 1
fi
exit 0
