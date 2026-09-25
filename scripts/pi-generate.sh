#!/usr/bin/env bash
# pi-generate.sh: shared Claude-to-pi surface generators.
#
# This file is a shell library, not a standalone installer. Callers provide
# TMP_GEN_DIR and then invoke the gen_* helpers below.

# Rewrite canonical Claude (and leftover Codex/omp) paths to installed pi
# paths. Slash /rite invocations stay slash-form: pi resolves them through the
# generated .pi/prompts/rite*.md templates, and /skill:rite-* always works.
# Dispatch wording uses the `subagent` tool (`{agent, task}`), matching the
# pi-subagents extension that loads .pi/agents/*.md.
gen_pi_markdown_file() {
  local _src="$1" _out="$2"
  mkdir -p "$(dirname "$_out")"
  sed -E \
    -e 's#Try post-install path first, fall back to pre-install:#Resolve the installed skill path:#g' \
    -e '/^\[ -f "\$F" \] \|\| F=[^ ].*SKILL\.md$/d' \
    -e 's#(pack/)?\.claude/agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.pi/agents/devrites-security-auditor.md`, `.pi/agents/devrites-performance-reviewer.md`, or `.pi/agents/devrites-simplifier-reviewer.md#g' \
    -e 's#(\.\./)+agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.pi/agents/devrites-security-auditor.md`, `.pi/agents/devrites-performance-reviewer.md`, or `.pi/agents/devrites-simplifier-reviewer.md#g' \
    -e 's#pack/\.claude/skills/devrites-lib/scripts/#.pi/skills/devrites-lib/scripts/#g' \
    -e 's#pack/\.claude/skills/#.pi/skills/#g' \
    -e 's#pack/\.claude/agents/([A-Za-z0-9_-]+)\.md#.pi/agents/\1.md#g' \
    -e 's#pack/\.claude/agents/#.pi/agents/#g' \
    -e 's#\.claude/skills/devrites-lib/scripts/#.pi/skills/devrites-lib/scripts/#g' \
    -e 's#\.claude/skills/#.pi/skills/#g' \
    -e 's#\.claude/agents/([A-Za-z0-9_-]+)\.md#.pi/agents/\1.md#g' \
    -e 's#\.claude/agents/#.pi/agents/#g' \
    -e 's#(\.\./)+agents/([A-Za-z0-9_-]+)\.md#.pi/agents/\2.md#g' \
    -e 's#\.agents/skills/devrites-lib/scripts/#.pi/skills/devrites-lib/scripts/#g' \
    -e 's#\.agents/skills/#.pi/skills/#g' \
    -e 's#\.codex/skills/devrites-lib/scripts/#.pi/skills/devrites-lib/scripts/#g' \
    -e 's#\.codex/skills/#.pi/skills/#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.toml#.pi/agents/\1.md#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.md#.pi/agents/\1.md#g' \
    -e 's#\.codex/agents/#.pi/agents/#g' \
    -e 's#\.omp/skills/#.pi/skills/#g' \
    -e 's#\.omp/agents/([A-Za-z0-9_-]+)\.md#.pi/agents/\1.md#g' \
    -e 's#\.omp/agents/#.pi/agents/#g' \
    -e '/^\(use the `[^`]*` mirror on Codex\)\.$/d' \
    -e '/^\(`[^`]+` mirror on Codex\)\.$/d' \
    -e 's# \(use the `[^`]*` mirror on Codex\)##g' \
    -e 's# \(`[^`]+` mirror on Codex\)##g' \
    -e 's#\(`[^`]+` mirror on Codex\); ##g' \
    -e 's#\(`[^`]+` mirror on Codex\)##g' \
    -e 's#^seeded verdicts#Seeded verdicts#' \
    -e '/Rules in scope/{N;s#;\n[[:space:]]*`[^`]+` mirror on Codex##;}' \
    -e '/On Codex, use the/{N;s# On Codex, use the mirrors under\n`[^`]+`\.##;s# On Codex, use the\nmirrors under `[^`]+`\.##;s# On Codex, use the\nmirror under `[^`]+`\.##;}' \
    -e 's#Host mapping: Claude Code uses `Agent` \(`Task` is its legacy alias\); Codex uses#Host mapping: pi fresh-context dispatch uses the `subagent` tool (`{agent, task}`); hosts use#g' \
    -e 's#`Agent` call#`subagent` call#g' \
    -e 's#`spawn_agent` call#`subagent` call#g' \
    -e 's#Claude: N concurrent Task wrights#pi: N concurrent `subagent` wrights#g' \
    -e 's#Task breakdown#Work-item breakdown#g' \
    -e 's#dispatch `devrites-source-driven`#invoke `devrites-source-driven`#g' \
    -e 's#Dispatch `devrites-source-driven`#Invoke `devrites-source-driven`#g' \
    -e 's#pass-through dispatch to the matching `rite-<verb>` skill#pass-through invocation of the matching `rite-<verb>` skill#g' \
    -e 's#verb dispatches to the matching `rite-<verb>` skill#verb invokes the matching `rite-<verb>` skill#g' \
    -e 's#the dispatch map only#the invocation map only#g' \
    -e 's#→ dispatch per the table above#→ invoke the matching skill per the table above#g' \
    "$_src" >"$_out"
}

