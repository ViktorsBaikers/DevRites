#!/usr/bin/env bash
# Dependabot titles use scope no-release and name the package, so they are
# longer than the 72-char header cap. Other scopes stay capped.
# usage: commitlint-dependabot-header-test.sh [repo-root]
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd -P)}"
cd "$ROOT"

lint() {
  printf '%s\n' "$1" | npx --no commitlint >/dev/null
}

expect() {
  local want="$1" msg="$2" status
  set +e
  lint "$msg"
  status=$?
  set -e
  if [ "$status" -eq "$want" ]; then
    echo "ok   exit $want (${#msg} chars)"
    return
  fi
  echo "FAIL exit $status want $want (${#msg} chars): $msg"
  printf '%s\n' "$msg" | npx --no commitlint || true
  exit 1
}

short='feat(skills): add rite-prove browser proof ladder'
pad73="$(printf 'feat(skills): %059d' 0)"
plain='ci(no-release): bump the npm-minor-and-patch group across 1 directory with 2 updates'
security='ci(no-release): [security] bump @semantic-release/release-notes-generator from 15.0.0-beta.3 to 15.0.0-beta.4 in the npm-security group across 1 directory'
depsdev='build(deps-dev): bump the npm-minor-and-patch group across 1 directory with 2 updates'
too_long="ci(no-release): $(printf 'a%.0s' $(seq 1 185))"

[ "${#pad73}" -eq 73 ] || { echo "FAIL pad73 is ${#pad73}"; exit 1; }
[ "${#too_long}" -eq 201 ] || { echo "FAIL too_long is ${#too_long}"; exit 1; }

expect 0 "$short"
expect 1 "$pad73"
expect 0 "$plain"
expect 0 "$security"
expect 1 "$depsdev"
expect 1 "$too_long"
