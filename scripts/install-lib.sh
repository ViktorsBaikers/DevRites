#!/usr/bin/env bash
# install-lib.sh - shared shell helpers for DevRites shims and scripts/pin.sh.
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

# ---- engine bootstrap ----------------------------------------------------
# These helpers are deliberately limited to release/binary acquisition for the
# shell shims. Install/update/uninstall policy lives in devrites-engine.
dr_pkg_version() {
  _dr_source_dir="$1"
  sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$_dr_source_dir/package.json" 2>/dev/null | head -n1
}

dr_release_tag() {
  _dr_source_dir="$1"
  if [ -n "${DEVRITES_REF:-}" ]; then
    printf '%s\n' "$DEVRITES_REF"
    return 0
  fi
  _dr_version="$(dr_pkg_version "$_dr_source_dir")"
  [ -n "$_dr_version" ] && printf 'v%s\n' "${_dr_version#v}"
}

dr_download_engine() {
  _dr_source_dir="$1"
  _dr_repo="$2"
  _dr_out="$3"
  _dr_tag="$(dr_release_tag "$_dr_source_dir")"
  [ -n "$_dr_tag" ] || return 1
  _dr_os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
  _dr_arch="$(uname -m 2>/dev/null)"
  case "$_dr_arch" in
    x86_64|amd64) _dr_arch=amd64 ;;
    arm64|aarch64) _dr_arch=arm64 ;;
    *) return 1 ;;
  esac
  case "$_dr_os" in
    linux|darwin) : ;;
    *) return 1 ;;
  esac

  _dr_url="https://github.com/$_dr_repo/releases/download/$_dr_tag/devrites-$_dr_os-$_dr_arch"
  curl -fsSL -o "$_dr_out" "$_dr_url" 2>/dev/null || return 1
  curl -fsSL -o "$_dr_out.sha256" "$_dr_url.sha256" 2>/dev/null || return 1
  _dr_want="$(awk '{print $1; exit}' "$_dr_out.sha256" 2>/dev/null)"
  if command -v shasum >/dev/null 2>&1; then
    _dr_got="$(shasum -a 256 "$_dr_out" | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    _dr_got="$(sha256sum "$_dr_out" | awk '{print $1}')"
  else
    _dr_got=""
  fi
  [ -n "$_dr_got" ] && [ "$_dr_got" = "$_dr_want" ] || return 1
  chmod +x "$_dr_out" 2>/dev/null || true
}

dr_build_engine() {
  _dr_source_dir="$1"
  _dr_out="$2"
  _dr_go_bin="$(command -v go 2>/dev/null || true)"
  [ -d "$_dr_source_dir/engine" ] && [ -n "$_dr_go_bin" ] || return 1
  _dr_tag="$(dr_release_tag "$_dr_source_dir")"
  [ -n "$_dr_tag" ] || _dr_tag="dev"
  ( cd "$_dr_source_dir/engine" && GOCACHE="$(dirname "$_dr_out")/go-cache" CGO_ENABLED=0 "$_dr_go_bin" build -trimpath -ldflags "-s -w -X github.com/devrites/devrites/internal/version.Version=$_dr_tag" -o "$_dr_out" . ) 2>/dev/null || return 1
  chmod +x "$_dr_out" 2>/dev/null || true
}

dr_engine_supports() {
  _dr_engine="$1"
  _dr_subcommand="$2"
  "$_dr_engine" "$_dr_subcommand" --help >/dev/null 2>&1
}

dr_acquire_engine() {
  _dr_source_dir="$1"
  _dr_subcommand="$2"
  _dr_repo="$3"

  if [ -n "${DEVRITES_ENGINE_CLI:-}" ] && [ -x "$DEVRITES_ENGINE_CLI" ]; then
    dr_engine_supports "$DEVRITES_ENGINE_CLI" "$_dr_subcommand" && { printf '%s\n' "$DEVRITES_ENGINE_CLI"; return 0; }
  fi

  case "$_dr_subcommand" in
    install|update) _dr_prefer_fresh=1 ;;
    *) _dr_prefer_fresh=0 ;;
  esac

  if [ "$_dr_prefer_fresh" -eq 0 ] && command -v devrites-engine >/dev/null 2>&1; then
    _dr_path="$(command -v devrites-engine)"
    dr_engine_supports "$_dr_path" "$_dr_subcommand" && { printf '%s\n' "$_dr_path"; return 0; }
  fi

  _dr_tmp="${TMPDIR:-/tmp}/devrites-engine-bootstrap.$$"
  mkdir -p "$_dr_tmp"
  if dr_build_engine "$_dr_source_dir" "$_dr_tmp/devrites-engine"; then
    dr_engine_supports "$_dr_tmp/devrites-engine" "$_dr_subcommand" && { printf '%s\n' "$_dr_tmp/devrites-engine"; return 0; }
  fi

  if command -v curl >/dev/null 2>&1; then
    if dr_download_engine "$_dr_source_dir" "$_dr_repo" "$_dr_tmp/devrites-engine"; then
      dr_engine_supports "$_dr_tmp/devrites-engine" "$_dr_subcommand" && { printf '%s\n' "$_dr_tmp/devrites-engine"; return 0; }
    fi
  fi

  if [ "$_dr_prefer_fresh" -eq 1 ] && command -v devrites-engine >/dev/null 2>&1; then
    _dr_path="$(command -v devrites-engine)"
    dr_engine_supports "$_dr_path" "$_dr_subcommand" && { printf '%s\n' "$_dr_path"; return 0; }
  fi

  return 1
}
# ---- manifest ------------------------------------------------------------
# Manifest = newline list of target-relative paths. Pin aliases append to the
# engine-owned manifest so the standard uninstaller can remove them.
DR_MANIFEST_NAME=".claude/devrites.manifest"

# True if a target-relative path is listed in the manifest file.
dr_manifest_contains() {
  _mf="$1"; _rel="$2"
  [ -f "$_mf" ] || return 1
  # exact line match, ignoring comment/blank lines
  grep -v '^#' "$_mf" 2>/dev/null | grep -v '^[[:space:]]*$' \
    | grep -Fxq "$_rel"
}

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
# Generate a thin SKILL.md that delegates to a DevRites skill. The engine
# installer owns built-in short aliases; this helper is only for user-pinned
# ad-hoc shortcuts managed by scripts/pin.sh.
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
