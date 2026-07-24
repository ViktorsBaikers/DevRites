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
      print "- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation."
      print "- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch."
      print "- For every DevRites specialist or writer dispatch, first call `spawn_agent` with the named `devrites-<role>` custom role. The matching project contract is `.codex/agents/devrites-<role>.toml`."
      print "- If `spawn_agent` is callable but a named read-only role is unavailable, use generic `explorer` only when the host proves that run has a runtime-enforced read-only sandbox. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. A missing read-only custom role is not evidence that spawning is unavailable."
      print "- Never dispatch generic `worker` for `devrites-slice-wright` unless the host proves that worker run carries exact DevRites identity and the same `.wright-allowlist` enforcement as the named role. Codex reports a generic run as `agent_type=worker`, so the generated global hooks cannot prove that binding. Reject that unsafe rung and use the documented labelled inline wright path with `.reconcile-inline` plus the full reconcile gate."
      print "- If the host cannot prove the generic explorer is runtime read-only, reject that rung too. Only when no spawn primitive exists or a higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, and apply every fallback risk gate. An unbound generic wright or unconfined generic explorer is such a safety rejection, not evidence that no agents exist."
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
        print "Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Invocation runs a skill inline; dispatch starts a fresh agent with `spawn_agent` and must be awaited. Try the named `devrites-<role>` first. A missing read-only role may fall back to generic `explorer` reading `.codex/agents/devrites-<role>.toml` only when the host proves runtime-enforced read-only sandboxing. Never use generic `worker` for the wright unless exact DevRites identity and `.wright-allowlist` enforcement are proven; Codex `agent_type=worker` does not prove them, so use the labelled inline `.reconcile-inline` path with full reconciliation. Other inline fallback requires spawning itself to be unavailable or a safe spawn to be rejected by higher-priority policy. Trust `.codex/hooks.json` and the inline hooks in `.codex/agents/devrites-*.toml` with `/hooks` before relying on hooks."
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
- Custom Codex subagents generated from the DevRites review agents live in `.codex/agents`.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `spawn_agent`, wait for it, and reconcile its result.
- For a DevRites dispatch, first use the named `devrites-<role>` custom role. If a read-only role is unavailable but `spawn_agent` still works, use generic `explorer` only when the host proves runtime-enforced read-only sandboxing; tell it to read `.codex/agents/devrites-<role>.toml` before executing the unchanged packet.
- Do not use generic `worker` for the wright unless exact DevRites identity and `.wright-allowlist` enforcement are proven. Codex exposes that fallback as `agent_type=worker`, which the generated leaf hooks intentionally do not treat as a declared DevRites run. Reject the unsafe worker rung and use the documented labelled inline `.reconcile-inline` path with full reconciliation.
- Other inline work is allowed only when no spawn primitive exists or higher-priority policy rejects a safe spawn; label it `independence: fallback`, never independent. A missing read-only custom role alone is not such a rejection, but an unconfined generic explorer is.
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
            "command": "command -v devrites-engine >/dev/null 2>&1 || exit 0; cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" 2>/dev/null || exit 0; exec devrites-engine hook a1-guard --harness=codex",
            "statusMessage": "DevRites: checking build write boundary"
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
# Codex has no SessionEnd hook event. Its lifecycle events are
# SessionStart/SubagentStart/PreToolUse/PermissionRequest/PostToolUse/PreCompact/
# PostCompact/UserPromptSubmit/SubagentStop/Stop. The `hook event session-end`
# trace remains Claude-only until Codex adds that event.
}
