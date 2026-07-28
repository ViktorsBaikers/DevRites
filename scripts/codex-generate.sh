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
  _src="$1"; _out="$2"; _internal="${3:-0}"
  _tmp="$TMP_GEN_DIR/codex-skill-raw-$(basename "$(dirname "$_src")").md"
  mkdir -p "$(dirname "$_out")"
  # Give the 12 internal skills a short description in the Codex mirror. Codex
  # uses descriptions for its 2% implicit-matching budget, but these skills are
  # invoked by name. Shorter internal descriptions leave that budget for the
  # public rite-* skills. Keep explicit invocation and each SKILL.md body
  # unchanged.
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
      print "- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task."
      print "- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing."
      print "- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation."
      print "- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch."
      print "- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns=\"none\"`. Codex loads that role TOML'\''s `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result."
      print "- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns=\"none\"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract'\''s exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard."
      print "- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns=\"none\"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role."
      print "- The invoked skill'\''s `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn."
      print "- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context."
      print "- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete."
      print "- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement."
      print "- When this skill asks a HITL question via `AskUserQuestion`: Codex'\''s equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK'\''s contract, gated by the `.devrites/AFK` sentinel."
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
        print "Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Invocation runs a skill inline; dispatch starts a fresh agent with `spawn_agent` and must be awaited. On V2 use the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns=\"none\"`; on V1 use guarded `explorer`/`worker` with the unchanged packet and exact role TOML. If any required dispatch is unavailable or rejected, stop for HITL; never execute the specialist role in the root context."
      }
    }
  ' "$_src" > "$_tmp"
  gen_codex_markdown_file "$_tmp" "$_out"
}

toml_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

gen_codex_leaf_hook_toml() {
  local _active_agent="$1" _required_var="$2" _subcommand="$3" _status="$4" _command
  _command="cd \"\$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || { printf '%s\\n' 'DevRites: cannot resolve the project root for a declared Codex leaf. (devrites-codex-leaf-guard)' >&2; exit 2; }; DEVRITES_AGENT_RUN=1 DEVRITES_ACTIVE_AGENT=${_active_agent} ${_required_var}=1 devrites-engine hook ${_subcommand} --harness=codex; rc=\$?; case \"\$rc\" in 0) exit 0 ;; 2) exit 2 ;; *) printf '%s\\n' 'DevRites: declared Codex leaf guard unavailable or crashed; install or repair devrites-engine. (devrites-codex-leaf-guard)' >&2; exit 2 ;; esac"
  printf '\n[[hooks.PreToolUse]]\n'
  printf 'matcher = "Bash|Shell|sh|exec_command|run_command|Edit|Write|MultiEdit|NotebookEdit|apply_patch|exec|js|python|computer|computer_use|write_stdin|run_code|Agent|Task|spawn_agent|delegate|dispatch_agent|create_agent"\n\n'
  printf '[[hooks.PreToolUse.hooks]]\n'
  printf 'type = "command"\n'
  printf "command = '''%s'''\n" "$_command"
  printf 'statusMessage = "%s"\n' "$_status"
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
    printf 'Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.\n\n'
    printf 'For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.\n\n'
    _body_tmp="$TMP_GEN_DIR/codex-agent-body-$(basename "$_src").md"
    _body_codex="$TMP_GEN_DIR/codex-agent-body-$(basename "$_src").codex.md"
    awk 'NR==1 && $0=="---"{fm=1; next} fm && $0=="---"{fm=0; body=1; next} body{print}' "$_src" > "$_body_tmp"
    gen_codex_markdown_file "$_body_tmp" "$_body_codex"
    cat "$_body_codex"
    printf "\n%s\n" "'''"
    case "$_name" in
      devrites-slice-wright)
        gen_codex_leaf_hook_toml \
          "$_name" \
          "DEVRITES_WRIGHT_AGENT_REQUIRED" \
          "wright-scope" \
          "DevRites: checking slice-wright boundary"
        ;;
      *)
        gen_codex_leaf_hook_toml \
          "$_name" \
          "DEVRITES_REVIEWER_AGENT_REQUIRED" \
          "reviewer-readonly" \
          "DevRites: checking reviewer read-only boundary"
        ;;
    esac
  } > "$_out"
}

