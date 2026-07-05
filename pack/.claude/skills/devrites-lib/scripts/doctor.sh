#!/usr/bin/env bash
# doctor.sh — DevRites health diagnose. READ-ONLY. Never writes, never blocks.
#
# Checks install integrity + the active .devrites/ workspace for the inconsistencies that
# silently waste a session: a stale ACTIVE pointer, a corrupt workspace, an orphaned gate,
# or broken hook wiring across Claude Code and Codex. Two surfaces wrap this one core:
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

# 2. Broken Claude hook wiring — every devrites hook referenced in settings.json must exist on disk.
SET="$ROOT/.claude/settings.json"
if [ -f "$SET" ]; then
  for h in $(grep -oE 'devrites-[a-z-]+\.sh' "$SET" 2>/dev/null | sort -u); do
    [ -f "$ROOT/.claude/hooks/$h" ] || issue "settings.json wires missing hook: .claude/hooks/$h — reinstall or remove the hook entry"
  done
  ok "Claude hook wiring checked"
fi

# 3. Codex support is optional. Only diagnose the Codex surface when DevRites
# manages it, so unrelated project AGENTS.md/.codex files and --no-codex installs
# do not produce false failures.
MF="$ROOT/.claude/devrites.manifest"
DR_FLAGS=""
[ -f "$MF" ] && DR_FLAGS="$(sed -n 's/^# devrites-flags:[[:space:]]*//p' "$MF" 2>/dev/null | head -n1)"
has_flag() {
  case " $DR_FLAGS " in *" $1 "*) return 0 ;; *) return 1 ;; esac
}
manifest_has() {
  [ -f "$MF" ] && grep -Eq -- "$1" "$MF" 2>/dev/null
}

codex_managed=0
if manifest_has '^(\.agents/|\.codex/|AGENTS\.md$|\.claude/devrites\.(agents|codex-config)-merge$)'; then
  codex_managed=1
fi
has_flag --no-codex && codex_managed=0

if [ "$codex_managed" -eq 1 ]; then
  if ! has_flag --no-skills && manifest_has '^\.agents/skills/'; then
    [ -f "$ROOT/.agents/skills/rite/SKILL.md" ] \
      || issue "Codex skill mirror incomplete — missing .agents/skills/rite/SKILL.md — reinstall DevRites"
  fi
  if ! has_flag --no-rules && manifest_has '^\.agents/devrites/rules/'; then
    [ -f "$ROOT/.agents/devrites/rules/core.md" ] \
      || issue "Codex rules mirror incomplete — missing .agents/devrites/rules/core.md — reinstall DevRites"
  fi
  if ! has_flag --no-agents && manifest_has '^\.codex/agents/'; then
    [ -f "$ROOT/.codex/agents/devrites-code-reviewer.toml" ] \
      || issue "Codex custom agents incomplete — missing .codex/agents/devrites-code-reviewer.toml — reinstall DevRites"
  fi
  if manifest_has '^\.codex/hooks\.json$|^\.codex/hooks/|^\.claude/devrites\.codex-hooks-merge$'; then
    [ -f "$ROOT/.codex/hooks.json" ] \
      || issue "Codex hooks incomplete — missing .codex/hooks.json — reinstall DevRites or install with --no-codex"
    if [ -f "$ROOT/.codex/hooks.json" ]; then
      grep -q 'devrites-' "$ROOT/.codex/hooks.json" 2>/dev/null \
        || issue "Codex hooks incomplete — .codex/hooks.json does not reference DevRites hooks — reinstall DevRites"
      for h in $(grep -oE 'devrites-[a-z-]+\.sh' "$ROOT/.codex/hooks.json" 2>/dev/null | sort -u); do
        [ -f "$ROOT/.codex/hooks/$h" ] || issue "Codex hooks.json wires missing hook: .codex/hooks/$h — reinstall DevRites"
      done
    fi
  fi
  if manifest_has '^\.codex/mcp/|^\.codex/config\.toml$|^\.claude/devrites\.codex-config-merge$'; then
    [ -f "$ROOT/.codex/mcp/devrites-mcp.mjs" ] \
      || issue "Codex MCP incomplete — missing .codex/mcp/devrites-mcp.mjs — reinstall DevRites or install with --no-codex"
    if [ -f "$ROOT/.codex/config.toml" ]; then
      grep -q 'mcp_servers.devrites' "$ROOT/.codex/config.toml" 2>/dev/null \
        || issue "Codex MCP config missing DevRites server — merge the DevRites block into .codex/config.toml or reinstall DevRites"
    else
      issue "Codex config incomplete — missing .codex/config.toml with DevRites MCP server — reinstall DevRites or install with --no-codex"
    fi
  fi
  if manifest_has '^AGENTS\.md$|^\.claude/devrites\.agents-merge$'; then
    if [ -f "$ROOT/AGENTS.md" ]; then
      grep -q 'DevRites' "$ROOT/AGENTS.md" 2>/dev/null \
        || issue "AGENTS.md does not include the DevRites Codex bridge — reinstall with --force or merge the DevRites AGENTS.md guidance"
    else
      issue "Codex guidance incomplete — missing AGENTS.md bridge — reinstall DevRites"
    fi
  fi
  ok "Codex support wiring checked"
fi

# 4. ACTIVE pointer — if set, it must name a real workspace.
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
