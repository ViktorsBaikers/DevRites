#!/usr/bin/env bash
# validate.sh must print "Skipped gates: N" on the line before its verdict, N must
# equal the skip lines validate.sh itself printed, and the verdict line stays exact.
# The real skip helper and summary block are lifted from validate.sh into a stub with
# three skipped gates, so no second full validation run is needed.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

V="$ROOT/scripts/validate.sh"
{
  sed -n '/^fail=0$/,/^skips=0$/p;/^bad() /,/^skip() /p' "$V"
  printf 'skip "stub one"\nskip "stub two"\nskip "stub three"\n'
  sed -n '/^# ---- summary/,$p' "$V"
} >"$T/stub.sh"

bash "$T/stub.sh" >"$T/out.txt" 2>&1 || true

verdict="$(tail -n 1 "$T/out.txt")"
summary="$(tail -n 2 "$T/out.txt" | head -n 1)"
expected="$(grep -c '^skip (validate): ' "$T/out.txt" || true)"

case "$verdict" in
  "VALIDATION PASSED" | "VALIDATION FAILED") ;;
  *)
    echo "FAIL: final line '$verdict' is not an exact verdict"
    exit 1
    ;;
esac

if [ "$expected" != 3 ]; then
  echo "FAIL: stub printed $expected skip lines, expected 3"
  exit 1
fi

if [ "$summary" != "Skipped gates: $expected" ]; then
  echo "FAIL: line before the verdict is '$summary', expected 'Skipped gates: $expected'"
  exit 1
fi

# The shellcheck gate: CI=true with shellcheck missing must fail and be listed in
# the failure list; without CI it stays a skip. PATH is empty so no shellcheck is
# found; the stub only uses shell builtins.
{
  sed -n '/^fail=0$/,/^skips=0$/p;/^bad() /,/^skip() /p' "$V"
  printf 'SH_LIST=()\n'
  sed -n '/^section "shellcheck/,/^# ---- 16\./p' "$V"
  sed -n '/^# ---- summary/,$p' "$V"
} >"$T/sc.sh"
BASH_BIN="$(command -v bash)"

rc=0
env -u CI PATH="$T/none" "$BASH_BIN" "$T/sc.sh" >"$T/sc-local.txt" 2>&1 || rc=$?
if [ "$rc" != 0 ] || ! grep -q '^skip (validate): shellcheck' "$T/sc-local.txt"; then
  echo "FAIL: without CI a missing shellcheck must skip and pass (exit $rc)"
  exit 1
fi

rc=0
env CI=true PATH="$T/none" "$BASH_BIN" "$T/sc.sh" >"$T/sc-ci.txt" 2>&1 || rc=$?
if [ "$rc" = 0 ] || grep -q '^skip (validate): shellcheck' "$T/sc-ci.txt" ||
  ! grep -q '^FAIL: shellcheck' "$T/sc-ci.txt" || ! grep -q '^  - shellcheck' "$T/sc-ci.txt"; then
  echo "FAIL: with CI=true a missing shellcheck must fail and be listed in the failure list (exit $rc)"
  exit 1
fi

echo "validate-skip-summary: ok ($expected skipped)"
