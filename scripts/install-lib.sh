#!/usr/bin/env bash
# install-lib.sh - shared shell helpers for DevRites shims and scripts/pin.sh.
# Sourced, not executed. Bash 3.2 compatible (no assoc arrays, no mapfile).

# ---- logging -------------------------------------------------------------
if [ -t 1 ]; then
  DR_B="$(printf '\033[1m')"; DR_R="$(printf '\033[0m')"
  DR_Y="$(printf '\033[33m')"; DR_G="$(printf '\033[32m')"
else
  DR_B=""; DR_R=""; DR_Y=""; DR_G=""
fi
dr_say()  { printf '%s\n' "$*"; }
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
# Shell shims own source and binary acquisition. The engine receives only local
# source, payload, and a version-compatible binary handoff.
dr_pkg_version() {
  _dr_source_dir="$1"
  sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$_dr_source_dir/package.json" 2>/dev/null | head -n1
}

dr_release_tag() {
  _dr_source_dir="$1"
  if [ -n "${DEVRITES_REF:-}" ]; then _dr_version="${DEVRITES_REF#v}"; else _dr_version="$(dr_pkg_version "$_dr_source_dir")"; fi
  printf '%s\n' "$_dr_version" | LC_ALL=C grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' || return 1
  _dr_pre="${_dr_version%%+*}"
  case "$_dr_pre" in *-*) _dr_pre="${_dr_pre#*-}" ;; *) _dr_pre="" ;; esac
  if [ -n "$_dr_pre" ]; then
    printf '%s\n' "$_dr_pre" | awk -F. '{ for (i=1;i<=NF;i++) if ($i ~ /^[0-9]+$/ && length($i)>1 && substr($i,1,1)=="0") exit 1 }' || return 1
  fi
  printf 'v%s\n' "$_dr_version"
}

dr_valid_repo() {
  printf '%s\n' "$1" | LC_ALL=C grep -Eq '^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$'
}

dr_bounded_download() {
  _dr_url="$1"; _dr_out="$2"; _dr_limit="$3"
  DR_DOWNLOAD_FAILURE=""
  rm -f "$_dr_out" "$_dr_out.part"
  curl -fL --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 10 --max-time 120 \
    "$_dr_url" 2>/dev/null | head -c "$((_dr_limit + 1))" > "$_dr_out.part"
  _dr_pipeline_status=("${PIPESTATUS[@]}")
  _dr_curl_status="${_dr_pipeline_status[0]}"
  _dr_head_status="${_dr_pipeline_status[1]}"
  _dr_bytes="$(wc -c < "$_dr_out.part" 2>/dev/null)" || {
    DR_DOWNLOAD_FAILURE="local write"
    rm -f "$_dr_out.part"
    return 1
  }
  if [ "$_dr_bytes" -gt "$_dr_limit" ]; then
    DR_DOWNLOAD_FAILURE="size limit"
    rm -f "$_dr_out.part"
    return 1
  fi
  if [ "$_dr_head_status" -ne 0 ]; then
    DR_DOWNLOAD_FAILURE="local write"
    rm -f "$_dr_out.part"
    return 1
  fi
  if [ "$_dr_curl_status" -ne 0 ]; then
    case "$_dr_curl_status" in
      22) DR_DOWNLOAD_FAILURE="HTTP status" ;;
      28) DR_DOWNLOAD_FAILURE="timeout" ;;
      47) DR_DOWNLOAD_FAILURE="redirect" ;;
      *) DR_DOWNLOAD_FAILURE="download" ;;
    esac
    rm -f "$_dr_out.part"
    return 1
  fi
  mv "$_dr_out.part" "$_dr_out" || { DR_DOWNLOAD_FAILURE="local write"; rm -f "$_dr_out.part"; return 1; }
}

