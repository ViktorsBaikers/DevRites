#!/usr/bin/env bash
# build-host-artifacts.sh: render host-native Claude, Codex, omp, pi, and
# Devin artifacts from the canonical pack.
#
# Default output is pack/generated/ so npm pack can ship prebuilt surfaces.
# Tests may set DEVRITES_HOST_ARTIFACT_DIR to a temporary directory.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
PACK_SRC="$ROOT/pack/.claude"
OUT_ROOT="${DEVRITES_HOST_ARTIFACT_DIR:-$ROOT/pack/generated}"

[ -d "$PACK_SRC/skills" ] || {
  echo "build-host-artifacts: missing $PACK_SRC/skills" >&2
  exit 1
}
[ -d "$PACK_SRC/agents" ] || {
  echo "build-host-artifacts: missing $PACK_SRC/agents" >&2
  exit 1
}
[ -d "$PACK_SRC/workflows" ] || {
  echo "build-host-artifacts: missing $PACK_SRC/workflows" >&2
  exit 1
}

TMP_GEN_DIR="$(mktemp -d)"
cleanup() { rm -rf "$TMP_GEN_DIR"; }
trap cleanup EXIT

# Build into a scratch dir, then swap into place: file watchers were observed
# deleting/recreating entries under the live output dir mid-build on macOS.
REAL_OUT="$OUT_ROOT"
OUT_ROOT="$TMP_GEN_DIR/out"

# shellcheck source=codex-generate.sh
. "$ROOT/scripts/codex-generate.sh"
# shellcheck source=omp-generate.sh
. "$ROOT/scripts/omp-generate.sh"
# shellcheck source=pi-generate.sh
. "$ROOT/scripts/pi-generate.sh"
# shellcheck source=devin-generate.sh
. "$ROOT/scripts/devin-generate.sh"

copy_tree() {
  _src="$1"
  _dest="$2"
  mkdir -p "$(dirname "$_dest")"
  rm -rf "$_dest"
  cp -R "$_src" "$_dest"
}

render_codex_skill_tree() {
  _sd="$1"
  _dest="$2"
  mkdir -p "$_dest"
  while IFS= read -r f; do
    _r="${f#$_sd/}"
    _out="$_dest/$_r"
    mkdir -p "$(dirname "$_out")"
    case "$_r" in
    SKILL.md)
      _implicit_disabled=0
      if awk 'NR==1&&$0=="---"{fm=1;next} fm&&$0=="---"{exit} fm&&/^disable-model-invocation:[[:space:]]*true/{u=1} END{exit !u}' "$f"; then
        _implicit_disabled=1
      fi
      gen_codex_skill_file "$f" "$_out"
      if [ "$_implicit_disabled" -eq 1 ]; then
        mkdir -p "$_dest/agents"
        printf 'policy:\n  allow_implicit_invocation: false\n' >"$_dest/agents/openai.yaml"
      fi
      ;;
    *.md)
      gen_codex_markdown_file "$f" "$_out"
      ;;
    *)
      cp "$f" "$_out"
      ;;
    esac
  done < <(find "$_sd" -type f)
}

render_omp_skill_tree() {
  _sd="$1"
  _dest="$2"
  mkdir -p "$_dest"
  while IFS= read -r f; do
    _r="${f#$_sd/}"
    _out="$_dest/$_r"
    mkdir -p "$(dirname "$_out")"
    case "$_r" in
    SKILL.md)
      gen_omp_skill_file "$f" "$_out"
      ;;
    *.md)
      gen_omp_markdown_file "$f" "$_out"
      ;;
    *)
      cp "$f" "$_out"
      ;;
    esac
  done < <(find "$_sd" -type f)
}

render_pi_skill_tree() {
  _sd="$1"
  _dest="$2"
  mkdir -p "$_dest"
  while IFS= read -r f; do
    _r="${f#$_sd/}"
    _out="$_dest/$_r"
    mkdir -p "$(dirname "$_out")"
    case "$_r" in
    SKILL.md)
      gen_pi_skill_file "$f" "$_out"
      ;;
    *.md)
      gen_pi_markdown_file "$f" "$_out"
      ;;
    *)
      cp "$f" "$_out"
      ;;
    esac
  done < <(find "$_sd" -type f)
}

