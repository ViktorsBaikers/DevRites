#!/usr/bin/env bash
# analyze.sh — cross-artifact consistency + coverage gate for /rite-vet, run read-only
# over spec.md / tasks.md (+ coverage.md) BEFORE any code. One pass proving the artifacts
# are mutually consistent and the plan fully covers the spec, so a gap is caught when it
# is a one-line plan edit, not a reslice mid-build. Deterministic where it can be; the
# model walks the rest.
#
# Checks (each finding: "[severity] <locus> — issue"):
#   Coverage      — every [ACn] in spec.md maps to >=1 slice (tasks.md Satisfies:)   CRITICAL
#   Dangling-ref  — a slice Satisfies: an AC id that does not exist in spec.md         CRITICAL
#   Orphan-slice  — a slice satisfies no acceptance criterion                          warn
#
# Usage: analyze.sh [slug]    (slug defaults to .devrites/ACTIVE)
# Exit:  0 clean · 1 CRITICAL found (blocks /rite-build) · 2 no workspace
set -u

slug="${1:-}"
[ -n "$slug" ] || slug="$(cat .devrites/ACTIVE 2>/dev/null || true)"
[ -n "$slug" ] || { echo "analyze: no active workspace." >&2; exit 2; }
d=".devrites/work/$slug"
spec="$d/spec.md"; tasks="$d/tasks.md"
[ -f "$spec" ] && [ -f "$tasks" ] || { echo "analyze: need spec.md + tasks.md in $d." >&2; exit 2; }

spec_acs="$(grep -oE '\[AC[0-9]+\]' "$spec" 2>/dev/null | tr -d '[]' | sort -u)"
task_acs="$(grep -oE '\bAC[0-9]+\b' "$tasks" 2>/dev/null | sort -u)"

crit=0
echo "# Cross-artifact analysis: $slug"
echo

# Coverage: spec ACs with no slice reference.
echo "## Coverage"
if [ -z "$spec_acs" ]; then
  echo "- [warn] spec.md — no [ACn] acceptance ids found; tag criteria as '[AC1] ...' for a machine-checkable gate."
else
  while IFS= read -r ac; do
    [ -n "$ac" ] || continue
    if printf '%s\n' "$task_acs" | grep -qxF "$ac"; then
      echo "- [ok] $ac — covered"
    else
      echo "- [CRITICAL] $ac (spec.md) — no slice Satisfies it (uncovered)"; crit=$((crit+1))
    fi
  done <<EOF
$spec_acs
EOF
fi

# Dangling refs: tasks Satisfies an AC the spec doesn't define.
echo
echo "## Consistency"
if [ -n "$task_acs" ]; then
  while IFS= read -r ac; do
    [ -n "$ac" ] || continue
    if ! printf '%s\n' "$spec_acs" | grep -qxF "$ac"; then
      echo "- [CRITICAL] $ac (tasks.md) — referenced by a slice but not defined in spec.md (dangling)"; crit=$((crit+1))
    fi
  done <<EOF
$task_acs
EOF
fi

# Orphan slices: a slice with no Satisfies line.
orphans="$(awk '
  /^##[[:space:]]*Slice/ { if (name!="" && !sat) print name; name=$0; sub(/^##[[:space:]]*/,"",name); sat=0 }
  /^[[:space:]]*Satisfies:/ { sat=1 }
  END { if (name!="" && !sat) print name }
' "$tasks" 2>/dev/null)"
if [ -n "$orphans" ]; then
  echo "$orphans" | while IFS= read -r s; do [ -n "$s" ] && echo "- [warn] slice '$s' — satisfies no acceptance criterion (add Satisfies: or justify)"; done
fi

echo
if [ "$crit" -gt 0 ]; then
  echo "## Verdict: BLOCKED — $crit CRITICAL finding(s). Resolve before /rite-build."
  exit 1
fi
echo "## Verdict: clear — spec/tasks consistent and fully mapped."
exit 0