gen_pi_skill_file() {
  gen_pi_markdown_file "$1" "$2"
}

pi_yaml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

# Map Claude Code tool names onto pi built-in tool names. Pi built-ins are
# read, write, edit, bash, grep, find, ls (plus powershell on Windows). Claude
# Glob maps to pi find; Skill is dropped (pi-subagents uses the skills:
# frontmatter field instead). Unmapped Claude tools (WebFetch, WebSearch) are
# dropped, same as the omp mirror.
_pi_map_tools() {
  local _raw="$1" _name="$2"
  local _tok _mapped _out="" _sep=""
  if [ -z "$_raw" ]; then
    if [[ "$_name" == devrites-slice-wright || "$_name" == overhaul-* ]]; then
      printf '%s' "read, edit, write, bash, grep, find, ls"
    else
      printf '%s' "read, grep, find, ls, bash"
    fi
    return
  fi
  while IFS= read -r _tok; do
    _tok="${_tok#"${_tok%%[![:space:]]*}"}"
    _tok="${_tok%"${_tok##*[![:space:]]}"}"
    [ -z "$_tok" ] && continue
    case "$_tok" in
    Read | read) _mapped="read" ;;
    Edit | edit) _mapped="edit" ;;
    Write | write) _mapped="write" ;;
    Bash | bash) _mapped="bash" ;;
    Grep | grep) _mapped="grep" ;;
    Glob | glob) _mapped="find" ;;
    LS | ls) _mapped="ls" ;;
    Skill | skill) continue ;;
    *) continue ;;
    esac
    _out="${_out}${_sep}${_mapped}"
    _sep=", "
  done < <(printf '%s\n' "$_raw" | tr ',' '\n')
  if [ -z "$_out" ]; then
    if [[ "$_name" == devrites-slice-wright || "$_name" == overhaul-* ]]; then
      _out="read, edit, write, bash, grep, find, ls"
    else
      _out="read, grep, find, ls, bash"
    fi
  fi
  printf '%s' "$_out"
}

# pi-subagents treats frontmatter tools: as a strict allowlist, so ambient
# pi-lean-ctx / pi-lens tools are dropped unless named. Append those names
# after the mapped builtins (preserve base order). Unknown names are pruned
# non-fatally by pi-subagents. Read-only agents stay read-only: ctx_edit /
# ctx_patch only when the base already has edit or write; ctx_shell only
# when the base already has bash.
_PI_EXTENSION_NAV_TOOLS="ctx_read, ctx_ls, ctx_find, ctx_grep, ctx_glob, ctx_search, ctx_compose, ctx_callgraph, ctx_tree, symbol_search, project_report, module_report, read_symbol, read_enclosing, lens_diagnostics"

