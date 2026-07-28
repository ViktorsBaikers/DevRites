#!/usr/bin/env bash
# scripts/pin.sh: manage user-pinned slash aliases for DevRites.
#
# Adopts the same "thin wrapper SKILL.md that delegates" pattern as the engine
# installer uses for --short-aliases=all, but exposes it as a runtime verb so
# users can add / remove arbitrary aliases against any rite-* skill without
# re-running the installer.
#
# Usage:
#   scripts/pin.sh add    <alias> <target>      # /alias → /target (target = rite-spec | rite-build | ...)
#   scripts/pin.sh remove <alias>               # remove a previously-pinned alias
#   scripts/pin.sh list                         # print currently-pinned aliases
#   scripts/pin.sh --help
#
# Examples:
#   ./scripts/pin.sh add b rite-build        # /b == /rite-build
#   ./scripts/pin.sh add ship rite-seal      # /ship == /rite-seal
#   ./scripts/pin.sh remove b
#
# Targets:
#   - Default: $PWD (the installed project, which holds .claude/skills/
#     and may also hold .agents/skills/ for Codex).
#   - --target <dir> to operate on a project elsewhere.
#
# Safety:
#   - Refuses to write inside global Claude/Codex homes (DevRites is project-local only).
#   - Refuses to overwrite a non-alias skill at .claude/skills/<alias>/.
#   - Refuses if <target> is not a known rite-* skill in the target's pack.
#   - Manifest-managed: pinned wrappers are recorded in .claude/devrites.manifest
#     so the standard uninstall.sh cleans them up automatically.

set -euo pipefail

# ---- locate install-lib + load helpers ----------------------------------
SELF_DIR="$( cd "$(dirname "$0")" && pwd -P )"
ROOT="$( cd "$SELF_DIR/.." && pwd -P )"
LIB="$SELF_DIR/install-lib.sh"
if [ ! -r "$LIB" ]; then
  printf 'error: cannot find %s\n' "$LIB" >&2
  exit 2
fi
# shellcheck source=scripts/install-lib.sh
. "$LIB"

# ---- parse args ----------------------------------------------------------
TARGET="$PWD"
SUBCMD=""
ALIAS=""
DEST=""
while [ $# -gt 0 ]; do
  case "$1" in
    --target) TARGET="$2"; shift 2 ;;
    --target=*) TARGET="${1#*=}"; shift ;;
    -h|--help)
      sed -n '2,30p' "$0"; exit 0 ;;
    add|remove|list)
      SUBCMD="$1"; shift
      [ "$SUBCMD" = "add" ]    && { ALIAS="${1:-}"; DEST="${2:-}"; shift 2 || true; }
      [ "$SUBCMD" = "remove" ] && { ALIAS="${1:-}"; shift || true; }
      ;;
    *)
      dr_die "unknown arg: $1 (try --help)" ;;
  esac
done

[ -z "$SUBCMD" ] && dr_die "missing subcommand (add | remove | list)"

TARGET="$(dr_abspath_dir "$TARGET")" || dr_die "target dir not found: $TARGET"
dr_is_global_claude "$TARGET" && dr_die "refusing to operate on ~/.claude: DevRites is project-local only"
dr_is_global_codex "$TARGET" && dr_die "refusing to operate on ~/.codex: DevRites is project-local only"

SKILLS_DIR="$TARGET/.claude/skills"
CODEX_SKILLS_DIR="$TARGET/.agents/skills"
MF="$TARGET/$DR_MANIFEST_NAME"

[ -d "$SKILLS_DIR" ] || dr_die "no .claude/skills at $TARGET: run install.sh first?"
[ -f "$MF" ]         || dr_die "no manifest at $MF: run install.sh first?"

# ---- helpers -------------------------------------------------------------
PIN_TMP=""
cleanup_pin_tmp() {
  [ -z "$PIN_TMP" ] || rm -rf "$PIN_TMP"
}
trap cleanup_pin_tmp EXIT

path_exists() {
  [ -e "$1" ] || [ -L "$1" ]
}

