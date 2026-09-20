#!/usr/bin/env bash
# scripts/issue-ready.sh: compute the ready frontier of the local .scratch
# Markdown issue tracker (docs/agents/issue-tracker.md).
#
# An issue is ready when its `Status:` line is open or ready-for-agent, every
# `Blocked by:` target is in a terminal state (resolved/done/wontfix), and it
# is not claimed. Prints ready issues as "<file> — <title>" sorted by number.
#
# Usage:
#   scripts/issue-ready.sh                 # scan every .scratch/*/issues/*.md
#   scripts/issue-ready.sh <feature-slug>  # scan .scratch/<slug>/issues/*.md
#
# Read-only. Exit 0 always; no matches prints nothing.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"

dirs=()
if [ $# -gt 0 ]; then
  for arg in "$@"; do
    if [ -d "$arg" ]; then
      dirs+=("$arg")
    elif [ -d "$ROOT/.scratch/$arg/issues" ]; then
      dirs+=("$ROOT/.scratch/$arg/issues")
    elif [ -d "$ROOT/.scratch/$arg" ]; then
      dirs+=("$ROOT/.scratch/$arg")
    else
      echo "issue-ready: no issues directory for '$arg'" >&2
      exit 2
    fi
  done
else
  for d in "$ROOT"/.scratch/*/issues; do
    [ -d "$d" ] && dirs+=("$d")
  done
fi

# status_of <file>: first Status line, lowercased, decoration stripped.
status_of() {
  sed -n -E 's/^[[:space:]]*\**Status:\**[[:space:]]*//Ip' "$1" 2>/dev/null \
    | head -1 \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[[:space:]]+$//; s/\.$//'
}

# blockers_of <file>: issue numbers on the first Blocked-by line.
blockers_of() {
  sed -n -E 's/^[[:space:]]*\**Blocked by:\**[[:space:]]*//Ip' "$1" 2>/dev/null \
    | head -1 \
    | tr ',' ' ' \
    | grep -oE '[0-9]+[a-z]?' \
    | sort -u
}

# issue_num <file>: numeric id from the filename prefix (08b -> 08b).
issue_num() {
  basename "$1" | grep -oE '^[0-9]+[a-z]?' || true
}

is_terminal() {
  case "$1" in
    resolved|done|done\ *|wontfix|closed) return 0 ;;
    *) return 1 ;;
  esac
}

is_open() {
  case "$1" in
    open|ready-for-agent) return 0 ;;
    *) return 1 ;;
  esac
}

# First pass: num -> status map ("num<TAB>status" lines).
map_file="$(mktemp)"
trap 'rm -f "$map_file"' EXIT

for d in "${dirs[@]}"; do
  for f in "$d"/*.md; do
    [ -f "$f" ] || continue
    n="$(issue_num "$f")"
    [ -n "$n" ] || continue
    printf '%s\t%s\n' "$n" "$(status_of "$f")" >>"$map_file"
  done
done

status_for_num() {
  awk -F '\t' -v n="$1" '$1 == n { print $2 }' "$map_file" | head -1
}

for d in "${dirs[@]}"; do
  for f in "$d"/*.md; do
    [ -f "$f" ] || continue
    n="$(issue_num "$f")"
    [ -n "$n" ] || continue
    st="$(status_of "$f")"
    is_open "$st" || continue
    ready=1
    for dep in $(blockers_of "$f"); do
      dep_st="$(status_for_num "$dep")"
      is_terminal "$dep_st" || { ready=0; break; }
    done
    [ "$ready" -eq 1 ] || continue
    title="$(sed -n -E 's/^#[[:space:]]+//p' "$f" | head -1)"
    printf '%s\t%s\n' "$n" "$f — $title"
  done
done | sort -t "$(printf '\t')" -k1,1 | cut -f2-
