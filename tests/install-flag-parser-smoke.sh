#!/usr/bin/env bash
# install-flag-parser-smoke.sh — keep core install.sh replayable flags parseable.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== install-flag-parser-smoke =="

run_flags() {
  local flags="$1"
  local target
  target="$(mktemp -d "$T/case.XXXXXX")" || return 1
  # shellcheck disable=SC2086
  if bash "$ROOT/install.sh" --target "$target" --dry-run $flags >/dev/null 2>&1; then
    ok "install.sh parses recorded flags: $flags"
    return 0
  else
    no "install.sh rejected recorded flags: $flags"
    return 1
  fi
}

pids=()
for flags in \
  "--no-skills --no-agents --no-short-aliases" \
  "--no-agents" \
  "--no-codex" ; do
  run_flags "$flags" & pids+=("$!")
done

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "install-flag-parser-smoke: PASS" || echo "install-flag-parser-smoke: FAIL"
exit "$fail"
