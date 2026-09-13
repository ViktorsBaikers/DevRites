#!/usr/bin/env bash
# Install pinned CI validate tools from release binaries (actionlint, osv-scanner)
# and pip (zizmor). Prefer binaries over `go install` so cold CI does not compile
# the scanners from source (~70s on ubuntu-24.04-arm).
set -euo pipefail

ACTIONLINT_VERSION="${ACTIONLINT_VERSION:-1.7.10}"
OSV_SCANNER_VERSION="${OSV_SCANNER_VERSION:-2.5.1}"
ZIZMOR_VERSION="${ZIZMOR_VERSION:-1.30.1}"

DEST="${CI_VALIDATE_TOOLS_DIR:-$HOME/.local/devrites-ci-tools}"
mkdir -p "$DEST/bin"
export PATH="$DEST/bin:$PATH"

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) goarch=amd64 ;;
  aarch64|arm64) goarch=arm64 ;;
  *)
    echo "unsupported arch: $arch" >&2
    exit 1
    ;;
esac

os="$(uname -s)"
case "$os" in
  Linux) goos=linux ;;
  Darwin) goos=darwin ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

need_actionlint=1
need_osv=1
if [[ -x "$DEST/bin/actionlint" ]] && "$DEST/bin/actionlint" -version 2>/dev/null | grep -q "$ACTIONLINT_VERSION"; then
  need_actionlint=0
fi
if [[ -x "$DEST/bin/osv-scanner" ]] && "$DEST/bin/osv-scanner" --version >/dev/null 2>&1; then
  need_osv=0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

if [[ "$need_actionlint" -eq 1 ]]; then
  base="actionlint_${ACTIONLINT_VERSION}_${goos}_${goarch}"
  url="https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/${base}.tar.gz"
  sum_url="https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION}_checksums.txt"
  curl -fsSL "$url" -o "$tmpdir/actionlint.tgz"
  curl -fsSL "$sum_url" -o "$tmpdir/actionlint.sums"
  expected="$(awk -v f="${base}.tar.gz" '$2 == f { print $1; exit }' "$tmpdir/actionlint.sums")"
  [[ -n "$expected" ]] || { echo "missing checksum for ${base}.tar.gz" >&2; exit 1; }
  echo "${expected}  $tmpdir/actionlint.tgz" | shasum -a 256 -c -
  tar -xzf "$tmpdir/actionlint.tgz" -C "$tmpdir" actionlint
  install -m 0755 "$tmpdir/actionlint" "$DEST/bin/actionlint"
fi

if [[ "$need_osv" -eq 1 ]]; then
  url="https://github.com/google/osv-scanner/releases/download/v${OSV_SCANNER_VERSION}/osv-scanner_${goos}_${goarch}"
  curl -fsSL "$url" -o "$tmpdir/osv-scanner"
  install -m 0755 "$tmpdir/osv-scanner" "$DEST/bin/osv-scanner"
fi

venv="$DEST/venv"
if [[ ! -x "$venv/bin/zizmor" ]]; then
  python3 -m venv "$venv"
  "$venv/bin/python" -m pip install --disable-pip-version-check --upgrade pip
  "$venv/bin/python" -m pip install --disable-pip-version-check "zizmor==${ZIZMOR_VERSION}"
fi
# Prefer the venv zizmor on PATH without shadowing system python.
ln -sfn "$venv/bin/zizmor" "$DEST/bin/zizmor"
export PATH="$DEST/bin:$PATH"

command -v actionlint
command -v osv-scanner
command -v zizmor
actionlint -version
osv-scanner --version
zizmor --version

# Persist PATH for later steps when GITHUB_PATH is available.
if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "$DEST/bin" >>"$GITHUB_PATH"
fi
