#!/usr/bin/env bash
# install.sh - bootstrap shim for the engine-owned DevRites installer.
# GUARD:no-global - project-local agent files are installed only under --target;
# the only sanctioned global write is the devrites-engine binary lifecycle.
set -u

SELF_DIR=""
SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
if [ -n "$SCRIPT_SOURCE" ] && [ -f "$SCRIPT_SOURCE" ]; then
  SELF_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" 2>/dev/null && pwd -P)" || SELF_DIR=""
fi
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"

BOOTSTRAP_DIR=""
BOOTSTRAP_MAX_METADATA=1048576
BOOTSTRAP_MAX_SIDECAR=4096
BOOTSTRAP_MAX_ARCHIVE=67108864
BOOTSTRAP_MAX_UNCOMPRESSED=335544320
BOOTSTRAP_MAX_EXPANDED=268435456

valid_repo() {
  printf '%s\n' "$1" | LC_ALL=C grep -Eq '^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$'
}

normalize_release_tag() {
  version="${1#v}"
  printf '%s\n' "$version" | LC_ALL=C grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' || return 1
  prerelease="${version%%+*}"
  case "$prerelease" in
    *-*) prerelease="${prerelease#*-}" ;;
    *) prerelease="" ;;
  esac
  if [ -n "$prerelease" ]; then
    printf '%s\n' "$prerelease" | awk -F. '{ for (i = 1; i <= NF; i++) if ($i ~ /^[0-9]+$/ && length($i) > 1 && substr($i, 1, 1) == "0") exit 1 }' || return 1
  fi
  printf 'v%s\n' "$version"
}

bounded_curl() {
  url="$1"
  out="$2"
  limit="$3"
  seconds="$4"
  DOWNLOAD_FAILURE=""
  rm -f "$out" "$out.part"
  curl -fL --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time "$seconds" \
    "$url" 2>/dev/null | head -c "$((limit + 1))" > "$out.part"
  pipeline_status=("${PIPESTATUS[@]}")
  curl_status="${pipeline_status[0]}"
  head_status="${pipeline_status[1]}"
  bytes="$(wc -c < "$out.part" 2>/dev/null)" || {
    DOWNLOAD_FAILURE="local write"
    rm -f "$out.part"
    return 1
  }
  if [ "$bytes" -gt "$limit" ]; then
    DOWNLOAD_FAILURE="size limit"
    rm -f "$out.part"
    return 1
  fi
  if [ "$head_status" -ne 0 ]; then
    DOWNLOAD_FAILURE="local write"
    rm -f "$out.part"
    return 1
  fi
  if [ "$curl_status" -ne 0 ]; then
    case "$curl_status" in
      22) DOWNLOAD_FAILURE="HTTP status" ;;
      28) DOWNLOAD_FAILURE="timeout" ;;
      47) DOWNLOAD_FAILURE="redirect" ;;
      *) DOWNLOAD_FAILURE="download" ;;
    esac
    rm -f "$out.part"
    return 1
  fi
  mv "$out.part" "$out" || { DOWNLOAD_FAILURE="local write"; rm -f "$out.part"; return 1; }
}

bounded_decompress() {
  archive="$1"
  out="$2"
  limit="$3"
  DECOMPRESS_FAILURE=""
  rm -f "$out" "$out.part"
  gzip -dc "$archive" 2>/dev/null | head -c "$((limit + 1))" > "$out.part"
  pipeline_status=("${PIPESTATUS[@]}")
  gzip_status="${pipeline_status[0]}"
  head_status="${pipeline_status[1]}"
  bytes="$(wc -c < "$out.part" 2>/dev/null)" || {
    DECOMPRESS_FAILURE="could not write bounded archive"
    rm -f "$out.part"
    return 1
  }
  if [ "$bytes" -gt "$limit" ]; then
    DECOMPRESS_FAILURE="decompressed archive exceeds $((limit / 1048576)) MiB limit"
    rm -f "$out.part"
    return 1
  fi
  if [ "$head_status" -ne 0 ]; then
    DECOMPRESS_FAILURE="could not write bounded archive"
    rm -f "$out.part"
    return 1
  fi
  if [ "$gzip_status" -ne 0 ]; then
    DECOMPRESS_FAILURE="gzip decompression failed"
    rm -f "$out.part"
    return 1
  fi
  mv "$out.part" "$out" || { DECOMPRESS_FAILURE="could not write bounded archive"; rm -f "$out.part"; return 1; }
}

