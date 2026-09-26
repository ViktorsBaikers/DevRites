#!/usr/bin/env bash
# devin-generate.sh: shared Claude-to-Devin surface generators.
#
# This file is a shell library, not a standalone installer. Callers provide
# TMP_GEN_DIR and then invoke the gen_* helpers below.

# Rewrite canonical Claude (and leftover Codex/omp/pi) paths to installed
# Devin paths. Slash /rite invocations stay slash-form: Devin discovers
# .devin/skills/<name>/SKILL.md as /name. Dispatch wording uses the
# `run_subagent` tool with `profile`, matching Devin CLI custom subagents.
gen_devin_markdown_file() {
  local _src="$1" _out="$2"
  mkdir -p "$(dirname "$_out")"
  sed -E \
    -e 's#Try post-install path first, fall back to pre-install:#Resolve the installed skill path:#g' \
    -e '/^\[ -f "\$F" \] \|\| F=[^ ].*SKILL\.md$/d' \
    -e 's#(pack/)?\.claude/agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.devin/agents/devrites-security-auditor.md`, `.devin/agents/devrites-performance-reviewer.md`, or `.devin/agents/devrites-simplifier-reviewer.md#g' \
    -e 's#(\.\./)+agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.devin/agents/devrites-security-auditor.md`, `.devin/agents/devrites-performance-reviewer.md`, or `.devin/agents/devrites-simplifier-reviewer.md#g' \
    -e 's#pack/\.claude/skills/devrites-lib/scripts/#.devin/skills/devrites-lib/scripts/#g' \
    -e 's#pack/\.claude/skills/#.devin/skills/#g' \
    -e 's#pack/\.claude/agents/([A-Za-z0-9_-]+)\.md#.devin/agents/\1.md#g' \
    -e 's#pack/\.claude/agents/#.devin/agents/#g' \
    -e 's#\.claude/skills/devrites-lib/scripts/#.devin/skills/devrites-lib/scripts/#g' \
    -e 's#\.claude/skills/#.devin/skills/#g' \
    -e 's#\.claude/agents/([A-Za-z0-9_-]+)\.md#.devin/agents/\1.md#g' \
    -e 's#\.claude/agents/#.devin/agents/#g' \
    -e 's#(\.\./)+agents/([A-Za-z0-9_-]+)\.md#.devin/agents/\2.md#g' \
    -e 's#\.agents/skills/devrites-lib/scripts/#.devin/skills/devrites-lib/scripts/#g' \
    -e 's#\.agents/skills/#.devin/skills/#g' \
    -e 's#\.codex/skills/devrites-lib/scripts/#.devin/skills/devrites-lib/scripts/#g' \
    -e 's#\.codex/skills/#.devin/skills/#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.toml#.devin/agents/\1.md#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.md#.devin/agents/\1.md#g' \
    -e 's#\.codex/agents/#.devin/agents/#g' \
    -e 's#\.omp/skills/#.devin/skills/#g' \
    -e 's#\.omp/agents/([A-Za-z0-9_-]+)\.md#.devin/agents/\1.md#g' \
    -e 's#\.omp/agents/#.devin/agents/#g' \
    -e 's#\.pi/skills/#.devin/skills/#g' \
    -e 's#\.pi/agents/([A-Za-z0-9_-]+)\.md#.devin/agents/\1.md#g' \
    -e 's#\.pi/agents/#.devin/agents/#g' \
    -e '/^\(use the `[^`]*` mirror on Codex\)\.$/d' \
    -e '/^\(`[^`]+` mirror on Codex\)\.$/d' \
    -e 's# \(use the `[^`]*` mirror on Codex\)##g' \
    -e 's# \(`[^`]+` mirror on Codex\)##g' \
    -e 's#\(`[^`]+` mirror on Codex\); ##g' \
    -e 's#\(`[^`]+` mirror on Codex\)##g' \
    -e 's#^seeded verdicts#Seeded verdicts#' \
    -e '/Rules in scope/{N;s#;\n[[:space:]]*`[^`]+` mirror on Codex##;}' \
    -e '/On Codex, use the/{N;s# On Codex, use the mirrors under\n`[^`]+`\.##;s# On Codex, use the\nmirrors under `[^`]+`\.##;s# On Codex, use the\nmirror under `[^`]+`\.##;}' \
    -e 's#Host mapping: Claude Code uses `Agent` \(`Task` is its legacy alias\); Codex uses#Host mapping: Devin fresh-context dispatch uses `run_subagent` with `profile`; hosts use#g' \
    -e 's#`Agent` call#`run_subagent` call#g' \
    -e 's#`spawn_agent` call#`run_subagent` call#g' \
    -e 's#Claude: N concurrent Task wrights#Devin: N concurrent `run_subagent` wrights#g' \
    -e 's#Task breakdown#Work-item breakdown#g' \
    -e 's#dispatch `devrites-source-driven`#invoke `devrites-source-driven`#g' \
    -e 's#Dispatch `devrites-source-driven`#Invoke `devrites-source-driven`#g' \
    -e 's#pass-through dispatch to the matching `rite-<verb>` skill#pass-through invocation of the matching `rite-<verb>` skill#g' \
    -e 's#verb dispatches to the matching `rite-<verb>` skill#verb invokes the matching `rite-<verb>` skill#g' \
    -e 's#the dispatch map only#the invocation map only#g' \
    -e 's#→ dispatch per the table above#→ invoke the matching skill per the table above#g' \
    "$_src" >"$_out"
}

