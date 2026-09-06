#!/usr/bin/env bash
# omp-generate.sh: shared Claude-to-omp surface generators.
#
# This file is a shell library, not a standalone installer. Callers provide
# TMP_GEN_DIR and then invoke the gen_* helpers below.

# Rewrite canonical Claude (and leftover Codex) paths to installed omp paths.
# Slash /rite invocations stay slash-form: omp discovers skills as /name.
# After paths are `.omp/skills/...`, drop Codex-only mirror asides.
# Dispatch wording uses the `task` tool / `tasks[]` batch, not a fictional `agent` tool.
gen_omp_markdown_file() {
  local _src="$1" _out="$2"
  mkdir -p "$(dirname "$_out")"
  sed -E \
    -e 's#Try post-install path first, fall back to pre-install:#Resolve the installed skill path:#g' \
    -e '/^\[ -f "\$F" \] \|\| F=[^ ].*SKILL\.md$/d' \
    -e 's#(pack/)?\.claude/agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.omp/agents/devrites-security-auditor.md`, `.omp/agents/devrites-performance-reviewer.md`, or `.omp/agents/devrites-simplifier-reviewer.md#g' \
    -e 's#(\.\./)+agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.omp/agents/devrites-security-auditor.md`, `.omp/agents/devrites-performance-reviewer.md`, or `.omp/agents/devrites-simplifier-reviewer.md#g' \
    -e 's#pack/\.claude/skills/devrites-lib/scripts/#.omp/skills/devrites-lib/scripts/#g' \
    -e 's#pack/\.claude/skills/#.omp/skills/#g' \
    -e 's#pack/\.claude/agents/([A-Za-z0-9_-]+)\.md#.omp/agents/\1.md#g' \
    -e 's#pack/\.claude/agents/#.omp/agents/#g' \
    -e 's#\.claude/skills/devrites-lib/scripts/#.omp/skills/devrites-lib/scripts/#g' \
    -e 's#\.claude/skills/#.omp/skills/#g' \
    -e 's#\.claude/agents/([A-Za-z0-9_-]+)\.md#.omp/agents/\1.md#g' \
    -e 's#\.claude/agents/#.omp/agents/#g' \
    -e 's#(\.\./)+agents/([A-Za-z0-9_-]+)\.md#.omp/agents/\2.md#g' \
    -e 's#\.agents/skills/devrites-lib/scripts/#.omp/skills/devrites-lib/scripts/#g' \
    -e 's#\.agents/skills/#.omp/skills/#g' \
    -e 's#\.codex/skills/devrites-lib/scripts/#.omp/skills/devrites-lib/scripts/#g' \
    -e 's#\.codex/skills/#.omp/skills/#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.toml#.omp/agents/\1.md#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.md#.omp/agents/\1.md#g' \
    -e 's#\.codex/agents/#.omp/agents/#g' \
    -e 's#\.omp/agents/([A-Za-z0-9_-]+)\.toml#.omp/agents/\1.md#g' \
    -e '/^\(use the `[^`]*` mirror on Codex\)\.$/d' \
    -e '/^\(`[^`]+` mirror on Codex\)\.$/d' \
    -e 's# \(use the `[^`]*` mirror on Codex\)##g' \
    -e 's# \(`[^`]+` mirror on Codex\)##g' \
    -e 's#\(`[^`]+` mirror on Codex\); ##g' \
    -e 's#\(`[^`]+` mirror on Codex\)##g' \
    -e 's#^seeded verdicts#Seeded verdicts#' \
    -e '/Rules in scope/{N;s#;\n[[:space:]]*`[^`]+` mirror on Codex##;}' \
    -e '/On Codex, use the/{N;s# On Codex, use the mirrors under\n`[^`]+`\.##;s# On Codex, use the\nmirrors under `[^`]+`\.##;s# On Codex, use the\nmirror under `[^`]+`\.##;}' \
    -e 's#Host mapping: Claude Code uses `Agent` \(`Task` is its legacy alias\); Codex uses#Host mapping: omp fresh-context dispatch uses the `task` tool (`tasks[]` batch); hosts use#g' \
    -e 's#`Agent` call#`task` call#g' \
    -e 's#Claude: N concurrent Task wrights#omp: N concurrent `task` tool / `tasks[]` batch wrights#g' \
    -e 's#Task breakdown#Work-item breakdown#g' \
    -e 's#dispatch `devrites-source-driven`#invoke `devrites-source-driven`#g' \
    -e 's#Dispatch `devrites-source-driven`#Invoke `devrites-source-driven`#g' \
    -e 's#pass-through dispatch to the matching `rite-<verb>` skill#pass-through invocation of the matching `rite-<verb>` skill#g' \
    -e 's#verb dispatches to the matching `rite-<verb>` skill#verb invokes the matching `rite-<verb>` skill#g' \
    -e 's#the dispatch map only#the invocation map only#g' \
    -e 's#→ dispatch per the table above#→ invoke the matching skill per the table above#g' \
    "$_src" > "$_out"
}

