#!/usr/bin/env bash
# rites.sh — locate and run a devrites-lib helper by name, across install layouts.
#
# Usage: rites.sh <name> [args...]
#   <name> = helper basename without extension (preamble, readiness, reconcile,
#            stuck, footprint, test-integrity, progress, conventions, tick-afk, ...).
#
# Runs <this-dir>/<name>.sh — or <name>.py via python3 — propagating its exit code
# unchanged, so callers that read `$?` (readiness/reconcile/test-integrity/tick-afk)
# keep working. A missing helper, or a .py helper with no python3, prints a degraded
# note to stderr and exits 0 (degraded-but-continue, the same standing as the inline
# `|| echo "(... unavailable)"` fallback this replaces).
#
# This is the single home for the three-layout script-resolution dance; callers only
# locate THIS script (one target) and name the helper they want.
set -u

dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)" || dir="."
name="${1:-}"
if [ -z "$name" ]; then
  echo "rites: usage: rites.sh <name> [args...]" >&2
  exit 2
fi
shift

sh="$dir/$name.sh"
py="$dir/$name.py"

if [ -f "$sh" ]; then
  exec bash "$sh" "$@"
fi
if [ -f "$py" ]; then
  if command -v python3 >/dev/null 2>&1; then
    exec python3 "$py" "$@"
  fi
  echo "(devrites helper '$name' needs python3 — skipped)" >&2
  exit 0
fi

echo "(devrites helper '$name' unavailable on this install)" >&2
exit 0
