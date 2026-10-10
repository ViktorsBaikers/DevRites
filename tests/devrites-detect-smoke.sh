#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

fail=0

expect_fail_contains() {
  local label="$1"
  local expected="$2"
  shift 2

  local output
  if output="$("$@" 2>&1)"; then
    echo "FAIL: $label (expected non-zero)"
    fail=1
  elif grep -Fq "$expected" <<<"$output"; then
    echo "PASS: $label"
  else
    echo "FAIL: $label (missing: $expected)"
    echo "$output"
    fail=1
  fi
}

expect_ok() {
  local label="$1"
  shift

  if "$@" >/dev/null 2>&1; then
    echo "PASS: $label"
  else
    echo "FAIL: $label"
    fail=1
  fi
}

cat >"$T/diff-narration.js" <<'JS'
// Updated to handle the empty response edge case.
export const normalizeResponse = (response) => response ?? [];
JS

cat >"$T/earned-comment.js" <<'JS'
// The provider returns null while leadership changes hands; callers need an empty list.
export const normalizeResponse = (response) => response ?? [];
JS

expect_fail_contains \
  "detector rejects diff-narrating comments" \
  "Diff-narrating comment" \
  "$ROOT/scripts/devrites-detect.sh" "$T/diff-narration.js"

expect_ok \
  "detector keeps comments that explain why" \
  "$ROOT/scripts/devrites-detect.sh" "$T/earned-comment.js"

TARGET_REPO="$T/target repo"
POISON_REPO="$T/poison repo"
mkdir -p "$TARGET_REPO" "$POISON_REPO"
git -C "$TARGET_REPO" init -q
git -C "$TARGET_REPO" config user.email devrites@example.invalid
git -C "$TARGET_REPO" config user.name 'DevRites Test'
printf 'export const base = true;\n' >"$TARGET_REPO/base.js"
git -C "$TARGET_REPO" add base.js
git -C "$TARGET_REPO" commit -qm base
cat >"$TARGET_REPO/target.js" <<'JS'
// Updated to handle the empty response edge case.
export const target = true;
JS
git -C "$TARGET_REPO" add target.js
git -C "$TARGET_REPO" commit -qm target
git -C "$TARGET_REPO" branch -m fixture

git -C "$POISON_REPO" init -q
git -C "$POISON_REPO" config user.email devrites@example.invalid
git -C "$POISON_REPO" config user.name 'DevRites Test'
printf 'export const base = true;\n' >"$POISON_REPO/base.js"
git -C "$POISON_REPO" add base.js
git -C "$POISON_REPO" commit -qm base
printf 'export const clean = true;\n' >"$POISON_REPO/clean.js"
git -C "$POISON_REPO" add clean.js
git -C "$POISON_REPO" commit -qm clean
git -C "$POISON_REPO" branch -m fixture

expect_fail_contains \
  "detector ignores inherited repository targets" \
  "Diff-narrating comment" \
  env GIT_DIR="$POISON_REPO/.git" GIT_WORK_TREE="$POISON_REPO" \
  bash -c 'cd "$1" && "$2"' _ "$TARGET_REPO" "$ROOT/scripts/devrites-detect.sh"

INDEX_REPO="$T/index repo"
mkdir -p "$INDEX_REPO"
git -C "$INDEX_REPO" init -q
git -C "$INDEX_REPO" config user.email devrites@example.invalid
git -C "$INDEX_REPO" config user.name 'DevRites Test'
printf 'one\n' >"$INDEX_REPO/LICENSE"
git -C "$INDEX_REPO" add LICENSE
git -C "$INDEX_REPO" commit -qm one
printf 'two\n' >"$INDEX_REPO/notes.txt"
git -C "$INDEX_REPO" add notes.txt
git -C "$INDEX_REPO" commit -qm two
git -C "$INDEX_REPO" update-index --refresh >/dev/null
sleep 1
touch "$INDEX_REPO/LICENSE"
index_before="$(shasum "$INDEX_REPO/.git/index")"
(cd "$INDEX_REPO" && "$ROOT/scripts/devrites-detect.sh" --advisory >/dev/null 2>&1)
index_after="$(shasum "$INDEX_REPO/.git/index")"
if [[ "$index_before" == "$index_after" ]]; then
  echo "PASS: detector leaves the git index untouched"
else
  echo "FAIL: detector rewrote the git index"
  fail=1
fi

STAT_REPO="$T/stat repo"
mkdir -p "$STAT_REPO"
git -C "$STAT_REPO" init -q
git -C "$STAT_REPO" config user.email devrites@example.invalid
git -C "$STAT_REPO" config user.name 'DevRites Test'
printf '// Updated to handle the empty response edge case.\nexport const old = true;\n' >"$STAT_REPO/old.js"
git -C "$STAT_REPO" add old.js
git -C "$STAT_REPO" commit -qm old
printf 'export const fresh = true;\n' >"$STAT_REPO/fresh.js"
git -C "$STAT_REPO" add fresh.js
git -C "$STAT_REPO" commit -qm fresh
git -C "$STAT_REPO" branch -m fixture
git -C "$STAT_REPO" update-index --refresh >/dev/null
sleep 1
touch "$STAT_REPO/old.js"
stat_index_before="$(shasum "$STAT_REPO/.git/index")"
stat_output="$(cd "$STAT_REPO" && "$ROOT/scripts/devrites-detect.sh" 2>&1)" && stat_exit=0 || stat_exit=$?
if [[ "$stat_exit" -eq 0 && "$stat_output" == *"clean"* \
  && "$stat_index_before" == "$(shasum "$STAT_REPO/.git/index")" ]]; then
  echo "PASS: detector ignores files that only changed timestamp"
else
  echo "FAIL: detector scanned a timestamp-only change (exit $stat_exit)"
  echo "$stat_output"
  fail=1
fi

printf '// Updated to handle the empty response edge case.\nexport const fresh = true;\n' >"$STAT_REPO/fresh.js"
expect_fail_contains \
  "detector still reports real content changes" \
  "fresh.js" \
  bash -c 'cd "$1" && "$2"' _ "$STAT_REPO" "$ROOT/scripts/devrites-detect.sh"

if [[ "$fail" -ne 0 ]]; then
  echo "DEVRITES DETECT SMOKE: FAIL"
  exit 1
fi

echo "DEVRITES DETECT SMOKE: PASS"
