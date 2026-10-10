#!/usr/bin/env bash
# validate-pack.sh: run the static pack validation once, show its output, keep its
# exit status, and require the "Skipped gates: N" line before an exact verdict line,
# with N equal to the skip lines validate.sh itself printed.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

bash "$ROOT/scripts/validate.sh" >"$T/out.txt" 2>&1
rc=$?
cat "$T/out.txt"

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

if [ "$summary" != "Skipped gates: $expected" ]; then
  echo "FAIL: line before the verdict is '$summary', expected 'Skipped gates: $expected'"
  exit 1
fi

exit "$rc"
