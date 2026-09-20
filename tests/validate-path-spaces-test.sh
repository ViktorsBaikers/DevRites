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
  git ls-files --cached --others --exclude-standard -z |
    while IFS= read -r -d '' file; do
      [ ! -e "$file" ] || printf '%s\0' "$file"
    done |
    tar --null -T - -cf -
) | tar -C "$SPACE_ROOT" -xf -

rm -rf "$SPACE_ROOT/pack/generated"
cp -R "$ROOT/pack/generated" "$SPACE_ROOT/pack/generated"
find "$SPACE_ROOT/pack/generated" -type d -empty -delete

git -C "$SPACE_ROOT" init -q
git -C "$SPACE_ROOT" add -f .
POISON_REPO="$TMP/poison repo"
mkdir -p "$POISON_REPO"
git -C "$POISON_REPO" init -q
printf 'poison\n' >"$POISON_REPO/README.md"
git -C "$POISON_REPO" add README.md

GIT_DIR="$POISON_REPO/.git" GIT_WORK_TREE="$POISON_REPO" \
  bash "$SPACE_ROOT/scripts/validate.sh" >"$TMP/validate.log" 2>&1 || {
  sed -n '1,100p' "$TMP/validate.log"
  exit 1
}
grep -q 'VALIDATION PASSED' "$TMP/validate.log"
echo "validate-path-spaces-test: PASS"
