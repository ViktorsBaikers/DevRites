#!/usr/bin/env bash
# codex-generate.sh: shared Claude-to-Codex surface generators.
#
# This file is a shell library, not a standalone installer. Callers provide
# TMP_GEN_DIR and then invoke the gen_* helpers below.

gen_codex_markdown_file() {
  local _src="$1" _out="$2"
  mkdir -p "$(dirname "$_out")"
  sed -E \
    -e 's#(pack/)?\.claude/agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.codex/agents/devrites-security-auditor.toml`, `.codex/agents/devrites-performance-reviewer.toml`, or `.codex/agents/devrites-simplifier-reviewer.toml#g' \
    -e 's#(\.\./)+agents/devrites-\{security-auditor,performance-reviewer,simplifier-reviewer\}\.md#.codex/agents/devrites-security-auditor.toml`, `.codex/agents/devrites-performance-reviewer.toml`, or `.codex/agents/devrites-simplifier-reviewer.toml#g' \
    -e 's#pack/\.claude/skills/devrites-lib/scripts/#.agents/skills/devrites-lib/scripts/#g' \
    -e 's#pack/\.claude/skills/#.agents/skills/#g' \
    -e 's#pack/\.claude/agents/([A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#pack/\.claude/agents/#.codex/agents/#g' \
    -e 's#\.claude/skills/devrites-lib/scripts/#.agents/skills/devrites-lib/scripts/#g' \
    -e 's#\.claude/skills/#.agents/skills/#g' \
    -e 's#\.claude/agents/([A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#\.claude/agents/#.codex/agents/#g' \
    -e 's#(\.\./)+agents/([A-Za-z0-9_-]+)\.md#.codex/agents/\2.toml#g' \
    -e 's#\.codex/agents/([A-Za-z0-9_-]+)\.md#.codex/agents/\1.toml#g' \
    -e 's#Host mapping: Claude Code uses `Agent` \(`Task` is its legacy alias\); Codex uses#Host mapping: Codex fresh-context dispatch uses#g' \
    -e 's#`Agent` call#`spawn_agent` call#g' \
    -e 's#Task breakdown#Work-item breakdown#g' \
    -e 's#(^|[^A-Za-z0-9_])Task([^A-Za-z0-9_]|$)#\1spawn_agent\2#g' \
    -e 's#dispatch `devrites-source-driven`#invoke `devrites-source-driven`#g' \
    -e 's#Dispatch `devrites-source-driven`#Invoke `devrites-source-driven`#g' \
    -e 's#pass-through dispatch to the matching `rite-<verb>` skill#pass-through invocation of the matching `rite-<verb>` skill#g' \
    -e 's#verb dispatches to the matching `rite-<verb>` skill#verb invokes the matching `rite-<verb>` skill#g' \
    -e 's#the dispatch map only#the invocation map only#g' \
    -e 's#→ dispatch per the table above#→ invoke the matching skill per the table above#g' \
    -e 's#(^|[^A-Za-z0-9_./-])/(rite(-[a-z0-9-]+)?)([^A-Za-z0-9_-]|$)#\1$\2\4#g' \
    "$_src" > "$_out"
}

gen_codex_skill_file() {
  gen_codex_markdown_file "$1" "$2"
}

toml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

# Generate a Codex custom-agent TOML file from a Claude Code markdown agent.
# Codex does not consume .claude/agents/*.md directly; it loads project agents
# from .codex/agents/*.toml.
gen_codex_agent() {
  local _src="$1" _out="$2"
  local _name _desc _desc_tmp _desc_codex _body_tmp _body_codex _permissions
  _name="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^name:[[:space:]]*/{sub(/^name:[[:space:]]*/, ""); print; exit}' "$_src")"
  _desc="$(awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{exit} fm && /^description:[[:space:]]*/{sub(/^description:[[:space:]]*/, ""); print; exit}' "$_src")"
  [ -n "$_name" ] || _name="$(basename "$_src" .md)"
  [ -n "$_desc" ] || _desc="DevRites custom agent."
  _permissions=":read-only"
  [ "$_name" = "devrites-slice-wright" ] && _permissions=":workspace"
  _desc_tmp="$TMP_GEN_DIR/codex-agent-desc-$(basename "$_src").txt"
  _desc_codex="$TMP_GEN_DIR/codex-agent-desc-$(basename "$_src").codex.txt"
  printf '%s' "$_desc" > "$_desc_tmp"
  gen_codex_markdown_file "$_desc_tmp" "$_desc_codex"
  _desc="$(cat "$_desc_codex")"
  mkdir -p "$(dirname "$_out")"
  {
    printf 'name = "%s"\n' "$(toml_escape "$_name")"
    printf 'description = "%s"\n' "$(toml_escape "$_desc")"
    printf 'default_permissions = "%s"\n' "$_permissions"
    printf "%s\n" "developer_instructions = '''"
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

- **Inspect before trust.** Codex skips project-scoped `.codex/` configuration and agents in an untrusted project. Before enabling them, inspect `.codex/config.toml`, `.codex/agents/`, and the project guidance. The human operator decides whether to trust the folder.
- DevRites workflow skills are available to Codex from `.agents/skills`.
- Use `$rite` or `$rite-<verb>` through Codex skills, or open `/skills` and select the matching DevRites skill.
- If the user mentions a DevRites slash command such as `/rite spec`, `/rite-build`, or `/rite-seal`, treat that as an explicit request to use the corresponding DevRites skill.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Before using any DevRites workflow skill, read `.agents/skills/devrites-lib/reference/standards/core.md`. Load other `.agents/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them. These are DevRites engineering standards, not Codex exec-policy `.rules` files.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Custom Codex subagents generated from the DevRites review agents live in `.codex/agents`.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `spawn_agent`, wait for it, and reconcile its result.
- Dispatch every exact named `devrites-<role>` required by the active workflow in a fresh subagent thread, wait for it, and reconcile its result. If that role is unavailable, stop for HITL; never skip it, never substitute a generic/default child, and never execute the specialist role in the root context.
- The root uses the workspace-capable `devrites-orchestrator` profile because Codex children cannot elevate above the parent permission ceiling. This permission is for native writer dispatch and exact path-bounded executable workflow artifacts under the active `.devrites/work/<slug>/`; the root must never edit product source or tests itself.
- Every generated specialist is hook-free. `devrites-slice-wright` alone uses `default_permissions = ":workspace"`; every other specialist uses `default_permissions = ":read-only"`. Exact paths are instruction-enforced: put the project-relative paths in the task, wait for the wright, compare its file list and `git diff --name-only` with that contract, and reject any extra path. Never bypass the wright, widen its contract, or edit source in the root; the root must never recreate an engine dispatch bridge.
- A seal GO, AFK mode, or autocomplete flag never authorizes an irreversible action. Disclose the exact commit/push/tag/PR plan and obtain fresh explicit user approval for that attempt; any changed or retried plan needs fresh approval. Native host permission and sandbox prompts remain authoritative and cannot be inferred or bypassed.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES CODEX -->
EOF
}

gen_codex_config_toml() {
  _out="$1"
  cat > "$_out" <<'EOF'
# BEGIN DEVRITES CODEX PERMISSIONS
default_permissions = "devrites-orchestrator"

[permissions.devrites-orchestrator]
description = "Workspace-capable orchestrator; source and test edits must be delegated to devrites-slice-wright."
extends = ":workspace"
# END DEVRITES CODEX PERMISSIONS
EOF
}
