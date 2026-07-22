#!/usr/bin/env bash
# codex-generator-test.sh: focused tests for the shared Claude-to-Codex
# generation helper used by host-artifact packaging.
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
Run /rite-build, then /rite-seal.
EOF

gen_codex_markdown_file "$sample" "$rewritten"
grep -q '.agents/skills/rite-build/SKILL.md' "$rewritten" && ok "rewrites pack skill paths" || no "did not rewrite pack skill paths"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$rewritten" && ok "rewrites installed skill paths" || no "did not rewrite installed skill paths"
grep -q '.codex/agents/devrites-code-reviewer.toml' "$rewritten" && ok "rewrites relative agent links" || no "did not rewrite relative agent links"
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
EOF

skill_out="$TMP_GEN_DIR/devrites-sample.codex.md"
gen_codex_skill_file "$skill_dir/SKILL.md" "$skill_out" 1
grep -q 'description: Internal DevRites skill; DevRites agents invoke it explicitly' "$skill_out" && ok "internal skill description stubbed" || no "internal skill description not stubbed"
grep -q '## Codex compatibility' "$skill_out" && ok "skill compatibility block injected" || no "skill compatibility block missing"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$skill_out" && ok "skill body uses mirrored rules path" || no "skill body did not use mirrored rules path"
grep -q '.codex/agents/devrites-code-reviewer.toml' "$skill_out" && ok "skill body rewrites agent reference" || no "skill body did not rewrite agent reference"
grep -q '\$rite-review' "$skill_out" && ok "skill body rewrites slash invocation" || no "skill body did not rewrite slash invocation"
grep -q 'pack/\.claude\|\.claude/skills\|\.claude/agents' "$skill_out" && no "generated skill kept Claude runtime paths" || ok "generated skill has no Claude runtime paths"

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

if command -v python3 >/dev/null 2>&1; then
  python3 - "$agent_out" <<'PY'
import pathlib, sys, tomllib
tomllib.loads(pathlib.Path(sys.argv[1]).read_text())
PY
  rc="$?"
  if [ "$rc" -eq 0 ]; then
    ok "generated agent TOML parses"
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

echo ""
[ "$fail" -eq 0 ] && echo "codex-generator-test: PASS" || echo "codex-generator-test: FAIL"
exit "$fail"
