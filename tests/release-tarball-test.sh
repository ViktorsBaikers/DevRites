#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

copy_release_source() {
  local destination="$1"
  mkdir -p "$destination"
  for item in pack engine scripts docs install.sh uninstall.sh update.sh README.md CHANGELOG.md LICENSE SECURITY.md NOTICE.md CODE_OF_CONDUCT.md CODEOWNERS package.json; do
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
bootstrap_summary='<summary><b>Install without Node.js (verified Bash bootstrap)</b></summary>'
[[ "$(grep -Fxc "$bootstrap_summary" "$ROOT/README.md")" == 1 ]] || {
  echo "README must contain the verified bootstrap heading line exactly once" >&2
  exit 1
}
# The fence opens at the first line after the heading that equals ```bash and ends at the next line
# that equals ```; its bytes are compared exactly with the pinned copy below.
actual_bootstrap="$(awk -v summary="$bootstrap_summary" '
  $0 == summary { seen = 1 }
  seen && !open { if ($0 == "```bash") { open = 1; print } ; next }
  open { print; if ($0 == "```") exit }
' "$ROOT/README.md")"
# The bootstrap runs inside a throwaway directory, so every live installer call must name the project.
untargeted_install="$(printf '%s\n' "$actual_bootstrap" | grep -E '^[[:space:]]*bash \./install\.sh' | grep -Fv -e '--target' || true)"
[[ -z "$untargeted_install" ]] || {
  echo "README bootstrap runs install.sh without --target: $untargeted_install" >&2
  exit 1
}
expected_bootstrap="$(cat <<'EOF_BOOTSTRAP'
```bash
bootstrap_dir="$(mktemp -d)"
project_dir="$PWD"
(
  set -e
  trap 'rm -rf "$bootstrap_dir"' EXIT HUP INT TERM
  cd "$bootstrap_dir"
  release=https://github.com/ViktorsBaikers/DevRites/releases/latest/download
  curl -q -fL --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 60 --max-filesize 1048576 "$release/install.sh" | head -c 1048577 > install.sh
  install_status="${PIPESTATUS[0]}"
  [ "$(wc -c < install.sh)" -le 1048576 ] || { echo 'error: install.sh exceeds 1 MiB' >&2; exit 1; }
  [ "$install_status" -eq 0 ] || { echo 'error: install.sh download failed' >&2; exit 1; }
  curl -q -fL --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 30 --max-filesize 4096 "$release/install.sh.sha256" | head -c 4097 > install.sh.sha256
  sidecar_status="${PIPESTATUS[0]}"
  [ "$(wc -c < install.sh.sha256)" -le 4096 ] || { echo 'error: install.sh.sha256 exceeds 4 KiB' >&2; exit 1; }
  [ "$sidecar_status" -eq 0 ] || { echo 'error: install.sh.sha256 download failed' >&2; exit 1; }
  want="$(awk '
    NF == 0 { next }
    { records++ }
    NF == 2 && length($1) == 64 && $1 ~ /^[0-9A-Fa-f]+$/ && $2 == "install.sh" { valid++; hash=tolower($1) }
    END { if (records == 1 && valid == 1) print hash; else exit 1 }
  ' install.sh.sha256)"
  if command -v shasum >/dev/null 2>&1; then
    got="$(shasum -a 256 install.sh | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    got="$(sha256sum install.sh | awk '{print $1}')"
  else
    echo 'error: shasum or sha256sum is required' >&2
    exit 1
  fi
  [ "$got" = "$want" ] || { echo 'error: install.sh checksum mismatch' >&2; exit 1; }
  command -v gh >/dev/null 2>&1 || { echo 'error: gh (GitHub CLI) is required and was not found' >&2; exit 1; }
  gh attestation verify install.sh --repo ViktorsBaikers/DevRites --signer-workflow ViktorsBaikers/DevRites/.github/workflows/ci.yml@refs/heads/main || { if gh auth status >/dev/null 2>&1; then echo 'error: install.sh attestation verification failed: no build provenance from ViktorsBaikers/DevRites/.github/workflows/ci.yml@refs/heads/main' >&2; else echo 'error: gh is not authenticated; run gh auth login' >&2; fi; exit 1; }

  # Choose one Node-free operation:
  bash ./install.sh --target "$project_dir"  # install into the directory you ran this from
  # bash ./install.sh --dry-run --target "$project_dir"
  # bash ./install.sh update --target "$project_dir"
  # bash ./install.sh uninstall --target "$project_dir"
)
```
EOF_BOOTSTRAP
)"
# Whole-file checks on README.md: each release= line equals the origin; each line with lower-case
# curl, wget or aria2c followed by whitespace, or gh release|api, equals the pinned fence's; each
# lower-case http(s):// URL outside Markdown link and href/src syntax is under the origin.
release_origin='https://github.com/ViktorsBaikers/DevRites/releases/latest/download'
bad_origin="$(grep -E '(^|[^[:alnum:]_])release[+[:space:]]*=' "$ROOT/README.md" | sed 's/^[[:space:]]*//' | grep -Fxv "release=$release_origin" || true)"
[[ -z "$bad_origin" ]] || {
  echo "README bootstrap download origin is not release=$release_origin: $bad_origin" >&2
  exit 1
}
fetch_re='(^|[^[:alnum:]_.-])(curl|wget|aria2c)[[:space:]]|(^|[^[:alnum:]_.-])gh[[:space:]]+(release|api)[[:space:]]'
[[ "$(grep -E "$fetch_re" "$ROOT/README.md")" == "$(printf '%s\n' "$expected_bootstrap" | grep -E "$fetch_re")" ]] || {
  echo "README has a fetch command line outside the pinned bootstrap fence" >&2
  exit 1
}
# curl reads ~/.curlrc unless -q is its first argument, so every README and release-guide curl
# invocation must start with it; the mutants (no -q, -q not first) must be flagged.
curl_without_leading_q() {
  { grep -E '(^|[^[:alnum:]_.-])curl[[:space:]]' "$1" || true; } |
    { grep -Ev '(^|[^[:alnum:]_.-])curl -q ' || true; }
}
[[ -z "$(curl_without_leading_q "$ROOT/README.md")" ]] || {
  echo "README curl invocation does not pass -q as its first argument" >&2
  exit 1
}
[[ -z "$(curl_without_leading_q "$ROOT/docs/release.md")" ]] || {
  echo "docs/release.md curl invocation does not pass -q as its first argument" >&2
  exit 1
}
sed 's/curl -q /curl /' "$ROOT/README.md" >"$TMP/readme-no-q.md"
sed 's/curl -q -fL/curl -fL -q/' "$ROOT/README.md" >"$TMP/readme-q-not-first.md"
sed 's/curl -q /curl /' "$ROOT/docs/release.md" >"$TMP/release-doc-no-q.md"
for mutant in readme-no-q readme-q-not-first release-doc-no-q; do
  [[ -n "$(curl_without_leading_q "$TMP/$mutant.md")" ]] || {
    echo "curl -q check accepts mutant $mutant" >&2
    exit 1
  }
