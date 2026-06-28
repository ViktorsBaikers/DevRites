#!/usr/bin/env bash
# install.sh — install the DevRites skills pack into a project (project-local only).
#
#   ./install.sh                      install into the current directory
#   ./install.sh --target DIR         install into DIR
#   ./install.sh --dry-run            show planned operations, change nothing
#   ./install.sh --force              overwrite existing non-DevRites files
#   ./install.sh --no-agents          skip the review subagents
#   ./install.sh --no-rules           skip the DevRites engineering rules
#   ./install.sh --rules-only         install only the engineering rules
#   ./install.sh --short-aliases=all  install /define /build /prove /seal aliases
#
# Network install (no git clone needed):
#   curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash -s -- --target /path
#
# Never writes to ~/.claude. Records every installed file in
# .claude/devrites.manifest so uninstall removes exactly what it added.

set -u

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd -P)" || SELF_DIR=""

# ---- bootstrap (curl | bash) --------------------------------------------
# When run via `curl ... | bash`, the script has no sibling files. Detect that
# and re-exec ourselves from a downloaded release tarball so the rest of the
# script (which needs scripts/install-lib.sh + pack/.claude) just works.
DEVRITES_REPO="${DEVRITES_REPO:-ViktorsBaikers/DevRites}"
DEVRITES_REF="${DEVRITES_REF:-}"  # optional override: a release tag or branch
if [ -z "$SELF_DIR" ] || [ ! -d "$SELF_DIR/pack/.claude" ] || [ ! -r "$SELF_DIR/scripts/install-lib.sh" ]; then
  if [ "${DEVRITES_BOOTSTRAPPED:-0}" = "1" ]; then
    echo "error: bootstrap re-exec did not find pack/ — aborting to avoid a loop." >&2
    exit 1
  fi
  command -v curl >/dev/null 2>&1 || { echo "error: curl is required for the network installer." >&2; exit 1; }
  command -v tar  >/dev/null 2>&1 || { echo "error: tar is required for the network installer."  >&2; exit 1; }

  BOOT_TMP="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-bootstrap.$$")"
  mkdir -p "$BOOT_TMP"
  # Resolve ref → latest release tag, unless DEVRITES_REF is set.
  if [ -n "$DEVRITES_REF" ]; then
    BOOT_TAG="$DEVRITES_REF"
  else
    BOOT_TAG="$(curl -fsSL "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  fi

  BOOT_DOWNLOADED=0
  BOOT_FROM_RELEASE=0   # 1 only when we fetched the published release artifact (the one with a .sha256)
  if [ -n "$BOOT_TAG" ]; then
    # 1) try the published release artifact (tarball produced by build-release-tarball.sh)
    BOOT_URL="https://github.com/$DEVRITES_REPO/releases/download/$BOOT_TAG/devrites-$BOOT_TAG.tar.gz"
    if curl -fsSL -o "$BOOT_TMP/devrites.tar.gz" "$BOOT_URL" 2>/dev/null; then
      BOOT_DOWNLOADED=1; BOOT_FROM_RELEASE=1
    else
      # 2) fall back to the tag's source tarball
      BOOT_URL="https://github.com/$DEVRITES_REPO/archive/refs/tags/$BOOT_TAG.tar.gz"
      curl -fsSL -o "$BOOT_TMP/devrites.tar.gz" "$BOOT_URL" 2>/dev/null && BOOT_DOWNLOADED=1
    fi
  fi
  if [ "$BOOT_DOWNLOADED" -ne 1 ]; then
    # 3) no release yet (or refs failed) → grab main
    BOOT_URL="https://github.com/$DEVRITES_REPO/archive/refs/heads/main.tar.gz"
    curl -fsSL -o "$BOOT_TMP/devrites.tar.gz" "$BOOT_URL" || { echo "error: could not download DevRites from $BOOT_URL" >&2; exit 1; }
  fi

  # Best-effort checksum verification for the published release artifact: if the
  # sibling .sha256 asset is present, the tarball MUST match it; if it's absent
  # (older release, offline mirror, source/main fallback) we warn and proceed.
  if [ "$BOOT_FROM_RELEASE" -eq 1 ]; then
    BOOT_SUM_URL="https://github.com/$DEVRITES_REPO/releases/download/$BOOT_TAG/devrites-$BOOT_TAG.tar.gz.sha256"
    if curl -fsSL -o "$BOOT_TMP/devrites.tar.gz.sha256" "$BOOT_SUM_URL" 2>/dev/null; then
      BOOT_WANT="$(awk '{print $1; exit}' "$BOOT_TMP/devrites.tar.gz.sha256" 2>/dev/null)"
      if command -v shasum >/dev/null 2>&1; then
        BOOT_GOT="$(shasum -a 256 "$BOOT_TMP/devrites.tar.gz" | awk '{print $1}')"
      elif command -v sha256sum >/dev/null 2>&1; then
        BOOT_GOT="$(sha256sum "$BOOT_TMP/devrites.tar.gz" | awk '{print $1}')"
      else
        BOOT_GOT=""
      fi
      if [ -z "$BOOT_GOT" ]; then
        echo "DevRites: warning: no sha256 tool found; skipping checksum verification." >&2
      elif [ "$BOOT_GOT" = "$BOOT_WANT" ]; then
        echo "DevRites: checksum verified (sha256)."
      else
        echo "error: checksum mismatch for devrites-$BOOT_TAG.tar.gz — refusing to install." >&2
        echo "  expected: $BOOT_WANT" >&2
        echo "  got:      $BOOT_GOT" >&2
        exit 1
      fi
    else
      echo "DevRites: warning: no .sha256 asset for $BOOT_TAG; skipping checksum verification." >&2
    fi
  fi

  tar -C "$BOOT_TMP" -xzf "$BOOT_TMP/devrites.tar.gz" || { echo "error: could not extract DevRites tarball" >&2; exit 1; }
  BOOT_BUNDLE="$(find "$BOOT_TMP" -mindepth 1 -maxdepth 1 -type d | head -n1)"
  if [ -z "$BOOT_BUNDLE" ] || [ ! -f "$BOOT_BUNDLE/install.sh" ]; then
    echo "error: extracted bundle is missing install.sh: $BOOT_BUNDLE" >&2
    exit 1
  fi
  # GitHub source tarballs strip the executable bit; restore it for our scripts.
  chmod +x "$BOOT_BUNDLE/install.sh" "$BOOT_BUNDLE/uninstall.sh" "$BOOT_BUNDLE/update.sh" 2>/dev/null || true
  chmod +x "$BOOT_BUNDLE"/scripts/*.sh 2>/dev/null || true

  echo "DevRites: bootstrapped from ${BOOT_TAG:-main} → $BOOT_BUNDLE"
  export DEVRITES_BOOTSTRAPPED=1
  exec bash "$BOOT_BUNDLE/install.sh" "$@"
fi

# shellcheck source=scripts/install-lib.sh
. "$SELF_DIR/scripts/install-lib.sh"

PACK_SRC="$SELF_DIR/pack/.claude"

# ---- defaults ------------------------------------------------------------
TARGET="$PWD"
DRYRUN=0
FORCE=0
WITH_SKILLS=1
WITH_AGENTS=1
WITH_RULES=1
ALIAS_MODE="safe"     # safe | off | all

# ---- parse args ----------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --target) shift; [ $# -gt 0 ] || dr_die "--target needs a directory"; TARGET="$1" ;;
    --target=*) TARGET="${1#*=}" ;;
    --dry-run) DRYRUN=1 ;;
    --force) FORCE=1 ;;
    --no-skills) WITH_SKILLS=0 ;;                 # accepted so a --rules-only manifest round-trips through update.sh's flag replay
    --no-agents) WITH_AGENTS=0 ;;
    --no-rules) WITH_RULES=0 ;;
    --rules-only) WITH_SKILLS=0; WITH_AGENTS=0; WITH_RULES=1; ALIAS_MODE="off" ;;
    --no-short-aliases) ALIAS_MODE="off" ;;       # accepted for backward compat; aliases are off by default now
    --short-aliases) dr_warn "--short-aliases with no value is a no-op (aliases default off); use --short-aliases=all to install /define /build /prove /seal."; ALIAS_MODE="off" ;;  # /polish + /normalize were removed in favor of /rite-polish argument-hint modes
    --short-aliases=all) ALIAS_MODE="all" ;;
    -h|--help) sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) dr_die "unknown option: $1 (try --help)" ;;
  esac
  shift
done

# ---- validate source + target -------------------------------------------
[ -d "$PACK_SRC/skills" ] || dr_die "pack not found at $PACK_SRC (run from the DevRites repo)"
[ -e "$TARGET" ] || dr_die "target does not exist: $TARGET"
[ -d "$TARGET" ] || dr_die "target is not a directory: $TARGET"
TARGET="$(dr_abspath_dir "$TARGET")" || dr_die "cannot resolve target: $TARGET"

# GUARD:no-global — refuse to install into the user's global ~/.claude tree.
TARGET_CLAUDE="$TARGET/.claude"
if dr_is_global_claude "$TARGET_CLAUDE" || [ "$TARGET" = "$HOME" ]; then
  dr_die "refusing to target the global ~/.claude. DevRites is project-local; choose a project directory."
fi

PREV_MANIFEST="$TARGET/$DR_MANIFEST_NAME"
MANIFEST_TMP="$(mktemp 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites.manifest.$$")"
: > "$MANIFEST_TMP"
TMP_GEN_DIR="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-gen.$$")"
mkdir -p "$TMP_GEN_DIR"
cleanup() { rm -f "$MANIFEST_TMP"; rm -rf "$TMP_GEN_DIR"; }
trap cleanup EXIT

N_INSTALL=0; N_OVERWRITE=0; N_SKIP=0; N_PRUNE=0

# install_file SRC RELDEST — copy one file, honoring conflict + manifest rules.
install_file() {
  _src="$1"; _rel="$2"; _dest="$TARGET/$_rel"
  if [ -e "$_dest" ]; then
    if dr_manifest_contains "$PREV_MANIFEST" "$_rel"; then
      _action="overwrite"
    elif [ "$FORCE" -eq 1 ]; then
      _action="overwrite(force)"
    else
      [ "$DRYRUN" -eq 1 ] && dr_say "  [skip] $_rel ${DR_Y}(exists, not DevRites-managed)${DR_R}"
      [ "$DRYRUN" -eq 1 ] || dr_warn "skip $_rel (exists, not DevRites-managed; use --force to overwrite)"
      N_SKIP=$((N_SKIP+1))
      return 0
    fi
  else
    _action="install"
  fi

  if [ "$DRYRUN" -eq 1 ]; then
    dr_say "  [$_action] $_rel"
  else
    mkdir -p "$(dirname "$_dest")" || dr_die "cannot create $(dirname "$_dest")"
    cp "$_src" "$_dest" || dr_die "cannot write $_dest"
  fi
  printf '%s\n' "$_rel" >> "$MANIFEST_TMP"
  case "$_action" in
    install) N_INSTALL=$((N_INSTALL+1)) ;;
    *) N_OVERWRITE=$((N_OVERWRITE+1)) ;;
  esac
}

# install_tree SRC_DIR REL_PREFIX — install every file under SRC_DIR.
install_tree() {
  _sd="$1"; _pre="$2"
  [ -d "$_sd" ] || return 0
  while IFS= read -r f; do
    _r="${f#$_sd/}"
    install_file "$f" "$_pre/$_r"
  done < <(find "$_sd" -type f)
}

# ---- alias wrapper generator (for --short-aliases=all) -------------------
# Thin pass-through to the shared template in install-lib.sh (also used by
# scripts/pin.sh for ad-hoc user pins).
gen_alias_wrapper() { dr_gen_alias_wrapper "$@"; }

# ---- plan + execute ------------------------------------------------------
dr_info "DevRites installer"
dr_say  "  source : $SELF_DIR"
dr_say  "  target : $TARGET"
dr_say  "  skills : $([ "$WITH_SKILLS" -eq 1 ] && echo yes || echo no)"
dr_say  "  rules  : $([ "$WITH_RULES" -eq 1 ] && echo 'DevRites engineering rules' || echo 'skipped')"
dr_say  "  agents : $([ "$WITH_AGENTS" -eq 1 ] && echo yes || echo no)"
dr_say  "  aliases: $ALIAS_MODE"
[ "$DRYRUN" -eq 1 ] && dr_info "  (dry run — no changes will be made)"
dr_say ""

# 1) skills — every dir under pack/.claude/skills/ (the deleted /polish and
# /normalize aliases are no longer present; /rite-polish carries the modes via
# argument-hint).
if [ "$WITH_SKILLS" -eq 1 ]; then
  while IFS= read -r d; do
    name="$(basename "$d")"
    install_tree "$d" ".claude/skills/$name"
  done < <(find "$PACK_SRC/skills" -mindepth 1 -maxdepth 1 -type d)
fi

# 2) optional --short-aliases=all wrappers for /define /build /prove /seal.
if [ "$WITH_SKILLS" -eq 1 ] && [ "$ALIAS_MODE" = "all" ]; then
  for pair in "define:rite-define" "build:rite-build" "prove:rite-prove" "seal:rite-seal"; do
    a="${pair%%:*}"; to="${pair#*:}"
    gen_alias_wrapper "$a" "$to" "$TMP_GEN_DIR/$a.md"
    install_file "$TMP_GEN_DIR/$a.md" ".claude/skills/$a/SKILL.md"
  done
fi

# 3) agents
if [ "$WITH_AGENTS" -eq 1 ]; then
  install_tree "$PACK_SRC/agents" ".claude/agents"
fi

# 4) DevRites engineering rules
if [ "$WITH_RULES" -eq 1 ]; then
  install_tree "$PACK_SRC/rules" ".claude/rules"
fi

# 4b) hooks — auto-approve the read-only orientation/gate scripts (no per-run permission
# prompts) + a SessionStart orientation injector. Ship the scripts (manifest-managed, updated
# on reinstall); SEED .claude/settings.json only when absent and never record it in the
# manifest, so it is preserved on uninstall/update and the user's own settings stay safe.
if [ "$WITH_SKILLS" -eq 1 ] && [ -d "$PACK_SRC/hooks" ]; then
  install_tree "$PACK_SRC/hooks" ".claude/hooks"
  if [ ! -e "$TARGET/.claude/settings.json" ]; then
    if [ "$DRYRUN" -eq 1 ]; then
      dr_say "  [seed] .claude/settings.json ${DR_Y}(DevRites hooks — preserved on uninstall/update)${DR_R}"
    else
      mkdir -p "$TARGET/.claude"; cp "$PACK_SRC/settings.json" "$TARGET/.claude/settings.json"
    fi
  else
    dr_warn "skip .claude/settings.json (exists) — to silence the per-run script prompts, add the \"hooks\" block from $PACK_SRC/settings.json to your settings, or run /hooks"
  fi
fi

# 5) .devrites seed (README always managed; ACTIVE seeded but never manifest-managed)
DEV_README="$TMP_GEN_DIR/devrites-readme.md"
cat > "$DEV_README" <<'EOF'
# .devrites/ — DevRites runtime state

This directory holds DevRites feature workspaces. Each feature lives in
`work/<feature-slug>/` as human-readable Markdown (spec, plan, tasks, state,
evidence, drift, decisions, review, seal). It survives context compaction and new
sessions — it's the durable memory of in-flight work.

- `ACTIVE` names the currently active feature slug (or is empty).
- `work/<slug>/` is created by `/rite-define`; other phases read it.
- `AFK` (optional, per-developer) — presence flips the session to AFK mode.
  Empty file = safe defaults; optional YAML body sets `max_slices`, `notify`,
  `allow_gates`. Full contract: `.claude/rules/afk-hitl.md`.
  **Gitignore `AFK` unless your team agrees on shared AFK defaults** — it's a
  local toggle, not project state.
- `conventions.md` (optional) — the conventions ledger: facts a sealed slice proved
  about this codebase (commands, idioms, placement, gotchas), written on a GO seal and
  read at orient so the wright stops re-deriving them. The band is earned from how many
  slices corroborated each entry. **Gitignore `conventions.md`** — it's local, per-clone
  learning, not shared project state.

Safe to commit `work/` so the team and future sessions share feature state.
Delete a feature's folder when it's shipped and you no longer need the trail.
EOF
install_file "$DEV_README" ".devrites/README.md"
# .devrites/ACTIVE is runtime state: seed it if absent, but NEVER record it in the
# manifest, so uninstall preserves it (and work/) like the feature data it is.
if [ ! -e "$TARGET/.devrites/ACTIVE" ]; then
  if [ "$DRYRUN" -eq 1 ]; then
    dr_say "  [seed] .devrites/ACTIVE ${DR_Y}(runtime state — preserved on uninstall)${DR_R}"
  else
    mkdir -p "$TARGET/.devrites"; : > "$TARGET/.devrites/ACTIVE"
  fi
fi

# ---- detect installed version + flags (recorded in manifest header) -----
DR_INSTALLED_VERSION="unknown"
_pkg_json="$SELF_DIR/package.json"
if [ -r "$_pkg_json" ]; then
  # Best-effort: extract the top-level "version" field without depending on jq.
  # package.json's first "version" key is the top-level one (devDependency keys
  # are package names, not "version"), so head -n1 is safe.
  DR_INSTALLED_VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$_pkg_json" | head -n1)"
  [ -n "$DR_INSTALLED_VERSION" ] || DR_INSTALLED_VERSION="unknown"
fi

DR_INSTALL_FLAGS=""
[ "$WITH_SKILLS" -eq 0 ] && DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --no-skills"
[ "$WITH_AGENTS" -eq 0 ] && DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --no-agents"
[ "$WITH_RULES"  -eq 0 ] && DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --no-rules"
case "$ALIAS_MODE" in
  off) DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --no-short-aliases" ;;
  all) DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --short-aliases=all" ;;
esac
DR_INSTALL_FLAGS="$(printf '%s' "$DR_INSTALL_FLAGS" | sed 's/^ //')"

# ---- prune files dropped since the previous install ----------------------
# install_file only adds/overwrites; a file the prior manifest tracked but the
# current pack no longer ships would orphan on an in-place update (the path
# update.sh takes: download release → re-run install.sh --force). Remove
# exactly those: managed paths only (listed in the OLD manifest, absent from the
# new one), never unmanaged user files and never .devrites/ runtime state.
if [ -f "$PREV_MANIFEST" ]; then
  while IFS= read -r _rel; do
    case "$_rel" in ''|\#*) continue ;; esac           # skip comments + blank lines
    case "$_rel" in .devrites/*) continue ;; esac       # never touch runtime state
    grep -Fxq -- "$_rel" "$MANIFEST_TMP" && continue    # still shipped → keep
    _dead="$TARGET/$_rel"
    [ -e "$_dead" ] || continue                         # already gone
    if [ "$DRYRUN" -eq 1 ]; then
      dr_say "  [prune] $_rel ${DR_Y}(dropped from pack)${DR_R}"
    else
      rm -f "$_dead"
      # tidy now-empty parent dirs, bounded strictly below .claude/
      _d="$(dirname "$_dead")"
      while [ "$_d" != "$TARGET/.claude" ] && [ "$_d" != "$TARGET" ] && [ "$_d" != "/" ]; do
        rmdir "$_d" 2>/dev/null || break
        _d="$(dirname "$_d")"
      done
      dr_say "  [prune] $_rel"
    fi
    N_PRUNE=$((N_PRUNE+1))
  done < "$PREV_MANIFEST"
fi

# ---- write manifest ------------------------------------------------------
if [ "$DRYRUN" -eq 0 ]; then
  mkdir -p "$TARGET/.claude"
  {
    echo "# DevRites install manifest — do not edit by hand."
    echo "# Generated $(date -u '+%Y-%m-%dT%H:%M:%SZ'). Uninstall removes exactly these paths."
    echo "# devrites-version: $DR_INSTALLED_VERSION"
    echo "# devrites-flags: $DR_INSTALL_FLAGS"
    sort -u "$MANIFEST_TMP"
  } > "$PREV_MANIFEST"
fi

# ---- summary -------------------------------------------------------------
dr_say ""
dr_ok "DevRites $([ "$DRYRUN" -eq 1 ] && echo 'plan complete (dry run)' || echo installed)"
dr_say "  installed: $N_INSTALL   overwritten: $N_OVERWRITE   skipped(conflict): $N_SKIP   pruned: $N_PRUNE"
dr_say ""
dr_say "${DR_B}Commands:${DR_R} /rite  /rite-spec  /rite-define  /rite-plan  /rite-build  /rite-prove  /rite-polish  /rite-review  /rite-seal  /rite-status"
dr_say "${DR_B}Utilities:${DR_R} /rite-pressure-test  /rite-zoom-out  /rite-prototype  /rite-handoff"
if [ "$ALIAS_MODE" = "all" ]; then
  dr_say "${DR_B}Aliases:${DR_R}  /define  /build  /prove  /seal"
fi
[ "$WITH_RULES" -eq 1 ] && dr_say "${DR_B}Rules:${DR_R}    engineering rules installed to .claude/rules/"
dr_say ""
dr_say "${DR_B}Next:${DR_R}"
dr_say "  1. Open the project in Claude Code and accept the workspace trust prompt."
dr_say "  2. Run ${DR_C}/rite${DR_R} for the menu, or ${DR_C}/rite-spec <feature>${DR_R} to start."
dr_say "  3. ${DR_B}HITL by default${DR_R} — HITL slices pause at typed gates; resume with ${DR_C}/rite-resolve <qid> \"<answer>\"${DR_R}."
dr_say "     For unattended runs: ${DR_C}touch .devrites/AFK${DR_R} (see .claude/rules/afk-hitl.md for the contract)."
[ "$N_SKIP" -gt 0 ] && dr_say "  • $N_SKIP file(s) were skipped (already present). Re-run with --force to overwrite."
[ "$DRYRUN" -eq 1 ] && dr_say "  (dry run only — re-run without --dry-run to apply.)"
exit 0
