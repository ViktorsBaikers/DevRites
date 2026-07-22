#!/usr/bin/env bash
# Build the DevRites release tarball: the artifact attached to the GitHub Release
# by semantic-release. Extracting yields a `devrites-v<version>/` directory with
# everything an end-user needs (pack/, install.sh, uninstall.sh, scripts/, docs).
#
# Usage: build-release-tarball.sh <version>
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: build-release-tarball.sh <version>" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="${DEVRITES_RELEASE_DIST_DIR:-$ROOT/dist}"
NAME="devrites-v${VERSION}"
STAGE="$DIST/$NAME"

cd "$ROOT"

echo "Building release tarball: ${NAME}.tar.gz"

rm -rf "$STAGE"
mkdir -p "$STAGE"

# Files and directories shipped to end-users.
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
    if [[ "$item" == "engine" ]] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      while IFS= read -r -d '' path; do
        if [[ "$path" == engine/testdata/golden/* ]]; then
          continue
        fi
        mkdir -p "$STAGE/$(dirname "$path")"
        cp "$path" "$STAGE/$path"
      done < <(git ls-files -z -- "$item")
    else
      cp -R "$item" "$STAGE/"
    fi
  fi
done

# Ship the same prebuilt host-native artifacts that npm pack includes.
DEVRITES_HOST_ARTIFACT_DIR="$STAGE/pack/generated" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null

# Drop dev-only artifacts that may have been copied transitively.
rm -rf "$STAGE/docs/internal" "$STAGE/scripts/.cache" 2>/dev/null || true

tar -C "$DIST" -czf "$DIST/${NAME}.tar.gz" "$NAME"
rm -rf "$STAGE"

# Emit a sibling checksum so install.sh can verify the artifact when present.
# Write just "<sha256>  <filename>" (no path) so it verifies from any cwd.
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
