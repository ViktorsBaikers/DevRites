#!/usr/bin/env bash
# DevRites Stop hook — keep the code-intelligence indexes fresh after code changes, so the
# NEXT structural lookup (see .claude/rules/tooling.md) reads current code, not a stale graph.
# Reindexes the three optional indexes that track this repo — codebase-memory-mcp, codegraph,
# graphify — each only if its binary is on PATH AND it already tracks the repo. Missing = no-op.
#
# Enhancement, never a dependency: silent on projects with no index, FAIL-OPEN, NEVER blocks.
# The turn never waits — work runs in a DETACHED background process; the Stop hook returns at
# once. Self-guards on a change-stamp so idle turns cost nothing. ON by default; disable with
# DEVRITES_REFRESH_INDEXES=off. Does NOT touch agentmemory (separate, judgment-based store).
#
# Modes:
#   (no args)        trigger: guard on changes, then spawn the detached worker. Used by the hook.
#   --force [ROOT]   run synchronously NOW, ignore the change-guard, print a report (the skill).
#   --worker ROOT    internal: the reindex work itself. Detached.
set -u

[ "${DEVRITES_REFRESH_INDEXES:-on}" = "off" ] && exit 0

# ---- portable bounded-time runner (macOS has no `timeout` by default) ----
run_timeout() { # <seconds> <cmd...>
  secs="$1"; shift
  if   command -v timeout  >/dev/null 2>&1; then timeout  "$secs" "$@"
  elif command -v gtimeout >/dev/null 2>&1; then gtimeout "$secs" "$@"
  else "$@"; fi
}

# ---- state-file locations derived from a repo root (all gitignore-safe) ----
compute_state() { # <root>
  ROOT="$1"
  if   [ -d "$ROOT/.codegraph" ]; then STATE="$ROOT/.codegraph"
  elif [ -d "$ROOT/.git"       ]; then STATE="$ROOT/.git"
  else STATE="$ROOT"; fi
  STAMP="$STATE/.index_refresh_stamp"
  LOCK="$STATE/.index_refresh.lock"
  LOG="$STATE/index_refresh.log"
}

# ---- is this repo tracked by codebase-memory-mcp? ----
# Cheap repo-local signal first (exported artifact); fall back to the project registry.
cbm_registered() { # <root>
  command -v codebase-memory-mcp >/dev/null 2>&1 || return 1
  [ -d "$1/.codebase-memory" ] && return 0
  codebase-memory-mcp cli --raw list_projects 2>/dev/null | grep -qF "$1"
}

# ---- the reindex work; runs detached (or synchronously under --force) ----
worker() { # <root>
  _root="$1"
  echo "[$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo '?')] refresh start: $_root"
  if cbm_registered "$_root"; then
    echo "[codebase-memory] index_repository"
    run_timeout 300 codebase-memory-mcp cli index_repository "{\"repo_path\": \"$_root\"}" 2>&1 \
      || echo "[codebase-memory] index failed or timed out"
  fi
  if [ -d "$_root/.codegraph" ] && command -v codegraph >/dev/null 2>&1; then
    echo "[codegraph] sync"
    run_timeout 120 codegraph sync "$_root" 2>&1 || echo "[codegraph] sync failed or timed out"
  fi
  if [ -d "$_root/graphify-out" ] && command -v graphify >/dev/null 2>&1; then
    echo "[graphify] update"
    run_timeout 300 graphify update "$_root" 2>&1 || echo "[graphify] update failed or timed out"
  fi
  echo "[$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo '?')] refresh done"
}

# ============================ entrypoint ============================

# --- worker branch: do the work, release the lock, exit ---
if [ "${1:-}" = "--worker" ]; then
  ROOT="${2:-$PWD}"
  compute_state "$ROOT"
  worker "$ROOT"
  rm -rf "$LOCK" 2>/dev/null || true
  exit 0
fi

# --- parse mode ---
FORCE=0
if [ "${1:-}" = "--force" ]; then FORCE=1; shift; fi
ROOT="${1:-${CLAUDE_PROJECT_DIR:-$PWD}}"
compute_state "$ROOT"

# --- availability guard: do nothing unless at least one index tracks this repo ---
if [ ! -d "$ROOT/.codegraph" ] && [ ! -d "$ROOT/graphify-out" ] && ! cbm_registered "$ROOT"; then
  exit 0
fi

# --- change guard: skip if nothing changed since last refresh (trigger mode only) ---
if [ "$FORCE" -ne 1 ] && [ -f "$STAMP" ]; then
  changed=$(find "$ROOT" -type f -newer "$STAMP" \
    -not -path '*/.git/*' \
    -not -path '*/.codegraph/*' \
    -not -path '*/graphify-out/*' \
    -not -path '*/.codebase-memory/*' \
    -not -path '*/.devrites/*' \
    -not -path '*/node_modules/*' \
    -not -path '*/.venv/*' \
    -not -path '*/__pycache__/*' \
    -not -path '*/.research/*' \
    -not -path '*/.cursor/*' \
    -not -path '*/dist/*' \
    -not -path '*/build/*' \
    -print 2>/dev/null | head -n 1)
  [ -z "$changed" ] && exit 0
fi

# --- lock: never run two refreshes at once; steal a lock older than 30 min ---
if ! mkdir "$LOCK" 2>/dev/null; then
  if [ -d "$LOCK" ] && find "$LOCK" -prune -mmin +30 2>/dev/null | grep -q .; then
    rm -rf "$LOCK" 2>/dev/null
    mkdir "$LOCK" 2>/dev/null || exit 0
  else
    exit 0   # another refresh in progress
  fi
fi

# debounce: stamp now so a no-op turn doesn't re-trigger
touch "$STAMP" 2>/dev/null || true

if [ "$FORCE" -eq 1 ]; then
  worker "$ROOT"                                   # synchronous: the skill wants the report
  rm -rf "$LOCK" 2>/dev/null || true
else
  nohup "$0" --worker "$ROOT" >>"$LOG" 2>&1 &      # detached: Stop returns immediately
fi
exit 0
