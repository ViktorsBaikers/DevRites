#!/usr/bin/env bash
# Build-readiness gate for DevRites — turns /rite-build's step-0 prose checks
# ("if Status is awaiting_human → STOP", "if no Plan approved → STOP") into a
# deterministic exit code, the same way tick-afk.sh enforces the AFK cap. A gate
# the model *is* stopped by beats a gate the model is *asked* to honor.
# Read-only; never mutates the workspace.
#
# Usage: readiness.sh [slug]
#   slug — optional; defaults to .devrites/ACTIVE
#
# Checks .devrites/work/<slug>/state.md, in order:
#   - workspace + state.md exist
#   - Status != awaiting_human   (a HITL gate is open)
#   - Status != blocked
#   - "Plan approved:" is set     (/rite-define writes it when the human confirms)
#
# Exit codes:
#   0  ready to build
#   2  plan not approved        → /rite-define (then confirm the plan)
#   3  awaiting human           → /rite-resolve <qid> "<answer>"
#   4  blocked                  → /rite-plan repair | unblock
#   5  no workspace / state.md  → /rite-spec <feature>

set -euo pipefail

slug="${1:-$(cat .devrites/ACTIVE 2>/dev/null || true)}"
d=".devrites/work/$slug"
s="$d/state.md"
if [ -z "$slug" ] || [ ! -f "$s" ]; then
  printf 'readiness: no active workspace/state.md (slug=%s) — run /rite-spec <feature>\n' "${slug:-<unset>}" >&2
  exit 5
fi

# Read a state.md field. Tolerates the markdown-bullet rendering ("- Key: val")
# and a trailing " | none" / "# comment".
field() {
  awk -v k="$1" 'match($0, "^[[:space:]]*-?[[:space:]]*" k ":[[:space:]]*") {
    rest = substr($0, RLENGTH + 1)
    sub(/[[:space:]]*(#|\|).*$/, "", rest)
    sub(/[[:space:]]+$/, "", rest)
    print rest; exit
  }' "$s"
}

status="$(field Status)"
case "$status" in
  awaiting_human)
    printf 'readiness: STOP — Status: awaiting_human. Resume with /rite-resolve <qid> "<answer>".\n' >&2
    exit 3 ;;
  blocked)
    printf 'readiness: STOP — Status: blocked. Repair with /rite-plan repair (or unblock).\n' >&2
    exit 4 ;;
esac

approved="$(field 'Plan approved')"
if [ -z "$approved" ] || [ "$approved" = "none" ]; then
  printf 'readiness: STOP — plan not approved (state.md has no "Plan approved"). Run /rite-define.\n' >&2
  exit 2
fi

printf 'readiness: OK — plan approved %s, status %s. Ready to build.\n' "$approved" "${status:-running}"
exit 0
