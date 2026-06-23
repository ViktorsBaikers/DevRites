#!/usr/bin/env bash
# test-integrity.sh — anti-reward-hacking gate for /rite-build, /rite-prove, /rite-seal.
# Proves the slice did not reach green by WEAKENING its tests. Diffs the test files
# against the slice base (the reconcile.sh snapshot tree if present, else HEAD) and
# flags any test file that was deleted, had a skip/focus marker added, or lost
# assertions. Turns "never weaken a failing test" (rules/testing.md) into an exit code
# instead of a prose promise the model can rationalize past — same doctrine as
# reconcile.sh / readiness.sh. Read-only.
#
# Usage:
#   test-integrity.sh [slug]            (alias: test-integrity.sh check [slug])
#     slug — optional; defaults to .devrites/ACTIVE
#
# Exit codes:
#   0  clean — no test file deleted, skipped, or de-asserted since the base
#   2  no active workspace / bad args
#   3  WEAKENED — a test was deleted, skip/.only-marked, or lost assertions: Critical.
#      Revert the weakening; fix the code (or route the change as a blocking question).
set -u

a1="${1:-}"
case "$a1" in
  check) slug="${2:-}" ;;
  ""|-*) slug="" ;;
  *) slug="$a1" ;;
esac
[ -n "$slug" ] || slug="$(cat .devrites/ACTIVE 2>/dev/null || true)"
[ -n "$slug" ] || { echo "test-integrity: no active workspace (pass a slug or set .devrites/ACTIVE)." >&2; exit 2; }

d=".devrites/work/$slug"
[ -d "$d" ] || { echo "test-integrity: no workspace at $d." >&2; exit 2; }

root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$root" ]; then
  echo "test-integrity: not a git repo — gate skipped, eyeball the test diff by hand." >&2
  exit 0
fi

base_tree="$(cat "$d/.reconcile-base" 2>/dev/null || true)"
ref="${base_tree:-HEAD}"

is_test_file() {
  case "$1" in
    *_test.*|*.test.*|*.spec.*|*_spec.*|*Test.*|*Tests.*|*Spec.*) return 0 ;;
    */__tests__/*|*/tests/*|*/test/*|*/spec/*|test_*) return 0 ;;
    *test*|*spec*) return 0 ;;
  esac
  return 1
}

SKIP_RE='(\bit|\bdescribe|\btest|\bcontext)\.skip|\bxit\b|\bxdescribe\b|\.only\b|@pytest\.mark\.skip|@unittest\.skip|\bpytest\.skip\b|t\.Skip\(|#\[ignore\]|@Ignore\b|@Disabled\b|\bfdescribe\b|\bfit\('
ASSERT_RE='assert|expect\(|\.should\b|\brequire\.|EXPECT_|ASSERT_|XCTAssert|t\.Error|t\.Fatal'

count() { printf '%s' "${1:-}" | grep -oE "$2" 2>/dev/null | wc -l | tr -d ' '; }

weak=0
report=""

while IFS= read -r f; do
  [ -z "$f" ] && continue
  is_test_file "$f" || continue
  old="$(git -C "$root" show "$ref:$f" 2>/dev/null || true)"
  [ -n "$old" ] || continue           # new test file — adding tests is never a weakening
  if [ ! -f "$root/$f" ]; then
    weak=$((weak+1)); report="${report}  - ${f}: test file DELETED
"; continue
  fi
  new="$(cat "$root/$f")"
  os="$(count "$old" "$SKIP_RE")"; ns="$(count "$new" "$SKIP_RE")"
  oa="$(count "$old" "$ASSERT_RE")"; na="$(count "$new" "$ASSERT_RE")"
  if [ "${ns:-0}" -gt "${os:-0}" ]; then
    weak=$((weak+1)); report="${report}  - ${f}: skip/focus markers added (${os}→${ns})
"
  elif [ "${na:-0}" -lt "${oa:-0}" ]; then
    weak=$((weak+1)); report="${report}  - ${f}: assertions dropped (${oa}→${na})
"
  fi
done < <(git -C "$root" diff --name-only "$ref" 2>/dev/null)

if [ "$weak" -gt 0 ]; then
  echo "test-integrity: WEAKENED — $weak test file(s) lost coverage since the slice base:" >&2
  printf '%s' "$report" >&2
  echo "A test weakened to pass a gate is a Critical finding. Revert it; fix the code or raise a blocking question." >&2
  exit 3
fi
echo "test-integrity: OK — no test deleted, skipped, or de-asserted since the base."
exit 0
