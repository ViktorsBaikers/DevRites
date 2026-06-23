#!/usr/bin/env bash
# DevRites UserPromptSubmit hook — re-inject the active feature's cursor each turn so the
# load-bearing next-action sits at the high-attention end of context (fights lost-in-the-middle
# on a long phase; pairs with the SessionStart orientation). Tiny + read-only; silent on
# non-DevRites projects. Its stdout is added to the model's context for this turn.
set -u

root="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -f "$root/.devrites/ACTIVE" ] || exit 0
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null)"
[ -n "$slug" ] || exit 0
d="$root/.devrites/work/$slug"
[ -d "$d" ] || exit 0
state="$d/state.md"

next="$(grep -iE '^[-[:space:]]*Next step:' "$state" 2>/dev/null | head -1 | sed -E 's/^[-[:space:]]*Next step:[[:space:]]*//')"
status="$(grep -iE '^[-[:space:]]*Status:' "$state" 2>/dev/null | head -1 | sed -E 's/^[-[:space:]]*Status:[[:space:]]*//')"
gates="$(grep -ciE '^[[:space:]]*status:[[:space:]]*open' "$d/questions.md" 2>/dev/null || echo 0)"
afk=""
[ -f "$root/.devrites/AFK" ] && afk="$(grep -iE 'AFK slices remaining:' "$state" 2>/dev/null | head -1 | sed -E 's/.*remaining:[[:space:]]*//')"

echo "DevRites cursor — active feature: $slug"
[ -n "$status" ] && echo "  status: $status"
[ -n "$next" ] && echo "  next: $next"
echo "  open questions: ${gates:-0}"
[ -n "$afk" ] && echo "  AFK slices remaining: $afk"
[ -f "$d/.red" ] && echo "  ⚠ tests/build RED — resolve before stopping"
exit 0
