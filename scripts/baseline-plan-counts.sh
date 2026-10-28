#!/usr/bin/env bash
# Re-derive the plan-shape counts recorded in tests/parallel-baseline-counts.json.
#
# Each checked-in plan input under tests/baseline-plans/<plan>/tasks.md is placed in a
# throwaway workspace so `check task-graph` can schedule it, then the schedule line is
# printed. No repository state is written; the temp workspace is removed on exit.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

strict=""
[ "${1:-}" = "--strict" ] && strict=" --strict"

(cd "$root/engine" && go build -o "$tmp/devrites-engine" .)

for plan_dir in "$root"/tests/baseline-plans/*/; do
  name=$(basename "$plan_dir")
  ws="$tmp/$name"
  mkdir -p "$ws/.devrites/work/$name"
  cp "$plan_dir/tasks.md" "$ws/.devrites/work/$name/tasks.md"
  out=$(cd "$ws" && "$tmp/devrites-engine" check task-graph "$name"$strict 2>&1) && code=0 || code=$?
  line=$(printf '%s\n' "$out" | grep -E 'schedule:|BLOCKED|REFUSED' | head -1)
  printf '%-24s exit=%s %s\n' "$name" "$code" "${line:-no schedule or refusal line}"
done
