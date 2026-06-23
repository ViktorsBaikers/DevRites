#!/usr/bin/env bash
# doctor.sh — DevRites health diagnose. READ-ONLY. Never writes, never blocks.
#
# Checks install integrity + the active .devrites/ workspace for the inconsistencies that
# silently waste a session: a stale ACTIVE pointer, a corrupt workspace, an orphaned gate,
# or broken hook wiring. Two surfaces wrap this one core:
#   - the SessionStart orient hook calls it and surfaces issues only when there are any
#     (silent-when-healthy);
#   - the /rite-doctor skill calls it with --verbose for a full report on demand.
#
# Usage: doctor.sh [--root DIR] [--verbose]
# Output: one "issue: <what> — <fix>" line per problem (always); with --verbose, also a
#   "ok: <check>" line per passing check.
# Exit:   0 = healthy (no issues);  1 = one or more issues found.
set -u

ROOT="."
VERBOSE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --root) ROOT="$2"; shift 2 ;;
    --root=*) ROOT="${1#--root=}"; shift ;;
    --verbose|-v) VERBOSE=1; shift ;;
    *) shift ;;
  esac
done

issues=0
issue() { printf 'issue: %s\n' "$1"; issues=$((issues + 1)); }
ok()    { [ "$VERBOSE" -eq 1 ] && printf 'ok: %s\n' "$1"; return 0; }

DR="$ROOT/.devrites"

# Not a DevRites project at all → nothing to diagnose (stay silent, healthy).
if [ ! -d "$DR" ]; then
  ok "no .devrites/ — not a DevRites project"
  exit 0
fi

# Resolve the installed pack base (user project = .claude/; dev repo = pack/.claude/).
BASE="$ROOT/.claude"
[ -d "$BASE/skills/devrites-lib" ] || BASE="$ROOT/pack/.claude"

# 1. Install integrity — the core shared scripts must be present.
missing=""
for s in preamble.sh progress.sh readiness.sh conventions.py; do
  [ -f "$BASE/skills/devrites-lib/scripts/$s" ] || missing="$missing $s"
done
if [ -n "$missing" ]; then
  issue "install incomplete — missing devrites-lib script(s):$missing — reinstall DevRites"
else
  ok "devrites-lib core scripts present"
fi

# 2. Broken hook wiring — every devrites hook referenced in settings.json must exist on disk.
SET="$ROOT/.claude/settings.json"
if [ -f "$SET" ]; then
  for h in $(grep -oE 'devrites-[a-z-]+\.sh' "$SET" 2>/dev/null | sort -u); do
    [ -f "$ROOT/.claude/hooks/$h" ] || issue "settings.json wires missing hook: .claude/hooks/$h — reinstall or remove the hook entry"
  done
  ok "hook wiring checked"
fi

# 3. ACTIVE pointer — if set, it must name a real workspace.
slug="$(cat "$DR/ACTIVE" 2>/dev/null | tr -d '[:space:]')"
if [ -z "$slug" ]; then
  ok "no active feature (ACTIVE empty)"
else
  WS="$DR/work/$slug"
  if [ ! -d "$WS" ]; then
    issue "stale ACTIVE — points at '$slug' but .devrites/work/$slug/ is gone — run /rite-status or 'rite use <slug>'"
  else
    ok "active feature '$slug' workspace present"
    # 4. Corrupt workspace — an active workspace must have state.md.
    if [ ! -f "$WS/state.md" ]; then
      issue "corrupt workspace '$slug' — state.md missing — the phase cursor is lost"
    else
      ok "workspace '$slug' has state.md"
      # 5. Orphaned gate — an open validating/blocking question while the feature reads done/shipped.
      if [ -f "$WS/questions.md" ] && grep -qiE '^status:[[:space:]]*open' "$WS/questions.md" 2>/dev/null \
         && grep -qiE '^gate:[[:space:]]*(validating|blocking)' "$WS/questions.md" 2>/dev/null \
         && grep -qiE '(phase|status):[[:space:]]*(done|shipped)' "$WS/state.md" 2>/dev/null; then
        issue "orphaned gate in '$slug' — an open validating/blocking question remains while state reads done/shipped — resolve it via /rite-resolve"
      else
        ok "no orphaned gates in '$slug'"
      fi
    fi
  fi
fi

[ "$issues" -gt 0 ] && exit 1
exit 0
