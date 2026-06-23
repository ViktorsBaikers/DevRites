#!/usr/bin/env bash
# DevRites PostToolUse hook — fail-on-red sentinel. After a Bash command that runs the
# project's tests / build / typecheck / lint, set or clear .devrites/work/<slug>/.red so the
# Stop gate (devrites-stop-gate.sh) can refuse to end a turn while the suite is red. Turns
# "evidence over confidence / fail-on-red" from prose into a tracked flag. Heuristic on the
# command + its output; FAIL-OPEN and silent on any doubt.
set -u

input="$(cat)"
case "$input" in *'"tool_name"'*) : ;; *) exit 0 ;; esac
root="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -f "$root/.devrites/ACTIVE" ] || exit 0
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null)"
[ -n "$slug" ] || exit 0
d="$root/.devrites/work/$slug"
[ -d "$d" ] || exit 0

command -v node >/dev/null 2>&1 || exit 0
parsed="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const j=JSON.parse(s);const ti=j.tool_input||{};let r=j.tool_response;if(r&&typeof r==="object")r=JSON.stringify(r);process.stdout.write((j.tool_name||"")+""+(ti.command||"")+""+(r||""))}catch(e){}})' 2>/dev/null)"
[ -n "$parsed" ] || exit 0
tool="${parsed%%$'\001'*}"; rest="${parsed#*$'\001'}"; cmd="${rest%%$'\001'*}"; out="${rest#*$'\001'}"
[ "$tool" = "Bash" ] || exit 0

# Only react to test/build/lint/typecheck commands.
printf '%s' "$cmd" | grep -qiE '(npm|pnpm|yarn|bun)([[:space:]]+run)?[[:space:]]+(test|build|lint|typecheck|check)|jest|vitest|pytest|go[[:space:]]+test|cargo[[:space:]]+(test|build|clippy)|\bmvn\b|gradle|eslint|ruff|mypy|\btsc\b|\bmake[[:space:]]+(test|build|check)' || exit 0

red="$d/.red"
combo="$cmd
$out"
if printf '%s' "$combo" | grep -qiE '(^|[^A-Za-z])FAIL([^A-Za-z]|$)|[1-9][0-9]*[[:space:]]+(failed|failing|failures?|errors?)|not ok|panic:|error TS[0-9]|Traceback|AssertionError|✗|✘|BUILD FAILED|exit code [1-9]'; then
  printf '%s\n' "$cmd" > "$red" 2>/dev/null || true
  safe="$(printf '%s' "$cmd" | head -c 80 | tr -d '"\\')"
  printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"DevRites: tests/build are RED (%s). Fix to green or record the failure + next step in state.md before stopping — the Stop gate blocks an end-of-turn while .red is set."}}\n' "$safe" 2>/dev/null || true
  exit 0
fi
if printf '%s' "$combo" | grep -qiE 'PASS(ED)?|0[[:space:]]+(failed|failing|errors?)|all tests passed|BUILD SUCC(ESS|EEDED)|succeeded|no (errors|problems|issues)|(^|[^a-z])ok([^a-z]|$)|✓'; then
  rm -f "$red" 2>/dev/null || true
fi
exit 0
