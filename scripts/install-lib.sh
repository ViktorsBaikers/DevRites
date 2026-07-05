#!/usr/bin/env bash
# install-lib.sh — shared helpers for install.sh / uninstall.sh.
# Sourced, not executed. Bash 3.2 compatible (no assoc arrays, no mapfile).

# ---- logging -------------------------------------------------------------
if [ -t 1 ]; then
  DR_B="$(printf '\033[1m')"; DR_R="$(printf '\033[0m')"
  DR_Y="$(printf '\033[33m')"; DR_G="$(printf '\033[32m')"; DR_C="$(printf '\033[36m')"
else
  DR_B=""; DR_R=""; DR_Y=""; DR_G=""; DR_C=""
fi
dr_say()  { printf '%s\n' "$*"; }
dr_info() { printf '%s%s%s\n' "$DR_C" "$*" "$DR_R"; }
dr_ok()   { printf '%s%s%s\n' "$DR_G" "$*" "$DR_R"; }
dr_warn() { printf '%swarning:%s %s\n' "$DR_Y" "$DR_R" "$*" >&2; }
dr_err()  { printf 'error: %s\n' "$*" >&2; }
dr_die()  { dr_err "$*"; exit 1; }

# ---- paths ---------------------------------------------------------------
# Canonical absolute path of an existing directory.
dr_abspath_dir() {
  ( cd "$1" 2>/dev/null && pwd -P ) || return 1
}
# Canonical absolute path of a path whose parent exists (file may not).
dr_canon() {
  _p="$1"
  case "$_p" in
    /*) : ;;
    *)  _p="$PWD/$_p" ;;
  esac
  _d="$( cd "$(dirname "$_p")" 2>/dev/null && pwd -P )" || return 1
  printf '%s/%s\n' "$_d" "$(basename "$_p")"
}

# ---- manifest ------------------------------------------------------------
# Manifest = newline list of target-relative paths, plus comment header lines.
DR_MANIFEST_NAME=".claude/devrites.manifest"

# True if a target-relative path is listed in the manifest file.
dr_manifest_contains() {
  _mf="$1"; _rel="$2"
  [ -f "$_mf" ] || return 1
  # exact line match, ignoring comment/blank lines
  grep -v '^#' "$_mf" 2>/dev/null | grep -v '^[[:space:]]*$' \
    | grep -Fxq "$_rel"
}

# ---- misc ----------------------------------------------------------------
dr_lc() { printf '%s' "$1" | tr '[:upper:]' '[:lower:]'; }

# Is $1 (canonical) equal to, or inside, a global agent home?
# Used to refuse global writes. No write verbs here — read-only comparison.
dr_is_global_claude() {
  _t="$1"
  _home_claude="$HOME/.claude"
  [ "$_t" = "$_home_claude" ] && return 0
  case "$_t/" in
    "$_home_claude"/*) return 0 ;;
  esac
  return 1
}

dr_is_global_codex() {
  _t="$1"
  _home_codex="$HOME/.codex"
  [ "$_t" = "$_home_codex" ] && return 0
  case "$_t/" in
    "$_home_codex"/*) return 0 ;;
  esac
  return 1
}

# ---- alias wrapper template ----------------------------------------------
# Generate a thin SKILL.md that delegates to a DevRites skill. Used by:
#   - install.sh (--short-aliases=all → /define, /build, /prove, /seal)
#   - scripts/pin.sh (user-pinned ad-hoc shortcuts)
# Args: $1=alias-name, $2=target-skill-name (e.g. rite-build), $3=output-file
dr_gen_alias_wrapper() {
  _name="$1"; _to="$2"; _out="$3"
  cat > "$_out" <<EOF
---
name: $_name
description: Alias of DevRites /$_to. Use when the user runs /$_name. Delegates to /$_to.
argument-hint: "[args]"
user-invocable: true
---

# /$_name — alias of /$_to

This command **delegates to DevRites \`/$_to\`**. State that to the user, then load and
run \`$_to/SKILL.md\` with the given arguments, following it exactly.
EOF
}
