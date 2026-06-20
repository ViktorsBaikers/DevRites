#!/usr/bin/env bash
# DevRites SessionStart hook — two read-only jobs, both silent unless they have something to say:
#   1. surface DevRites health issues (stale ACTIVE, corrupt workspace, broken hook wiring, …)
#      via doctor.sh — only when there ARE issues (silent-when-healthy);
#   2. inject the active feature's orientation digest so the model knows a workflow is in flight.
# Silent on non-DevRites projects and on any failure — this is an enhancement, never a
# dependency. Read-only; never blocks.
set -u

# Resolve shared scripts across install layouts (mirror the skill resolver).
P=".claude/skills/devrites-lib/scripts/preamble.sh"
[ -f "$P" ] || P="${CLAUDE_PLUGIN_ROOT:-}/pack/.claude/skills/devrites-lib/scripts/preamble.sh"
[ -f "$P" ] || P="pack/.claude/skills/devrites-lib/scripts/preamble.sh"
D=".claude/skills/devrites-lib/scripts/doctor.sh"
[ -f "$D" ] || D="${CLAUDE_PLUGIN_ROOT:-}/pack/.claude/skills/devrites-lib/scripts/doctor.sh"
[ -f "$D" ] || D="pack/.claude/skills/devrites-lib/scripts/doctor.sh"

# Not a DevRites project -> stay silent.
[ -d ".devrites" ] || exit 0

# 1. Health sweep — prints issue lines only (empty when healthy).
health=""
[ -f "$D" ] && health="$(bash "$D" 2>/dev/null)"

# 2. Orientation digest — only when there's an active workspace.
digest=""
if [ -f "$P" ]; then
  slug="$(cat .devrites/ACTIVE 2>/dev/null | tr -d '[:space:]')"
  if [ -n "$slug" ] && [ -d ".devrites/work/$slug" ]; then
    digest="$(bash "$P" 2>/dev/null)"
  fi
fi

# Nothing to say -> stay silent.
[ -z "$health" ] && [ -z "$digest" ] && exit 0

ctx=""
[ -n "$health" ] && ctx="⚠ DevRites health — issues found at session start (advisory; run /rite-doctor for the full report):"$'\n'"$health"$'\n\n'
[ -n "$digest" ] && ctx="${ctx}DevRites workflow active — live workspace orientation (from preamble.sh):"$'\n\n'"$digest"$'\n\n'"This is the current state; a /rite-* command will re-run its own step-0 preamble."

# Emit as additionalContext, JSON-encoded via node (skip if node is absent — each skill's
# own step-0 preamble still orients, and the health block degrades to nothing).
command -v node >/dev/null 2>&1 || exit 0
printf '%s' "$ctx" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{process.stdout.write(JSON.stringify({hookSpecificOutput:{hookEventName:"SessionStart",additionalContext:s}}));});' 2>/dev/null || exit 0
exit 0