render_devin_skill_tree() {
  _sd="$1"
  _dest="$2"
  mkdir -p "$_dest"
  while IFS= read -r f; do
    _r="${f#$_sd/}"
    _out="$_dest/$_r"
    mkdir -p "$(dirname "$_out")"
    case "$_r" in
    SKILL.md)
      gen_devin_skill_file "$f" "$_out"
      ;;
    *.md)
      gen_devin_markdown_file "$f" "$_out"
      ;;
    *)
      cp "$f" "$_out"
      ;;
    esac
  done < <(find "$_sd" -type f)
}

mkdir -p "$OUT_ROOT/claude" "$OUT_ROOT/codex" "$OUT_ROOT/omp" "$OUT_ROOT/pi" "$OUT_ROOT/devin"

# Canonical sources may use <!-- include:REL --> markers for shared blocks
# (e.g. agents/_shared/). Expand them on a scratch copy so every host renders
# identical text while the canonical pack stays deduplicated.
EXPANDED_SRC="$TMP_GEN_DIR/canonical"
mkdir -p "$EXPANDED_SRC"
cp -R "$PACK_SRC/." "$EXPANDED_SRC/"
python3 "$ROOT/scripts/expand-includes.py" "$EXPANDED_SRC"
PACK_SRC="$EXPANDED_SRC"

# `_shared` dirs exist only to feed expansion; no host ships them.
find "$PACK_SRC" -type d -name _shared -prune -exec rm -rf {} +

# Claude artifacts are host-native copies of the canonical pack.
copy_tree "$PACK_SRC/skills" "$OUT_ROOT/claude/skills"
copy_tree "$PACK_SRC/agents" "$OUT_ROOT/claude/agents"
rm -rf "$OUT_ROOT/claude/agents/_shared" "$OUT_ROOT/claude/skills/devrites-lib/reference/_shared"
copy_tree "$PACK_SRC/workflows" "$OUT_ROOT/claude/workflows"
# Local editor caches under .impeccable/ must never ship in generated hosts.
rm -rf "$OUT_ROOT/claude/agents/.impeccable" "$OUT_ROOT/claude/skills/.impeccable"
cp "$PACK_SRC/settings.json" "$OUT_ROOT/claude/settings.json"

# Codex artifacts are generated host-native mirrors.
mkdir -p "$OUT_ROOT/codex/skills" "$OUT_ROOT/codex/agents"
while IFS= read -r d; do
  name="$(basename "$d")"
  [ "$name" = ".impeccable" ] && continue
  render_codex_skill_tree "$d" "$OUT_ROOT/codex/skills/$name"
done < <(find "$PACK_SRC/skills" -mindepth 1 -maxdepth 1 -type d)

while IFS= read -r f; do
  name="$(basename "$f" .md)"
  gen_codex_agent "$f" "$OUT_ROOT/codex/agents/$name.toml"
done < <(find "$PACK_SRC/agents" -maxdepth 1 -type f -name '*.md')

gen_codex_agents_bridge "$OUT_ROOT/codex/AGENTS.md"
gen_codex_config_toml "$OUT_ROOT/codex/config.toml"

# omp artifacts are generated host-native mirrors (Markdown agents, no TOML).
mkdir -p "$OUT_ROOT/omp/skills" "$OUT_ROOT/omp/agents" "$OUT_ROOT/omp/commands"
while IFS= read -r d; do
  name="$(basename "$d")"
  [ "$name" = ".impeccable" ] && continue
  render_omp_skill_tree "$d" "$OUT_ROOT/omp/skills/$name"
  case "$name" in
  rite | rite-*)
    gen_omp_command_stub "$d/SKILL.md" "$name" "$OUT_ROOT/omp/commands/$name.md"
    ;;
  esac
done < <(find "$PACK_SRC/skills" -mindepth 1 -maxdepth 1 -type d)

while IFS= read -r f; do
  name="$(basename "$f" .md)"
  gen_omp_agent "$f" "$OUT_ROOT/omp/agents/$name.md"
done < <(find "$PACK_SRC/agents" -maxdepth 1 -type f -name '*.md')

gen_omp_plugin_json "$OUT_ROOT/omp/.omp-plugin/plugin.json"

