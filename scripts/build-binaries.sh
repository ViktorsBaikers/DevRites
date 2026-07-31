#!/usr/bin/env bash
# Cross-compile devrites-engine for every supported target and emit a sha256
# sidecar per binary: the artifacts attached to the GitHub Release alongside the
# pack tarball. Pure-Go (CGO_ENABLED=0) so every target builds
# from one Linux runner with no cross-toolchain. The version is stamped into the
# binary via -ldflags so `devrites-engine version` reports the release it came from.
#
# Usage: build-binaries.sh <version>          # version WITHOUT a leading "v"
#
# Output: dist/bin/devrites-<os>-<arch>[.exe] plus a "<sha>  <name>" .sha256 each.
# Binary names are UNVERSIONED so the installer can fetch
# github.com/<repo>/releases/latest/download/devrites-<os>-<arch> directly.
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: build-binaries.sh <version>" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENGINE="$ROOT/engine"
OUT="$ROOT/dist/bin"
MODULE="github.com/devrites/devrites"

if [[ ! -d "$ENGINE" ]]; then
  echo "build-binaries: no engine/ directory at $ENGINE" >&2
  exit 1
fi

# The release matrix: darwin arm64/amd64, linux amd64/arm64, windows amd64.
TARGETS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

rm -rf "$OUT"
mkdir -p "$OUT"

sha256_of() {
  # Emit "<sha256>  <filename>" (no path) so it verifies from any cwd.
  local f="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$f"
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$f"
  else
    echo "build-binaries: no sha256 tool found" >&2
    return 1
  fi
}

echo "Cross-compiling devrites v${VERSION} (CGO_ENABLED=0)"
cd "$ENGINE"

for target in "${TARGETS[@]}"; do
  os="${target%%/*}"
  arch="${target##*/}"
  name="devrites-${os}-${arch}"
  [[ "$os" == "windows" ]] && name="${name}.exe"

  echo "  → ${name}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath \
      -ldflags "-s -w -X ${MODULE}/internal/version.Version=v${VERSION}" \
      -o "$OUT/$name" .

  ( cd "$OUT" && sha256_of "$name" > "${name}.sha256" )
done

echo "Built $(find "$OUT" -maxdepth 1 -name 'devrites-*' ! -name '*.sha256' | wc -l | tr -d ' ') binaries in $OUT"
ls -lh "$OUT"
