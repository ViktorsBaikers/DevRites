#!/usr/bin/env bash
# install.sh — install the DevRites skills pack into a project (project-local only).
#
#   ./install.sh                      install into the current directory
#   ./install.sh --target DIR         install into DIR
#   ./install.sh --dry-run            show planned operations, change nothing
#   ./install.sh --force              overwrite existing non-DevRites files
#   ./install.sh --no-agents          skip the review subagents
#   ./install.sh --no-codex           skip Codex support files (.agents, .codex, AGENTS.md)
#   ./install.sh --no-rules           deprecated no-op (standards ship inside the devrites-lib skill)
#   ./install.sh --rules-only         deprecated no-op (standards ship inside the devrites-lib skill)
#   ./install.sh --short-aliases=all  install /define /build /prove /seal aliases
#
# Network install (no git clone needed):
#   curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash -s -- --target /path
#
# Never writes to ~/.claude or ~/.codex. Records every installed file in
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
WITH_CODEX=1
WITH_BINARY=1         # install the global devrites-engine binary (see install_binary)
ALIAS_MODE="safe"     # safe | off | all

# ---- parse args ----------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --target) shift; [ $# -gt 0 ] || dr_die "--target needs a directory"; TARGET="$1" ;;
    --target=*) TARGET="${1#*=}" ;;
    --dry-run) DRYRUN=1 ;;
    --force) FORCE=1 ;;
    --no-skills) WITH_SKILLS=0 ;;
    --no-agents) WITH_AGENTS=0 ;;
    --no-codex) WITH_CODEX=0 ;;
    --no-binary) WITH_BINARY=0 ;;               # skip the global engine binary (hooks fail open without it)
    --no-rules|--rules-only) dr_warn "$1 is deprecated and now a no-op — DevRites engineering standards ship inside the devrites-lib skill (installed with --skills)." ;;
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

# GUARD:no-global — refuse to install into global agent configuration trees.
TARGET_CLAUDE="$TARGET/.claude"
TARGET_CODEX="$TARGET/.codex"
if dr_is_global_claude "$TARGET_CLAUDE" || dr_is_global_codex "$TARGET_CODEX" || [ "$TARGET" = "$HOME" ]; then
  dr_die "refusing to target a global agent home. DevRites is project-local; choose a project directory."
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

