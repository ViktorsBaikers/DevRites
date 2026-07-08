#!/usr/bin/env bash
# install.sh - bootstrap shim for the engine-owned DevRites installer.
# GUARD:no-global - project-local agent files are installed only under --target;
# the only sanctioned global write is the devrites-engine binary lifecycle.
set -u

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd -P)" || SELF_DIR=""
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"

verify_sha256() {
  file="$1"
  sumfile="$2"
  want="$(awk '{print $1; exit}' "$sumfile" 2>/dev/null)"
  if command -v shasum >/dev/null 2>&1; then
    got="$(shasum -a 256 "$file" | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    got="$(sha256sum "$file" | awk '{print $1}')"
  else
    got=""
  fi
  [ -n "$want" ] && [ -n "$got" ] && [ "$got" = "$want" ]
}

bootstrap_bundle() {
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ - aborting to avoid a loop." >&2
    exit 1
  fi
  command -v curl >/dev/null 2>&1 || { echo "error: curl is required for the network installer." >&2; exit 1; }
  command -v tar >/dev/null 2>&1 || { echo "error: tar is required for the network installer." >&2; exit 1; }
  tmp="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-bootstrap.$$")"
  mkdir -p "$tmp"
  if [ -n "$DEVRITES_REF" ]; then
    tag="$DEVRITES_REF"
  else
    tag="$(curl -fsSL "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" 2>/dev/null | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  got=0
  if [ -n "$tag" ]; then
    url="https://github.com/$DEVRITES_REPO/releases/download/$tag/devrites-$tag.tar.gz"
    if curl -fsSL -o "$tmp/devrites.tar.gz" "$url" 2>/dev/null; then
      if curl -fsSL -o "$tmp/devrites.tar.gz.sha256" "$url.sha256" 2>/dev/null && verify_sha256 "$tmp/devrites.tar.gz" "$tmp/devrites.tar.gz.sha256"; then
        got=1
      else
        echo "warning: release artifact checksum verification failed for $url; falling back to source archive." >&2
        rm -f "$tmp/devrites.tar.gz" "$tmp/devrites.tar.gz.sha256"
      fi
    fi
    if [ "$got" -ne 1 ]; then
      url="https://github.com/$DEVRITES_REPO/archive/refs/tags/$tag.tar.gz"
      curl -fsSL -o "$tmp/devrites.tar.gz" "$url" 2>/dev/null && got=1
    fi
  fi
  if [ "$got" -ne 1 ]; then
    url="https://github.com/$DEVRITES_REPO/archive/refs/heads/main.tar.gz"
    curl -fsSL -o "$tmp/devrites.tar.gz" "$url" || { echo "error: could not download DevRites from $url" >&2; exit 1; }
  fi
  tar -C "$tmp" -xzf "$tmp/devrites.tar.gz" || { echo "error: could not extract DevRites tarball" >&2; exit 1; }
  bundle="$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  [ -n "$bundle" ] && [ -f "$bundle/install.sh" ] || { echo "error: extracted bundle is missing install.sh" >&2; exit 1; }
  chmod +x "$bundle/install.sh" "$bundle/uninstall.sh" "$bundle/update.sh" 2>/dev/null || true
  echo "DevRites: bootstrapped from ${tag:-main} -> $bundle"
  export DEVRITES_BOOTSTRAPPED=1
  exec bash "$bundle/install.sh" "$@"
}

if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack" ]; then
  bootstrap_bundle "$@"
fi

INSTALL_LIB="$SELF_DIR/scripts/install-lib.sh"
[ -f "$INSTALL_LIB" ] || { echo "error: extracted bundle is missing scripts/install-lib.sh" >&2; exit 1; }
. "$INSTALL_LIB"

ENGINE="$(dr_acquire_engine "$SELF_DIR" install "$DEVRITES_REPO")" || { echo "error: could not acquire devrites-engine (no usable installed binary and no matching verified release binary)." >&2; exit 1; }
PAYLOAD="${DEVRITES_HOST_ARTIFACT_DIR:-$SELF_DIR/pack/generated}"
if [ ! -d "$PAYLOAD/claude/skills" ] || [ ! -d "$PAYLOAD/codex/skills" ] || [ ! -f "$PAYLOAD/codex/hooks.json" ]; then
  BUILDER="$SELF_DIR/scripts/build-host-artifacts.sh"
  [ -f "$BUILDER" ] || { echo "error: generated install payload missing at $PAYLOAD and builder missing at $BUILDER" >&2; exit 1; }
  DEVRITES_HOST_ARTIFACT_DIR="$PAYLOAD" bash "$BUILDER" >/dev/null || { echo "error: could not generate install payload at $PAYLOAD" >&2; exit 1; }
fi
export DEVRITES_ENGINE_CLI="$ENGINE"
exec "$ENGINE" install --source-dir "$SELF_DIR" --payload-dir "$PAYLOAD" "$@"
