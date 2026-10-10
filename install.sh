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
RUN_JOB=""
RUN_WATCH=""
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
    "$url" 2>/dev/null | head -c "$((limit + 1))" >"$out.part"
  pipeline_status=("${PIPESTATUS[@]}")
  curl_status="${pipeline_status[0]}"
  head_status="${pipeline_status[1]}"
  bytes="$(wc -c <"$out.part" 2>/dev/null)" || {
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
  mv "$out.part" "$out" || {
    DOWNLOAD_FAILURE="local write"
    rm -f "$out.part"
    return 1
  }
}

bounded_decompress() {
  archive="$1"
  out="$2"
  limit="$3"
  DECOMPRESS_FAILURE=""
  rm -f "$out" "$out.part"
  gzip -dc "$archive" 2>/dev/null | head -c "$((limit + 1))" >"$out.part"
  pipeline_status=("${PIPESTATUS[@]}")
  gzip_status="${pipeline_status[0]}"
  head_status="${pipeline_status[1]}"
  bytes="$(wc -c <"$out.part" 2>/dev/null)" || {
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
  mv "$out.part" "$out" || {
    DECOMPRESS_FAILURE="could not write bounded archive"
    rm -f "$out.part"
    return 1
  }
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

# Runs a command for at most $1 seconds with output discarded. Past the limit it
# is sent SIGTERM and, if it is still alive two seconds later, SIGKILL. A command
# stopped this way sets RUN_TIMED_OUT=1 and returns 124; any other failure keeps
# the command's own status, so a crash is never mistaken for a timeout. Same
# watchdog idea as scripts/install-lib.sh dr_run_bounded, which is not available
# until the bundle is verified and extracted. The timeout marker lives in
# $BOOTSTRAP_DIR, which the EXIT trap removes, so a run without that directory
# fails closed; stop_bounded ends a command and watchdog still in flight when the
# bootstrap is interrupted.
run_bounded() {
  limit="$1"; shift
  RUN_TIMED_OUT=0
  [ -n "$BOOTSTRAP_DIR" ] || return 125
  fired="$(mktemp "$BOOTSTRAP_DIR/bound.XXXXXX" 2>/dev/null)" || return 125
  "$@" >/dev/null 2>&1 </dev/null &
  job=$!
  RUN_JOB=$job
  {
    trap 'kill "$nap"; exit 0' TERM
    sleep "$limit" & nap=$!; wait "$nap"
    kill -0 "$job" || exit 0
    echo 1 >"$fired"
    kill "$job"
    sleep 2 & nap=$!; wait "$nap"
    kill -9 "$job"
  } >/dev/null 2>&1 </dev/null &
  watch=$!
  RUN_WATCH=$watch
  wait "$job" 2>/dev/null
  rc=$?
  kill "$watch" >/dev/null 2>&1
  wait "$watch" 2>/dev/null
  RUN_JOB=""; RUN_WATCH=""
  if [ "$rc" -ne 0 ] && [ -s "$fired" ]; then
    RUN_TIMED_OUT=1
    rc=124
  fi
  rm -f "$fired"
  return "$rc"
}

stop_bounded() {
  # A signal can land after a job is forked but before its PID is recorded, so
  # every background job is ended, not only the recorded ones. run_bounded is the
  # only place this script backgrounds anything.
  for pid in $(jobs -p); do
    [ "$pid" = "$RUN_WATCH" ] || kill -9 "$pid" >/dev/null 2>&1
  done
  [ -z "$RUN_WATCH" ] || kill "$RUN_WATCH" >/dev/null 2>&1
  RUN_JOB=""; RUN_WATCH=""
}

# The checksum sidecar above travels with the tarball it vouches for: it is
# fetched from "$url.sha256", one suffix away from the artifact's own origin. An
# attacker who substitutes the release origin therefore supplies both the
# payload and a digest that matches it, and the digest check passes. The
# release job already mints the proof that does not travel with the artifact --
# a build-provenance attestation signed by that job's OIDC identity and bound to
# the artifact digest (.github/workflows/ci.yml, 'Attest release artifacts').
# Verify it here, pinned to the repository and to the signer workflow, so an
# attestation minted by any other workflow in this repository is rejected too.
#
# There is deliberately no bypass for a missing verifier: without gh there is no
# proof of origin, and the sidecar is not a substitute for one.
#
# The lookup is bounded: DEVRITES_ATTEST_TIMEOUT is a whole number of seconds
# (1 to 99999, default 120). Anything else is ignored with a warning and the
# default applies, so a bad value can never remove the bound. The follow-up
# `gh auth status` is bounded by 30 seconds or that limit, whichever is shorter.
# A timeout, a crash and a plain rejection each fail closed and are reported as
# what they are.
verify_attestation() {
  file="$1"
  signer="$DEVRITES_REPO/.github/workflows/ci.yml@refs/heads/main"
  ATTESTATION_FAILURE=""
  if ! command -v gh >/dev/null 2>&1; then
    ATTESTATION_FAILURE="attestation verification failed: gh (GitHub CLI) is required and was not found"
    return 1
  fi
  attest_limit=120
  case "${DEVRITES_ATTEST_TIMEOUT:-}" in
  "") ;;
  *[!0-9]* | 0* | [0-9][0-9][0-9][0-9][0-9][0-9]*)
    echo "warning: ignoring invalid DEVRITES_ATTEST_TIMEOUT '$DEVRITES_ATTEST_TIMEOUT' (expected whole seconds, 1 to 99999); using $attest_limit" >&2
    ;;
  *) attest_limit="$DEVRITES_ATTEST_TIMEOUT" ;;
  esac
  auth_limit=30
  [ "$attest_limit" -lt "$auth_limit" ] && auth_limit="$attest_limit"
  verify_rc=0
  run_bounded "$attest_limit" gh attestation verify "$file" --repo "$DEVRITES_REPO" \
    --signer-workflow "$signer" || verify_rc=$?
  if [ "$verify_rc" -ne 0 ]; then
    if [ "$RUN_TIMED_OUT" -eq 1 ]; then
      ATTESTATION_FAILURE="attestation verification timed out after ${attest_limit}s; check connectivity or raise DEVRITES_ATTEST_TIMEOUT and retry"
    elif [ "$verify_rc" -ge 128 ]; then
      ATTESTATION_FAILURE="attestation verification failed: gh exited with status $verify_rc"
    elif run_bounded "$auth_limit" gh auth status; then
      ATTESTATION_FAILURE="attestation verification failed: no build provenance from $signer"
    elif [ "$RUN_TIMED_OUT" -eq 1 ]; then
      ATTESTATION_FAILURE="attestation verification timed out: gh auth status did not answer within ${auth_limit}s; check connectivity or raise DEVRITES_ATTEST_TIMEOUT and retry"
    else
      ATTESTATION_FAILURE="attestation verification failed: gh is not authenticated (attestation lookup requires a GitHub login); run 'gh auth login' and retry"
    fi
    return 1
  fi
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
  install | update | uninstall)
    script="$1"
    shift
    ;;
  esac
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ - aborting to avoid a loop." >&2
    exit 1
  fi
  command -v curl >/dev/null 2>&1 || {
    echo "error: curl is required for the network installer." >&2
    exit 1
  }
  command -v gzip >/dev/null 2>&1 || {
    echo "error: gzip is required for the network installer." >&2
    exit 1
  }
  command -v tar >/dev/null 2>&1 || {
    echo "error: tar is required for the network installer." >&2
    exit 1
  }
  valid_repo "$DEVRITES_REPO" || {
    echo "error: DEVRITES_REPO must be an owner/repository name." >&2
    exit 1
  }
  old_umask="$(umask)"
  umask 077
  BOOTSTRAP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/devrites-bootstrap.XXXXXX" 2>/dev/null)" || {
    umask "$old_umask"
    echo "error: could not create a private bootstrap directory." >&2
    exit 1
  }
  umask "$old_umask"
  trap 'stop_bounded; exit 1' HUP INT TERM
  trap 'rm -rf "$BOOTSTRAP_DIR"' EXIT
  if [ -n "$DEVRITES_REF" ]; then
    tag="$(normalize_release_tag "$DEVRITES_REF")" || {
      echo "error: DEVRITES_REF must be an exact semantic version." >&2
      exit 1
    }
  else
    metadata="$BOOTSTRAP_DIR/latest.json"
    bounded_curl "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" "$metadata" "$BOOTSTRAP_MAX_METADATA" 30 || {
      echo "error: latest release metadata for $DEVRITES_REPO: ${DOWNLOAD_FAILURE:-download} failed." >&2
      exit 1
    }
    metadata_tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$metadata" | head -n1)"
    tag="$(normalize_release_tag "$metadata_tag")" || {
      echo "error: latest release did not provide an exact semantic version." >&2
      exit 1
    }
  fi
  asset="devrites-$tag.tar.gz"
  archive="$BOOTSTRAP_DIR/$asset"
  sidecar="$archive.sha256"
  url="https://github.com/$DEVRITES_REPO/releases/download/$tag/$asset"
  bounded_curl "$url" "$archive" "$BOOTSTRAP_MAX_ARCHIVE" 120 || {
    echo "error: release $tag asset $asset: ${DOWNLOAD_FAILURE:-download} failed." >&2
    exit 1
  }
  bounded_curl "$url.sha256" "$sidecar" "$BOOTSTRAP_MAX_SIDECAR" 30 || {
    rm -f "$archive"
    echo "error: release $tag asset $asset.sha256: ${DOWNLOAD_FAILURE:-download} failed." >&2
    exit 1
  }
  verify_sha256 "$archive" "$sidecar" "$asset" || {
    rm -f "$archive" "$sidecar"
    echo "error: release $tag asset $asset: checksum failed." >&2
    exit 1
  }
  verify_attestation "$archive" || {
    rm -f "$archive" "$sidecar"
    echo "error: release $tag asset $asset: ${ATTESTATION_FAILURE:-attestation verification failed}." >&2
    exit 1
  }
  uncompressed="$BOOTSTRAP_DIR/devrites-$tag.tar"
  bounded_decompress "$archive" "$uncompressed" "$BOOTSTRAP_MAX_UNCOMPRESSED" || {
    rm -f "$archive" "$sidecar"
    echo "error: release $tag asset $asset: ${DECOMPRESS_FAILURE:-decompression failed}." >&2
    exit 1
  }
  preflight_archive "$uncompressed" "$tag" || {
    rm -f "$archive" "$sidecar" "$uncompressed"
    echo "error: release $tag asset $asset: archive preflight failed." >&2
    exit 1
  }
  extract="$BOOTSTRAP_DIR/extract"
  mkdir "$extract" || exit 1
  tar -C "$extract" -xf "$uncompressed" || {
    rm -rf "$extract"
    echo "error: could not extract bounded DevRites tarball" >&2
    exit 1
  }
  bundle="$extract/devrites-$tag"
  [ -f "$bundle/$script.sh" ] || {
    rm -rf "$extract"
    echo "error: extracted bundle is missing $script.sh" >&2
    exit 1
  }
  chmod +x "$bundle/install.sh" "$bundle/uninstall.sh" "$bundle/update.sh" 2>/dev/null || true
  echo "DevRites: bootstrapped from $tag"
  export DEVRITES_BOOTSTRAPPED=1
  bash "$bundle/$script.sh" "$@"
  rc="$?"
  exit "$rc"
}

