#!/usr/bin/env bash
# Deterministic outcome grader for a completed DevRites feature workspace.
#
# Trigger evals in evals/*.json check skill selection. This script checks
# whether a completed run is shippable and has proof. It reads only committed
# Markdown artifacts, so CI can grade a golden fixture without an API key or
# model harness.
#
# Git fixtures do not preserve the mtimes needed to compare proof with code.
# The `devrites-engine evidence-fresh` gate performs that check in a live
# workspace.
#
# Usage: grade-feature.sh [--json] <workspace-dir>
#   e.g. evals/golden/shippable-feature | .devrites/work/<slug> | .devrites/archive/<slug>
#
# Checks from rite-seal/reference/{seal-template,go-no-go,final-evidence}.md:
#   1. seal.md present with "Verdict: GO" (not NO-GO)
#   2. seal.md "## Acceptance Criteria" has no unchecked "- [ ]" item
#   3. seal.md "## Blockers" is empty / "none"
#   4. evidence.md present and non-empty
#   5. review.md present
#   6. questions.md has no open question (later lifecycle phases block them)
#   7. state.md Phase in {seal, ship, done}; Status not awaiting_human / blocked
#
# Exit codes: 0 shippable; 1 one or more invariants failed; 2 bad usage.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
json=0
if [ "${1:-}" = "--json" ]; then
  json=1
  shift
fi
ws="${1:-}"
if [ "$#" -ne 1 ] || [ ! -d "$ws" ]; then
  printf 'usage: grade-feature.sh [--json] <workspace-dir>\n' >&2
  exit 2
fi

seal="$ws/seal.md"; ev="$ws/evidence.md"; rev="$ws/review.md"
q="$ws/questions.md"; st="$ws/state.md"
problems=()
rules=()
unchecked=0
acceptance_ready=0

add_problem() {
  rules+=("$1")
  problems+=("$2")
}

# Checks 1 through 3: seal verdict, acceptance, and blockers.
if [ ! -f "$seal" ]; then
  add_problem "final.seal.missing" "seal.md missing: feature never sealed"
else
  grep -qiE '^[[:space:]]*Verdict:[[:space:]]*GO[[:space:]]*$' "$seal" \
    || add_problem "final.verdict.not-go" "seal.md Verdict is not GO"
  unchecked=$(awk '/^## /{insec=($0 ~ /^## Acceptance Criteria/)} insec && /^- \[ \]/{c++} END{print c+0}' "$seal")
  if [ "${unchecked:-0}" -gt 0 ]; then
    add_problem "final.acceptance.unchecked" "seal.md has ${unchecked} unchecked acceptance criterion(s)"
  else
    acceptance_ready=1
  fi
  if awk '
      /^## /{ insec=($0 ~ /^## Blockers/); next }
      insec { l=$0; gsub(/^[[:space:]]+|[[:space:]]+$/,"",l)
              if (l=="") next
              ll=tolower(l); if (ll ~ /^-?[[:space:]]*(none|n\/a)$/) next
              nz=1 }
      END { exit(nz?0:1) }' "$seal"; then
    add_problem "final.blockers.unresolved" "seal.md lists unresolved blockers"
  fi
fi

# Check 4: evidence.
{ [ -f "$ev" ] && [ -s "$ev" ]; } \
  || add_problem "final.evidence.missing-or-empty" "evidence.md missing or empty: acceptance unproven"

# Check 5: review.
[ -f "$rev" ] || add_problem "final.review.missing" "review.md missing: feature not reviewed"

# Check 6: open questions block sealing.
if [ -f "$q" ]; then
  if python3 - "$q" <<'PY'
from pathlib import Path
import sys

lines = Path(sys.argv[1]).read_text().splitlines()
in_question = False
status = ""
for line in lines:
    stripped = line.strip()
    if stripped.startswith("## "):
        if in_question and status.lower() == "open":
            raise SystemExit(0)
        in_question = stripped[3:].lower().startswith("q-")
        status = ""
    elif in_question and stripped.lower().startswith("status:"):
        status = stripped.split(":", 1)[1].strip()
