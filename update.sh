#!/usr/bin/env bash
# update.sh — upgrade an existing DevRites install to the latest GitHub Release.
#
#   ./update.sh                       upgrade the install in the current directory
#   ./update.sh --target DIR          upgrade the install in DIR
#   ./update.sh --to vX.Y.Z           pin to a specific release tag instead of latest
#   ./update.sh --check               report installed vs latest, do not change anything
#   ./update.sh --force               re-install even if already at the latest version
#   ./update.sh --pre                 allow pre-release tags (default: stable only)
#
# Env:
#   DEVRITES_UPDATE_BUNDLE=PATH       use a local .tar.gz instead of downloading
#                                     (air-gapped upgrades; also used by tests)
#
# How it works:
#   1. Reads the installed version + flags from .claude/devrites.manifest.
#   2. Asks the GitHub API for the latest release tag (or the tag passed via --to).
#   3. If newer, downloads the release tarball, extracts to a temp dir,
#      and re-runs the bundled install.sh with the same flags the user
#      originally installed with — preserving .devrites/ feature state.
#
# Requires: curl, tar. (jq optional; falls back to sed parsing.)

set -u

REPO="ViktorsBaikers/DevRites"
API="https://api.github.com/repos/$REPO"

# ---- args ----------------------------------------------------------------
TARGET="$PWD"
PIN_TAG=""
CHECK_ONLY=0
FORCE=0
ALLOW_PRE=0

while [ $# -gt 0 ]; do
  case "$1" in
    --target)  shift; [ $# -gt 0 ] || { echo "error: --target needs a directory" >&2; exit 1; }; TARGET="$1" ;;
    --target=*) TARGET="${1#*=}" ;;
    --to)      shift; [ $# -gt 0 ] || { echo "error: --to needs a tag" >&2; exit 1; }; PIN_TAG="$1" ;;
    --to=*)    PIN_TAG="${1#*=}" ;;
    --check)   CHECK_ONLY=1 ;;
    --force)   FORCE=1 ;;
    --pre)     ALLOW_PRE=1 ;;
    -h|--help) sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)         echo "error: unknown option: $1 (try --help)" >&2; exit 1 ;;
  esac
  shift
done

# ---- pretty -------------------------------------------------------------
if [ -t 1 ]; then
  B="$(printf '\033[1m')"; R="$(printf '\033[0m')"
  G="$(printf '\033[32m')"; C="$(printf '\033[36m')"; Y="$(printf '\033[33m')"
else
  B=""; R=""; G=""; C=""; Y=""
fi
say()  { printf '%s\n' "$*"; }
info() { printf '%s%s%s\n' "$C" "$*" "$R"; }
ok()   { printf '%s%s%s\n' "$G" "$*" "$R"; }
warn() { printf '%swarning:%s %s\n' "$Y" "$R" "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

# ---- prerequisites -------------------------------------------------------
command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tar  >/dev/null 2>&1 || die "tar is required"

# ---- locate install ------------------------------------------------------
[ -d "$TARGET" ] || die "target is not a directory: $TARGET"
TARGET="$(cd "$TARGET" && pwd -P)"
MANIFEST="$TARGET/.claude/devrites.manifest"
[ -f "$MANIFEST" ] || die "no DevRites install found at $TARGET (missing .claude/devrites.manifest). Run install.sh first."

INSTALLED_VERSION="$(sed -n 's/^# devrites-version:[[:space:]]*//p' "$MANIFEST" | head -n1)"
INSTALL_FLAGS="$(sed -n 's/^# devrites-flags:[[:space:]]*//p' "$MANIFEST" | head -n1)"
[ -n "$INSTALLED_VERSION" ] || INSTALLED_VERSION="unknown"

info "DevRites update"
say  "  project:   $TARGET"
say  "  installed: ${B}$INSTALLED_VERSION${R}"
[ -n "$INSTALL_FLAGS" ] && say "  flags:     $INSTALL_FLAGS"

# ---- resolve target tag --------------------------------------------------
if [ -n "$PIN_TAG" ]; then
  LATEST_TAG="$PIN_TAG"
else
  if [ "$ALLOW_PRE" -eq 1 ]; then
    # All releases, take the first one
    LATEST_TAG="$(curl -fsSL "$API/releases" 2>/dev/null \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
      | head -n1)"
  else
    LATEST_TAG="$(curl -fsSL "$API/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
      | head -n1)"
  fi
  [ -n "$LATEST_TAG" ] || die "could not resolve the latest release tag from GitHub. Check connectivity, or pass --to vX.Y.Z."
fi
LATEST_VERSION="${LATEST_TAG#v}"
say "  latest:    ${B}$LATEST_VERSION${R}"