install_marker_file() {
  _rel="$1"; _text="$2"; _dest="$TARGET/$_rel"
  if [ "$DRYRUN" -eq 1 ]; then
    dr_say "  [install] $_rel"
  else
    mkdir -p "$(dirname "$_dest")" || dr_die "cannot create $(dirname "$_dest")"
    if [ -f "$_dest" ] && printf '%s\n' "$_text" | cmp -s - "$_dest"; then
      printf '%s\n' "$_rel" >> "$MANIFEST_TMP"
      return 0
    fi
    printf '%s\n' "$_text" > "$_dest" || dr_die "cannot write $_dest"
  fi
  printf '%s\n' "$_rel" >> "$MANIFEST_TMP"
  N_INSTALL=$((N_INSTALL+1))
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

gen_codex_markdown_file() {
  local _src="$1" _out="$2"
  mkdir -p "$(dirname "$_out")"
  sed -E \
    -e 's#pack/\.claude/skills/devrites-lib/scripts/#.agents/skills/devrites-lib/scripts/#g' \
    -e 's#pack/\.claude/skills/#.agents/skills/#g' \
    -e 's#pack/\.claude/agents/(devrites-[A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#pack/\.claude/agents/#.codex/agents/#g' \
    -e 's#\.claude/skills/devrites-lib/scripts/#.agents/skills/devrites-lib/scripts/#g' \
    -e 's#\.claude/skills/#.agents/skills/#g' \
    -e 's#\.claude/agents/(devrites-[A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#\.claude/agents/#.codex/agents/#g' \
    -e 's#\.\./\.\./\.\./agents/(devrites-[A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#\.\./\.\./agents/(devrites-[A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#(^|[^A-Za-z0-9_./-])/(rite(-[a-z0-9-]+)?)([^A-Za-z0-9_-]|$)#\1$\2\4#g' \
    "$_src" > "$_out"
}

strip_codex_agents_md_block() {
  _file="$1"
  [ -f "$_file" ] || return 0
  _tmp="$TMP_GEN_DIR/AGENTS.strip.md"
  awk '
    $0 == "<!-- BEGIN DEVRITES CODEX -->" { in_block = 1; next }
    $0 == "<!-- END DEVRITES CODEX -->" { in_block = 0; next }
    in_block { next }
    { print }
  ' "$_file" > "$_tmp" || dr_die "cannot strip DevRites block from $_file"
  cp "$_tmp" "$_file" || dr_die "cannot write $_file"
  grep -q '[^[:space:]]' "$_file" 2>/dev/null || rm -f "$_file"
}

strip_codex_config_block() {
  _file="$1"
  [ -f "$_file" ] || return 0
  _tmp="$TMP_GEN_DIR/codex-config.strip.toml"
  awk '
    $0 == "# BEGIN DEVRITES CODEX MCP" { in_block = 1; next }
    $0 == "# END DEVRITES CODEX MCP" { in_block = 0; next }
    in_block { next }
    { print }
  ' "$_file" > "$_tmp" || dr_die "cannot strip DevRites MCP block from $_file"
  cp "$_tmp" "$_file" || dr_die "cannot write $_file"
  grep -q '[^[:space:]]' "$_file" 2>/dev/null || rm -f "$_file"
}

strip_codex_hooks_entries() {
  _file="$1"
  [ -f "$_file" ] || return 0
  _tmp="$TMP_GEN_DIR/codex-hooks.strip.json"
  command -v node >/dev/null 2>&1 || dr_die "node is required to merge existing .codex/hooks.json"
  node "$SELF_DIR/scripts/merge-codex-hooks.mjs" strip "$_file" "$_tmp" || dr_die "cannot strip DevRites hooks from $_file"
  cp "$_tmp" "$_file" || dr_die "cannot write $_file"
  [ "$(tr -d '[:space:]' < "$_file" 2>/dev/null)" != "{}" ] || rm -f "$_file"
}

gen_codex_skill_file() {
  _src="$1"; _out="$2"; _internal="${3:-0}"
  _tmp="$TMP_GEN_DIR/codex-skill-raw-$(basename "$(dirname "$_src")").md"
  mkdir -p "$(dirname "$_out")"
  # Internal (explicit-only) skills: replace the frontmatter description with a short stub in the
  # Codex mirror. Codex uses the description ONLY to build its 2% skills-list for implicit matching;
  # an explicit-only skill is never matched, so its full description is pure budget waste. Stubbing
  # the 12 internal skills frees that budget so the user-facing rite-* descriptions fit un-shortened
  # and keep matching reliably. Explicit `$name` invocation and the full SKILL.md body are untouched.
  awk -v internal="$_internal" '
    BEGIN { fm = 0; inserted = 0 }
    NR == 1 && $0 == "---" { print; fm = 1; next }
    fm && internal == "1" && /^description:/ {
      print "description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match."
      next
    }
    fm && $0 == "---" {
      print
      print ""
      print "## Codex compatibility"
      print ""
      print "This is the Codex mirror of a DevRites skill. In Codex:"
      print ""
      print "- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them."
      print "- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation."
      print "- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional."
      print "- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence."
      print "- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement."
      print ""
      fm = 0
      inserted = 1
      next
    }
    { print }
    END {
      if (!inserted) {
        print ""
        print "## Codex compatibility"
        print ""
        print "Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`, use matching `.codex/agents/devrites-*.toml` custom agents for DevRites subagent dispatch, and trust `.codex/hooks.json` with `/hooks` before relying on hooks."
      }
    }
  ' "$_src" > "$_tmp"
  gen_codex_markdown_file "$_tmp" "$_out"
}

install_codex_skill_tree() {
  _sd="$1"; _pre="$2"
  [ -d "$_sd" ] || return 0
  while IFS= read -r f; do
    _r="${f#$_sd/}"
    case "$_r" in
    SKILL.md)
      # Internal (user-invocable: false) skills are invoked EXPLICITLY by DevRites agents
      # ($devrites-*), never implicitly from a user prompt. For those, on the Codex mirror we (a)
      # stub the description (gen_codex_skill_file, to free the 2% skills-list budget for the
      # user-facing rite-* skills) and (b) emit `agents/openai.yaml` with allow_implicit_invocation:
      # false so Codex won't auto-fire an internal specialist from a stray prompt. Explicit $skill
      # invocation still works either way. This is the Codex analog of Claude's `user-invocable: false`.
      _internal=0
      if awk 'NR==1&&$0=="---"{fm=1;next} fm&&$0=="---"{exit} fm&&/^user-invocable:[[:space:]]*false/{u=1} END{exit !u}' "$f"; then
        _internal=1
      fi
      _gen="$TMP_GEN_DIR/codex-skill-$(basename "$_sd").md"
      gen_codex_skill_file "$f" "$_gen" "$_internal"
      install_file "$_gen" "$_pre/$_r"
      if [ "$_internal" -eq 1 ]; then
        _pol="$TMP_GEN_DIR/codex-policy-$(basename "$_sd").yaml"
        printf 'policy:\n  allow_implicit_invocation: false\n' > "$_pol"
        install_file "$_pol" "$_pre/agents/openai.yaml"
      fi
      ;;
    *.md)
      _gen="$TMP_GEN_DIR/codex-md-$(basename "$_sd")-$(printf '%s' "$_r" | tr '/ ' '__').md"
      gen_codex_markdown_file "$f" "$_gen"
      install_file "$_gen" "$_pre/$_r"
      ;;
    *)
      install_file "$f" "$_pre/$_r"
      ;;
    esac
  done < <(find "$_sd" -type f)
}

toml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