done
bad_url="$(sed -E 's/\]\(https?:[^)]*\)//g; s/(href|src)="[^"]*"//g' "$ROOT/README.md" |
  { grep -Eo 'https?://[^[:space:]"<>)]*' || true; } |
  awk -v origin="$release_origin" '$0 != origin && index($0, origin "/") != 1')"
[[ -z "$bad_url" ]] || {
  echo "README has a URL outside the release origin $release_origin (Markdown links and href/src attributes are exempt): $bad_url" >&2
  exit 1
}
[[ "$actual_bootstrap" == "$expected_bootstrap" ]] || {
  echo "README verified bootstrap fence differs from the pinned copy (< pinned, > README):" >&2
  diff <(printf '%s\n' "$expected_bootstrap") <(printf '%s\n' "$actual_bootstrap") >&2 || true
  exit 1
}
# The install.sh attestation must be one exact, fatal, pinned line that runs before the first
# live installer call; each way of weakening it must be flagged.
attest_line="  gh attestation verify install.sh --repo ViktorsBaikers/DevRites --signer-workflow ViktorsBaikers/DevRites/.github/workflows/ci.yml@refs/heads/main || { if gh auth status >/dev/null 2>&1; then echo 'error: install.sh attestation verification failed: no build provenance from ViktorsBaikers/DevRites/.github/workflows/ci.yml@refs/heads/main' >&2; else echo 'error: gh is not authenticated; run gh auth login' >&2; fi; exit 1; }"
attest_before_install() {
  printf '%s\n' "$1" | awk -v want="$2" '
    $0 == want { if (!installed) ok++; else late++ }
    /^[[:space:]]*bash \.\/install\.sh/ { installed = 1 }
    END { exit !(ok == 1 && !late) }
  '
}
attest_before_install "$actual_bootstrap" "$attest_line" || {
  echo "README bootstrap does not run the pinned fatal install.sh attestation before bash ./install.sh" >&2
  exit 1
}
attest_mutants=(
  "$(printf '%s\n' "$actual_bootstrap" | grep -Fvx "$attest_line")"
  "$(printf '%s\n' "$actual_bootstrap" | grep -Fvx "$attest_line" | awk -v l="$attest_line" '{ print } /^[[:space:]]*bash \.\/install\.sh/ { print l }')"
  "$(printf '%s\n' "$actual_bootstrap" | sed 's/ || { if gh auth status.*$/ || true/')"
  "$(printf '%s\n' "$actual_bootstrap" | sed 's/ --repo ViktorsBaikers\/DevRites//')"
  "$(printf '%s\n' "$actual_bootstrap" | sed 's/ --signer-workflow [^ ]*//')"
)
for mutant in "${attest_mutants[@]}"; do
  ! attest_before_install "$mutant" "$attest_line" || {
    echo "install.sh attestation check accepts a weakened bootstrap" >&2
    exit 1
  }
