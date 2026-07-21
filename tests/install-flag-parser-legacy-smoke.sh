#!/usr/bin/env bash
# install-flag-parser-legacy-smoke.sh — retired/no-op flags still parse.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== install-flag-parser-legacy-smoke =="

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
  "--no-rules" \
  "--rules-only" \
  "--no-short-aliases" \
  "--short-aliases=all" ; do
  run_flags "$flags" & pids+=("$!")
done

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "install-flag-parser-legacy-smoke: PASS" || echo "install-flag-parser-legacy-smoke: FAIL"
exit "$fail"