# Generate a Codex custom-agent TOML file from a Claude Code markdown agent.
# Codex does not consume .claude/agents/*.md directly; it loads project agents
# from .codex/agents/*.toml.
gen_codex_agent() {
  local _src="$1" _out="$2"
  local _name _desc _desc_tmp _desc_codex _tools _body_tmp _body_codex
  _name="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^name:[[:space:]]*/{sub(/^name:[[:space:]]*/, ""); print; exit}' "$_src")"
  _desc="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^description:[[:space:]]*/{sub(/^description:[[:space:]]*/, ""); print; exit}' "$_src")"
  [ -n "$_name" ] || _name="$(basename "$_src" .md)"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  _desc_tmp="$TMP_GEN_DIR/codex-agent-desc-$(basename "$_src").txt"
  _desc_codex="$TMP_GEN_DIR/codex-agent-desc-$(basename "$_src").codex.txt"
  printf '%s' "$_desc" > "$_desc_tmp"
  gen_codex_markdown_file "$_desc_tmp" "$_desc_codex"
  _desc="$(cat "$_desc_codex")"
  _tools="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^tools:[[:space:]]*/{sub(/^tools:[[:space:]]*/, ""); print; exit}' "$_src")"
  mkdir -p "$(dirname "$_out")"
  {
    printf 'name = "%s"\n' "$(toml_escape "$_name")"
    printf 'description = "%s"\n' "$(toml_escape "$_desc")"
    case "$_tools" in
      *Edit*|*Write*|*MultiEdit*) : ;;
      *) printf 'sandbox_mode = "read-only"\n' ;;
    esac
    printf "%s\n" "developer_instructions = '''"
    printf 'You are the Codex custom-agent version of DevRites `%s`.\n' "$_name"
    printf 'Follow the source agent instructions below. Treat any Claude Code-specific hook/tool metadata as unavailable in Codex unless the current session exposes an equivalent capability.\n\n'
    _body_tmp="$TMP_GEN_DIR/codex-agent-body-$(basename "$_src").md"
    _body_codex="$TMP_GEN_DIR/codex-agent-body-$(basename "$_src").codex.md"
    awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{fm=0; body=1; next} body{print}' "$_src" > "$_body_tmp"
    gen_codex_markdown_file "$_body_tmp" "$_body_codex"
    cat "$_body_codex"
    printf "\n%s\n" "'''"
  } > "$_out"
}