_pi_csv_has() {
  case ", ${1}, " in
  *", ${2}, "*) return 0 ;;
  *) return 1 ;;
  esac
}

_pi_csv_append() {
  local _list="$1" _name="$2"
  if [ -z "$_name" ]; then
    printf '%s' "$_list"
    return
  fi
  if [ -z "$_list" ]; then
    printf '%s' "$_name"
    return
  fi
  if _pi_csv_has "$_list" "$_name"; then
    printf '%s' "$_list"
    return
  fi
  printf '%s, %s' "$_list" "$_name"
}

_pi_append_extension_tools() {
  local _base="$1"
  local _out="$_base" _tok
  while IFS= read -r _tok; do
    _tok="${_tok#"${_tok%%[![:space:]]*}"}"
    _tok="${_tok%"${_tok##*[![:space:]]}"}"
    [ -z "$_tok" ] && continue
    _out="$(_pi_csv_append "$_out" "$_tok")"
  done < <(printf '%s\n' "$_PI_EXTENSION_NAV_TOOLS" | tr ',' '\n')
  if _pi_csv_has "$_base" "edit" || _pi_csv_has "$_base" "write"; then
    _out="$(_pi_csv_append "$_out" "ctx_edit")"
    _out="$(_pi_csv_append "$_out" "ctx_patch")"
  fi
  if _pi_csv_has "$_base" "bash"; then
    _out="$(_pi_csv_append "$_out" "ctx_shell")"
  fi
  printf '%s' "$_out"
}

# Generate a pi-subagents Markdown agent from a Claude Code markdown agent.
# pi-subagents loads project agents from .pi/agents/**/*.md (YAML frontmatter +
# body). The tools allowlist carries the Claude permission boundary: only
# devrites-slice-wright and the /overhaul agents (overhaul-*) get write/edit; reviewers stay read-only.
# inheritProjectContext keeps AGENTS.md/CLAUDE.md visible, matching Claude
# Code subagent semantics. A canonical `skills:` block list passes through.
gen_pi_agent() {
  local _src="$1" _out="$2"
  local _name _desc _tools_raw _tools _skills _desc_tmp _desc_pi _body_tmp _body_pi
  _name="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^name:[[:space:]]*/{sub(/^name:[[:space:]]*/, ""); print; exit}' "$_src")"
  _desc="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^description:[[:space:]]*/{sub(/^description:[[:space:]]*/, ""); print; exit}' "$_src")"
  _tools_raw="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^tools:[[:space:]]*/{sub(/^tools:[[:space:]]*/, ""); print; exit}' "$_src")"
  _skills="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^skills:[[:space:]]*$/{list=1; next} list && /^[[:space:]]*-[[:space:]]*/{sub(/^[[:space:]]*-[[:space:]]*/, ""); print; next} list{exit}' "$_src" | tr '\n' ' ' | sed 's/ $//; s/ /, /g')"
  [ -n "$_name" ] || _name="$(basename "$_src" .md)"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  _tools="$(_pi_append_extension_tools "$(_pi_map_tools "$_tools_raw" "$_name")")"
  _desc_tmp="$TMP_GEN_DIR/pi-agent-desc-$(basename "$_src").txt"
  _desc_pi="$TMP_GEN_DIR/pi-agent-desc-$(basename "$_src").pi.txt"
  printf '%s' "$_desc" >"$_desc_tmp"
  gen_pi_markdown_file "$_desc_tmp" "$_desc_pi"
  _desc="$(cat "$_desc_pi")"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  mkdir -p "$(dirname "$_out")"
  {
    printf '%s\n' "---"
    printf 'name: %s\n' "$_name"
    printf 'description: "%s"\n' "$(pi_yaml_escape "$_desc")"
    printf 'tools: %s\n' "$_tools"
    printf 'inheritProjectContext: true\n'
    [ -n "$_skills" ] && printf 'skills: %s\n' "$_skills"
    printf '%s\n' "---"
    _body_tmp="$TMP_GEN_DIR/pi-agent-body-$(basename "$_src").md"
    _body_pi="$TMP_GEN_DIR/pi-agent-body-$(basename "$_src").pi.md"
    awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{fm=0; body=1; next} body{print}' "$_src" >"$_body_tmp"
    gen_pi_markdown_file "$_body_tmp" "$_body_pi"
    cat "$_body_pi"
  } >"$_out"
}

# Write a pi prompt template that preserves the public /<name> command form.
# Pi expands .pi/prompts/<name>.md into the prompt; the stub hands off to the
# installed skill so /rite-build behaves like /skill:rite-build.
gen_pi_prompt_stub() {
  local _skill_md="$1" _name="$2" _out="$3"
  local _desc _hint
  _desc="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^description:[[:space:]]*/{sub(/^description:[[:space:]]*/, ""); print; exit}' "$_skill_md")"
  _hint="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^argument-hint:[[:space:]]*/{sub(/^argument-hint:[[:space:]]*/, ""); print; exit}' "$_skill_md")"
  [ -n "$_desc" ] || _desc="DevRites $_name."
  mkdir -p "$(dirname "$_out")"
  {
    printf '%s\n' "---"
    printf 'description: "%s"\n' "$(pi_yaml_escape "$_desc")"
    # An already double-quoted YAML scalar passes through verbatim.
    case "$_hint" in
    \"*\") printf 'argument-hint: %s\n' "$_hint" ;;
    ?*) printf 'argument-hint: "%s"\n' "$(pi_yaml_escape "$_hint")" ;;
    esac
    printf '%s\n' "---"
    printf 'Read and follow the DevRites skill at `.pi/skills/%s/SKILL.md`, applying it to: ${ARGUMENTS:-the current request}.\n' "$_name"
  } >"$_out"
}

