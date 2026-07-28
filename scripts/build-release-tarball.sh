#!/usr/bin/env bash
# Build the archive that semantic-release attaches to a GitHub Release.
# It extracts to `devrites-v<version>/` with the pack, engine, scripts,
# documentation, and install tools.
#
# Usage: build-release-tarball.sh <version>
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: build-release-tarball.sh <version>" >&2
  exit 1
fi
if [[ ! "$VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z._+-]*$ ]]; then
  echo "error: version must be a portable release asset name" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
DIST="${DEVRITES_RELEASE_DIST_DIR:-$ROOT/dist}"
NAME="devrites-v${VERSION}"

cd "$ROOT"
mkdir -p "$DIST"
DIST="$(cd "$DIST" && pwd -P)"
case "$DIST/" in
  "$ROOT/pack/"* | "$ROOT/engine/"* | "$ROOT/scripts/"* | "$ROOT/mcp/"* | "$ROOT/docs/"*)
    echo "error: release output directory overlaps the release payload" >&2
    exit 1
    ;;
esac
STAGE="$(mktemp -d "$DIST/.devrites-release-stage.XXXXXX")"
cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT

echo "Building release tarball: ${NAME}.tar.gz"

# Release contents.
PAYLOAD=(
  pack
  engine
  scripts
  mcp
  docs
  install.sh
  uninstall.sh
  update.sh
  README.md
  CHANGELOG.md
  LICENSE
  SECURITY.md
  NOTICE.md
  CODE_OF_CONDUCT.md
  CODEOWNERS
  package.json
)

for item in "${PAYLOAD[@]}"; do
  if [[ -e "$item" ]]; then
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      while IFS= read -r -d '' path; do
        [[ -e "$path" || -L "$path" ]] || continue
        if [[ "$path" == engine/testdata/golden/* ]]; then
          continue
        fi
        mkdir -p "$STAGE/$(dirname "$path")"
        cp -P "$path" "$STAGE/$path"
      done < <(git ls-files -z -- "$item")
    else
      cp -R "$item" "$STAGE/"
    fi
  fi
done

# Include the same prebuilt host artifacts as the npm package.
DEVRITES_HOST_ARTIFACT_DIR="$STAGE/pack/generated" bash "$STAGE/scripts/build-host-artifacts.sh" >/dev/null

# Remove development files copied with the payload.
rm -rf "$STAGE/docs/internal" "$STAGE/scripts/.cache" 2>/dev/null || true

(
  cd "$ROOT/engine"
  go run ./cmd/releasepack \
    -root "$STAGE" \
    -output "$DIST/${NAME}.tar.gz" \
    -prefix "$NAME" \
    -epoch "${SOURCE_DATE_EPOCH:-0}"
)

# Write a sibling checksum for install.sh to verify when available.
# Store only "<sha256>  <filename>" so verification works from any directory.
(
  cd "$DIST"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${NAME}.tar.gz" > "${NAME}.tar.gz.sha256"
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${NAME}.tar.gz" > "${NAME}.tar.gz.sha256"
  else
    echo "warning: no sha256 tool found; skipping ${NAME}.tar.gz.sha256" >&2
  fi
)

echo "  → $DIST/${NAME}.tar.gz"
ls -lh "$DIST/${NAME}.tar.gz"
[[ -f "$DIST/${NAME}.tar.gz.sha256" ]] && { echo "  → $DIST/${NAME}.tar.gz.sha256"; cat "$DIST/${NAME}.tar.gz.sha256"; } || true
