#!/usr/bin/env bash
# codex-generate.sh: shared Claude-to-Codex surface generators.
#
# This file is a shell library, not a standalone installer. Callers provide
# TMP_GEN_DIR and then invoke the gen_* helpers below.

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
      print "- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here: Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional."
      print "- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review**: an inline pass shares the calling context and is weaker evidence."
      print "- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement."
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
        print "Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`, use matching `.codex/agents/devrites-*.toml` custom agents for DevRites subagent dispatch, and trust `.codex/hooks.json` with `/hooks` before relying on hooks."
      }
    }
  ' "$_src" > "$_tmp"
  gen_codex_markdown_file "$_tmp" "$_out"
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

- **Inspect before trust.** Codex skips project-scoped `.codex/` hooks, agents, and rules in an untrusted project. Before enabling them, inspect `.codex/hooks.json`, `.codex/agents/`, and the project guidance; use `/hooks` to review the commands Codex would run. The human operator decides whether to trust the folder. Until then, use DevRites only through explicitly inspected commands and treat hook enforcement as unavailable.
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
- Follow the DevRites lifecycle: frame -> spec -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
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
# NOTE: no SessionEnd entry: Codex does not emit a SessionEnd hook event (its lifecycle events
# are SessionStart/SubagentStart/PreToolUse/PermissionRequest/PostToolUse/PreCompact/PostCompact/
# UserPromptSubmit/SubagentStop/Stop). The `hook event session-end` trace is Claude-only; re-add
# here if Codex ever ships the event.
}