valid_alias_name() {
  # lowercase ASCII, digits, hyphens. No /, ., spaces. Not "rite" or "rite-*".
  case "$1" in
    ""|/*|*/*|*[!a-z0-9-]*) return 1 ;;
    rite|rite-*) return 1 ;;
  esac
  return 0
}

is_known_target() {
  # Target must be a rite-* skill present in the installed pack.
  case "$1" in
    rite-*) [ -f "$SKILLS_DIR/$1/SKILL.md" ] ;;
    *) return 1 ;;
  esac
}

is_pinned_alias() {
  # Detect a wrapper we generated: SKILL.md description must contain "Alias of DevRites /<target>".
  [ -f "$SKILLS_DIR/$1/SKILL.md" ] || return 1
  grep -q 'description: Alias of DevRites /' "$SKILLS_DIR/$1/SKILL.md"
}

is_pinned_alias_file() {
  [ -f "$1" ] || return 1
  grep -q 'description: Alias of DevRites /' "$1"
}

preflight_alias_destination() {
  _dir="$1"
  _file="$2"
  if [ -L "$_dir" ] || { path_exists "$_dir" && [ ! -d "$_dir" ]; }; then
    dr_die "$_dir exists and is not a managed alias directory: refusing to overwrite"
  fi
  if path_exists "$_file"; then
    [ ! -L "$_file" ] && is_pinned_alias_file "$_file" \
      || dr_die "$_file exists and is NOT a pinned alias: refusing to overwrite"
  elif [ -d "$_dir" ]; then
    dr_die "$_dir exists without a pinned alias: refusing to overwrite"
  fi
}

# ---- subcommands ---------------------------------------------------------
do_add() {
  valid_alias_name "$ALIAS"    || dr_die "invalid alias name '$ALIAS' (lowercase / digits / hyphens; not 'rite' or 'rite-*')"
  is_known_target "$DEST"      || dr_die "unknown target '$DEST': not a rite-* skill in $SKILLS_DIR"
  [ "$ALIAS" = "$DEST" ]       && dr_die "alias and target are the same"

  ALIAS_DIR="$SKILLS_DIR/$ALIAS"
  ALIAS_FILE="$ALIAS_DIR/SKILL.md"
  ALIAS_REL=".claude/skills/$ALIAS/SKILL.md"
  CODEX_ALIAS_DIR="$CODEX_SKILLS_DIR/$ALIAS"
  CODEX_ALIAS_FILE="$CODEX_ALIAS_DIR/SKILL.md"
  CODEX_ALIAS_REL=".agents/skills/$ALIAS/SKILL.md"

  preflight_alias_destination "$ALIAS_DIR" "$ALIAS_FILE"
  if path_exists "$ALIAS_FILE"; then
    dr_warn "already pinned: /$ALIAS: overwriting"
  fi
  if [ -d "$CODEX_SKILLS_DIR" ]; then
    preflight_alias_destination "$CODEX_ALIAS_DIR" "$CODEX_ALIAS_FILE"
    if path_exists "$CODEX_ALIAS_FILE"; then
      dr_warn "already pinned for Codex: /$ALIAS: overwriting"
    fi
  fi

  PIN_TMP="$(mktemp -d "$SKILLS_DIR/.devrites-pin.XXXXXX")"
  dr_gen_alias_wrapper "$ALIAS" "$DEST" "$PIN_TMP/claude"
  if [ -d "$CODEX_SKILLS_DIR" ]; then
    dr_gen_alias_wrapper "$ALIAS" "$DEST" "$PIN_TMP/codex"
  fi
  cp -p "$MF" "$PIN_TMP/manifest"
  if ! dr_manifest_contains "$PIN_TMP/manifest" "$ALIAS_REL"; then
    printf '%s\n' "$ALIAS_REL" >> "$PIN_TMP/manifest"
  fi
  if [ -d "$CODEX_SKILLS_DIR" ] && ! dr_manifest_contains "$PIN_TMP/manifest" "$CODEX_ALIAS_REL"; then
    printf '%s\n' "$CODEX_ALIAS_REL" >> "$PIN_TMP/manifest"
  fi

  mkdir -p "$ALIAS_DIR"
  if [ -d "$CODEX_SKILLS_DIR" ]; then
    mkdir -p "$CODEX_ALIAS_DIR"
  fi
  [ -w "$ALIAS_DIR" ] || dr_die "$ALIAS_DIR is not writable"
  [ -w "$(dirname "$MF")" ] || dr_die "$(dirname "$MF") is not writable"
  if [ -d "$CODEX_SKILLS_DIR" ]; then
    [ -w "$CODEX_ALIAS_DIR" ] || dr_die "$CODEX_ALIAS_DIR is not writable"
  fi

  mv "$PIN_TMP/claude" "$ALIAS_FILE"
  if [ -d "$CODEX_SKILLS_DIR" ]; then
    mv "$PIN_TMP/codex" "$CODEX_ALIAS_FILE"
  fi
  mv "$PIN_TMP/manifest" "$MF"
  rm -rf "$PIN_TMP"
  PIN_TMP=""

  if [ -d "$CODEX_SKILLS_DIR" ]; then
    dr_ok "pinned: /$ALIAS → /$DEST   ($ALIAS_FILE, $CODEX_ALIAS_FILE)"
  else
    dr_ok "pinned: /$ALIAS → /$DEST   ($ALIAS_FILE)"
  fi
}

