#!/usr/bin/env bash
# Focused checks for the Claude-to-Codex generator used by host packaging.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

TMP_GEN_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_GEN_DIR"' EXIT

# shellcheck source=../scripts/codex-generate.sh
. "$ROOT/scripts/codex-generate.sh"

echo "== codex-generator-test =="

sample="$TMP_GEN_DIR/sample.md"
rewritten="$TMP_GEN_DIR/sample.codex.md"
cat > "$sample" <<'EOF'
See pack/.claude/skills/rite-build/SKILL.md and .claude/skills/devrites-lib/reference/standards/core.md.
Dispatch ../../agents/devrites-code-reviewer.md or ../../../agents/devrites-slice-wright.md.
Audit `.claude/agents/devrites-{security-auditor,performance-reviewer,simplifier-reviewer}.md`.
Use one `Task` call, then wait for the result.
Run /rite-build, then /rite-seal.
EOF

gen_codex_markdown_file "$sample" "$rewritten"
grep -q '.agents/skills/rite-build/SKILL.md' "$rewritten" && ok "rewrites pack skill paths" || no "did not rewrite pack skill paths"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$rewritten" && ok "rewrites installed skill paths" || no "did not rewrite installed skill paths"
grep -q '.codex/agents/devrites-code-reviewer.toml' "$rewritten" && ok "rewrites relative agent links" || no "did not rewrite relative agent links"
if grep -q '.codex/agents/devrites-security-auditor.toml' "$rewritten" \
  && grep -q '.codex/agents/devrites-performance-reviewer.toml' "$rewritten" \
  && grep -q '.codex/agents/devrites-simplifier-reviewer.toml' "$rewritten"; then
  ok "expands brace-compressed agent paths"
else
  no "did not expand brace-compressed agent paths"
fi
grep -Fq '.codex/agents/devrites-{' "$rewritten" && no "kept brace-compressed Codex agent path" || ok "Codex agent paths are explicit"
grep -Eq '(^|[^[:alnum:]_])Task([^[:alnum:]_]|$)' "$rewritten" && no "kept legacy Task orchestration wording" || ok "legacy Task wording removed"
grep -q '`spawn_agent` call' "$rewritten" && ok "rewrites legacy dispatch primitive" || no "did not rewrite legacy dispatch primitive"
grep -q '\$rite-build' "$rewritten" && grep -q '\$rite-seal' "$rewritten" && ok "rewrites slash invocations" || no "did not rewrite slash invocations"
grep -q 'pack/\.claude\|\.claude/skills\|\.claude/agents' "$rewritten" && no "rewritten markdown kept Claude runtime paths" || ok "rewritten markdown has no Claude runtime paths"

skill_dir="$TMP_GEN_DIR/devrites-sample"
mkdir -p "$skill_dir"
cat > "$skill_dir/SKILL.md" <<'EOF'
---
name: devrites-sample
description: Expensive internal description that should be stubbed.
user-invocable: false
---

Read .claude/skills/devrites-lib/reference/standards/core.md and ask /rite-review.
Spawn ../../agents/devrites-code-reviewer.md.
If the named role is unavailable, preserve fresh-context execution.
EOF

