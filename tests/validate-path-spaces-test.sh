#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
SPACE_ROOT="$TMP/repository with spaces"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$SPACE_ROOT"
(
  cd "$ROOT"
  git ls-files -z | tar --null -T - -cf -
) | tar -C "$SPACE_ROOT" -xf -

bash "$SPACE_ROOT/scripts/validate.sh" >"$TMP/validate.log" 2>&1 || {
  sed -n '1,100p' "$TMP/validate.log"
  exit 1
}
grep -q 'VALIDATION PASSED' "$TMP/validate.log"
echo "validate-path-spaces-test: PASS"
