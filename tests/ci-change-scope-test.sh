#!/usr/bin/env bash
# Path-scope outputs for required-check-safe CI filtering.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SH="$ROOT/scripts/ci-change-scope.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

# GitHub Actions exports GITHUB_OUTPUT for every step. The scope script
# appends key=value to that file when it is set, so capturing stdout would
# miss the real contract and would write into the runner's output file.
# Point the script at a private OUTPUT so this test never mutates runner output.
OUTPUT="$(mktemp)"
trap 'rm -f "$OUTPUT"' EXIT

scope() {
  : >"$OUTPUT"
  GITHUB_OUTPUT="$OUTPUT" GITHUB_EVENT_NAME=pull_request DEVRITES_CI_CHANGED_PATHS="$1" bash "$SH" || return
  cat "$OUTPUT"
}

expect() {
  local paths="$1" key="$2" want="$3"
  local out
  out="$(scope "$paths")" || { no "script failed for [$paths]"; return; }
  printf '%s\n' "$out" | grep -qx "${key}=${want}" && ok "$key=$want for [$paths]" || {
    no "expected $key=$want for [$paths]; got:"
    printf '%s\n' "$out"
  }
}

echo "== ci-change-scope-test =="

# Push/default (no PR): full matrix.
: >"$OUTPUT"
GITHUB_OUTPUT="$OUTPUT" GITHUB_EVENT_NAME=push bash "$SH" || no "script failed for push"
out="$(cat "$OUTPUT")"
printf '%s\n' "$out" | grep -qx 'run_tests=true' && ok "push keeps run_tests=true" || no "push should run tests"

expect $'engine/main.go\nengine/go.mod' run_tests false
expect $'engine/main.go\nengine/go.mod' run_engine true
expect $'engine/main.go\nengine/go.mod' run_full true
expect $'engine/main.go\ntests/eval-coverage-ledger-test.sh' run_tests true
expect $'README.md\ndocs/release.md' run_full false
expect $'README.md\ndocs/release.md' run_tests false
expect $'pack/.claude/skills/rite-build/SKILL.md' run_tests true
expect $'pack/.claude/skills/rite-build/SKILL.md' run_engine false
expect $'.github/workflows/ci.yml' run_tests true
expect $'engine/main.go\n.github/workflows/ci.yml' run_tests true

echo ""
[ "$fail" -eq 0 ] && echo "ci-change-scope-test: PASS" || echo "ci-change-scope-test: FAIL"
exit "$fail"