skill_out="$TMP_GEN_DIR/devrites-sample.codex.md"
gen_codex_skill_file "$skill_dir/SKILL.md" "$skill_out" 1
grep -q 'description: Internal DevRites skill; DevRites agents invoke it explicitly' "$skill_out" && ok "internal skill description stubbed" || no "internal skill description not stubbed"
grep -q '## Codex compatibility' "$skill_out" && ok "skill compatibility block injected" || no "skill compatibility block missing"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$skill_out" && ok "skill body uses mirrored rules path" || no "skill body did not use mirrored rules path"
grep -q '.codex/agents/devrites-code-reviewer.toml' "$skill_out" && ok "skill body rewrites agent reference" || no "skill body did not rewrite agent reference"
grep -q '\$rite-review' "$skill_out" && ok "skill body rewrites slash invocation" || no "skill body did not rewrite slash invocation"
grep -q 'pack/\.claude\|\.claude/skills\|\.claude/agents' "$skill_out" && no "generated skill kept Claude runtime paths" || ok "generated skill has no Claude runtime paths"
grep -q 'invoke means run a skill in this context' "$skill_out" && ok "skill distinguishes invocation from dispatch" || no "skill does not distinguish invocation from dispatch"
if grep -q 'named read-only role is unavailable, use generic `explorer` only when the host proves' "$skill_out" \
  && grep -q 'runtime-enforced read-only sandbox' "$skill_out" \
  && grep -q 'read `.codex/agents/devrites-<role>.toml`' "$skill_out"; then
  ok "skill gates generic explorer on runtime-enforced read-only"
else
  no "skill permits an unconfined generic explorer"
fi
if grep -q 'Never dispatch generic `worker` for `devrites-slice-wright` unless' "$skill_out" \
  && grep -q 'agent_type=worker' "$skill_out" \
  && grep -q '\.wright-allowlist' "$skill_out" \
  && grep -q '\.reconcile-inline' "$skill_out"; then
  ok "skill rejects an unbound generic wright and names the safe fallback"
else
  no "skill promises wright enforcement for an unbound generic worker"
fi
grep -q 'A missing read-only custom role is not evidence that spawning is unavailable' "$skill_out" \
  && ok "read-only custom-role absence does not permit inline fallback" \
  || no "read-only custom-role absence can still permit inline fallback"
grep -q 'Only when no spawn primitive exists or a higher-priority policy rejects a safe spawn' "$skill_out" \
  && grep -q 'independence: fallback' "$skill_out" \
  && ok "inline fallback is labelled and capability-gated" \
  || no "inline fallback is not capability-gated"

bridge="$TMP_GEN_DIR/AGENTS.md"
gen_codex_agents_bridge "$bridge"
grep -q 'invoke.*run a skill inline.*dispatch.*fresh agent' "$bridge" \
  && ok "AGENTS bridge defines invoke and dispatch" \
  || no "AGENTS bridge does not define invoke and dispatch"
if grep -q 'read-only role is unavailable but `spawn_agent` still works, use generic `explorer` only when the host proves runtime-enforced read-only' "$bridge" \
  && grep -q 'agent_type=worker.*generated leaf hooks intentionally do not treat as a declared DevRites run' "$bridge" \
  && grep -q 'labelled inline `.reconcile-inline` path' "$bridge"; then
  ok "AGENTS bridge preserves the safe spawn capability ladder"
else
  no "AGENTS bridge promises an unenforceable generic-wright boundary"
fi

agent="$TMP_GEN_DIR/devrites-sample-reviewer.md"
agent_out="$TMP_GEN_DIR/devrites-sample-reviewer.toml"
cat > "$agent" <<'EOF'
---
name: devrites-sample-reviewer
description: Review .claude/skills/devrites-lib/reference/standards/core.md before /rite-seal.
tools: Read, Grep
---

Read .claude/skills/devrites-lib/reference/standards/core.md.
Do not run /rite-build.
EOF

gen_codex_agent "$agent" "$agent_out"
grep -q 'name = "devrites-sample-reviewer"' "$agent_out" && ok "agent name preserved" || no "agent name missing"
grep -q 'sandbox_mode = "read-only"' "$agent_out" && ok "read-only agent sandbox emitted" || no "read-only agent sandbox missing"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$agent_out" && ok "agent paths rewritten" || no "agent paths not rewritten"
grep -q '\$rite-seal' "$agent_out" && grep -q '\$rite-build' "$agent_out" && ok "agent invocations rewritten" || no "agent invocations not rewritten"
grep -q 'pack/\.claude\|\.claude/skills\|\.claude/agents' "$agent_out" && no "generated agent kept Claude runtime paths" || ok "generated agent has no Claude runtime paths"

