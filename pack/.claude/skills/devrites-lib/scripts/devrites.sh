#!/usr/bin/env bash
# devrites — tool-agnostic CLI over the .devrites/ workflow state.
#
# The DevRites discipline lives in the .devrites/ Markdown files + these state
# scripts, NOT in the Claude Code harness. This CLI is the portable surface over
# them: any agent (Cursor, Codex, Gemini CLI, CI) or a human can orient, gate,
# and advance a DevRites workflow by calling `devrites <command>` — no skill
# prose required. (The MCP wrapper in mcp/devrites-mcp.mjs exposes the same ops
# as MCP tools.)
#
# Run from the project root. Sibling scripts resolve by this file's location;
# the .devrites/ workspace resolves relative to CWD, like the rest of the pack.
#
# Commands & exit codes are listed by `devrites help`.

set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  cat <<'EOF'
devrites — drive the .devrites/ workflow from any tool (run at the project root)

  orient [slug]          orientation digest for the active feature (read-only)
  status [slug]          alias for orient
  progress [slug]        progress footer — slice meter + flow ribbon (read-only)
  ready [slug]           build-readiness gate    (0 ready · 2 no-plan · 3 awaiting · 4 blocked · 5 none)
  evidence-fresh [slug]  evidence-freshness gate (0 fresh · 3 stale)
  acceptance [slug|dir]  acceptance-criteria gate (0 all proven · 1 gap)
  tick-afk [slug]        decrement the AFK slice budget (3 = exhausted)
  resolve <args...>      answer / --drop / --batch a questions.md entry
  close [slug]           archive the workspace + clear ACTIVE
  active                 print the active slug
  list                   list workspace slugs under .devrites/work/
  use <slug>             re-point .devrites/ACTIVE to <slug>
  help                   this text
EOF
}

active_slug() { { cat .devrites/ACTIVE 2>/dev/null || true; } | tr -d '[:space:]'; }
resolve_slug() { local s="${1:-}"; [ -n "$s" ] || s="$(active_slug)"; printf '%s' "$s"; }

cmd="${1:-help}"; [ $# -gt 0 ] && shift || true

case "$cmd" in
  orient|status)  exec bash "$HERE/preamble.sh" "$@" ;;
  progress)       exec bash "$HERE/progress.sh" "$@" ;;
  ready)          exec bash "$HERE/readiness.sh" "$@" ;;
  evidence-fresh) exec bash "$HERE/evidence-fresh.sh" "$@" ;;
  acceptance)
    a="${1:-}"
    if [ -n "$a" ] && [ -d "$a" ]; then d="$a"
    else s="$(resolve_slug "${a:-}")"; [ -n "$s" ] || { echo "devrites: no active feature" >&2; exit 5; }; d=".devrites/work/$s"; fi
    exec bash "$HERE/check-acceptance.sh" "$d" ;;
  tick-afk)
    s="$(resolve_slug "${1:-}")"; [ -n "$s" ] || { echo "devrites: no active feature" >&2; exit 5; }
    exec bash "$HERE/tick-afk.sh" ".devrites/work/$s/state.md" ;;
  resolve)        exec bash "$HERE/resolve.sh" "$@" ;;
  close)
    s="$(resolve_slug "${1:-}")"; [ -n "$s" ] || { echo "devrites: no slug and no ACTIVE" >&2; exit 4; }
    exec bash "$HERE/close-out.sh" "$s" ;;
  active)         a="$(active_slug)"; [ -n "$a" ] && printf '%s\n' "$a" || echo "(none)" ;;
  list)           if [ -d .devrites/work ]; then ls -1 .devrites/work; else echo "(no .devrites/work)"; fi ;;
  use)
    s="${1:-}"; [ -n "$s" ] || { echo "usage: devrites use <slug>" >&2; exit 2; }
    [ -d ".devrites/work/$s" ] || { echo "devrites: no workspace .devrites/work/$s" >&2; exit 5; }
    printf '%s\n' "$s" > .devrites/ACTIVE; echo "active: $s" ;;
  help|-h|--help) usage ;;
  *) echo "devrites: unknown command '$cmd' (try: devrites help)" >&2; usage >&2; exit 2 ;;
esac
