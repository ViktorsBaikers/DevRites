#!/usr/bin/env bash
# update.sh - acquire a release locally, then hand it to the offline updater.
set -u

SELF_DIR=""
SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
if [ -n "$SCRIPT_SOURCE" ] && [ -f "$SCRIPT_SOURCE" ]; then
  SELF_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" 2>/dev/null && pwd -P)" || SELF_DIR=""
fi
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"

delegate_bootstrap() {
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ - aborting to avoid a loop." >&2
    exit 1
  fi
  [ -n "$SELF_DIR" ] && [ -f "$SELF_DIR/install.sh" ] || {
    echo "error: update requires the canonical install bootstrap; run the documented curl command for install.sh." >&2
    exit 1
  }
  bash "$SELF_DIR/install.sh" update "$@"
  exit "$?"
}

if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack" ]; then
  delegate_bootstrap "$@"
fi

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || { echo "error: extracted bundle is missing scripts/install-lib.sh" >&2; exit 1; }
. "$INSTALL_LIB"

DR_ENGINE_PATH=""; DR_ENGINE_TMP=""
trap dr_cleanup_engine EXIT
trap 'exit 1' HUP INT TERM
dr_acquire_engine "$SELF_DIR" update "$DEVRITES_REPO" || { echo "error: could not acquire a version-compatible devrites-engine${DR_ACQUIRE_FAILURE:+ ($DR_ACQUIRE_FAILURE)}." >&2; exit 1; }
ENGINE="$DR_ENGINE_PATH"
PAYLOAD="${DEVRITES_HOST_ARTIFACT_DIR:-$SELF_DIR/pack/generated}"
if [ ! -d "$PAYLOAD/claude/skills" ] || [ ! -d "$PAYLOAD/codex/skills" ] || [ ! -f "$PAYLOAD/codex/config.toml" ]; then
  BUILDER="$SELF_DIR/scripts/build-host-artifacts.sh"
  [ -f "$BUILDER" ] || { echo "error: generated update payload missing at $PAYLOAD and builder missing at $BUILDER" >&2; exit 1; }
  DEVRITES_HOST_ARTIFACT_DIR="$PAYLOAD" bash "$BUILDER" >/dev/null || { echo "error: could not generate update payload at $PAYLOAD" >&2; exit 1; }
fi
export DEVRITES_ENGINE_CLI="$ENGINE"
"$ENGINE" update --source-dir "$SELF_DIR" --payload-dir "$PAYLOAD" "$@"
rc="$?"
exit "$rc"
