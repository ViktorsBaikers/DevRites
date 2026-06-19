#!/usr/bin/env bash
# DevRites SessionStart hook — inject the active feature's orientation digest at session start
# so the model knows a DevRites workflow is in flight, deterministically and prompt-free
# (the step-0 preamble in each skill is advisory; this makes orientation happen every time).
#
# Silent when there's no active workspace (don't nag on non-DevRites projects) and on any
# failure — this is an enhancement, never a dependency. Read-only.
set -u

# Resolve preamble.sh across install layouts (mirror the skill resolver).
P=".claude/skills/devrites-lib/scripts/preamble.sh"
[ -f "$P" ] || P="${CLAUDE_PLUGIN_ROOT:-}/pack/.claude/skills/devrites-lib/scripts/preamble.sh"
[ -f "$P" ] || P="pack/.claude/skills/devrites-lib/scripts/preamble.sh"
[ -f "$P" ] || exit 0

# No active feature -> stay silent.
slug="$(cat .devrites/ACTIVE 2>/dev/null | tr -d '[:space:]')"
[ -n "$slug" ] && [ -d ".devrites/work/$slug" ] || exit 0

digest="$(bash "$P" 2>/dev/null)"
[ -n "$digest" ] || exit 0

# Emit as additionalContext, JSON-encoded via node (skip if node is absent — each skill's
# own step-0 preamble still orients).
command -v node >/dev/null 2>&1 || exit 0
printf '%s' "$digest" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{const ctx="DevRites workflow active — live workspace orientation (from preamble.sh):\n\n"+s+"\n\nThis is the current state; a /rite-* command will re-run its own step-0 preamble.";process.stdout.write(JSON.stringify({hookSpecificOutput:{hookEventName:"SessionStart",additionalContext:ctx}}));});' 2>/dev/null || exit 0
exit 0