# pi artifacts are generated host-native mirrors: skills under .pi/skills,
# pi-subagents Markdown agents under .pi/agents, prompt templates under
# .pi/prompts (preserving the public /rite* command forms), and an AGENTS.md
# bridge block.
mkdir -p "$OUT_ROOT/pi/skills" "$OUT_ROOT/pi/agents" "$OUT_ROOT/pi/prompts"
while IFS= read -r d; do
  name="$(basename "$d")"
  [ "$name" = ".impeccable" ] && continue
  render_pi_skill_tree "$d" "$OUT_ROOT/pi/skills/$name"
  case "$name" in
  rite | rite-*)
    gen_pi_prompt_stub "$d/SKILL.md" "$name" "$OUT_ROOT/pi/prompts/$name.md"
    ;;
  esac
done < <(find "$PACK_SRC/skills" -mindepth 1 -maxdepth 1 -type d)

while IFS= read -r f; do
  name="$(basename "$f" .md)"
  gen_pi_agent "$f" "$OUT_ROOT/pi/agents/$name.md"
done < <(find "$PACK_SRC/agents" -maxdepth 1 -type f -name '*.md')

gen_pi_agents_bridge "$OUT_ROOT/pi/AGENTS.md"

# Devin artifacts are generated host-native mirrors: skills under
# .devin/skills and custom subagent profiles under .devin/agents, plus an
# AGENTS.md bridge block (Devin reads project AGENTS.md as always-on rules).
mkdir -p "$OUT_ROOT/devin/skills" "$OUT_ROOT/devin/agents"
while IFS= read -r d; do
  name="$(basename "$d")"
  [ "$name" = ".impeccable" ] && continue
  render_devin_skill_tree "$d" "$OUT_ROOT/devin/skills/$name"
done < <(find "$PACK_SRC/skills" -mindepth 1 -maxdepth 1 -type d)

while IFS= read -r f; do
  name="$(basename "$f" .md)"
  gen_devin_agent "$f" "$OUT_ROOT/devin/agents/$name.md"
done < <(find "$PACK_SRC/agents" -maxdepth 1 -type f -name '*.md')

gen_devin_agents_bridge "$OUT_ROOT/devin/AGENTS.md"

cat >"$OUT_ROOT/README.md" <<'EOF'
# Generated DevRites Host Artifacts

This directory is generated by `scripts/build-host-artifacts.sh`.

- `claude/` contains Claude Code-native skills, agents, read-only workflow adapters, and settings.
- `codex/` contains Codex-native skill mirrors, custom-agent TOML, the
  read-only-root permission profile, and AGENTS bridge content.
- `omp/` contains omp-native skill mirrors, Markdown agents, command stubs
  preserving the `/rite*` commands, and `.omp-plugin/plugin.json`.
- `pi/` contains pi-native skill mirrors, pi-subagents Markdown agents,
  prompt templates preserving the `/rite*` commands, and an `AGENTS.md`
  bridge block.
- `devin/` contains Devin CLI skill mirrors, custom subagent profiles with
  `allowed-tools` allowlists, and an `AGENTS.md` bridge block.

Do not edit these files by hand. Edit `pack/.claude/`,
`scripts/codex-generate.sh`, `scripts/omp-generate.sh`,
`scripts/pi-generate.sh`, or `scripts/devin-generate.sh`, then rebuild.
EOF

# Swap the completed tree into place. Watchers can still re-create entries
# mid-delete, so retry the clear until the old tree is gone, then rename. The
# held-output security harness runs with the output root as the current
# directory; that directory cannot be renamed away, so replace only its
# generated children while leaving the held source tree in place.
if [ "$REAL_OUT" = "." ]; then
  for _name in claude codex devin omp pi README.md; do
    rm -rf "$_name"
  done
  for _entry in "$OUT_ROOT"/*; do
    [ -e "$_entry" ] || continue
    mv "$_entry" .
  done
else
  for _ in 1 2 3 4 5; do
    rm -rf "$REAL_OUT" && break
    sleep 1
  done
  [ ! -e "$REAL_OUT" ] || {
    echo "build-host-artifacts: cannot clear $REAL_OUT" >&2
    exit 1
  }
  mv "$OUT_ROOT" "$REAL_OUT"
fi
[ -f "$REAL_OUT/README.md" ] || {
  echo "build-host-artifacts: swap left no tree at $REAL_OUT" >&2
  exit 1
}

echo "build-host-artifacts: wrote $REAL_OUT"
