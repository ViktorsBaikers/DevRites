#!/usr/bin/env bash
# scripts/pin.sh — manage user-pinned slash aliases for DevRites.
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

set -u

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
dr_is_global_claude "$TARGET" && dr_die "refusing to operate on ~/.claude — DevRites is project-local only"
dr_is_global_codex "$TARGET" && dr_die "refusing to operate on ~/.codex — DevRites is project-local only"

SKILLS_DIR="$TARGET/.claude/skills"
CODEX_SKILLS_DIR="$TARGET/.agents/skills"
MF="$TARGET/$DR_MANIFEST_NAME"

[ -d "$SKILLS_DIR" ] || dr_die "no .claude/skills at $TARGET — run install.sh first?"
[ -f "$MF" ]         || dr_die "no manifest at $MF — run install.sh first?"

# ---- helpers -------------------------------------------------------------
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

# ---- subcommands ---------------------------------------------------------
do_add() {
  valid_alias_name "$ALIAS"    || dr_die "invalid alias name '$ALIAS' (lowercase / digits / hyphens; not 'rite' or 'rite-*')"
  is_known_target "$DEST"      || dr_die "unknown target '$DEST' — not a rite-* skill in $SKILLS_DIR"
  [ "$ALIAS" = "$DEST" ]       && dr_die "alias and target are the same"

  ALIAS_DIR="$SKILLS_DIR/$ALIAS"
  ALIAS_FILE="$ALIAS_DIR/SKILL.md"
  ALIAS_REL=".claude/skills/$ALIAS/SKILL.md"
  CODEX_ALIAS_DIR="$CODEX_SKILLS_DIR/$ALIAS"
  CODEX_ALIAS_FILE="$CODEX_ALIAS_DIR/SKILL.md"
  CODEX_ALIAS_REL=".agents/skills/$ALIAS/SKILL.md"

  if [ -e "$ALIAS_FILE" ]; then
    if is_pinned_alias "$ALIAS"; then
      dr_warn "already pinned: /$ALIAS — overwriting"
    else
      dr_die "$ALIAS_FILE exists and is NOT a pinned alias — refusing to overwrite"
    fi
  fi
  if [ -d "$CODEX_SKILLS_DIR" ] && [ -e "$CODEX_ALIAS_FILE" ]; then
    if is_pinned_alias_file "$CODEX_ALIAS_FILE"; then
      dr_warn "already pinned for Codex: /$ALIAS — overwriting"
    else
      dr_die "$CODEX_ALIAS_FILE exists and is NOT a pinned alias — refusing to overwrite"
    fi
  fi

  mkdir -p "$ALIAS_DIR"
  dr_gen_alias_wrapper "$ALIAS" "$DEST" "$ALIAS_FILE"
  if [ -d "$CODEX_SKILLS_DIR" ]; then
    mkdir -p "$CODEX_ALIAS_DIR"
    dr_gen_alias_wrapper "$ALIAS" "$DEST" "$CODEX_ALIAS_FILE"
  fi

  if ! dr_manifest_contains "$MF" "$ALIAS_REL"; then
    printf '%s\n' "$ALIAS_REL" >> "$MF"
  fi
  if [ -d "$CODEX_SKILLS_DIR" ] && ! dr_manifest_contains "$MF" "$CODEX_ALIAS_REL"; then
    printf '%s\n' "$CODEX_ALIAS_REL" >> "$MF"
  fi

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
  is_pinned_alias "$ALIAS"  || dr_die "$ALIAS_FILE exists but is not a pinned alias — refusing to remove"

  rm -f "$ALIAS_FILE"
  rmdir "$ALIAS_DIR" 2>/dev/null || true
  if [ -f "$CODEX_ALIAS_FILE" ]; then
    is_pinned_alias_file "$CODEX_ALIAS_FILE" || dr_die "$CODEX_ALIAS_FILE exists but is not a pinned alias — refusing to remove"
    rm -f "$CODEX_ALIAS_FILE"
    rmdir "$CODEX_ALIAS_DIR" 2>/dev/null || true
  fi

  # Drop the alias line from the manifest (preserve header + the rest)
  TMP="$(mktemp)"
  grep -Fvx "$ALIAS_REL" "$MF" | grep -Fvx "$CODEX_ALIAS_REL" > "$TMP" && mv "$TMP" "$MF"

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
  [ "$found" -eq 0 ] && dr_say "(no pinned aliases at $TARGET)"
}

case "$SUBCMD" in
  add)    [ -n "$ALIAS" ] && [ -n "$DEST" ] || dr_die "usage: pin.sh add <alias> <target>"; do_add ;;
  remove) [ -n "$ALIAS" ]                   || dr_die "usage: pin.sh remove <alias>";     do_remove ;;
  list)   do_list ;;
  *)      dr_die "unknown subcommand: $SUBCMD" ;;
esac
