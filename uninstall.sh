#!/usr/bin/env bash
# uninstall.sh — remove DevRites from a project. Removes ONLY the files listed in
# .claude/devrites.manifest (plus the manifest itself) and prunes dirs left empty.
# Never touches your feature data in .devrites/work/.
#
#   ./uninstall.sh                 uninstall from the current directory
#   ./uninstall.sh --target DIR    uninstall from DIR
#   ./uninstall.sh --dry-run       show what would be removed, change nothing
#
# Network uninstall (no git clone needed):
#   curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/uninstall.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/uninstall.sh | bash -s -- --target /path

set -u
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd -P)" || SELF_DIR=""

# ---- bootstrap (curl | bash) --------------------------------------------
# Same pattern as install.sh: when piped from curl there are no sibling files,
# so download a tarball and re-exec.
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"
if [ -z "$SELF_DIR" ] || [ ! -r "$SELF_DIR/scripts/install-lib.sh" ]; then
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find scripts/install-lib.sh — aborting." >&2
    exit 1
  fi
  command -v curl >/dev/null 2>&1 || { echo "error: curl is required for the network uninstaller." >&2; exit 1; }
  command -v tar  >/dev/null 2>&1 || { echo "error: tar is required for the network uninstaller."  >&2; exit 1; }

  BOOT_TMP="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-bootstrap.$$")"
  if [ -n "$DEVRITES_REF" ]; then
    BOOT_TAG="$DEVRITES_REF"
  else
    BOOT_TAG="$(curl -fsSL "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  BOOT_DOWNLOADED=0
  if [ -n "$BOOT_TAG" ]; then
    BOOT_URL="https://github.com/$DEVRITES_REPO/releases/download/$BOOT_TAG/devrites-$BOOT_TAG.tar.gz"
    curl -fsSL -o "$BOOT_TMP/devrites.tar.gz" "$BOOT_URL" 2>/dev/null && BOOT_DOWNLOADED=1
    if [ "$BOOT_DOWNLOADED" -ne 1 ]; then
      BOOT_URL="https://github.com/$DEVRITES_REPO/archive/refs/tags/$BOOT_TAG.tar.gz"
      curl -fsSL -o "$BOOT_TMP/devrites.tar.gz" "$BOOT_URL" 2>/dev/null && BOOT_DOWNLOADED=1
    fi
  fi
  if [ "$BOOT_DOWNLOADED" -ne 1 ]; then
    BOOT_URL="https://github.com/$DEVRITES_REPO/archive/refs/heads/main.tar.gz"
    curl -fsSL -o "$BOOT_TMP/devrites.tar.gz" "$BOOT_URL" || { echo "error: could not download DevRites from $BOOT_URL" >&2; exit 1; }
  fi
  tar -C "$BOOT_TMP" -xzf "$BOOT_TMP/devrites.tar.gz" || { echo "error: could not extract DevRites tarball" >&2; exit 1; }
  BOOT_BUNDLE="$(find "$BOOT_TMP" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  [ -n "$BOOT_BUNDLE" ] && [ -f "$BOOT_BUNDLE/uninstall.sh" ] || { echo "error: extracted bundle is missing uninstall.sh" >&2; exit 1; }
  chmod +x "$BOOT_BUNDLE/uninstall.sh" "$BOOT_BUNDLE"/scripts/*.sh 2>/dev/null || true
  export DEVRITES_BOOTSTRAPPED=1
  exec bash "$BOOT_BUNDLE/uninstall.sh" "$@"
fi

. "$SELF_DIR/scripts/install-lib.sh"

TARGET="$PWD"
DRYRUN=0
while [ $# -gt 0 ]; do
  case "$1" in
    --target) shift; [ $# -gt 0 ] || dr_die "--target needs a directory"; TARGET="$1" ;;
    --target=*) TARGET="${1#*=}" ;;
    --dry-run) DRYRUN=1 ;;
    -h|--help) sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) dr_die "unknown option: $1 (try --help)" ;;
  esac
  shift
done

[ -d "$TARGET" ] || dr_die "target is not a directory: $TARGET"
TARGET="$(dr_abspath_dir "$TARGET")" || dr_die "cannot resolve target: $TARGET"
MF="$TARGET/$DR_MANIFEST_NAME"
[ -f "$MF" ] || dr_die "no DevRites manifest at $MF — nothing to uninstall."

dr_info "DevRites uninstaller"
dr_say  "  target  : $TARGET"
dr_say  "  manifest: $MF"
[ "$DRYRUN" -eq 1 ] && dr_info "  (dry run — no changes)"
dr_say ""

N_REMOVED=0; N_MISSING=0
PRUNE_TMP="$(mktemp 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-prune.$$")"
: > "$PRUNE_TMP"
trap 'rm -f "$PRUNE_TMP"' EXIT

# Remove each manifest-listed file.
while IFS= read -r rel; do
  case "$rel" in ''|\#*) continue ;; esac          # skip blanks/comments
  case "$rel" in /*|*..*) dr_warn "ignoring suspicious manifest entry: $rel"; continue ;; esac
  dest="$TARGET/$rel"
  if [ -e "$dest" ] || [ -L "$dest" ]; then
    if [ "$DRYRUN" -eq 1 ]; then
      dr_say "  [remove] $rel"
    else
      rm -f "$dest" || dr_warn "could not remove $dest"
    fi
    printf '%s\n' "$(dirname "$dest")" >> "$PRUNE_TMP"
    N_REMOVED=$((N_REMOVED+1))
  else
    N_MISSING=$((N_MISSING+1))
  fi
done < "$MF"

# Remove the manifest itself.
if [ "$DRYRUN" -eq 1 ]; then
  dr_say "  [remove] $DR_MANIFEST_NAME"
else
  rm -f "$MF"
  printf '%s\n' "$(dirname "$MF")" >> "$PRUNE_TMP"
fi

# Prune directories left empty (rmdir only removes empty dirs — safe). Two passes
# walk up to catch parents. Never removes .devrites/work or any non-empty dir.
if [ "$DRYRUN" -eq 0 ]; then
  pass=0
  while [ "$pass" -lt 6 ]; do
    sort -u "$PRUNE_TMP" | awk '{ print length, $0 }' | sort -rn | cut -d' ' -f2- \
    | while IFS= read -r d; do
        case "$d" in "$TARGET"|"$TARGET"/) continue ;; esac
        [ -d "$d" ] && rmdir "$d" 2>/dev/null && printf '%s\n' "$(dirname "$d")" >> "$PRUNE_TMP.next"
      done
    [ -f "$PRUNE_TMP.next" ] || break
    cat "$PRUNE_TMP.next" >> "$PRUNE_TMP"; rm -f "$PRUNE_TMP.next"
    pass=$((pass+1))
  done
fi

dr_say ""
dr_ok "DevRites $([ "$DRYRUN" -eq 1 ] && echo 'uninstall plan complete (dry run)' || echo uninstalled)"
dr_say "  removed: $N_REMOVED   already-absent: $N_MISSING"
[ -d "$TARGET/.devrites/work" ] && dr_say "  ${DR_Y}kept${DR_R} .devrites/work/ (your feature data) — delete it manually if you want it gone."
[ -f "$TARGET/.devrites/ACTIVE" ] && dr_say "  ${DR_Y}kept${DR_R} .devrites/ACTIVE (active-feature cursor) — runtime state."
[ -f "$TARGET/.devrites/AFK" ] && dr_say "  ${DR_Y}kept${DR_R} .devrites/AFK (per-developer AFK sentinel) — runtime state."
exit 0