dr_verify_checksum() {
  _dr_file="$1"; _dr_sidecar="$2"; _dr_asset="$3"
  _dr_want="$(awk -v asset="$_dr_asset" '
    NF == 0 { next }
    { records++ }
    NF == 2 && length($1) == 64 && $1 ~ /^[0-9A-Fa-f]+$/ && $2 == asset { valid++; hash=tolower($1) }
    END { if (records == 1 && valid == 1) print hash; else exit 1 }
  ' "$_dr_sidecar" 2>/dev/null)" || return 1
  if command -v shasum >/dev/null 2>&1; then
    _dr_got="$(shasum -a 256 "$_dr_file" | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    _dr_got="$(sha256sum "$_dr_file" | awk '{print $1}')"
  else
    return 1
  fi
  [ "$_dr_got" = "$_dr_want" ]
}

dr_download_engine() {
  _dr_source_dir="$1"
  _dr_repo="$2"
  _dr_out="$3"
  dr_valid_repo "$_dr_repo" || { DR_ACQUIRE_FAILURE="repository name validation failed"; rm -f "$_dr_out" "$_dr_out.sha256"; return 1; }
  _dr_tag="$(dr_release_tag "$_dr_source_dir")"
  [ -n "$_dr_tag" ] || { DR_ACQUIRE_FAILURE="release tag validation failed"; rm -f "$_dr_out" "$_dr_out.sha256"; return 1; }
  _dr_os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
  _dr_arch="$(uname -m 2>/dev/null)"
  case "$_dr_arch" in
    x86_64|amd64) _dr_arch=amd64 ;;
    arm64|aarch64) _dr_arch=arm64 ;;
    *) DR_ACQUIRE_FAILURE="release $_dr_tag has no asset for architecture $_dr_arch"; return 1 ;;
  esac
  case "$_dr_os" in
    linux|darwin) : ;;
    *) DR_ACQUIRE_FAILURE="release $_dr_tag has no asset for operating system $_dr_os"; return 1 ;;
  esac

  _dr_asset="devrites-$_dr_os-$_dr_arch"
  _dr_url="https://github.com/$_dr_repo/releases/download/$_dr_tag/$_dr_asset"
  dr_bounded_download "$_dr_url" "$_dr_out" 67108864 || { DR_ACQUIRE_FAILURE="release $_dr_tag asset $_dr_asset: ${DR_DOWNLOAD_FAILURE:-download} failed"; return 1; }
  dr_bounded_download "$_dr_url.sha256" "$_dr_out.sha256" 4096 || { DR_ACQUIRE_FAILURE="release $_dr_tag asset $_dr_asset.sha256: ${DR_DOWNLOAD_FAILURE:-download} failed"; rm -f "$_dr_out"; return 1; }
  dr_verify_checksum "$_dr_out" "$_dr_out.sha256" "$_dr_asset" || { DR_ACQUIRE_FAILURE="release $_dr_tag asset $_dr_asset: checksum failed"; rm -f "$_dr_out" "$_dr_out.sha256"; return 1; }
  chmod +x "$_dr_out" || { DR_ACQUIRE_FAILURE="release $_dr_tag asset $_dr_asset: executable preparation failed"; rm -f "$_dr_out" "$_dr_out.sha256"; return 1; }
}

dr_build_engine() {
  _dr_source_dir="$1"
  _dr_out="$2"
  _dr_go_bin="$(command -v go 2>/dev/null || true)"
  [ -d "$_dr_source_dir/engine" ] && [ -n "$_dr_go_bin" ] || return 1
  _dr_tag="$(dr_release_tag "$_dr_source_dir")"
  [ -n "$_dr_tag" ] || _dr_tag="dev"
  ( cd "$_dr_source_dir/engine" && GOCACHE="${GOCACHE:-$(dirname "$_dr_out")/go-cache}" CGO_ENABLED=0 "$_dr_go_bin" build -trimpath -ldflags "-s -w -X github.com/devrites/devrites/internal/version.Version=$_dr_tag" -o "$_dr_out" . ) 2>/dev/null || return 1
  chmod +x "$_dr_out" 2>/dev/null || true
}

dr_engine_supports() {
  _dr_engine="$1"
  _dr_subcommand="$2"
  "$_dr_engine" "$_dr_subcommand" --help >/dev/null 2>&1
}