SUBCOMMAND="install"
case "${1:-}" in
install | update | uninstall)
  SUBCOMMAND="$1"
  shift
  ;;
esac

if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack" ]; then
  bootstrap_bundle "$SUBCOMMAND" "$@"
fi

case "$SUBCOMMAND" in
update | uninstall)
  exec bash "$SELF_DIR/$SUBCOMMAND.sh" "$@"
  ;;
esac

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || {
  echo "error: extracted bundle is missing scripts/install-lib.sh" >&2
  exit 1
}
. "$INSTALL_LIB"

DR_ENGINE_PATH=""
DR_ENGINE_TMP=""
trap dr_cleanup_engine EXIT
trap 'exit 1' HUP INT TERM
dr_acquire_engine "$SELF_DIR" install "$DEVRITES_REPO" || {
  echo "error: could not acquire devrites-engine (no usable installed binary and no matching verified release binary${DR_ACQUIRE_FAILURE:+; $DR_ACQUIRE_FAILURE})." >&2
  exit 1
}
ENGINE="$DR_ENGINE_PATH"
PAYLOAD="${DEVRITES_HOST_ARTIFACT_DIR:-$SELF_DIR/pack/generated}"
if ! dr_payload_complete "$PAYLOAD"; then
  BUILDER="$SELF_DIR/scripts/build-host-artifacts.sh"
  [ -f "$BUILDER" ] || {
    echo "error: generated install payload missing at $PAYLOAD and builder missing at $BUILDER" >&2
    exit 1
  }
  DEVRITES_HOST_ARTIFACT_DIR="$PAYLOAD" bash "$BUILDER" >/dev/null || {
    echo "error: could not generate install payload at $PAYLOAD" >&2
    exit 1
  }
fi
export DEVRITES_ENGINE_CLI="$ENGINE"
"$ENGINE" install --source-dir "$SELF_DIR" --payload-dir "$PAYLOAD" "$@"
rc="$?"
exit "$rc"
