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
  command -v tar >/dev/null 2>&1 || { echo "error: tar is required for the network uninstaller." >&2; exit 1; }
  tmp="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-bootstrap.$$")"
  if [ -n "$DEVRITES_REF" ]; then
    tag="$DEVRITES_REF"
  else
    tag="$(curl -fsSL "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" 2>/dev/null | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  got=0
  if [ -n "$tag" ]; then
    curl -fsSL -o "$tmp/devrites.tar.gz" "https://github.com/$DEVRITES_REPO/releases/download/$tag/devrites-$tag.tar.gz" 2>/dev/null && got=1
    [ "$got" -eq 1 ] || curl -fsSL -o "$tmp/devrites.tar.gz" "https://github.com/$DEVRITES_REPO/archive/refs/tags/$tag.tar.gz" 2>/dev/null && got=1
  fi
  [ "$got" -eq 1 ] || curl -fsSL -o "$tmp/devrites.tar.gz" "https://github.com/$DEVRITES_REPO/archive/refs/heads/main.tar.gz" || { echo "error: could not download DevRites" >&2; exit 1; }
  tar -C "$tmp" -xzf "$tmp/devrites.tar.gz" || { echo "error: could not extract DevRites tarball" >&2; exit 1; }
  bundle="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  [ -n "$bundle" ] && [ -f "$bundle/uninstall.sh" ] || { echo "error: extracted bundle is missing uninstall.sh" >&2; exit 1; }
  export DEVRITES_BOOTSTRAPPED=1
  exec bash "$bundle/uninstall.sh" "$@"
fi

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || { echo "error: extracted bundle is missing scripts/install-lib.sh" >&2; exit 1; }
. "$INSTALL_LIB"

ENGINE="$(dr_acquire_engine "$SELF_DIR" uninstall "$DEVRITES_REPO")" || { echo "error: could not acquire devrites-engine." >&2; exit 1; }
export DEVRITES_ENGINE_CLI="$ENGINE"
exec "$ENGINE" uninstall --source-dir "$SELF_DIR" "$@"
