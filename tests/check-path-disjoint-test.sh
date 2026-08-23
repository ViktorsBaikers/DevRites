#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
CHECK="$ROOT/scripts/check-path-disjoint.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

run() {
  python3 "$CHECK" "$@" 2>&1
}

assert_ok() {
  local label="$1"
  shift
  if run "$@" >/tmp/devrites-path-disjoint-ok.txt; then
    grep -q "path-disjoint: ok" /tmp/devrites-path-disjoint-ok.txt
  else
    echo "FAIL: $label should pass"
    cat /tmp/devrites-path-disjoint-ok.txt
    exit 1
  fi
}

assert_fail() {
  local label="$1"
  local pattern="$2"
  shift 2
  if run "$@" >/tmp/devrites-path-disjoint-bad.txt; then
    echo "FAIL: $label should fail"
    cat /tmp/devrites-path-disjoint-bad.txt
    exit 1
  fi
  grep -q "$pattern" /tmp/devrites-path-disjoint-bad.txt
}

cat >"$T/disjoint.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src/a.go", "tests/a_test.go"]},
    {"id": "slice-b", "paths": ["src/b.go", "tests/b_test.go"]}
  ]
}
JSON
assert_ok "pairwise disjoint exact paths" "$T/disjoint.json"

cat >"$T/overlap.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src/shared.go"]},
    {"id": "slice-b", "paths": ["src/shared.go", "src/b.go"]}
  ]
}
JSON
assert_fail "overlap is rejected" "path sets overlap" "$T/overlap.json"

cat >"$T/empty-path.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src/a.go"]},
    {"id": "slice-b", "paths": [""]}
  ]
}
JSON
assert_fail "empty path is rejected" "empty path is not allowed" "$T/empty-path.json"

cat >"$T/parent-segment.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src/../secret.go"]},
    {"id": "slice-b", "paths": ["src/b.go"]}
  ]
}
JSON
assert_fail "parent segment is rejected" "must not contain '..'" "$T/parent-segment.json"

cat >"$T/backslash.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src\\a.go"]},
    {"id": "slice-b", "paths": ["src/b.go"]}
  ]
}
JSON
assert_ok "backslashes normalize to forward slashes" "$T/backslash.json"

cat >"$T/duplicate-within-slice.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src/a.go", "src/a.go"]},
    {"id": "slice-b", "paths": ["src/b.go"]}
  ]
}
JSON
assert_fail "duplicate within slice is rejected" "duplicate path" "$T/duplicate-within-slice.json"

mkdir -p "$T/repo/src"
echo real >"$T/repo/src/a.go"
ln -s "$T/repo/src/a.go" "$T/repo/src/link.go"
cat >"$T/symlink.json" <<'JSON'
{
  "slices": [
    {"id": "slice-a", "paths": ["src/link.go"]},
    {"id": "slice-b", "paths": ["src/b.go"]}
  ]
}
JSON
assert_fail "symlink path is rejected with --root" "symlink path is not allowed" \
  --root "$T/repo" "$T/symlink.json"

echo "ok: path-disjoint helper accepts disjoint sets and rejects overlap and dirty paths"