gen_pi_agents_bridge() {
  _out="$1"
  cat >"$_out" <<'EOF'
<!-- BEGIN DEVRITES PI -->
## DevRites For pi

This project has DevRites installed for pi.

## pi usage

- DevRites workflow skills live in `.pi/skills`. Public commands run as `/rite`, `/rite-spec`, ... (prompt templates under `.pi/prompts`) or `/skill:rite`, `/skill:rite-spec`, ...; the forms are equivalent.
- Before using any DevRites workflow skill, read `.pi/skills/devrites-lib/reference/standards/core.md`. Load other `.pi/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them.
- DevRites specialist agents live in `.pi/agents` and are provided by the `pi-subagents` extension. Before dispatch, run `subagent({ action: "list" })`; if a required `devrites-<role>` is absent or not executable, stop for HITL — never skip it, substitute a generic child, or run the specialist role in the root context.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `subagent({ agent, task })` (or `runs.run`/`runs.all` in a `workflowScript`), wait for it, and reconcile its result.
- Only `devrites-slice-wright` may edit source or tests; every other specialist is read-only by tool allowlist. Exact paths are instruction-enforced: put the project-relative paths in the task, wait for the wright, compare its file list and `git diff --name-only` with that contract, and reject any extra path.
- The explicit `/overhaul` skill ships its own `overhaul-*` agents outside the DevRites lifecycle. They carry the full write tool set, are dispatched only by that skill, and follow its own approval gate and path contracts instead of the lifecycle writer rules above.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Installed `.pi/` content loads only after the project is trusted. If pi has not trusted this project, the skills, agents, and prompts above are not active.
- A seal GO, AFK mode, or autocomplete flag never authorizes an irreversible action. Disclose the exact commit/push/tag/PR plan and obtain fresh explicit user approval for that attempt; any changed or retried plan needs fresh approval.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES PI -->
EOF
}
