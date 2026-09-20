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
    git add -f .
  )
  mkdir -p "$destination/scripts/.cache" "$destination/docs/internal" "$destination/dist"
  printf 'prior output\n' >"$destination/scripts/.cache/prior.tar.gz"
  printf 'internal only\n' >"$destination/docs/internal/private.md"
  printf 'prior output\n' >"$destination/dist/devrites-vprior.tar.gz"
  printf 'private\n' >"$destination/scripts/UNTRACKED_SECRET.txt"
  printf 'private\n' >"$destination/pack/UNTRACKED_SECRET.txt"
  printf 'private generated instruction\n' >"$destination/pack/.claude/agents/UNTRACKED_SECRET.md"
  printf 'private\n' >"$destination/docs/UNTRACKED_SECRET.txt"
}

poison_checkout_materialization() {
  local source="$1"
  cat > "$source/.git/info/attributes" <<'EOF'
install.sh text eol=crlf filter=release-poison
README.md text eol=crlf filter=release-poison
EOF
  cat > "$source/.git/release-poison-smudge.sh" <<'EOF'
#!/bin/sh
printf 'FILTERED-BY-CHECKOUT\n'
cat
EOF
  chmod +x "$source/.git/release-poison-smudge.sh"
  git -C "$source" config core.autocrlf true
  git -C "$source" config filter.release-poison.required true
  git -C "$source" config filter.release-poison.smudge .git/release-poison-smudge.sh
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
  local archive="$1" sidecar="$2" expected_name="$3" expected actual
  expected="$(awk '{print $1}' "$sidecar")"
  if command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
  else
    actual="$(sha256sum "$archive" | awk '{print $1}')"
  fi
  [[ "$actual" == "$expected" ]]
  [[ "$(awk '{print $2}' "$sidecar")" == "$expected_name" ]]
}

SOURCE_A="$TMP/source-a"
SOURCE_B="$TMP/different/source-b"
SOURCE_C="$TMP/source-no-checksum"
SOURCE_NONGIT="$TMP/source-no-git"
DIST_A="$TMP/dist-a"
DIST_B="$TMP/dist-b"
POISON_REPO="$TMP/poison-repo"
copy_release_source "$SOURCE_A"
copy_release_source "$SOURCE_B"
copy_release_source "$SOURCE_C"
copy_release_source "$SOURCE_NONGIT"
for source in "$SOURCE_A" "$SOURCE_B"; do
  printf 'index mode probe\n' > "$source/docs/index-mode-probe"
  chmod 0755 "$source/docs/index-mode-probe"
  git -C "$source" add -f docs/index-mode-probe
  chmod 0644 "$source/docs/index-mode-probe"
done
poison_checkout_materialization "$SOURCE_A"
POISON_CONTROL="$TMP/checkout-poison-control"
mkdir "$POISON_CONTROL"
git -C "$SOURCE_A" checkout-index --prefix="$POISON_CONTROL/" -- install.sh README.md
grep -Fq 'FILTERED-BY-CHECKOUT' "$POISON_CONTROL/install.sh"
python3 - "$POISON_CONTROL/install.sh" <<'PY'
from pathlib import Path
import sys

if b"\r\n" not in Path(sys.argv[1]).read_bytes():
    raise SystemExit("checkout poison control did not apply configured CRLF conversion")
PY
INDEX_README="$TMP/index-readme"
git -C "$SOURCE_A" show :README.md > "$INDEX_README"
printf 'worktree B must not ship\n' > "$SOURCE_A/README.md"
printf 'another worktree B must not ship\n' > "$SOURCE_B/README.md"
[[ "$(git -C "$SOURCE_A" show :README.md)" != "$(cat "$SOURCE_A/README.md")" ]]
rm -rf "$SOURCE_NONGIT/.git"
mkdir -p "$POISON_REPO"
git -C "$POISON_REPO" init -q
printf 'poison\n' >"$POISON_REPO/README.md"
git -C "$POISON_REPO" add README.md
set_tree_mtime "$SOURCE_A" 946684800
set_tree_mtime "$SOURCE_B" 1893456000