gen_codex_agents_bridge() {
  _out="$1"
  cat > "$_out" <<'EOF'
<!-- BEGIN DEVRITES CODEX -->
## DevRites For Codex

This project has DevRites installed for both Claude Code and Codex.

## Codex usage

- **Inspect before trust.** Codex skips project-scoped `.codex/` hooks, agents, and rules in an untrusted project. Before enabling them, inspect `.codex/hooks.json`, `.codex/agents/`, and the project guidance; use `/hooks` to review the commands Codex would run. The human operator decides whether to trust the folder. Until then, use DevRites only through explicitly inspected commands and treat hook enforcement as unavailable.
- DevRites workflow skills are available to Codex from `.agents/skills`.
- Use `$rite` or `$rite-<verb>` through Codex skills, or open `/skills` and select the matching DevRites skill.
- If the user mentions a DevRites slash command such as `/rite spec`, `/rite-build`, or `/rite-seal`, treat that as an explicit request to use the corresponding DevRites skill.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Before using any DevRites workflow skill, read `.agents/skills/devrites-lib/reference/standards/core.md`. Load other `.agents/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them. These are DevRites engineering standards, not Codex exec-policy `.rules` files.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Custom Codex subagents generated from the DevRites review agents live in `.codex/agents`.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `spawn_agent`, wait for it, and reconcile its result.
- Follow the invoked skill's generated **Codex compatibility** section for exact dispatch mechanics: V2 uses the named `devrites-<role>`; V1 alone may use the guarded generic compatibility path. Both use `fork_turns="none"`.
- The invoked skill's `required-agent-roles` frontmatter arms a fail-closed receipt. If a required dispatch is unavailable or rejected, stop for HITL; never execute a DevRites specialist role in the root context.
- Claude Code agent hook metadata is not active in Codex. The generated Codex agents preserve read-only intent with Codex sandbox settings where possible; still follow DevRites' scope and no-mutation rules explicitly.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES CODEX -->
EOF
}

gen_codex_hooks_json() {
  _out="$1"
  cat > "$_out" <<'EOF'
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Shell|sh|exec_command|run_command",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook git-guard --harness=codex",
            "statusMessage": "DevRites: checking destructive Git authority"
          },
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
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec env DEVRITES_A1_HOOK=enforce devrites-engine hook a1-guard --harness=codex",
            "statusMessage": "DevRites: checking build write boundary"
          }
        ]
      },
      {
        "matcher": "Bash|Shell|sh|exec_command|run_command|Edit|Write|MultiEdit|NotebookEdit|apply_patch|exec|js|python|computer|computer_use|write_stdin|run_code|Agent|Task|spawn_agent|delegate|dispatch_agent|create_agent|wait|wait_agent",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook agent-dispatch --harness=codex",
            "statusMessage": "DevRites: verifying agent dispatch receipt"
          },
          {
            "type": "command",
            "command": "payload=\"$(cat)\"; if ! command -v devrites-engine >/dev/null 2>&1; then if printf '%s' \"$payload\" | grep -Eq '\"agent_type\"[[:space:]]*:[[:space:]]*\"(explorer|worker)\"'; then printf '%s\\n' 'DevRites: generic Codex leaf guard unavailable; install or repair devrites-engine. (devrites-codex-generic-guard)' >&2; exit 2; fi; exit 0; fi; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || { printf '%s\\n' 'DevRites: cannot resolve the project root for a generic Codex leaf. (devrites-codex-generic-guard)' >&2; exit 2; }; printf '%s' \"$payload\" | DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1 devrites-engine hook reviewer-readonly --harness=codex; rc=$?; case \"$rc\" in 0) exit 0 ;; 2) exit 2 ;; *) printf '%s\\n' 'DevRites: generic Codex reviewer guard unavailable or crashed; install or repair devrites-engine. (devrites-codex-generic-guard)' >&2; exit 2 ;; esac",
            "statusMessage": "DevRites: checking generic reviewer boundary"
          },
          {
            "type": "command",
            "command": "payload=\"$(cat)\"; if ! command -v devrites-engine >/dev/null 2>&1; then if printf '%s' \"$payload\" | grep -Eq '\"agent_type\"[[:space:]]*:[[:space:]]*\"(explorer|worker)\"'; then printf '%s\\n' 'DevRites: generic Codex leaf guard unavailable; install or repair devrites-engine. (devrites-codex-generic-guard)' >&2; exit 2; fi; exit 0; fi; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || { printf '%s\\n' 'DevRites: cannot resolve the project root for a generic Codex leaf. (devrites-codex-generic-guard)' >&2; exit 2; }; printf '%s' \"$payload\" | DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1 devrites-engine hook wright-scope --harness=codex; rc=$?; case \"$rc\" in 0) exit 0 ;; 2) exit 2 ;; *) printf '%s\\n' 'DevRites: generic Codex wright guard unavailable or crashed; install or repair devrites-engine. (devrites-codex-generic-guard)' >&2; exit 2 ;; esac",
            "statusMessage": "DevRites: checking generic wright boundary"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Shell|sh|exec_command|run_command",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook redwatch --harness=codex",
            "statusMessage": "DevRites: checking test/build result"
          }
        ]
      },
      {
        "matcher": "Bash|Shell|sh|exec_command|run_command|Edit|Write|MultiEdit|NotebookEdit|apply_patch|exec|js|python|computer|computer_use|write_stdin|run_code|Agent|Task|spawn_agent|delegate|dispatch_agent|create_agent|wait|wait_agent",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook agent-dispatch --harness=codex",
            "statusMessage": "DevRites: retaining the pre-writer canonical boundary"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook handoff-snapshot",
            "statusMessage": "DevRites: snapshotting active handoff"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook agent-dispatch --harness=codex",
            "statusMessage": "DevRites: checking agent completion receipt"
          }
        ]
      },
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
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook agent-dispatch --harness=codex"
          },
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook cursor --harness=codex"
          }
        ]
      }
    ],
    "SubagentStart": [
      {
        "matcher": "devrites-.*|explorer|worker",
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
        "matcher": "devrites-.*|explorer|worker",
        "hooks": [
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook agent-dispatch --harness=codex",
            "statusMessage": "DevRites: recording subagent result"
          },
          {
            "type": "command",
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook event subagent-stop"
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
