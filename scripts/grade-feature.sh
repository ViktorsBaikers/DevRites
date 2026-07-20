#!/usr/bin/env bash
# Deterministic OUTCOME grader for a finished DevRites feature workspace.
#
# The trigger evals (evals/*.json) test whether the right SKILL *fires*. This
# grades whether a finished run actually reached a *shippable* state — the
# product claim DevRites sells ("won't claim done without proof"). It reads only
# committed Markdown artifacts (no API key, no Claude Code harness), so it runs
# in CI against a golden fixture and is the kind of deterministic grader
# Anthropic's eval guidance recommends for coding agents.
#
# Live evidence-freshness (mtime: does proof post-date code?) is NOT graded here
# — git fixtures don't preserve mtimes; that gate is enforced on a live
# workspace by the `devrites-engine evidence-fresh` engine gate.
#
# Usage: grade-feature.sh <workspace-dir>
#   e.g. evals/golden/shippable-feature | .devrites/work/<slug> | .devrites/archive/<slug>
#
# Invariants (from rite-seal/reference/{seal-template,go-no-go,final-evidence}.md):
#   1. seal.md present with "Verdict: GO" (not NO-GO)
#   2. seal.md "## Acceptance Criteria" has no unchecked "- [ ]" item
#   3. seal.md "## Blockers" is empty / "none"
#   4. evidence.md present and non-empty
#   5. review.md present
#   6. questions.md has no entry with gate: validating + status: open (NO-GO by definition)
#   7. state.md Phase in {seal, ship, done}; Status not awaiting_human / blocked
#
# Exit codes: 0 shippable; 1 one or more invariants failed; 2 bad usage.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ws="${1:-}"
if [ -z "$ws" ] || [ ! -d "$ws" ]; then
  printf 'usage: grade-feature.sh <workspace-dir>\n' >&2
  exit 2
fi

seal="$ws/seal.md"; ev="$ws/evidence.md"; rev="$ws/review.md"
q="$ws/questions.md"; st="$ws/state.md"
problems=()

# 1 / 2 / 3 — seal verdict, acceptance, blockers
if [ ! -f "$seal" ]; then
  problems+=("seal.md missing — feature never sealed")
else
  grep -qiE '^[[:space:]]*Verdict:[[:space:]]*GO[[:space:]]*$' "$seal" \
    || problems+=("seal.md Verdict is not GO")
  unchecked=$(awk '/^## /{insec=($0 ~ /^## Acceptance Criteria/)} insec && /^- \[ \]/{c++} END{print c+0}' "$seal")
  [ "${unchecked:-0}" -gt 0 ] && problems+=("seal.md has ${unchecked} unchecked acceptance criterion(s)")
  if awk '
      /^## /{ insec=($0 ~ /^## Blockers/); next }
      insec { l=$0; gsub(/^[[:space:]]+|[[:space:]]+$/,"",l)
              if (l=="") next
              ll=tolower(l); if (ll ~ /^-?[[:space:]]*(none|n\/a)$/) next
              nz=1 }
      END { exit(nz?0:1) }' "$seal"; then
    problems+=("seal.md lists unresolved blockers")
  fi
fi

# 4 — evidence
{ [ -f "$ev" ] && [ -s "$ev" ]; } || problems+=("evidence.md missing or empty — acceptance unproven")

# 5 — review
[ -f "$rev" ] || problems+=("review.md missing — feature not reviewed")

# 6 — open validating gate (merge-blocking by definition)
if [ -f "$q" ]; then
  if awk '
      /^## q-/{ if(inq && s=="open" && g=="validating") bad=1; inq=1; s=""; g=""; next }
      inq && /^status:/{ v=$0; sub(/^status:[[:space:]]*/,"",v); s=v }
      inq && /^gate:/  { v=$0; sub(/^gate:[[:space:]]*/,"",v);   g=v }
      END{ if(inq && s=="open" && g=="validating") bad=1; exit(bad?0:1) }' "$q"; then
    problems+=("open 'gate: validating' question — merge-blocking by definition (NO-GO)")
  fi
fi

# 7 — state phase / status
if [ -f "$st" ]; then
  ph="$(python3 "$ROOT/scripts/workflow_schema.py" field "$st" phase 2>/dev/null || true)"
  stt="$(python3 "$ROOT/scripts/workflow_schema.py" field "$st" status 2>/dev/null || true)"
  if ! python3 "$ROOT/scripts/workflow_schema.py" phase-property "$ph" shippable; then
    problems+=("state.md Phase='${ph}' (expected a shippable phase)")
  fi
  case "$stt" in awaiting_human|blocked) problems+=("state.md Status='${stt}' (not shippable)") ;; esac
else
  problems+=("state.md missing")
fi

# 8 — executable acceptance criteria: every spec [ACn] proven + checked in seal
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

if ! acout=$(run_acceptance_check 2>&1); then
  problems+=("acceptance: $(printf '%s' "$acout" | tail -1 | sed 's/^check-acceptance: //')")
fi

slug="$(basename "$ws")"
if [ "${#problems[@]}" -eq 0 ]; then
  printf 'GO    %s — shippable: sealed GO, acceptance proven, no blockers, no open validating gate.\n' "$slug"
  exit 0
fi
printf 'NO-GO %s — %d blocker(s):\n' "$slug" "${#problems[@]}"
for p in "${problems[@]}"; do printf '  - %s\n' "$p"; done
exit 1
