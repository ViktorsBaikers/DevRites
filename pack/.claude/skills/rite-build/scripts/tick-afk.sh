#!/usr/bin/env bash
# Decrement the AFK slice budget in state.md — called by /rite-build's record
# step (workflow step 6) after a slice is built under AFK. The mutable remaining count lives in
# state.md (`AFK slices remaining: <n>`); `.devrites/AFK`'s `max_slices` is the
# read-only initial budget and is never rewritten in place.
#
# Usage: tick-afk.sh <state.md path>
#
# Reads "AFK slices remaining: <n>", decrements it, writes it back in place,
# and prints the new value.
#
# Exit codes:
#   0  decremented, budget remains (new value > 0); or no AFK budget set (no-op)
#   3  budget exhausted (new value reached 0) — caller must force a HITL stop

set -euo pipefail

sfile="${1:-}"
if [ -z "$sfile" ] || [ ! -f "$sfile" ]; then
  printf 'tick-afk: state.md not found at %s\n' "${sfile:-<unset>}" >&2
  exit 5
fi

# Defensive: a state.md without the field means no AFK budget is in effect.
# Don't invent one; let the build proceed (HITL or unbudgeted AFK).
# state.md renders fields as markdown bullets ("- AFK slices remaining: <n>").
# Tolerate an optional leading "- " and a trailing " | none" / comment.
cur="$(awk 'match($0, /^[[:space:]]*-?[[:space:]]*AFK slices remaining:[[:space:]]*/) {
              rest = substr($0, RLENGTH + 1); sub(/[[:space:]].*$/, "", rest); print rest; exit }' "$sfile")"
if [ -z "$cur" ] || [ "$cur" = "none" ]; then
  printf 'tick-afk: no "AFK slices remaining" budget set in %s — no-op\n' "$sfile"
  exit 0
fi
if ! printf '%s' "$cur" | grep -qE '^[0-9]+$'; then
  printf 'tick-afk: "AFK slices remaining" is not a number (%s) in %s\n' "$cur" "$sfile" >&2
  exit 5
fi

new=$(( cur - 1 ))
[ "$new" -lt 0 ] && new=0

# Rewrite the field in place. Awk rewrites the file atomically via a temp file.
awk -v n="$new" '
  /^[[:space:]]*-?[[:space:]]*AFK slices remaining:/ { print "- AFK slices remaining: " n; next }
  { print }
' "$sfile" > "$sfile.tmp" && mv "$sfile.tmp" "$sfile"

printf '%s\n' "$new"

# Budget exhausted — force the caller to stop and revert to HITL.
[ "$new" -le 0 ] && exit 3
exit 0