gen_codex_agents_bridge() {
  _out="$1"
  cat > "$_out" <<'EOF'
<!-- BEGIN DEVRITES CODEX -->
## DevRites For Codex

This project has DevRites installed for both Claude Code and Codex.

## Codex usage

- **Trust this project.** Codex skips every project-scoped `.codex/` layer — config, hooks, agents, rules — in an **untrusted** project. If DevRites seems absent (no skills, no custom agents, no MCP), trust the folder when Codex prompts on first run, or mark the project trusted in your global Codex config (`projects."<abs-path>".trust_level = "trusted"`). Untrusted = DevRites silently does nothing.
- DevRites workflow skills are available to Codex from `.agents/skills`.
- Use `$rite` or `$rite-<verb>` through Codex skills, or open `/skills` and select the matching DevRites skill.
- If the user mentions a DevRites slash command such as `/rite spec`, `/rite-build`, or `/rite-seal`, treat that as an explicit request to use the corresponding DevRites skill.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Before using any DevRites workflow skill, read `.agents/skills/devrites-lib/reference/standards/core.md`. Load other `.agents/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them. These are DevRites engineering standards, not Codex exec-policy `.rules` files.
- Custom Codex subagents generated from the DevRites review agents live in `.codex/agents`.
- When DevRites skill prose asks for a DevRites specialist or writer agent, use the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents. If Codex subagents are unavailable in the current surface, run the skill's documented inline fallback and say that the result was not an independent subagent review.
- Claude Code agent hook metadata is not active in Codex. The generated Codex agents preserve read-only intent with Codex sandbox settings where possible; still follow DevRites' scope and no-mutation rules explicitly.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: spec -> define -> build -> prove -> polish -> review -> seal -> ship.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES CODEX -->
EOF
}

install_codex_agents_md() {
  _src="$1"
  _rel="AGENTS.md"
  _dest="$TARGET/$_rel"
  _marker=".claude/devrites.agents-merge"

  if [ -e "$_dest" ]; then
    if [ "$DRYRUN" -eq 1 ]; then
      if grep -q '<!-- BEGIN DEVRITES CODEX -->' "$_dest" 2>/dev/null; then
        dr_say "  [merge] AGENTS.md ${DR_Y}(refresh DevRites block)${DR_R}"
      else
        dr_say "  [merge] AGENTS.md ${DR_Y}(append DevRites block)${DR_R}"
      fi
    else
      _merged="$TMP_GEN_DIR/AGENTS.merged.md"
      awk -v block_file="$_src" '
        BEGIN {
          while ((getline line < block_file) > 0) block = block line "\n"
          close(block_file)
          in_block = 0
          found = 0
        }
        $0 == "<!-- BEGIN DEVRITES CODEX -->" {
          printf "%s", block
          in_block = 1
          found = 1
          next
        }
        $0 == "<!-- END DEVRITES CODEX -->" {
          in_block = 0
          next
        }
        in_block { next }
        { print }
        END {
          if (!found) {
            print ""
            printf "%s", block
          }
        }
      ' "$_dest" > "$_merged" || dr_die "cannot merge AGENTS.md"
      cp "$_merged" "$_dest" || dr_die "cannot write $_dest"
    fi
    install_marker_file "$_marker" "AGENTS.md contains a DevRites managed block between BEGIN/END DEVRITES CODEX markers."
    return 0
  fi

  if [ "$DRYRUN" -eq 1 ]; then
    dr_say "  [merge] AGENTS.md ${DR_Y}(create DevRites block)${DR_R}"
  else
    mkdir -p "$(dirname "$_dest")" || dr_die "cannot create $(dirname "$_dest")"
    cp "$_src" "$_dest" || dr_die "cannot write $_dest"
  fi
  install_marker_file "$_marker" "AGENTS.md contains a DevRites managed block between BEGIN/END DEVRITES CODEX markers."
}

gen_codex_config_block() {
  _out="$1"
  cat > "$_out" <<'EOF'
# BEGIN DEVRITES CODEX MCP
[mcp_servers.devrites]
command = "node"
args = [".codex/mcp/devrites-mcp.mjs"]
cwd = "."
enabled = true
default_tools_approval_mode = "prompt"
startup_timeout_sec = 10
tool_timeout_sec = 60
# END DEVRITES CODEX MCP
EOF
}

install_codex_config_toml() {
  _src="$1"
  _rel=".codex/config.toml"
  _dest="$TARGET/$_rel"
  _marker=".claude/devrites.codex-config-merge"

  if [ -e "$_dest" ]; then
    if grep -q '^\[mcp_servers\.devrites\]' "$_dest" 2>/dev/null && ! grep -q '# BEGIN DEVRITES CODEX MCP' "$_dest" 2>/dev/null; then
      [ "$DRYRUN" -eq 1 ] && dr_say "  [skip] .codex/config.toml ${DR_Y}(already defines mcp_servers.devrites)${DR_R}"
      [ "$DRYRUN" -eq 1 ] || dr_warn "skip .codex/config.toml DevRites MCP block (already defines mcp_servers.devrites)"
      N_SKIP=$((N_SKIP+1))
      return 0
    fi
    if [ "$DRYRUN" -eq 1 ]; then
      if grep -q '# BEGIN DEVRITES CODEX MCP' "$_dest" 2>/dev/null; then
        dr_say "  [merge] .codex/config.toml ${DR_Y}(refresh DevRites MCP block)${DR_R}"
      else
        dr_say "  [merge] .codex/config.toml ${DR_Y}(append DevRites MCP block)${DR_R}"
      fi
    else
      _merged="$TMP_GEN_DIR/codex-config.merged.toml"
      awk -v block_file="$_src" '
        BEGIN {
          while ((getline line < block_file) > 0) block = block line "\n"
          close(block_file)
          in_block = 0
          found = 0
        }
        $0 == "# BEGIN DEVRITES CODEX MCP" {
          printf "%s", block
          in_block = 1
          found = 1
          next
        }
        $0 == "# END DEVRITES CODEX MCP" {
          in_block = 0
          next
        }
        in_block { next }
        { print }
        END {
          if (!found) {
            print ""
            printf "%s", block
          }
        }
      ' "$_dest" > "$_merged" || dr_die "cannot merge .codex/config.toml"
      cp "$_merged" "$_dest" || dr_die "cannot write $_dest"
    fi
    install_marker_file "$_marker" ".codex/config.toml contains a DevRites managed block between BEGIN/END DEVRITES CODEX MCP markers."
    return 0
  fi

  if [ "$DRYRUN" -eq 1 ]; then
    dr_say "  [merge] .codex/config.toml ${DR_Y}(create DevRites MCP block)${DR_R}"
  else
    mkdir -p "$(dirname "$_dest")" || dr_die "cannot create $(dirname "$_dest")"
    cp "$_src" "$_dest" || dr_die "cannot write $_dest"
  fi
  install_marker_file "$_marker" ".codex/config.toml contains a DevRites managed block between BEGIN/END DEVRITES CODEX MCP markers."
}

gen_codex_hooks_json() {
  _out="$1"
  cat > "$_out" <<'EOF'
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec env DEVRITES_REVIEWER_AGENT_REQUIRED=1 devrites-engine hook reviewer-readonly --harness=codex",
            "statusMessage": "DevRites: checking reviewer read-only boundary"
          }
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook allow --harness=codex",
            "statusMessage": "DevRites: checking read-only helper approval"
          }
        ]
      },
      {
        "matcher": "Edit|Write|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook a1-guard --harness=codex",
            "statusMessage": "DevRites: checking build write boundary"
          },
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec env DEVRITES_WRIGHT_AGENT_REQUIRED=1 devrites-engine hook wright-scope --harness=codex",
            "statusMessage": "DevRites: checking slice file scope"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook redwatch --harness=codex",
            "statusMessage": "DevRites: checking test/build result"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook stop-gate --harness=codex",
            "statusMessage": "DevRites: checking workspace stop gate"
          }
        ]
      },
      {
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook refresh-indexes",
            "statusMessage": "DevRites: refreshing code indexes"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook cursor --harness=codex"
          }
        ]
      }
    ],
    "SubagentStart": [
      {
        "matcher": "devrites-.*",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook subagent-orient --harness=codex",
            "statusMessage": "DevRites: injecting subagent discipline"
          }
        ]
      },
      {
        "matcher": "devrites-slice-wright",
        "hooks": [
          {
            "type": "command",
            "command": "printf '%s\\n' 'DevRites: this subagent is the single write-capable slice-wright. Stay within the current slice contract and touched-files.md; write no .devrites bookkeeping.'"
          }
        ]
      },
      {
        "matcher": "devrites-(code|test|frontend|security|performance|devex|doubt|simplifier|strategy|plan|spec)-reviewer|devrites-retrospector|devrites-forge-judge",
        "hooks": [
          {
            "type": "command",
            "command": "printf '%s\\n' 'DevRites: this subagent is read-only. Inspect with read/search/test commands and return findings only; do not mutate the tree.'"
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "devrites-slice-wright|devrites-.*reviewer|devrites-retrospector|devrites-forge-judge",
        "hooks": [
          {
            "type": "command",
            "command": "printf '%s\\n' 'DevRites: reconcile this subagent result against the active DevRites skill contract before claiming completion.'"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook orient --harness=codex",
            "statusMessage": "DevRites: loading project orientation"
          }
        ]
      }
    ]
  }
}
EOF
}

install_codex_hooks_json() {
  _src="$1"
  _rel=".codex/hooks.json"
  _dest="$TARGET/$_rel"
  _marker=".claude/devrites.codex-hooks-merge"

  if [ -e "$_dest" ]; then
    if [ "$DRYRUN" -eq 1 ]; then
      if grep -q 'devrites-' "$_dest" 2>/dev/null; then
        dr_say "  [merge] .codex/hooks.json ${DR_Y}(refresh DevRites hooks)${DR_R}"
      else
        dr_say "  [merge] .codex/hooks.json ${DR_Y}(append DevRites hooks)${DR_R}"
      fi
    else
      command -v node >/dev/null 2>&1 || dr_die "node is required to merge existing .codex/hooks.json"
      _merged="$TMP_GEN_DIR/codex-hooks.merged.json"
      node "$SELF_DIR/scripts/merge-codex-hooks.mjs" merge "$_dest" "$_src" "$_merged" || dr_die "cannot merge .codex/hooks.json"
      cp "$_merged" "$_dest" || dr_die "cannot write $_dest"
    fi
    install_marker_file "$_marker" ".codex/hooks.json contains DevRites managed hook entries."
    return 0
  fi

  if [ "$DRYRUN" -eq 1 ]; then
    dr_say "  [merge] .codex/hooks.json ${DR_Y}(create DevRites hooks)${DR_R}"
  else
    mkdir -p "$(dirname "$_dest")" || dr_die "cannot create $(dirname "$_dest")"
    cp "$_src" "$_dest" || dr_die "cannot write $_dest"
  fi
  install_marker_file "$_marker" ".codex/hooks.json contains DevRites managed hook entries."
}

# ---- plan + execute ------------------------------------------------------
dr_info "DevRites installer"
dr_say  "  source : $SELF_DIR"
dr_say  "  target : $TARGET"
dr_say  "  skills : $([ "$WITH_SKILLS" -eq 1 ] && echo yes || echo no)"
dr_say  "  standards: ship inside the devrites-lib skill"
dr_say  "  agents : $([ "$WITH_AGENTS" -eq 1 ] && echo yes || echo no)"
dr_say  "  codex  : $([ "$WITH_CODEX" -eq 1 ] && echo yes || echo no)"
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
    [ "$WITH_CODEX" -eq 1 ] && install_codex_skill_tree "$d" ".agents/skills/$name"
  done < <(find "$PACK_SRC/skills" -mindepth 1 -maxdepth 1 -type d)
fi

# 2) optional --short-aliases=all wrappers for /define /build /prove /seal.
if [ "$WITH_SKILLS" -eq 1 ] && [ "$ALIAS_MODE" = "all" ]; then
  for pair in "define:rite-define" "build:rite-build" "prove:rite-prove" "seal:rite-seal"; do
    a="${pair%%:*}"; to="${pair#*:}"
    gen_alias_wrapper "$a" "$to" "$TMP_GEN_DIR/$a.md"
    install_file "$TMP_GEN_DIR/$a.md" ".claude/skills/$a/SKILL.md"
    [ "$WITH_CODEX" -eq 1 ] && install_file "$TMP_GEN_DIR/$a.md" ".agents/skills/$a/SKILL.md"
  done
fi

# 3) agents
if [ "$WITH_AGENTS" -eq 1 ]; then
  install_tree "$PACK_SRC/agents" ".claude/agents"
  if [ "$WITH_CODEX" -eq 1 ]; then
    mkdir -p "$TMP_GEN_DIR/codex-agents"
    while IFS= read -r f; do
      name="$(basename "$f" .md)"
      gen_codex_agent "$f" "$TMP_GEN_DIR/codex-agents/$name.toml"
      install_file "$TMP_GEN_DIR/codex-agents/$name.toml" ".codex/agents/$name.toml"
    done < <(find "$PACK_SRC/agents" -maxdepth 1 -type f -name '*.md')
  fi
fi

# 3b) Codex project instructions. Codex discovers repository guidance from
# AGENTS.md, so install or merge a small bridge when skills are installed.
if [ "$WITH_CODEX" -eq 1 ] && [ "$WITH_SKILLS" -eq 1 ]; then
  CODEX_AGENTS_MD="$TMP_GEN_DIR/AGENTS.md"
  gen_codex_agents_bridge "$CODEX_AGENTS_MD"
  install_codex_agents_md "$CODEX_AGENTS_MD"
  if [ -f "$SELF_DIR/mcp/devrites-mcp.mjs" ]; then
    install_file "$SELF_DIR/mcp/devrites-mcp.mjs" ".codex/mcp/devrites-mcp.mjs"
    CODEX_CONFIG_TOML="$TMP_GEN_DIR/codex-config.toml"
    gen_codex_config_block "$CODEX_CONFIG_TOML"
    install_codex_config_toml "$CODEX_CONFIG_TOML"
  fi
fi

# 4) DevRites engineering standards now live inside the devrites-lib skill
# (pack/.claude/skills/devrites-lib/reference/standards/) and install with the
# skills tree in step 1 — no separate rules install, and no reserved
# .claude/rules/ namespace to collide with a project's own rules.

# 4b) hooks — seed Claude's JSON hook config and merge Codex's generated hook config.
# Hooks call the global engine binary directly; no per-project hook scripts are shipped.
# SEED .claude/settings.json only when absent and never record it in the manifest, so it
# is preserved on uninstall/update and the user's own settings stay safe.
if [ "$WITH_SKILLS" -eq 1 ]; then
  if [ "$WITH_CODEX" -eq 1 ]; then
    CODEX_HOOKS_JSON="$TMP_GEN_DIR/codex-hooks.json"
    gen_codex_hooks_json "$CODEX_HOOKS_JSON"
    install_codex_hooks_json "$CODEX_HOOKS_JSON"
  fi
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
  `allow_gates`. Full contract: `.claude/skills/devrites-lib/reference/standards/afk-hitl.md`.
  **Gitignore `AFK` unless your team agrees on shared AFK defaults** — it's a
  local toggle, not project state.
- `conventions.md` (optional) — the conventions ledger: facts a sealed slice proved
  about this codebase (commands, idioms, placement, gotchas), written on a GO seal and
  read at orient so the wright stops re-deriving them. The band is earned from how many
  slices corroborated each entry. **Gitignore `conventions.md`** — it's local, per-clone
  learning, not shared project state.
- `specs/<capability>/spec.md` — the **capability ledger**: the living record of what the
  system already does, folded from each feature's proven spec deltas on ship. **Commit
  `specs/`** — it is durable, shared project truth (the next feature writes deltas against
  it), unlike the ephemeral, per-clone runtime state above. Your `.gitignore` should track
  it explicitly, e.g. `.devrites/*` + `!.devrites/specs/`.

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
[ "$WITH_CODEX" -eq 0 ] && DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --no-codex"
[ "$WITH_BINARY" -eq 0 ] && DR_INSTALL_FLAGS="$DR_INSTALL_FLAGS --no-binary"
# (--no-rules retired: engineering standards always ship with the skills)
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
    case "$_rel" in AGENTS.md|.codex/config.toml|.codex/hooks.json) continue ;; esac
    grep -Fxq -- "$_rel" "$MANIFEST_TMP" && continue    # still shipped → keep
    _dead="$TARGET/$_rel"
    [ -e "$_dead" ] || continue                         # already gone
    if [ "$DRYRUN" -eq 1 ]; then
      dr_say "  [prune] $_rel ${DR_Y}(dropped from pack)${DR_R}"
    else
      case "$_rel" in
        .claude/devrites.agents-merge)
          strip_codex_agents_md_block "$TARGET/AGENTS.md"
          ;;
        .claude/devrites.codex-config-merge)
          strip_codex_config_block "$TARGET/.codex/config.toml"
          ;;
        .claude/devrites.codex-hooks-merge)
          strip_codex_hooks_entries "$TARGET/.codex/hooks.json"
          ;;
      esac
      rm -f "$_dead"
      # tidy now-empty parent dirs, bounded strictly below the target
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

# ---- engine binary lifecycle --------------------------------------------
# GUARD:no-global — install_binary only ever writes the sanctioned
# `devrites-engine`
# leaf under /usr/local/bin or ~/.local/bin (see check-no-global-writes.sh); it
# never touches ~/.claude or ~/.codex. All bin paths flow through variables so the
# static guard's write-verb scan stays clean.

# dr_ver_gt A B — exit 0 iff semantic version A is strictly greater than B.
# Portable POSIX-ish comparator; a leading "v" is tolerated. Equal → not greater.
dr_ver_gt() {
  [ "$1" = "$2" ] && return 1
  _a="${1#v}"; _b="${2#v}"
  [ "$_a" = "$_b" ] && return 1
  _ap="${_a%%[-+]*}"; _bp="${_b%%[-+]*}"
  _old_ifs="$IFS"; IFS=.
  # shellcheck disable=SC2086
  set -- $_ap
  _a1="${1:-0}"; _a2="${2:-0}"; _a3="${3:-0}"
  # shellcheck disable=SC2086
  set -- $_bp
  _b1="${1:-0}"; _b2="${2:-0}"; _b3="${3:-0}"
  IFS="$_old_ifs"
  for _part in 1 2 3; do
    eval "_av=\$_a$_part"
    eval "_bv=\$_b$_part"
    case "$_av" in ''|*[!0-9]*) _av=0 ;; esac
    case "$_bv" in ''|*[!0-9]*) _bv=0 ;; esac
    if [ "$_av" -gt "$_bv" ]; then return 0; fi
    if [ "$_av" -lt "$_bv" ]; then return 1; fi
  done
  return 1
}

dr_packaged_release_tag() {
  case "$DR_INSTALLED_VERSION" in
    unknown|"") return 0 ;;
    v*) printf '%s\n' "$DR_INSTALLED_VERSION" ;;
    *) printf 'v%s\n' "$DR_INSTALLED_VERSION" ;;
  esac
}

install_binary() {
  if [ "$WITH_BINARY" -ne 1 ] || [ "${DEVRITES_NO_BINARY:-0}" = "1" ]; then
    dr_say "  engine binary: skipped (--no-binary); hooks fail open."
    return 0
  fi

  # This unix installer covers darwin/linux; windows users take the .exe from the
  # release page. Map uname → the release asset suffix.
  _os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
  _arch="$(uname -m 2>/dev/null)"
  case "$_os" in linux|darwin) : ;; *) dr_warn "engine binary: unsupported OS '$_os' — skipped (hooks fail open)."; return 0 ;; esac
  case "$_arch" in x86_64|amd64) _arch=amd64 ;; arm64|aarch64) _arch=arm64 ;; *) dr_warn "engine binary: unsupported arch '$_arch' — skipped."; return 0 ;; esac
  _plat="${_os}-${_arch}"

  # Resolve the release tag. npm packages carry package.json, so a pinned
  # `npx devrites@<version>` installs the matching binary instead of drifting to
  # GitHub's latest release.
  _tag="${DEVRITES_REF:-${BOOT_TAG:-}}"
  if [ -z "$_tag" ]; then
    _tag="$(dr_packaged_release_tag)"
  fi
  if [ -z "$_tag" ]; then
    _tag="$(curl -fsSL "https://api.github.com/repos/$DEVRITES_REPO/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  fi
  _incoming="${_tag#v}"

  # Downgrade guard: never replace a newer installed binary with an older one.
  # Only compare when BOTH sides are real semver releases — an unstamped source
  # build reports "dev", which must always be upgradable (never treated as newest).
  _existing_bin="$(command -v devrites-engine 2>/dev/null || true)"
  if [ -n "$_existing_bin" ]; then
    _existing_ver="$("$_existing_bin" version 2>/dev/null | head -n1 || true)"
    if printf '%s' "${_existing_ver#v}" | grep -qE '^[0-9]' \
      && printf '%s' "$_incoming" | grep -qE '^[0-9]' \
      && dr_ver_gt "$_existing_ver" "$_incoming"; then
      dr_warn "engine binary: installed ${_existing_ver} is newer than ${_tag:-<unknown>} — refusing to downgrade (kept)."
      return 0
    fi
  fi

  # Acquire the binary into a temp file: prefer the verified release asset, else
  # build from source when this is a full checkout with the Go toolchain present.
  _tmpd="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-bin.$$")"; mkdir -p "$_tmpd"
  _staged="$_tmpd/devrites"
  _got=0

  if [ -n "$_tag" ]; then
    _url="https://github.com/$DEVRITES_REPO/releases/download/$_tag/devrites-$_plat"
    if curl -fsSL -o "$_staged" "$_url" 2>/dev/null; then
      if curl -fsSL -o "$_staged.sha256" "$_url.sha256" 2>/dev/null; then
        _want="$(awk '{print $1; exit}' "$_staged.sha256" 2>/dev/null)"
        if command -v shasum >/dev/null 2>&1; then _got_sum="$(shasum -a 256 "$_staged" | awk '{print $1}')"
        elif command -v sha256sum >/dev/null 2>&1; then _got_sum="$(sha256sum "$_staged" | awk '{print $1}')"
        else _got_sum=""; fi
        if [ -z "$_got_sum" ]; then dr_warn "engine binary: no sha256 tool — refusing unverified release asset."
        elif [ "$_got_sum" = "$_want" ]; then _got=1; dr_say "  engine binary: checksum verified (sha256)."
        else dr_warn "engine binary: checksum mismatch for devrites-$_plat — refusing to install."; rm -rf "$_tmpd"; return 0; fi
      else
        dr_warn "engine binary: no .sha256 for devrites-$_plat — refusing unverified release asset."
      fi
    fi
  fi

  if [ "$_got" -ne 1 ] && [ -d "$SELF_DIR/engine" ] && command -v go >/dev/null 2>&1; then
    dr_say "  engine binary: no release asset — building from source…"
    if ( cd "$SELF_DIR/engine" && CGO_ENABLED=0 go build -trimpath \
          -ldflags "-s -w -X github.com/devrites/devrites/internal/version.Version=${_tag:-dev}" \
          -o "$_staged" . ) 2>/dev/null; then
      _got=1
    fi
  fi

  if [ "$_got" -ne 1 ]; then
    dr_warn "engine binary: could not obtain a binary (no release asset, no Go toolchain) — hooks fail open until you install it."
    rm -rf "$_tmpd"; return 0
  fi
  chmod +x "$_staged" 2>/dev/null || true

  # Destination: an explicit DEVRITES_BIN_DIR override (used by tests and by users
  # who keep a custom bin dir), else the system bin dir when writable (directly or
  # via sudo on a tty), else the per-user bin dir. Only ever the `devrites-engine` leaf.
  _sys_dir="/usr/local/bin"        # GUARD:no-global — sanctioned bin dir
  _user_dir="$HOME/.local/bin"     # GUARD:no-global — sanctioned bin dir
  _use_sudo=0
  if [ -n "${DEVRITES_BIN_DIR:-}" ]; then
    _dest_dir="$DEVRITES_BIN_DIR"
  elif [ -w "$_sys_dir" ] 2>/dev/null; then
    _dest_dir="$_sys_dir"
  elif [ -d "$_sys_dir" ] && command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    _dest_dir="$_sys_dir"; _use_sudo=1
  else
    _dest_dir="$_user_dir"
  fi
  _dest="$_dest_dir/devrites-engine"

  if [ "$_use_sudo" -eq 1 ]; then
    if ! sudo install -m 0755 "$_staged" "$_dest" 2>/dev/null; then
      dr_warn "engine binary: sudo install to $_dest_dir failed — falling back to the per-user bin dir."
      _dest_dir="$_user_dir"; _dest="$_dest_dir/devrites-engine"; _use_sudo=0
    fi
  fi
  if [ "$_use_sudo" -eq 0 ]; then
    _final_tmp="$_dest.tmp.$$"
    if ! { mkdir -p "$_dest_dir" 2>/dev/null && cp "$_staged" "$_final_tmp" 2>/dev/null \
           && chmod 0755 "$_final_tmp" 2>/dev/null && mv "$_final_tmp" "$_dest" 2>/dev/null; }; then
      rm -f "$_final_tmp" 2>/dev/null || true
      dr_warn "engine binary: cannot write $_dest — skipped (hooks fail open)."
      rm -rf "$_tmpd"; return 0
    fi
  fi
  rm -rf "$_tmpd"

  dr_ok "engine binary installed: $_dest ${_tag:+($_tag)}"
  case ":$PATH:" in
    *":$_dest_dir:"*) : ;;
    *) dr_warn "engine binary: $_dest_dir is not on your PATH — add it (export PATH=\"$_dest_dir:\$PATH\") so 'devrites-engine' resolves." ;;
  esac
}

if [ "$DRYRUN" -eq 1 ]; then
  [ "$WITH_BINARY" -eq 1 ] && dr_say "  [plan] would install the global devrites-engine control-plane binary (see install_binary)"
else
  install_binary
fi

# ---- mirror project extensions -------------------------------------------
# Any project-local extensions (.devrites/extensions/) are mirrored into the
# harness skill/agent dirs now the pack and engine binary are in place. Fail-open:
# no extensions, no binary on PATH, or a sync error never fails the install — the
# user can always run `devrites-engine extensions sync` or /rite-doctor later.
if [ "$DRYRUN" -eq 0 ] && command -v devrites-engine >/dev/null 2>&1; then
  if [ -d "$TARGET/.devrites/extensions" ]; then
    DEVRITES_ROOT="$TARGET/.devrites" devrites-engine extensions sync >/dev/null 2>&1 \
      && dr_ok "project extensions synced (.devrites/extensions/ → .claude/)" || true
  fi
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
dr_say "${DR_B}Standards:${DR_R} engineering standards ship in .claude/skills/devrites-lib/reference/standards/$([ "$WITH_CODEX" -eq 1 ] && echo ' (Codex: .agents/skills/devrites-lib/reference/standards/)')"
[ "$WITH_CODEX" -eq 1 ] && [ "$WITH_SKILLS" -eq 1 ] && dr_say "${DR_B}Codex:${DR_R}   skills mirrored to .agents/skills/; project guidance in AGENTS.md"
dr_say ""
dr_say "${DR_B}Next:${DR_R}"
dr_say "  1. Open the project in Claude Code or Codex and accept the workspace trust prompt."
dr_say "  2. Claude Code: run ${DR_C}/rite${DR_R} for the menu, or ${DR_C}/rite-spec <feature>${DR_R} to start."
[ "$WITH_CODEX" -eq 1 ] && [ "$WITH_SKILLS" -eq 1 ] && dr_say "     Codex: use ${DR_C}\$rite${DR_R} / ${DR_C}\$rite-spec${DR_R} through skills, or pick DevRites from ${DR_C}/skills${DR_R}."
dr_say "  3. ${DR_B}HITL by default${DR_R} — HITL slices pause at typed gates; resume with ${DR_C}/rite-resolve <qid> \"<answer>\"${DR_R}."
dr_say "     For unattended runs: ${DR_C}touch .devrites/AFK${DR_R} (see .claude/skills/devrites-lib/reference/standards/afk-hitl.md for the contract)."
[ "$N_SKIP" -gt 0 ] && dr_say "  • $N_SKIP file(s) were skipped (already present). Re-run with --force to overwrite."
[ "$DRYRUN" -eq 1 ] && dr_say "  (dry run only — re-run without --dry-run to apply.)"
exit 0
