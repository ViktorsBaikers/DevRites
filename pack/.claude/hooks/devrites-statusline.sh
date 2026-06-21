#!/usr/bin/env bash
# DevRites statusline — one-line workspace HUD for settings.json "statusLine". Ships UNWIRED
# (opt-in) so it never clobbers a user's own statusline; point a "statusLine" command at it to
# enable. Reads the session JSON on stdin (ignored) + .devrites/ and prints:
#   DevRites: <slug> · <phase> · gates:<n> · AFK|HITL
set -u

cat >/dev/null 2>&1 || true
root="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -f "$root/.devrites/ACTIVE" ] || exit 0
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null)"
[ -n "$slug" ] || exit 0
d="$root/.devrites/work/$slug"; state="$d/state.md"
phase="$(grep -iE '^[-[:space:]]*Phase:' "$state" 2>/dev/null | head -1 | sed -E 's/^[-[:space:]]*Phase:[[:space:]]*//')"
gates="$(grep -ciE '^[[:space:]]*status:[[:space:]]*open' "$d/questions.md" 2>/dev/null || echo 0)"
mode="HITL"; [ -f "$root/.devrites/AFK" ] && mode="AFK"
red=""; [ -f "$d/.red" ] && red=" · RED"
printf 'DevRites: %s · %s · gates:%s · %s%s\n' "$slug" "${phase:-?}" "${gates:-0}" "$mode" "$red"
