#!/usr/bin/env bash
# uninstall.sh - bootstrap shim for the engine-owned DevRites uninstaller.
set -u

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd -P)" || SELF_DIR=""
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"

if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack" ]; then
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ - aborting." >&2
    exit 1
  fi
  command -v curl >/dev/null 2>&1 || { echo "error: curl is required for the network uninstaller." >&2; exit 1; }
  if [ -n "$DEVRITES_REF" ]; then
    ref="$DEVRITES_REF"
  else
    ref="$(curl -fsSL "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" 2>/dev/null | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  ref="${ref:-main}"
  url="https://raw.githubusercontent.com/$DEVRITES_REPO/$ref/install.sh"
  tmp="$(mktemp 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-install.$$")"
  curl -fsSL -o "$tmp" "$url" || { echo "error: could not download DevRites bootstrap from $url" >&2; rm -f "$tmp"; exit 1; }
  bash "$tmp" uninstall "$@"
  rc="$?"
  rm -f "$tmp"
  exit "$rc"
fi

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || { echo "error: extracted bundle is missing scripts/install-lib.sh" >&2; exit 1; }
. "$INSTALL_LIB"

ENGINE="$(dr_acquire_engine "$SELF_DIR" uninstall "$DEVRITES_REPO")" || { echo "error: could not acquire devrites-engine." >&2; exit 1; }
export DEVRITES_ENGINE_CLI="$ENGINE"
exec "$ENGINE" uninstall --source-dir "$SELF_DIR" "$@"