(
  umask 022
  TZ=UTC SOURCE_DATE_EPOCH=1700000000 DEVRITES_RELEASE_DIST_DIR="$DIST_A" \
    GIT_DIR="$POISON_REPO/.git" GIT_WORK_TREE="$POISON_REPO" \
    bash "$SOURCE_A/scripts/build-release-tarball.sh" 0.0.0-repro >/dev/null
)
(
  umask 077
  TZ=Pacific/Honolulu SOURCE_DATE_EPOCH=1700000000 DEVRITES_RELEASE_DIST_DIR="$DIST_B" \
    GIT_DIR="$POISON_REPO/.git" GIT_WORK_TREE="$POISON_REPO" \
    bash "$SOURCE_B/scripts/build-release-tarball.sh" 0.0.0-repro >/dev/null
)

ARCHIVE_A="$DIST_A/devrites-v0.0.0-repro.tar.gz"
ARCHIVE_B="$DIST_B/devrites-v0.0.0-repro.tar.gz"
SIDECAR_A="$ARCHIVE_A.sha256"
SIDECAR_B="$ARCHIVE_B.sha256"
cmp "$ARCHIVE_A" "$ARCHIVE_B"
cmp "$SIDECAR_A" "$SIDECAR_B"
verify_checksum "$ARCHIVE_A" "$SIDECAR_A" "devrites-v0.0.0-repro.tar.gz"
cmp "$DIST_A/install.sh" "$DIST_B/install.sh"
cmp "$DIST_A/install.sh.sha256" "$DIST_B/install.sh.sha256"
verify_checksum "$DIST_A/install.sh" "$DIST_A/install.sh.sha256" install.sh
git -C "$SOURCE_A" show :install.sh > "$TMP/index-install.sh"
cmp "$TMP/index-install.sh" "$DIST_A/install.sh"

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
if grep -q 'UNTRACKED_SECRET' "$TMP/members"; then
  echo "release archive contains an untracked working-tree file" >&2
  exit 1
fi

mkdir "$TMP/extracted"
tar -C "$TMP/extracted" -xzf "$ARCHIVE_A"
BUNDLE="$TMP/extracted/devrites-v0.0.0-repro"
cmp "$INDEX_README" "$BUNDLE/README.md"
cmp "$TMP/index-install.sh" "$BUNDLE/install.sh"
[[ -x "$BUNDLE/install.sh" && -x "$BUNDLE/update.sh" && -x "$BUNDLE/uninstall.sh" ]]
[[ -x "$BUNDLE/docs/index-mode-probe" ]]
[[ ! -x "$BUNDLE/README.md" ]]

if DEVRITES_RELEASE_DIST_DIR="$SOURCE_A/scripts/release-dist" \
  bash "$SOURCE_A/scripts/build-release-tarball.sh" 0.0.0-overlap >"$TMP/overlap.log" 2>&1; then
  echo "release build accepted output inside its payload" >&2
  exit 1
fi
grep -q 'output directory overlaps the release payload' "$TMP/overlap.log"

if DEVRITES_RELEASE_DIST_DIR="$TMP/no-git-dist" \
  bash "$SOURCE_NONGIT/scripts/build-release-tarball.sh" 0.0.0-no-git >"$TMP/no-git.log" 2>&1; then
  echo "release build succeeded without a Git index" >&2
  exit 1
fi
grep -q 'Git index' "$TMP/no-git.log"
[[ ! -e "$TMP/no-git-dist/devrites-v0.0.0-no-git.tar.gz" ]]
[[ ! -e "$TMP/no-git-dist/install.sh" ]]

