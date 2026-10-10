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
  "$ROOT/pack/"* | "$ROOT/engine/"* | "$ROOT/scripts/"* | "$ROOT/docs/"*)
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

# Materialize blobs from the index through one `cat-file --batch` process;
# like `cat-file blob`, --batch applies no smudge or eol filters.
WRITE_BLOBS=$(cat <<'PY'
import os, subprocess, sys
stage = sys.argv[1]
fields = sys.stdin.buffer.read().split(b"\0")[:-1]
with subprocess.Popen(
    ["git", "--no-replace-objects", "cat-file", "--batch"],
    stdin=subprocess.PIPE, stdout=subprocess.PIPE,
) as batch:
    for i in range(0, len(fields), 3):
        mode, obj, path = fields[i:i + 3]
        batch.stdin.write(obj + b"\n")
        batch.stdin.flush()
        header = batch.stdout.readline().split()
        if len(header) != 3 or header[1] != b"blob":
            sys.exit("error: release payload object is not a readable blob: " + path.decode(errors="replace"))
        size = int(header[2])
        data = batch.stdout.read(size)
        if len(data) != size or batch.stdout.read(1) != b"\n":
            sys.exit("error: git cat-file ended before the release payload object was fully read: " + path.decode(errors="replace"))
        dest = os.path.join(stage, os.fsdecode(path))
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        with open(dest, "wb") as f:
            f.write(data)
        os.chmod(dest, int(mode, 8))
    batch.stdin.close()
    if batch.wait() != 0:
        sys.exit("error: git cat-file --batch exited with status %d" % batch.returncode)
PY
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
      printf '%s\0%s\0%s\0' "$permissions" "$object" "$path"
    done \
  | python3 -c "$WRITE_BLOBS" "$STAGE"

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