wright="$TMP_GEN_DIR/devrites-slice-wright.md"
wright_out="$TMP_GEN_DIR/devrites-slice-wright.toml"
cat > "$wright" <<'EOF'
---
name: devrites-slice-wright
description: Implement one allowlisted slice.
tools: Read, Grep, Edit, Write
---

Write only the allowlisted slice.
EOF
gen_codex_agent "$wright" "$wright_out"

if command -v python3 >/dev/null 2>&1; then
  python3 - "$agent_out" "$wright_out" <<'PY'
import pathlib, sys, tomllib

required = {
    "Bash",
    "Edit",
    "Write",
    "apply_patch",
    "exec",
    "Task",
    "spawn_agent",
    "delegate",
    "dispatch_agent",
    "create_agent",
}
for path, subcommand, required_var in (
    (
        pathlib.Path(sys.argv[1]),
        "reviewer-readonly",
        "DEVRITES_REVIEWER_AGENT_REQUIRED=1",
    ),
    (
        pathlib.Path(sys.argv[2]),
        "wright-scope",
        "DEVRITES_WRIGHT_AGENT_REQUIRED=1",
    ),
):
    data = tomllib.loads(path.read_text())
    groups = data["hooks"]["PreToolUse"]
    assert len(groups) == 1
    assert required <= set(groups[0]["matcher"].split("|"))
    handlers = groups[0]["hooks"]
    assert len(handlers) == 1
    command = handlers[0]["command"]
    assert f"{required_var} devrites-engine hook {subcommand} --harness=codex" in command
    assert "DEVRITES_AGENT_RUN=1" in command
    assert f"DEVRITES_ACTIVE_AGENT={data['name']}" in command
    assert "command -v devrites-engine" not in command
    assert "node " not in command
    assert 'case "$rc" in' in command
    assert "exit 2" in command
PY
  rc="$?"
  if [ "$rc" -eq 0 ]; then
    ok "generated agent TOML parses with scoped fail-closed leaf hooks"
  else
    if python3 - <<'PY' >/dev/null 2>&1
try:
    import tomllib  # noqa: F401
except ModuleNotFoundError:
    raise SystemExit(1)
PY
    then
      no "generated agent TOML does not parse"
    else
      ok "generated agent TOML parse skipped (python has no tomllib)"
    fi
  fi
else
  ok "generated agent TOML parse skipped (python3 not found)"
fi

reviewer_command="$(sed -n "s/^command = '''\\(.*\\)'''$/\\1/p" "$agent_out")"
wright_command="$(sed -n "s/^command = '''\\(.*\\)'''$/\\1/p" "$wright_out")"
if [ -n "$reviewer_command" ] && [ -n "$wright_command" ]; then
  ok "generated leaf hook commands are present in agent profiles"
else
  no "generated agent profile is missing its leaf hook"
fi

hooks="$TMP_GEN_DIR/codex-hooks.json"
gen_codex_hooks_json "$hooks"
if command -v node >/dev/null 2>&1; then
  node -e "JSON.parse(require('fs').readFileSync(process.argv[1], 'utf8'))" "$hooks" \
    && ok "generated hooks JSON parses" || no "generated hooks JSON invalid"
else
  ok "generated hooks JSON parse skipped (node not found)"
fi
grep -q 'devrites-engine hook stop-gate --harness=codex' "$hooks" && ok "hooks call engine hook subcommands" || no "hooks do not call engine hook subcommands"
grep -q 'pack/\.claude/hooks\|\.sh' "$hooks" && no "hooks reference shell hook scripts" || ok "hooks do not reference shell hook scripts"
if grep -q 'hook reviewer-readonly\|hook wright-scope' "$hooks"; then
  no "global hooks can shadow root or generic agents with leaf policy"
else
  ok "declared leaf guards are scoped to agent profiles, not root hooks"
fi