verify_sha256() {
  file="$1"
  sumfile="$2"
  asset="$3"
  want="$(awk -v asset="$asset" '
    NF == 0 { next }
    { records++ }
    NF == 2 && length($1) == 64 && $1 ~ /^[0-9A-Fa-f]+$/ && $2 == asset { valid++; hash=tolower($1) }
    END { if (records == 1 && valid == 1) print hash; else exit 1 }
  ' "$sumfile" 2>/dev/null)" || return 1
  if command -v shasum >/dev/null 2>&1; then
    got="$(shasum -a 256 "$file" | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    got="$(sha256sum "$file" | awk '{print $1}')"
  else
    got=""
  fi
  [ -n "$want" ] && [ -n "$got" ] && [ "$got" = "$want" ]
}

preflight_archive() {
  archive_path="$1"
  tag="$2"
  prefix="devrites-$tag"

  LC_ALL=C tar -tvf "$archive_path" 2>/dev/null | LC_ALL=C awk -v max="$BOOTSTRAP_MAX_EXPANDED" '
    {
      count++
      if (count > 10000) exit 1
      type=substr($1, 1, 1)
      if (type != "-" && type != "d") exit 1
      if (type == "-") {
        size=($2 ~ /^[0-9]+$/ ? $5 : $3)
        if (size !~ /^[0-9]+$/) exit 1
        total += size
        if (total > max) exit 1
      }
    }
    END { if (count == 0) exit 1 }
  '
  metadata_status=("${PIPESTATUS[@]}")
  [ "${metadata_status[0]}" -eq 0 ] && [ "${metadata_status[1]}" -eq 0 ] || return 1

  LC_ALL=C tar -tf "$archive_path" 2>/dev/null | LC_ALL=C awk -v prefix="$prefix" '
    length($0) > 4096 || $0 == "" || $0 ~ /^\// || $0 ~ /\\/ || $0 ~ /[[:cntrl:]]/ { exit 1 }
    {
      count++
      if (count > 10000) exit 1
      name=$0
      while (length(name) > 1 && substr(name, length(name), 1) == "/") name=substr(name, 1, length(name)-1)
      if (name == prefix) roots++
      else if (index(name, prefix "/") != 1) exit 1
      if (name ~ /(^|\/)\.{1,2}(\/|$)/ || name ~ /\/\// || seen[name]++) exit 1
    }
    END { if (roots != 1) exit 1 }
  '
  path_status=("${PIPESTATUS[@]}")
  [ "${path_status[0]}" -eq 0 ] && [ "${path_status[1]}" -eq 0 ]
}

bootstrap_bundle() {
  script="install"
  case "${1:-}" in
    install|update|uninstall)
      script="$1"
      shift
      ;;
  esac
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ - aborting to avoid a loop." >&2
    exit 1
  fi
  command -v curl >/dev/null 2>&1 || { echo "error: curl is required for the network installer." >&2; exit 1; }
  command -v gzip >/dev/null 2>&1 || { echo "error: gzip is required for the network installer." >&2; exit 1; }
  command -v tar >/dev/null 2>&1 || { echo "error: tar is required for the network installer." >&2; exit 1; }
  valid_repo "$DEVRITES_REPO" || { echo "error: DEVRITES_REPO must be an owner/repository name." >&2; exit 1; }
  old_umask="$(umask)"
  umask 077
  BOOTSTRAP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/devrites-bootstrap.XXXXXX" 2>/dev/null)" || {
    umask "$old_umask"
    echo "error: could not create a private bootstrap directory." >&2
    exit 1
  }
  umask "$old_umask"
  trap 'exit 1' HUP INT TERM
  trap 'rm -rf "$BOOTSTRAP_DIR"' EXIT
  if [ -n "$DEVRITES_REF" ]; then
    tag="$(normalize_release_tag "$DEVRITES_REF")" || { echo "error: DEVRITES_REF must be an exact semantic version." >&2; exit 1; }
  else
    metadata="$BOOTSTRAP_DIR/latest.json"
    bounded_curl "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" "$metadata" "$BOOTSTRAP_MAX_METADATA" 30 || {
      echo "error: latest release metadata for $DEVRITES_REPO: ${DOWNLOAD_FAILURE:-download} failed." >&2
      exit 1
    }
    metadata_tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$metadata" | head -n1)"
    tag="$(normalize_release_tag "$metadata_tag")" || { echo "error: latest release did not provide an exact semantic version." >&2; exit 1; }
  fi
  asset="devrites-$tag.tar.gz"
  archive="$BOOTSTRAP_DIR/$asset"
  sidecar="$archive.sha256"
  url="https://github.com/$DEVRITES_REPO/releases/download/$tag/$asset"
  bounded_curl "$url" "$archive" "$BOOTSTRAP_MAX_ARCHIVE" 120 || { echo "error: release $tag asset $asset: ${DOWNLOAD_FAILURE:-download} failed." >&2; exit 1; }
  bounded_curl "$url.sha256" "$sidecar" "$BOOTSTRAP_MAX_SIDECAR" 30 || { rm -f "$archive"; echo "error: release $tag asset $asset.sha256: ${DOWNLOAD_FAILURE:-download} failed." >&2; exit 1; }
  verify_sha256 "$archive" "$sidecar" "$asset" || { rm -f "$archive" "$sidecar"; echo "error: release $tag asset $asset: checksum failed." >&2; exit 1; }
  uncompressed="$BOOTSTRAP_DIR/devrites-$tag.tar"
  bounded_decompress "$archive" "$uncompressed" "$BOOTSTRAP_MAX_UNCOMPRESSED" || { rm -f "$archive" "$sidecar"; echo "error: release $tag asset $asset: ${DECOMPRESS_FAILURE:-decompression failed}." >&2; exit 1; }
  preflight_archive "$uncompressed" "$tag" || { rm -f "$archive" "$sidecar" "$uncompressed"; echo "error: release $tag asset $asset: archive preflight failed." >&2; exit 1; }
  extract="$BOOTSTRAP_DIR/extract"
  mkdir "$extract" || exit 1
  tar -C "$extract" -xf "$uncompressed" || { rm -rf "$extract"; echo "error: could not extract bounded DevRites tarball" >&2; exit 1; }
  bundle="$extract/devrites-$tag"
  [ -f "$bundle/$script.sh" ] || { rm -rf "$extract"; echo "error: extracted bundle is missing $script.sh" >&2; exit 1; }
  chmod +x "$bundle/install.sh" "$bundle/uninstall.sh" "$bundle/update.sh" 2>/dev/null || true
  echo "DevRites: bootstrapped from $tag"
  export DEVRITES_BOOTSTRAPPED=1
  bash "$bundle/$script.sh" "$@"
  rc="$?"
  exit "$rc"
}

