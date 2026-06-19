#!/usr/bin/env bash
# DevRites progress footer — render the active feature's position deterministically.
# Run LAST (output step) by every workspace-operating rite-* skill via the standard
# resolution snippet, the mirror of preamble.sh (which runs first). Read-only; never
# mutates workspace files.
#
# Prints three lines of "chrome" the skill's own fact lines sit under:
#   ── rite-<phase> ───────────────────────────
#   Slice <built>/<total>  ██████░░░░  <last-built name> ✓     (or "✅ ALL BUILT" at N/N)
#   Flow   spec ✓ define ✓ build ◉ prove ○ … ship ○
#
# The slice meter answers "how many steps left" (built/total); the ✅ ALL BUILT marker
# + the `build ✓` ribbon glyph answer "is the build phase complete". The skill prints the
# what-was-done / next-step / hygiene lines beneath; this script owns only the visuals so
# every rite-* footer looks the same with zero model drift.
#
# Usage: progress.sh [slug]
#   slug — optional override; defaults to .devrites/ACTIVE
#
# Paths resolve relative to CWD (the project root), like preamble.sh. Prints nothing and
# exits 0 when there is no active workspace, so pre-spec callers stay silent.

set -u
slug="${1:-$(cat .devrites/ACTIVE 2>/dev/null)}"
d=".devrites/work/$slug"
s="$d/state.md"

[ -n "$slug" ] && [ -f "$s" ] || exit 0

# --- phase ----------------------------------------------------------------------------
# First whitespace token after "Phase:" (ignores any trailing "  # comment").
phase="$(awk -F': *' '/^- Phase:/{print $2; exit}' "$s" | awk '{print $1}')"
[ -n "$phase" ] || phase="spec"

# --- slice tally ----------------------------------------------------------------------
# total / built counts + the name of the last built slice, parsed from "## Slice progress".
read -r total built lastbuilt <<EOF
$(awk '
  /^## Slice progress/ { inp=1; next }
  inp && /^## /         { inp=0 }
  inp && /^- \[/ {
    total++
    name=$0
    sub(/^- \[[^]]*\][[:space:]]*/, "", name)              # drop "- [x] "
    sub(/^Slice[[:space:]]*[0-9]+:[[:space:]]*/, "", name) # drop "Slice N: "
    sub(/[[:space:]]+(built|pending)[[:space:]]*$/, "", name)        # drop trailing "built|pending"
    sub(/[[:space:]]+[^[:alnum:][:space:]]+[[:space:]]*$/, "", name) # drop the " — " separator
    if ($0 ~ /\[[xX]\]/) { built++; last=name }
  }
  END { printf "%d %d %s", total+0, built+0, last }
' "$s")
EOF
total="${total:-0}"; built="${built:-0}"

# --- header rule ----------------------------------------------------------------------
title="rite-${phase}"
header="── ${title} "
while [ "${#header}" -lt 44 ]; do header="${header}─"; done
printf '%s\n' "$header"

# --- slice meter ----------------------------------------------------------------------
# Bar is 10 cells; filled = round(built/total*10). Skipped entirely before any slices exist.
if [ "$total" -gt 0 ]; then
  filled=$(( (built * 10 + total / 2) / total ))
  [ "$filled" -gt 10 ] && filled=10
  bar=""; i=0
  while [ "$i" -lt "$filled" ]; do bar="${bar}█"; i=$((i+1)); done
  while [ "$i" -lt 10 ];       do bar="${bar}░"; i=$((i+1)); done
  if [ "$built" -ge "$total" ]; then
    printf 'Slices %d/%d  %s  ✅ ALL BUILT\n' "$built" "$total" "$bar"
  else
    tail=""
    [ -n "$lastbuilt" ] && tail="  ${lastbuilt} ✓"
    printf 'Slice %d/%d  %s%s\n' "$built" "$total" "$bar" "$tail"
  fi
fi

# --- flow ribbon ----------------------------------------------------------------------
# Lifecycle spine; optional temper/vet appear only once their artifact exists.
order=(spec)
[ -f "$d/strategy.md" ]   && order+=(temper)
order+=(define)
[ -f "$d/eng-review.md" ] && order+=(vet)
order+=(build prove polish review seal ship)

# Current index in the spine. `plan` is replan → sits at build; `done` is past ship.
cur="$phase"; [ "$cur" = "plan" ] && cur="build"
idx=-1
if [ "$phase" = "done" ]; then
  idx=${#order[@]}
else
  for n in "${!order[@]}"; do [ "${order[$n]}" = "$cur" ] && { idx=$n; break; }; done
fi

ribbon="Flow  "
for n in "${!order[@]}"; do
  if   [ "$n" -lt "$idx" ]; then g="✓"
  elif [ "$n" -eq "$idx" ]; then g="◉"
  else                           g="○"
  fi
  ribbon="${ribbon} ${order[$n]} ${g}"
done
printf '%s\n' "$ribbon"