# Translate the Claude skill invocation flags into Devin's `triggers` list:
# user-invocable maps to the `user` trigger and model invocation maps to the
# `model` trigger. Both flags restrictive yields `triggers: []` (never
# auto- or slash-invoked); neither restrictive omits the field so Devin's
# default (user + model) applies.
gen_devin_skill_file() {
  local _src="$1" _out="$2"
  local _ui _dmi _tmp="$TMP_GEN_DIR/devin-skill-$(basename "$(dirname "$_src")")-$(basename "$_src")"
  _ui="$(awk 'NR==1&&$0=="---"{fm=1;next} fm&&$0=="---"{exit} fm&&/^user-invocable:[[:space:]]*/{sub(/^user-invocable:[[:space:]]*/,""); gsub(/[[:space:]]/,""); print; exit}' "$_src")"
  _dmi="$(awk 'NR==1&&$0=="---"{fm=1;next} fm&&$0=="---"{exit} fm&&/^disable-model-invocation:[[:space:]]*/{sub(/^disable-model-invocation:[[:space:]]*/,""); gsub(/[[:space:]]/,""); print; exit}' "$_src")"
  gen_devin_markdown_file "$_src" "$_tmp"
  mkdir -p "$(dirname "$_out")"
  awk -v ui="$_ui" -v dmi="$_dmi" '
    NR==1 && $0=="---" { fm=1; print; next }
    fm==1 && $0=="---" {
      u = (ui != "false")
      m = (dmi != "true")
      if (!u && !m) {
        print "triggers: []"
      } else if (!u || !m) {
        print "triggers:"
        if (u) print "  - user"
        if (m) print "  - model"
      }
      fm=0; print; next
    }
    fm==1 && (/^user-invocable:[[:space:]]/ || /^disable-model-invocation:[[:space:]]/) { next }
    { print }
  ' "$_tmp" >"$_out"
}

devin_yaml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

# Map Claude Code tool names onto Devin tool names. Devin's allowed-tools
# field takes its builtin names: read, edit, write, exec, grep, glob,
# webfetch, web_search, and skill. Unmapped Claude tools are dropped.
_devin_map_tools() {
  local _raw="$1" _name="$2"
  local _tok _mapped _out="" _sep=""
  if [ -z "$_raw" ]; then
    if [[ "$_name" == devrites-slice-wright || "$_name" == overhaul-* || "$_name" == fast-builder ]]; then
      printf '%s' "read, edit, write, exec, grep, glob, skill"
    else
      printf '%s' "read, grep, glob, exec"
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
      Bash | bash | exec) _mapped="exec" ;;
      Glob | glob) _mapped="glob" ;;
      Grep | grep) _mapped="grep" ;;
      Skill | skill) _mapped="skill" ;;
      WebFetch | webfetch) _mapped="webfetch" ;;
      WebSearch | web_search | websearch) _mapped="web_search" ;;
      *) continue ;;
    esac
    _out="${_out}${_sep}${_mapped}"
    _sep=", "
  done < <(printf '%s\n' "$_raw" | tr ',' '\n')
  if [ -z "$_out" ]; then
    if [[ "$_name" == devrites-slice-wright || "$_name" == overhaul-* || "$_name" == fast-builder ]]; then
      _out="read, edit, write, exec, grep, glob, skill"
    else
      _out="read, grep, glob, exec"
    fi
  fi
  printf '%s' "$_out"
}

_devin_csv_has() {
  case ", ${1}, " in
  *", ${2}, "*) return 0 ;;
  *) return 1 ;;
  esac
}