# ---- semver compare ------------------------------------------------------
# Returns 0 if $1 > $2, 1 if equal, 2 if $1 < $2. Pre-release tags compare as lower.
semver_cmp() {
  _a="${1%%-*}"; _b="${2%%-*}"
  _ap="${1#${_a}}"; _bp="${2#${_b}}"
  IFS=. read -r a1 a2 a3 <<EOF
$_a
EOF
  IFS=. read -r b1 b2 b3 <<EOF
$_b
EOF
  a1=${a1:-0}; a2=${a2:-0}; a3=${a3:-0}
  b1=${b1:-0}; b2=${b2:-0}; b3=${b3:-0}
  if   [ "$a1" -gt "$b1" ]; then return 0
  elif [ "$a1" -lt "$b1" ]; then return 2
  elif [ "$a2" -gt "$b2" ]; then return 0
  elif [ "$a2" -lt "$b2" ]; then return 2
  elif [ "$a3" -gt "$b3" ]; then return 0
  elif [ "$a3" -lt "$b3" ]; then return 2
  fi
  # core equal — pre-release < release
  if   [ -z "$_ap" ] && [ -n "$_bp" ]; then return 0
  elif [ -n "$_ap" ] && [ -z "$_bp" ]; then return 2
  fi
  return 1
}

NEEDS_UPDATE=0
if [ "$INSTALLED_VERSION" = "unknown" ]; then
  NEEDS_UPDATE=1
  warn "installed version unknown — treating as out-of-date."
else
  semver_cmp "$LATEST_VERSION" "$INSTALLED_VERSION"
  case $? in
    0) NEEDS_UPDATE=1 ;;
    1) NEEDS_UPDATE=0 ;;
    2) warn "installed version ($INSTALLED_VERSION) is newer than latest release ($LATEST_VERSION)." ;;
  esac
fi

if [ "$CHECK_ONLY" -eq 1 ]; then
  if [ "$NEEDS_UPDATE" -eq 1 ]; then
    info "update available: $INSTALLED_VERSION → $LATEST_VERSION"
    exit 10
  else
    ok "up to date."
    exit 0
  fi
fi

if [ "$NEEDS_UPDATE" -eq 0 ] && [ "$FORCE" -eq 0 ]; then
  ok "already at $LATEST_VERSION — nothing to do (use --force to reinstall)."
  exit 0
fi

# ---- download release tarball -------------------------------------------
TMP="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-update.$$")"
mkdir -p "$TMP"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

ARTIFACT="devrites-${LATEST_TAG}.tar.gz"
URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$ARTIFACT"
SRC_TARBALL_URL="https://github.com/$REPO/archive/refs/tags/$LATEST_TAG.tar.gz"

# DEVRITES_UPDATE_BUNDLE — optional escape hatch for air-gapped upgrades and
# tests: a readable local .tar.gz to use instead of downloading from GitHub.
if [ -n "${DEVRITES_UPDATE_BUNDLE:-}" ]; then
  [ -r "$DEVRITES_UPDATE_BUNDLE" ] || die "DEVRITES_UPDATE_BUNDLE not readable: $DEVRITES_UPDATE_BUNDLE"
  info "using local bundle $DEVRITES_UPDATE_BUNDLE (DEVRITES_UPDATE_BUNDLE set) …"
  cp "$DEVRITES_UPDATE_BUNDLE" "$TMP/$ARTIFACT" || die "could not copy $DEVRITES_UPDATE_BUNDLE"
else
  info "downloading $ARTIFACT …"
  if ! curl -fsSL -o "$TMP/$ARTIFACT" "$URL" 2>/dev/null; then
    warn "release artifact $ARTIFACT not found; falling back to the source tarball ($LATEST_TAG)."
    ARTIFACT="src-$LATEST_TAG.tar.gz"
    curl -fsSL -o "$TMP/$ARTIFACT" "$SRC_TARBALL_URL" || die "could not download $SRC_TARBALL_URL"
  fi
fi

tar -C "$TMP" -xzf "$TMP/$ARTIFACT"
EXTRACTED="$(find "$TMP" -maxdepth 1 -mindepth 1 -type d | head -n1)"
[ -n "$EXTRACTED" ] && [ -f "$EXTRACTED/install.sh" ] || die "extracted bundle has no install.sh: $EXTRACTED"
# GitHub source tarballs strip +x; restore it before re-exec.
chmod +x "$EXTRACTED/install.sh" "$EXTRACTED"/scripts/*.sh 2>/dev/null || true

# ---- re-run installer with the original flags + --force -----------------
info "running installer (with --force) — preserving .devrites/ feature state …"
# DEVRITES_REF pins the engine-binary download to the SAME release as the pack, so
# install_binary installs the matching version (and its downgrade guard sees the
# right target) instead of independently resolving "latest".
# shellcheck disable=SC2086
DEVRITES_REF="$LATEST_TAG" bash "$EXTRACTED/install.sh" --target "$TARGET" --force $INSTALL_FLAGS \
  || die "installer failed; the previous install is still in place."

ok "DevRites upgraded: $INSTALLED_VERSION → $LATEST_VERSION"
say "  run ${C}/rite${R} or ${C}/rite-status${R} to continue where you left off."
exit 0