done
# The bootstrap itself must name each cause: gh absent (checked before the attestation) and gh
# logged out (distinguished from a provenance rejection), both fatal; install.sh never runs then.
gh_line="  command -v gh >/dev/null 2>&1 || { echo 'error: gh (GitHub CLI) is required and was not found' >&2; exit 1; }"
attest_before_install "$actual_bootstrap" "$gh_line" || {
  echo "README bootstrap does not name a missing gh before bash ./install.sh" >&2
  exit 1
}
[[ "$(printf '%s\n' "$actual_bootstrap" | awk -v g="$gh_line" -v a="$attest_line" '$0 == g { gi = NR } $0 == a { ai = NR } END { print (gi && ai && gi < ai) }')" == 1 ]] || {
  echo "README bootstrap does not check for gh before the attestation" >&2
  exit 1
}
for cause in "echo 'error: gh is not authenticated; run gh auth login' >&2" "gh auth status >/dev/null 2>&1" \
  "echo 'error: install.sh attestation verification failed: no build provenance from ViktorsBaikers/DevRites/.github/workflows/ci.yml@refs/heads/main' >&2"; do
  printf '%s\n' "$actual_bootstrap" | grep -Fx "$attest_line" | grep -Fq -- "$cause" || {
    echo "README bootstrap attestation failure does not name its cause: $cause" >&2
    exit 1
  }
done
cause_mutants=(
  "$(printf '%s\n' "$actual_bootstrap" | grep -Fvx "$gh_line")"
  "$(printf '%s\n' "$actual_bootstrap" | sed "s/^  command -v gh >.*$/  command -v gh >\/dev\/null 2>\&1 || true/")"
  "$(printf '%s\n' "$actual_bootstrap" | grep -Fvx "$gh_line" | awk -v l="$gh_line" '{ print } /^[[:space:]]*bash \.\/install\.sh/ { print l }')"
)
for mutant in "${cause_mutants[@]}"; do
  ! { attest_before_install "$mutant" "$gh_line" && [[ "$(printf '%s\n' "$mutant" | awk -v g="$gh_line" -v a="$attest_line" '$0 == g { gi = NR } $0 == a { ai = NR } END { print (gi && ai && gi < ai) }')" == 1 ]]; } || {
    echo "gh-missing check accepts a weakened bootstrap" >&2
    exit 1
  }
done

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

node --input-type=module - "$ROOT" <<'JS'
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
const root = process.argv[2];
const require = createRequire(`${root}/package.json`);
const config = require(`${root}/.releaserc.json`);
const { analyzeCommits } = await import(pathToFileURL(require.resolve('@semantic-release/commit-analyzer')).href);
const analyzer = config.plugins.find(([name]) => name === '@semantic-release/commit-analyzer')[1];
const expected = {
  'build(deps-dev): bump the npm group with 3 updates': null,
  'build(deps): bump js-yaml from 4.3.1 to 4.3.2': 'patch',
  'build(installer): pin go toolchain': 'patch',
  'revert(skills): undo the change\n\nThis reverts commit 22beb4b2.': 'patch',
  'revert(no-release): undo the change': null,
  'style(skills): reformat tables': null,
};
for (const [message, want] of Object.entries(expected)) {
  const got = await analyzeCommits(analyzer, { commits: [{ message, hash: 'h' }], logger: { log() {} }, cwd: root });
  if (got !== want) throw new Error(`${message.split('\n')[0]} releases ${got}, want ${want}`);
}
const npm = config.plugins.find(([name]) => name === '@semantic-release/npm')[1];
if ('provenance' in npm) throw new Error('semantic-release npm plugin has no provenance option; publishConfig.provenance owns it');
JS

# semantic-release loads plugins from node_modules/semantic-release/lib/plugins first, so the
# pinned analyzer must be the only lockfile copy that lookup can reach.
node - "$ROOT" <<'JS'
const fs = require('fs');
const root = process.argv[2];
const read = (file) => JSON.parse(fs.readFileSync(`${root}/${file}`, 'utf8'));
const pkg = read('package.json');
const packages = read('package-lock.json').packages;
const name = '@semantic-release/commit-analyzer';
const pin = pkg.devDependencies[name];
if (pkg.overrides[name] !== pin) throw new Error(`${name} override must equal the pin ${pin}`);
let dir = 'node_modules/semantic-release/lib/plugins';
let found;
for (;;) {
  found = packages[`${dir}/node_modules/${name}`];
  const cut = dir.lastIndexOf('/');
  if (found || cut === -1) break;
  dir = dir.slice(0, cut);
}
found = found || packages[`node_modules/${name}`];
if (!found || found.version !== pin) throw new Error(`release loads ${name}@${found && found.version}, not ${pin}`);
const copies = Object.keys(packages).filter((key) => key.endsWith(`node_modules/${name}`));
if (copies.length !== 1) throw new Error(`${name} has ${copies.length} lockfile copies: ${copies.join(', ')}`);
JS

echo "release-tarball-test: PASS"