gen_omp_skill_file() {
  gen_omp_markdown_file "$1" "$2"
}

omp_yaml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

# Map Claude Code tool names onto omp tool names. Skill is dropped (omp
# discovers skills from the plugin tree). Unmapped Claude tools are dropped.
_omp_map_tools() {
  local _raw="$1" _name="$2"
  local _tok _mapped _out="" _sep=""
  if [ -z "$_raw" ]; then
    if [ "$_name" = "devrites-slice-wright" ]; then
      printf '%s' "write, edit, bash, read, grep, glob"
    else
      printf '%s' "read, grep, glob, bash"
    fi
    return
  fi
  while IFS= read -r _tok; do
    _tok="${_tok#"${_tok%%[![:space:]]*}"}"
    _tok="${_tok%"${_tok##*[![:space:]]}"}"
    [ -z "$_tok" ] && continue
    case "$_tok" in
      Read|read) _mapped="read" ;;
      Edit|edit) _mapped="edit" ;;
      Write|write) _mapped="write" ;;
      Bash|bash) _mapped="bash" ;;
      Glob|glob) _mapped="glob" ;;
      Grep|grep) _mapped="grep" ;;
      Skill|skill) continue ;;
      *) continue ;;
    esac
    _out="${_out}${_sep}${_mapped}"
    _sep=", "
  done < <(printf '%s\n' "$_raw" | tr ',' '\n')
  if [ -z "$_out" ]; then
    if [ "$_name" = "devrites-slice-wright" ]; then
      _out="write, edit, bash, read, grep, glob"
    else
      _out="read, grep, glob, bash"
    fi
  fi
  printf '%s' "$_out"
}

# Generate an omp Markdown agent from a Claude Code markdown agent.
# omp loads project/plugin agents from agents/*.md (YAML frontmatter + body).
gen_omp_agent() {
  local _src="$1" _out="$2"
  local _name _desc _tools_raw _tools _desc_tmp _desc_omp _body_tmp _body_omp
  _name="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^name:[[:space:]]*/{sub(/^name:[[:space:]]*/, ""); print; exit}' "$_src")"
  _desc="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^description:[[:space:]]*/{sub(/^description:[[:space:]]*/, ""); print; exit}' "$_src")"
  _tools_raw="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^tools:[[:space:]]*/{sub(/^tools:[[:space:]]*/, ""); print; exit}' "$_src")"
  [ -n "$_name" ] || _name="$(basename "$_src" .md)"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  _tools="$(_omp_map_tools "$_tools_raw" "$_name")"
  _desc_tmp="$TMP_GEN_DIR/omp-agent-desc-$(basename "$_src").txt"
  _desc_omp="$TMP_GEN_DIR/omp-agent-desc-$(basename "$_src").omp.txt"
  printf '%s' "$_desc" > "$_desc_tmp"
  gen_omp_markdown_file "$_desc_tmp" "$_desc_omp"
  _desc="$(cat "$_desc_omp")"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  mkdir -p "$(dirname "$_out")"
  {
    printf '%s\n' "---"
    printf 'name: %s\n' "$_name"
    printf 'description: "%s"\n' "$(omp_yaml_escape "$_desc")"
    printf 'tools: %s\n' "$_tools"
    printf '%s\n' "---"
    _body_tmp="$TMP_GEN_DIR/omp-agent-body-$(basename "$_src").md"
    _body_omp="$TMP_GEN_DIR/omp-agent-body-$(basename "$_src").omp.md"
    awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{fm=0; body=1; next} body{print}' "$_src" > "$_body_tmp"
    gen_omp_markdown_file "$_body_tmp" "$_body_omp"
    cat "$_body_omp"
  } > "$_out"
}

# Write omp plugin manifest. Do not emit an Agent Plugins $schema URL: that
# routes the package through the strict agent-plugins loader, which exclusivizes
# skills. commands/ is omitted (no command tree is generated).
gen_omp_plugin_json() {
  local _out="$1"
  mkdir -p "$(dirname "$_out")"
  cat > "$_out" <<'EOF'
{
  "name": "devrites",
  "description": "DevRites: a disciplined senior-engineer workflow pack for omp",
  "skills": "./skills",
  "agents": "./agents"
}
EOF
}
