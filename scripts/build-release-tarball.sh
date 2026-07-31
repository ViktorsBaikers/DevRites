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
. "$ROOT/scripts/git-env.sh"
DIST="${DEVRITES_RELEASE_DIST_DIR:-$ROOT/dist}"
NAME="devrites-v${VERSION}"

cd "$ROOT"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "error: release payload requires a Git index at $ROOT" >&2
  exit 1
}
REPO_ROOT="$(cd "$REPO_ROOT" && pwd -P)"
if [[ "$REPO_ROOT" != "$ROOT" ]]; then
  echo "error: release payload requires the Git index rooted at $ROOT" >&2
  exit 1
fi
mkdir -p "$DIST"
DIST="$(cd "$DIST" && pwd -P)"
case "$DIST/" in
  "$ROOT/pack/"* | "$ROOT/engine/"* | "$ROOT/scripts/"* | "$ROOT/mcp/"* | "$ROOT/docs/"*)
    echo "error: release output directory overlaps the release payload" >&2
    exit 1
    ;;
esac
STAGE="$(mktemp -d "$DIST/.devrites-release-stage.XXXXXX")"
ARCHIVE="$DIST/${NAME}.tar.gz"
SIDECAR="$ARCHIVE.sha256"
INSTALLER="$DIST/install.sh"
INSTALLER_SIDECAR="$DIST/install.sh.sha256"
SUCCESS=0
rm -f "$ARCHIVE" "$SIDECAR" "$INSTALLER" "$INSTALLER_SIDECAR"
cleanup() {
  rm -rf "$STAGE"
  if [[ "$SUCCESS" -ne 1 ]]; then
    rm -f "$ARCHIVE" "$SIDECAR" "$INSTALLER" "$INSTALLER_SIDECAR"
  fi
}
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

git ls-files --stage -z -- "${PAYLOAD[@]}" \
  | while IFS= read -r -d '' entry; do
      metadata="${entry%%$'\t'*}"
      path="${entry#*$'\t'}"
      if [[ "$metadata" == "$entry" ]]; then
        echo "error: malformed Git index entry" >&2
        exit 1
      fi
      case "$path" in
        engine/testdata/golden/* | docs/internal/* | scripts/.cache/*) continue ;;
      esac
      read -r mode object stage extra <<< "$metadata"
      if [[ "$stage" != 0 || -n "${extra:-}" ]]; then
        echo "error: release payload requires a stage-0 Git index entry: $path" >&2
        exit 1
      fi
      case "$mode" in
        100644) permissions=0644 ;;
        100755) permissions=0755 ;;
        120000)
          echo "error: release payload symlink is not allowed: $path" >&2
          exit 1
          ;;
        *)
          echo "error: release payload has unsupported Git index mode $mode: $path" >&2
          exit 1
          ;;
      esac
      destination="$STAGE/$path"
      mkdir -p "$(dirname "$destination")"
      git --no-replace-objects cat-file blob "$object" > "$destination"
      chmod "$permissions" "$destination"
    done

[[ -f "$STAGE/install.sh" ]] || {
  echo "error: Git index release payload is missing install.sh" >&2
  exit 1
}

(
  cd "$STAGE/engine"
  go run ./cmd/releasepack \
    -root "$STAGE" \
    -output "$ARCHIVE" \
    -prefix "$NAME" \
    -epoch "${SOURCE_DATE_EPOCH:-0}"
)

cp "$STAGE/install.sh" "$INSTALLER"
chmod 0755 "$INSTALLER"

# Write mandatory sidecars as "<sha256>  <filename>" records.
(
  cd "$DIST"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${NAME}.tar.gz" > "${NAME}.tar.gz.sha256" || { rm -f "$ARCHIVE" "$SIDECAR"; exit 1; }
    shasum -a 256 install.sh > install.sh.sha256 || { rm -f "$ARCHIVE" "$SIDECAR" "$INSTALLER" "$INSTALLER_SIDECAR"; exit 1; }
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${NAME}.tar.gz" > "${NAME}.tar.gz.sha256" || { rm -f "$ARCHIVE" "$SIDECAR"; exit 1; }
    sha256sum install.sh > install.sh.sha256 || { rm -f "$ARCHIVE" "$SIDECAR" "$INSTALLER" "$INSTALLER_SIDECAR"; exit 1; }
  else
    rm -f "$ARCHIVE" "$SIDECAR" "$INSTALLER" "$INSTALLER_SIDECAR"
    echo "error: no SHA-256 tool found; release checksum is mandatory" >&2
    exit 1
  fi
)

SUCCESS=1

echo "  → $ARCHIVE"
ls -lh "$ARCHIVE"
echo "  → $SIDECAR"
cat "$SIDECAR"
echo "  → $INSTALLER"
echo "  → $INSTALLER_SIDECAR"
cat "$INSTALLER_SIDECAR"