do_remove() {
  valid_alias_name "$ALIAS" || dr_die "invalid alias name '$ALIAS'"
  ALIAS_DIR="$SKILLS_DIR/$ALIAS"
  ALIAS_FILE="$ALIAS_DIR/SKILL.md"
  ALIAS_REL=".claude/skills/$ALIAS/SKILL.md"
  CODEX_ALIAS_DIR="$CODEX_SKILLS_DIR/$ALIAS"
  CODEX_ALIAS_FILE="$CODEX_ALIAS_DIR/SKILL.md"
  CODEX_ALIAS_REL=".agents/skills/$ALIAS/SKILL.md"

  [ -f "$ALIAS_FILE" ]      || dr_die "no pinned alias at $ALIAS_FILE"
  is_pinned_alias "$ALIAS"  || dr_die "$ALIAS_FILE exists but is not a pinned alias: refusing to remove"

  if path_exists "$CODEX_ALIAS_FILE"; then
    is_pinned_alias_file "$CODEX_ALIAS_FILE" || dr_die "$CODEX_ALIAS_FILE exists but is not a pinned alias: refusing to remove"
  fi

  PIN_TMP="$(mktemp -d "$SKILLS_DIR/.devrites-pin.XXXXXX")"
  awk -v claude="$ALIAS_REL" -v codex="$CODEX_ALIAS_REL" \
    '$0 != claude && $0 != codex' "$MF" > "$PIN_TMP/manifest"

  rm -f "$ALIAS_FILE"
  rmdir "$ALIAS_DIR" 2>/dev/null || true
  if path_exists "$CODEX_ALIAS_FILE"; then
    rm -f "$CODEX_ALIAS_FILE"
    rmdir "$CODEX_ALIAS_DIR" 2>/dev/null || true
  fi

  mv "$PIN_TMP/manifest" "$MF"
  rm -rf "$PIN_TMP"
  PIN_TMP=""

  dr_ok "unpinned: /$ALIAS"
}

do_list() {
  found=0
  for d in "$SKILLS_DIR"/*/; do
    [ -d "$d" ] || continue
    nm="$(basename "$d")"
    if is_pinned_alias "$nm"; then
      to="$(awk '/^description: Alias of DevRites \//{ sub(/.*Alias of DevRites \//,""); sub(/\..*/,""); print; exit }' "$d/SKILL.md")"
      printf '  /%-20s → /%s\n' "$nm" "$to"
      found=1
    fi
  done
  if [ "$found" -eq 0 ]; then
    dr_say "(no pinned aliases at $TARGET)"
  fi
}

case "$SUBCMD" in
  add)    [ -n "$ALIAS" ] && [ -n "$DEST" ] || dr_die "usage: pin.sh add <alias> <target>"; do_add ;;
  remove) [ -n "$ALIAS" ]                   || dr_die "usage: pin.sh remove <alias>";     do_remove ;;
  list)   do_list ;;
  *)      dr_die "unknown subcommand: $SUBCMD" ;;
esac