dr_engine_compatible() {
  _dr_engine="$1"
  _dr_source_dir="$2"
  _dr_subcommand="$3"
  dr_engine_supports "$_dr_engine" "$_dr_subcommand" || return 1
  case "$_dr_subcommand" in
    install|update)
      _dr_want="$(dr_release_tag "$_dr_source_dir")"
      [ -n "$_dr_want" ] || return 1
      _dr_got="$("$_dr_engine" version 2>/dev/null)"
      [ "${_dr_got#v}" = "${_dr_want#v}" ] || return 1
      ;;
  esac
  return 0
}

dr_acquire_engine() {
  _dr_source_dir="$1"
  _dr_subcommand="$2"
  _dr_repo="$3"

  DR_ENGINE_PATH=""
  DR_ENGINE_TMP=""
  DR_ACQUIRE_FAILURE=""
  if [ -n "${DEVRITES_ENGINE_CLI:-}" ] && [ -x "$DEVRITES_ENGINE_CLI" ]; then
    dr_engine_compatible "$DEVRITES_ENGINE_CLI" "$_dr_source_dir" "$_dr_subcommand" && { DR_ENGINE_PATH="$DEVRITES_ENGINE_CLI"; return 0; }
  fi

  case "$_dr_subcommand" in
    install|update) _dr_prefer_fresh=1 ;;
    *) _dr_prefer_fresh=0 ;;
  esac

  if [ "$_dr_prefer_fresh" -eq 0 ] && command -v devrites-engine >/dev/null 2>&1; then
    _dr_path="$(command -v devrites-engine)"
    dr_engine_compatible "$_dr_path" "$_dr_source_dir" "$_dr_subcommand" && { DR_ENGINE_PATH="$_dr_path"; return 0; }
  fi

  _dr_old_umask="$(umask)"
  umask 077
  DR_ENGINE_TMP="$(mktemp -d "${TMPDIR:-/tmp}/devrites-engine-bootstrap.XXXXXX" 2>/dev/null)" || {
    umask "$_dr_old_umask"
    DR_ENGINE_TMP=""
    return 1
  }
  umask "$_dr_old_umask"
  if command -v curl >/dev/null 2>&1; then
    if dr_download_engine "$_dr_source_dir" "$_dr_repo" "$DR_ENGINE_TMP/devrites-engine"; then
      dr_engine_compatible "$DR_ENGINE_TMP/devrites-engine" "$_dr_source_dir" "$_dr_subcommand" && { DR_ENGINE_PATH="$DR_ENGINE_TMP/devrites-engine"; return 0; }
    fi
  fi

  if dr_build_engine "$_dr_source_dir" "$DR_ENGINE_TMP/devrites-engine"; then
    dr_engine_compatible "$DR_ENGINE_TMP/devrites-engine" "$_dr_source_dir" "$_dr_subcommand" && { DR_ENGINE_PATH="$DR_ENGINE_TMP/devrites-engine"; return 0; }
  fi

  if [ "$_dr_prefer_fresh" -eq 1 ] && command -v devrites-engine >/dev/null 2>&1; then
    _dr_path="$(command -v devrites-engine)"
    dr_engine_compatible "$_dr_path" "$_dr_source_dir" "$_dr_subcommand" && { rm -rf "$DR_ENGINE_TMP"; DR_ENGINE_TMP=""; DR_ENGINE_PATH="$_dr_path"; return 0; }
  fi

  rm -rf "$DR_ENGINE_TMP"
  DR_ENGINE_TMP=""
  return 1
}

dr_cleanup_engine() {
  [ -z "${DR_ENGINE_TMP:-}" ] || rm -rf "$DR_ENGINE_TMP"
  DR_ENGINE_TMP=""
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
# Used to refuse global writes. No write verbs here: read-only comparison.
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

# /$_name: alias of /$_to

This command **delegates to DevRites \`/$_to\`**. State that to the user, then load and
run \`$_to/SKILL.md\` with the given arguments, following it exactly.
EOF
}
