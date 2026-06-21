#!/usr/bin/env bash
# DevRites Stop hook — state-integrity gate. Refuse to end the turn while the active workspace
# is provably inconsistent: tests are red (.red, set by devrites-redwatch.sh), or an open
# blocking/validating gate is not surfaced as Awaiting human. Turns "persistence before
# stopping" + "fail-on-red" from prose into a real gate. FAIL-OPEN, OBSERVE by default (logs to
# .stop-gate.log); enable with DEVRITES_STOP_GATE=enforce. Loop-guarded by stop_hook_active so
# it can never wedge the session.
set -u

input="$(cat)"
root="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -f "$root/.devrites/ACTIVE" ] || exit 0
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null)"
[ -n "$slug" ] || exit 0
d="$root/.devrites/work/$slug"
[ -d "$d" ] || exit 0

# Loop guard: if we already blocked once this stop cycle, let it stop.
case "$input" in *'"stop_hook_active":true'*) exit 0 ;; esac

reason=""
# 1) Red tests/build.
if [ -f "$d/.red" ]; then
  rc="$(head -c 100 "$d/.red" 2>/dev/null | tr -d '"\\')"
  reason="tests/build are RED ($rc) — fix to green, or record the failure + the next step in state.md, before stopping"
fi
# 2) Open blocking/validating gate not surfaced as awaiting_human.
if [ -z "$reason" ] && [ -f "$d/questions.md" ]; then
  if grep -qiE '^[[:space:]]*gate:[[:space:]]*(blocking|validating)' "$d/questions.md" 2>/dev/null \
     && grep -qiE '^[[:space:]]*status:[[:space:]]*open' "$d/questions.md" 2>/dev/null; then
    if ! grep -qiE 'Status:[[:space:]]*awaiting_human|## Awaiting human' "$d/state.md" 2>/dev/null; then
      reason="an open blocking/validating question is not surfaced — write the Awaiting human block to state.md (and set Status: awaiting_human) before stopping"
    fi
  fi
fi

[ -n "$reason" ] || exit 0

mode="${DEVRITES_STOP_GATE:-observe}"
if [ "$mode" = "enforce" ]; then
  printf '{"decision":"block","reason":"DevRites stop-gate: %s. (devrites-stop-gate)"}\n' "$reason"
  exit 0
fi
printf '%s\tWOULD-BLOCK\t%s\n' "$(date '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || echo '?')" "$reason" >> "$d/.stop-gate.log" 2>/dev/null || true
exit 0