# Generate a Devin custom subagent profile from a Claude Code markdown agent.
# Devin loads project profiles from .devin/agents/*.md (YAML frontmatter +
# body). The allowed-tools allowlist carries the Claude permission boundary:
# only devrites-slice-wright, the /overhaul agents (overhaul-*), and /rite-fast fast-builder get write/edit; reviewers stay read-only.
# Claude's permissionMode has no Devin equivalent and is dropped. A canonical
# `skills:` block grants the agent the `skill` tool so it can invoke that
# skill inline (Devin has no agent-level skills preload field).
gen_devin_agent() {
  local _src="$1" _out="$2"
  local _name _desc _tools_raw _tools _skills _desc_tmp _desc_devin _body_tmp _body_devin _tok
  _name="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^name:[[:space:]]*/{sub(/^name:[[:space:]]*/, ""); print; exit}' "$_src")"
  _desc="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^description:[[:space:]]*/{sub(/^description:[[:space:]]*/, ""); print; exit}' "$_src")"
  _tools_raw="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^tools:[[:space:]]*/{sub(/^tools:[[:space:]]*/, ""); print; exit}' "$_src")"
  _skills="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^skills:[[:space:]]*$/{list=1; next} list && /^[[:space:]]*-[[:space:]]*/{found=1; exit} list{exit} END{if(found) print "yes"}' "$_src")"
  [ -n "$_name" ] || _name="$(basename "$_src" .md)"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  _tools="$(_devin_map_tools "$_tools_raw" "$_name")"
  if [ -n "$_skills" ] && ! _devin_csv_has "$_tools" "skill"; then
    _tools="$_tools, skill"
  fi
  _desc_tmp="$TMP_GEN_DIR/devin-agent-desc-$(basename "$_src").txt"
  _desc_devin="$TMP_GEN_DIR/devin-agent-desc-$(basename "$_src").devin.txt"
  printf '%s' "$_desc" >"$_desc_tmp"
  gen_devin_markdown_file "$_desc_tmp" "$_desc_devin"
  _desc="$(cat "$_desc_devin")"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  mkdir -p "$(dirname "$_out")"
  {
    printf '%s\n' "---"
    printf 'name: %s\n' "$_name"
    printf 'description: "%s"\n' "$(devin_yaml_escape "$_desc")"
    printf 'allowed-tools:\n'
    printf '%s\n' "$_tools" | tr ',' '\n' | while IFS= read -r _tok; do
      _tok="${_tok#"${_tok%%[![:space:]]*}"}"
      _tok="${_tok%"${_tok##*[![:space:]]}"}"
      [ -n "$_tok" ] && printf '  - %s\n' "$_tok"
    done
    printf '%s\n' "---"
    _body_tmp="$TMP_GEN_DIR/devin-agent-body-$(basename "$_src").md"
    _body_devin="$TMP_GEN_DIR/devin-agent-body-$(basename "$_src").devin.md"
    awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{fm=0; body=1; next} body{print}' "$_src" >"$_body_tmp"
    gen_devin_markdown_file "$_body_tmp" "$_body_devin"
    cat "$_body_devin"
  } >"$_out"
}

gen_devin_agents_bridge() {
  _out="$1"
  cat >"$_out" <<'EOF'
<!-- BEGIN DEVRITES DEVIN -->
## DevRites For Devin

This project has DevRites installed for Devin CLI.

## Devin usage

- DevRites workflow skills live in `.devin/skills`. Run them as `/rite`, `/rite-spec`, and the other `/rite-*` slash commands.
- Before using any DevRites workflow skill, read `.devin/skills/devrites-lib/reference/standards/core.md`. Load other `.devin/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them.
- DevRites specialist agents live in `.devin/agents` as custom subagent profiles offered to `run_subagent`. If a required `devrites-<role>` profile is not among the offered profiles, stop for HITL - never skip the role, never substitute `subagent_general`, `subagent_explore`, or another generic profile, never inline the specialist prompt, and never execute the specialist role in the root context.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `run_subagent` using the exact `profile` name, wait for it with `read_subagent`, and reconcile its result.
- Only `devrites-slice-wright` may edit source or tests; every other specialist is read-only by `allowed-tools`. Exact paths are instruction-enforced: put the project-relative paths in the task, wait for the wright, compare its file list and `git diff --name-only` with that contract, and reject any extra path.
- The explicit `/overhaul` skill ships its own `overhaul-*` agents outside the DevRites lifecycle. They carry the full write tool set, are dispatched only by that skill, and follow its own approval gate and path contracts instead of the lifecycle writer rules above.
- The explicit `/rite-fast` skill ships its own `fast-*` agents. Only `fast-builder` carries write tools; `fast-planner`, `fast-checker`, and `fast-critic` stay read-only. `/rite-fast` dispatches them itself and applies its own exact-path contracts.
- Custom profiles do not inherit interactive tool grants, and background subagents auto-deny unapproved tools. Dispatch the write-capable wright in the foreground, or make sure its `edit`, `write`, and `exec` tools are approved first. Specialists cannot use `ask_user_question`; the root relays user questions and answers through the task text and returned result.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Skills and agent profiles are loaded when a session starts. If they were installed while this session was open, restart the session or reopen the project before relying on them.
- A seal GO, AFK mode, or autocomplete flag never authorizes an irreversible action. Disclose the exact commit/push/tag/PR plan and obtain fresh explicit user approval for that attempt; any changed or retried plan needs fresh approval.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES DEVIN -->
EOF
}