if [ -n "$reviewer_command" ] && [ -n "$wright_command" ]; then
  hook_bin="$TMP_GEN_DIR/hook-bin"
  mkdir -p "$hook_bin"
  hook_path="$hook_bin"

  reviewer_payload='{"agent_type":"devrites-sample-reviewer","tool_name":"Edit","tool_input":{}}'
  reviewer_missing_err="$TMP_GEN_DIR/reviewer-missing.err"
  printf '%s\n' "$reviewer_payload" | PATH="$hook_path" /bin/sh -c "$reviewer_command" >/dev/null 2>"$reviewer_missing_err"
  reviewer_missing_rc="$?"
  if [ "$reviewer_missing_rc" -eq 2 ] \
    && grep -Fq 'devrites-codex-leaf-guard' "$reviewer_missing_err"; then
    ok "declared reviewer fails closed when engine is missing"
  else
    no "declared reviewer fails open when engine is missing"
  fi

  wright_payload='{"agent_type":"devrites-slice-wright","tool_name":"Write","tool_input":{}}'
  wright_missing_err="$TMP_GEN_DIR/wright-missing.err"
  printf '%s\n' "$wright_payload" | PATH="$hook_path" /bin/sh -c "$wright_command" >/dev/null 2>"$wright_missing_err"
  wright_missing_rc="$?"
  if [ "$wright_missing_rc" -eq 2 ] \
    && grep -Fq 'devrites-codex-leaf-guard' "$wright_missing_err"; then
    ok "declared wright fails closed when engine is missing"
  else
    no "declared wright fails open when engine is missing"
  fi

  cat > "$hook_bin/devrites-engine" <<'EOF'
#!/bin/sh
/bin/cat > "${CAPTURE_PATH:-/dev/null}"
printf '%s\n' "${DEVRITES_AGENT_RUN:-}:${DEVRITES_ACTIVE_AGENT:-}" > "${IDENTITY_PATH:-/dev/null}"
exit "${FAKE_ENGINE_EXIT:-0}"
EOF
  chmod +x "$hook_bin/devrites-engine"

  exact_payload="$TMP_GEN_DIR/exact-payload.json"
  captured_payload="$TMP_GEN_DIR/captured-payload.json"
  captured_identity="$TMP_GEN_DIR/captured-identity.txt"
  exact_out="$TMP_GEN_DIR/exact.out"
  printf '%s\n' "$reviewer_payload" > "$exact_payload"
  PATH="$hook_path" CAPTURE_PATH="$captured_payload" IDENTITY_PATH="$captured_identity" FAKE_ENGINE_EXIT=0 \
    /bin/sh -c "$reviewer_command" <"$exact_payload" >"$exact_out" 2>/dev/null
  exact_rc="$?"
  if [ "$exact_rc" -eq 0 ] \
    && cmp -s "$exact_payload" "$captured_payload" \
    && grep -Fxq '1:devrites-sample-reviewer' "$captured_identity" \
    && [ ! -s "$exact_out" ]; then
    ok "leaf wrapper forwards the payload and supplies scoped identity"
  else
    no "leaf wrapper loses the payload or scoped identity"
  fi

  crashed_out="$TMP_GEN_DIR/crashed.out"
  crashed_err="$TMP_GEN_DIR/crashed.err"
  PATH="$hook_path" CAPTURE_PATH="$TMP_GEN_DIR/crashed-payload.json" FAKE_ENGINE_EXIT=7 \
    /bin/sh -c "$reviewer_command" <"$exact_payload" >"$crashed_out" 2>"$crashed_err"
  crashed_rc="$?"
  if [ "$crashed_rc" -eq 2 ] \
    && grep -Fq 'devrites-codex-leaf-guard' "$crashed_err"; then
    ok "declared leaf fails closed when engine crashes"
  else
    no "declared leaf fails open when engine crashes"
  fi
else
  no "leaf hook behavior checks unavailable because commands were not generated"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-generator-test: PASS" || echo "codex-generator-test: FAIL"
exit "$fail"