if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack" ]; then
  bootstrap_bundle "$@"
fi

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || { echo "error: extracted bundle is missing scripts/install-lib.sh" >&2; exit 1; }
. "$INSTALL_LIB"

DR_ENGINE_PATH=""; DR_ENGINE_TMP=""
trap dr_cleanup_engine EXIT
trap 'exit 1' HUP INT TERM
dr_acquire_engine "$SELF_DIR" install "$DEVRITES_REPO" || { echo "error: could not acquire devrites-engine (no usable installed binary and no matching verified release binary${DR_ACQUIRE_FAILURE:+; $DR_ACQUIRE_FAILURE})." >&2; exit 1; }
ENGINE="$DR_ENGINE_PATH"
PAYLOAD="${DEVRITES_HOST_ARTIFACT_DIR:-$SELF_DIR/pack/generated}"
if [ ! -d "$PAYLOAD/claude/skills" ] || [ ! -d "$PAYLOAD/codex/skills" ] \
  || [ ! -f "$PAYLOAD/claude/skills/devrites-lib/reference/standards/agents.md" ] \
  || [ ! -f "$PAYLOAD/codex/skills/devrites-lib/reference/standards/agents.md" ] \
  || [ ! -f "$PAYLOAD/codex/config.toml" ]; then
  BUILDER="$SELF_DIR/scripts/build-host-artifacts.sh"
  [ -f "$BUILDER" ] || { echo "error: generated install payload missing at $PAYLOAD and builder missing at $BUILDER" >&2; exit 1; }
  DEVRITES_HOST_ARTIFACT_DIR="$PAYLOAD" bash "$BUILDER" >/dev/null || { echo "error: could not generate install payload at $PAYLOAD" >&2; exit 1; }
fi
export DEVRITES_ENGINE_CLI="$ENGINE"
"$ENGINE" install --source-dir "$SELF_DIR" --payload-dir "$PAYLOAD" "$@"
rc="$?"
exit "$rc"