ln -s "$TMP/outside" "$SOURCE_A/pack/unsafe-link"
git -C "$SOURCE_A" add -f pack/unsafe-link
if TZ=UTC SOURCE_DATE_EPOCH=1700000000 DEVRITES_RELEASE_DIST_DIR="$TMP/unsafe-dist" \
  bash "$SOURCE_A/scripts/build-release-tarball.sh" 0.0.0-unsafe >"$TMP/unsafe.log" 2>&1; then
  echo "release build accepted a symlink payload" >&2
  exit 1
fi
grep -q 'symlink is not allowed' "$TMP/unsafe.log"
[[ ! -e "$TMP/unsafe-dist/devrites-v0.0.0-unsafe.tar.gz" ]]
[[ ! -e "$TMP/unsafe-dist/devrites-v0.0.0-unsafe.tar.gz.sha256" ]]

cat > "$TMP/no-checksum-env.sh" <<'EOF'
command() {
  if [[ "$1" == -v && ( "$2" == shasum || "$2" == sha256sum ) ]]; then
    return 1
  fi
  builtin command "$@"
}
EOF
if BASH_ENV="$TMP/no-checksum-env.sh" TZ=UTC SOURCE_DATE_EPOCH=1700000000 \
  DEVRITES_RELEASE_DIST_DIR="$TMP/no-checksum-dist" \
  bash "$SOURCE_C/scripts/build-release-tarball.sh" 0.0.0-no-checksum >"$TMP/no-checksum.log" 2>&1; then
  echo "release build succeeded without a SHA-256 tool" >&2
  exit 1
fi
grep -q 'release checksum is mandatory' "$TMP/no-checksum.log"
[[ ! -e "$TMP/no-checksum-dist/devrites-v0.0.0-no-checksum.tar.gz" ]]
[[ ! -e "$TMP/no-checksum-dist/devrites-v0.0.0-no-checksum.tar.gz.sha256" ]]

if grep -Eq 'raw\.githubusercontent\.com/.*/main/install\.sh|archive/refs/heads/main' "$ROOT/README.md"; then
  echo "README recommends a mutable default-branch installer" >&2
  exit 1
fi
for phrase in 'releases/latest/download' 'install.sh.sha256' 'bash ./install.sh update' 'bash ./install.sh uninstall'; do
  grep -Fq "$phrase" "$ROOT/README.md" || {
    echo "README misses Node-free release installer guidance: $phrase" >&2
    exit 1
  }
done
installer_curl="$(grep -F 'curl -fL' "$ROOT/README.md" | grep -F 'install.sh"' | grep -v 'install.sh.sha256"')"
sidecar_curl="$(grep -F 'curl -fL' "$ROOT/README.md" | grep -F 'install.sh.sha256"')"
assert_download_bounds() {
  local command="$1"
  shift
  for required in "$@"; do
    [[ "$command" == *"$required"* ]] || {
      echo "README bootstrap download misses bound: $required" >&2
      exit 1
    }
  done
}
assert_download_bounds "$installer_curl" '--connect-timeout 10' '--max-time 60' '--max-filesize 1048576' 'head -c 1048577'
assert_download_bounds "$sidecar_curl" '--connect-timeout 10' '--max-time 30' '--max-filesize 4096' 'head -c 4097'

node - "$ROOT/.releaserc.json" <<'JS'
const config = require(process.argv[2]);
const exec = config.plugins.find(([name]) => name === '@semantic-release/exec')[1].prepareCmd;
const stage = 'git add -- CHANGELOG.md README.md package.json';
if (!exec.includes(stage) || exec.indexOf(stage) > exec.indexOf('build-release-tarball.sh') || (exec.match(/git add/g) || []).length !== 1) {
  throw new Error('release prepare must stage only known overlays before the index-owned builder');
}
const assets = config.plugins.find(([name]) => name === '@semantic-release/github')[1].assets.map((asset) => asset.path);
for (const path of ['dist/install.sh', 'dist/install.sh.sha256']) {
  if (!assets.includes(path)) throw new Error(`semantic-release does not publish ${path}`);
}
JS

echo "release-tarball-test: PASS"
