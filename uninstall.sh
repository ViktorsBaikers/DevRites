#!/usr/bin/env bash
# uninstall.sh - bootstrap shim for the engine-owned DevRites uninstaller.
set -u

SELF_DIR=""
SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
if [ -n "$SCRIPT_SOURCE" ] && [ -f "$SCRIPT_SOURCE" ]; then
  SELF_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" 2>/dev/null && pwd -P)" || SELF_DIR=""
fi
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"

if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack" ]; then
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ - aborting." >&2
    exit 1
  fi
  [ -n "$SELF_DIR" ] && [ -f "$SELF_DIR/install.sh" ] || {
    echo "error: uninstall requires the canonical install bootstrap; run the documented curl command for install.sh." >&2
    exit 1
  }
  bash "$SELF_DIR/install.sh" uninstall "$@"
  exit "$?"
fi

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || { echo "error: extracted bundle is missing scripts/install-lib.sh" >&2; exit 1; }
. "$INSTALL_LIB"

DR_ENGINE_PATH=""; DR_ENGINE_TMP=""
trap dr_cleanup_engine EXIT
trap 'exit 1' HUP INT TERM
dr_acquire_engine "$SELF_DIR" uninstall "$DEVRITES_REPO" || { echo "error: could not acquire devrites-engine${DR_ACQUIRE_FAILURE:+ ($DR_ACQUIRE_FAILURE)}." >&2; exit 1; }
ENGINE="$DR_ENGINE_PATH"
export DEVRITES_ENGINE_CLI="$ENGINE"
"$ENGINE" uninstall --source-dir "$SELF_DIR" "$@"
rc="$?"
exit "$rc"