if in_question and status.lower() == "open":
    raise SystemExit(0)

status_index = None
for line in lines:
    stripped = line.strip()
    if not (stripped.startswith("|") and stripped.endswith("|")):
        status_index = None
        continue
    cells = [cell.strip() for cell in stripped.strip("|").split("|")]
    lowered = [cell.lower() for cell in cells]
    if status_index is None:
        status_index = lowered.index("status") if "status" in lowered else None
        continue
    if status_index < len(cells) and cells[status_index].lower() == "open":
        raise SystemExit(0)
raise SystemExit(1)
PY
  then
    add_problem "final.questions.open" "questions.md contains an open question"
  fi
fi

# Check 7: state phase and status.
if [ -f "$st" ]; then
  ph="$(python3 "$ROOT/scripts/workflow_schema.py" field "$st" phase 2>/dev/null || true)"
  stt="$(python3 "$ROOT/scripts/workflow_schema.py" field "$st" status 2>/dev/null || true)"
  if ! python3 "$ROOT/scripts/workflow_schema.py" phase-property "$ph" shippable >/dev/null; then
    add_problem "final.state.phase" "state.md Phase='${ph}' (expected a shippable phase)"
  fi
  case "$stt" in
    awaiting_human|blocked)
      add_problem "final.state.status" "state.md Status='${stt}' (not shippable)"
      ;;
  esac
else
  add_problem "final.state.missing" "state.md missing"
fi

# Check 8: every spec [ACn] is proven and checked in the seal.
run_acceptance_check() {
  if [ -n "${DEVRITES_CLI:-}" ]; then
    "$DEVRITES_CLI" check-acceptance "$ws"
    return $?
  fi
  if command -v devrites-engine >/dev/null 2>&1; then
    devrites-engine check-acceptance "$ws"
    return $?
  fi
  if [ -x "$ROOT/engine/devrites" ]; then
    "$ROOT/engine/devrites" check-acceptance "$ws"
    return $?
  fi
  if [ -d "$ROOT/engine" ] && command -v go >/dev/null 2>&1; then
    ( cd "$ROOT/engine" && go run . check-acceptance "$ws" )
    return $?
  fi
  printf 'check-acceptance: devrites-engine unavailable (install devrites or run from a checkout with Go)\n' >&2
  return 127
}

if [ "$acceptance_ready" -eq 1 ] && ! acout=$(run_acceptance_check 2>&1); then
  add_problem "final.acceptance.ids" \
    "acceptance: $(printf '%s' "$acout" | tail -1 | sed 's/^check-acceptance: //')"
fi

slug="$(basename "$ws")"
if [ "$json" -eq 1 ]; then
  status="GO"
  [ "${#problems[@]}" -gt 0 ] && status="NO-GO"
  pairs=()
  for ((i = 0; i < ${#problems[@]}; i++)); do
    pairs+=("${rules[$i]}" "${problems[$i]}")
  done
  python3 - "$slug" "$status" "${pairs[@]}" <<'PY'
import json
import sys

slug, status, *pairs = sys.argv[1:]
items = [
    {"rule_id": pairs[i], "message": pairs[i + 1]}
    for i in range(0, len(pairs), 2)
]
print(json.dumps({
    "schema": "devrites-outcome-grade/v1",
    "workspace": slug,
    "status": status,
    "rule_ids": [item["rule_id"] for item in items],
    "problems": items,
}, separators=(",", ":")))
PY
  if [ "${#problems[@]}" -eq 0 ]; then
    exit 0
  fi
  exit 1
fi

if [ "${#problems[@]}" -eq 0 ]; then
  printf 'GO    %s: shippable: sealed GO, acceptance proven, no blockers, no open question.\n' "$slug"
  exit 0
fi
printf 'NO-GO %s: %d blocker(s):\n' "$slug" "${#problems[@]}"
for p in "${problems[@]}"; do printf '  - %s\n' "$p"; done
exit 1
