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
Use one `Task` call, then wait for the result.
Run /rite-build, then /rite-seal.
EOF

gen_codex_markdown_file "$sample" "$rewritten"
grep -q '.agents/skills/rite-build/SKILL.md' "$rewritten" && ok "rewrites pack skill paths" || no "did not rewrite pack skill paths"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$rewritten" && ok "rewrites installed skill paths" || no "did not rewrite installed skill paths"
grep -q '.codex/agents/devrites-code-reviewer.toml' "$rewritten" && ok "rewrites agent links" || no "did not rewrite agent links"
grep -Eq '(^|[^[:alnum:]_])Task([^[:alnum:]_]|$)' "$rewritten" && no "kept legacy Task wording" || ok "legacy Task wording removed"
grep -q '`spawn_agent` call' "$rewritten" && ok "rewrites the dispatch primitive" || no "did not rewrite the dispatch primitive"
grep -q '\$rite-build' "$rewritten" && grep -q '\$rite-seal' "$rewritten" && ok "rewrites slash invocations" || no "did not rewrite slash invocations"
grep -q 'pack/\.claude\|\.claude/skills\|\.claude/agents' "$rewritten" && no "rewritten markdown kept Claude paths" || ok "rewritten markdown has no Claude paths"

skill_dir="$TMP_GEN_DIR/devrites-sample"
mkdir -p "$skill_dir"
cat > "$skill_dir/SKILL.md" <<'EOF'
---
name: devrites-sample
description: Internal description that Codex can shorten natively when needed.
user-invocable: false
---

Read .claude/skills/devrites-lib/reference/standards/core.md.
Dispatch ../../agents/devrites-code-reviewer.md and wait for its result.
EOF

skill_out="$TMP_GEN_DIR/devrites-sample.codex.md"
gen_codex_skill_file "$skill_dir/SKILL.md" "$skill_out"
grep -q 'description: Internal description that Codex can shorten natively when needed.' "$skill_out" \
  && ok "native skill description preserved" \
  || no "native skill description was rewritten"
grep -q '## Codex compatibility' "$skill_out" \
  && no "skill duplicated project-wide Codex guidance" \
  || ok "skill relies on the AGENTS bridge for project-wide guidance"
if grep -Eq 'MultiAgent V1|MultiAgent V2|required-agent-roles|agent-dispatch|generic `explorer`|generic `worker`' "$skill_out"; then
  no "skill kept versioned receipt compatibility"
else
  ok "skill has no versioned receipt compatibility"
fi

bridge="$TMP_GEN_DIR/AGENTS.md"
gen_codex_agents_bridge "$bridge"
grep -q 'invoke.*run a skill inline.*dispatch.*fresh agent' "$bridge" && ok "AGENTS bridge defines invoke and dispatch" || no "AGENTS bridge does not define invoke and dispatch"
grep -q 'never substitute a generic/default child' "$bridge" \
  && grep -q 'Dispatch every exact named `devrites-<role>`' "$bridge" \
  && ok "AGENTS bridge delegates agent lifecycle to Codex" \
  || no "AGENTS bridge keeps a custom agent lifecycle"
grep -q 'devrites-slice-wright.*default_permissions = ":workspace"' "$bridge" \
  && grep -q 'every other specialist.*default_permissions = ":read-only"' "$bridge" \
  && grep -q 'root must never edit product source or tests itself' "$bridge" \
  && grep -q 'exact path-bounded executable workflow artifacts' "$bridge" \
  && grep -q '`git diff --name-only`' "$bridge" \
  && grep -q 'reject any extra path' "$bridge" \
  && grep -q 'never recreate an engine dispatch bridge' "$bridge" \
  && ok "AGENTS bridge routes Codex source writing through the exact wright" \
  || no "AGENTS bridge has the wrong Codex writer boundary"
grep -Eq 'MultiAgent V1|MultiAgent V2|required-agent-roles|agent-dispatch' "$bridge" && no "AGENTS bridge kept receipt machinery" || ok "AGENTS bridge has no receipt machinery"

reviewer="$TMP_GEN_DIR/devrites-sample-reviewer.md"
reviewer_out="$TMP_GEN_DIR/devrites-sample-reviewer.toml"
cat > "$reviewer" <<'EOF'
---
name: devrites-sample-reviewer
description: Review one bounded change.
tools: Read, Grep
---

Read .claude/skills/devrites-lib/reference/standards/core.md and return findings.
EOF
gen_codex_agent "$reviewer" "$reviewer_out"

wright="$TMP_GEN_DIR/devrites-slice-wright.md"
wright_out="$TMP_GEN_DIR/devrites-slice-wright.toml"
cat > "$wright" <<'EOF'
---
name: devrites-slice-wright
description: Implement one path-bounded slice.
tools: Read, Grep, Edit, Write
---

Write only the task's exact paths.
EOF
gen_codex_agent "$wright" "$wright_out"

if command -v python3 >/dev/null 2>&1; then
  python3 - "$reviewer_out" "$wright_out" <<'PY'
import pathlib, sys, tomllib

reviewer = tomllib.loads(pathlib.Path(sys.argv[1]).read_text())
wright = tomllib.loads(pathlib.Path(sys.argv[2]).read_text())

assert reviewer["name"] == "devrites-sample-reviewer"
assert reviewer["default_permissions"] == ":read-only"
assert "sandbox_mode" not in reviewer
assert "hooks" not in reviewer
assert ".agents/skills/devrites-lib/reference/standards/core.md" in reviewer["developer_instructions"]
assert "Codex custom-agent version" not in reviewer["developer_instructions"]

assert wright["name"] == "devrites-slice-wright"
assert wright["default_permissions"] == ":workspace"
assert "sandbox_mode" not in wright
assert "hooks" not in wright
assert "Codex custom-agent version" not in wright["developer_instructions"]
PY
  [ "$?" -eq 0 ] && ok "generated wright is workspace-capable; reviewer is read-only; both are hook-free" || no "generated agent TOML contract is wrong"
else
  ok "generated agent TOML parse skipped (python3 not found)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-generator-test: PASS" || echo "codex-generator-test: FAIL"
exit "$fail"
