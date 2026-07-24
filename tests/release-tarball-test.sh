#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

copy_release_source() {
  local destination="$1"
  mkdir -p "$destination"
  for item in pack engine scripts mcp docs install.sh uninstall.sh update.sh README.md CHANGELOG.md LICENSE SECURITY.md NOTICE.md CODE_OF_CONDUCT.md CODEOWNERS package.json; do
    [[ ! -e "$ROOT/$item" ]] || cp -R "$ROOT/$item" "$destination/"
  done
  (
    cd "$destination"
    git init -q
    git add -f engine
  )
  mkdir -p "$destination/scripts/.cache" "$destination/docs/internal" "$destination/dist"
  printf 'prior output\n' >"$destination/scripts/.cache/prior.tar.gz"
  printf 'internal only\n' >"$destination/docs/internal/private.md"
  printf 'prior output\n' >"$destination/dist/devrites-vprior.tar.gz"
}

set_tree_mtime() {
  local root="$1" epoch="$2"
  python3 - "$root" "$epoch" <<'PY'
import os
import sys

root, epoch = sys.argv[1], int(sys.argv[2])
for parent, dirs, files in os.walk(root, topdown=False):
    if os.path.basename(parent) == ".git" or f"{os.sep}.git{os.sep}" in parent:
        continue
    for name in files + dirs:
        path = os.path.join(parent, name)
        if not os.path.islink(path):
            os.utime(path, (epoch, epoch))
os.utime(root, (epoch, epoch))
PY
}

verify_checksum() {
  local archive="$1" sidecar="$2" expected actual
  expected="$(awk '{print $1}' "$sidecar")"
  if command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
  else
    actual="$(sha256sum "$archive" | awk '{print $1}')"
  fi
  [[ "$actual" == "$expected" ]]
  [[ "$(awk '{print $2}' "$sidecar")" == "devrites-v0.0.0-repro.tar.gz" ]]
}

SOURCE_A="$TMP/source-a"
SOURCE_B="$TMP/different/source-b"
DIST_A="$TMP/dist-a"
DIST_B="$TMP/dist-b"
copy_release_source "$SOURCE_A"
copy_release_source "$SOURCE_B"
set_tree_mtime "$SOURCE_A" 946684800
set_tree_mtime "$SOURCE_B" 1893456000

(
  umask 022
  TZ=UTC SOURCE_DATE_EPOCH=1700000000 DEVRITES_RELEASE_DIST_DIR="$DIST_A" \
    bash "$SOURCE_A/scripts/build-release-tarball.sh" 0.0.0-repro >/dev/null
)
(
  umask 077
  TZ=Pacific/Honolulu SOURCE_DATE_EPOCH=1700000000 DEVRITES_RELEASE_DIST_DIR="$DIST_B" \
    bash "$SOURCE_B/scripts/build-release-tarball.sh" 0.0.0-repro >/dev/null
)

ARCHIVE_A="$DIST_A/devrites-v0.0.0-repro.tar.gz"
ARCHIVE_B="$DIST_B/devrites-v0.0.0-repro.tar.gz"
SIDECAR_A="$ARCHIVE_A.sha256"
SIDECAR_B="$ARCHIVE_B.sha256"
cmp "$ARCHIVE_A" "$ARCHIVE_B"
cmp "$SIDECAR_A" "$SIDECAR_B"
verify_checksum "$ARCHIVE_A" "$SIDECAR_A"

tar -tzf "$ARCHIVE_A" >"$TMP/members"
[[ "$(head -n 1 "$TMP/members")" == "devrites-v0.0.0-repro/" ]]
LC_ALL=C sort -c "$TMP/members"
while IFS= read -r member; do
  normalized="${member%/}"
  case "$normalized" in
    devrites-v0.0.0-repro | devrites-v0.0.0-repro/*) ;;
    *) echo "unsafe archive member: $member" >&2; exit 1 ;;
  esac
  case "/$normalized/" in
    */../* | */./*) echo "unsafe archive member: $member" >&2; exit 1 ;;
  esac
  case "$normalized" in
    *//*) echo "unsafe archive member: $member" >&2; exit 1 ;;
  esac
done <"$TMP/members"
grep -qx 'devrites-v0.0.0-repro/install.sh' "$TMP/members"
grep -qx 'devrites-v0.0.0-repro/pack/generated/README.md' "$TMP/members"
if grep -Eq '/(docs/internal|scripts/\.cache|dist)(/|$)|/engine/testdata/golden/' "$TMP/members"; then
  echo "release archive contains excluded development or prior output" >&2
  exit 1
fi

mkdir "$TMP/extracted"
tar -C "$TMP/extracted" -xzf "$ARCHIVE_A"
BUNDLE="$TMP/extracted/devrites-v0.0.0-repro"
cmp "$SOURCE_A/README.md" "$BUNDLE/README.md"
[[ -x "$BUNDLE/install.sh" && -x "$BUNDLE/update.sh" && -x "$BUNDLE/uninstall.sh" ]]
[[ ! -x "$BUNDLE/README.md" ]]

if DEVRITES_RELEASE_DIST_DIR="$SOURCE_A/scripts/release-dist" \
  bash "$SOURCE_A/scripts/build-release-tarball.sh" 0.0.0-overlap >"$TMP/overlap.log" 2>&1; then
  echo "release build accepted output inside its payload" >&2
  exit 1
fi
grep -q 'output directory overlaps the release payload' "$TMP/overlap.log"

ln -s "$TMP/outside" "$SOURCE_A/pack/unsafe-link"
if TZ=UTC SOURCE_DATE_EPOCH=1700000000 DEVRITES_RELEASE_DIST_DIR="$TMP/unsafe-dist" \
  bash "$SOURCE_A/scripts/build-release-tarball.sh" 0.0.0-unsafe >"$TMP/unsafe.log" 2>&1; then
  echo "release build accepted a symlink payload" >&2
  exit 1
fi
grep -q 'symlink is not allowed' "$TMP/unsafe.log"
[[ ! -e "$TMP/unsafe-dist/devrites-v0.0.0-unsafe.tar.gz" ]]
[[ ! -e "$TMP/unsafe-dist/devrites-v0.0.0-unsafe.tar.gz.sha256" ]]

echo "release-tarball-test: PASS"
