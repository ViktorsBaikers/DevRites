#!/usr/bin/env bash
# fixture-install.sh — DevRites installs identically regardless of project stack.
# Rules are common / stack-agnostic, so there is one fixture, not one per language:
# it proves a clean install and that the installed rule set does not vary by stack.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

EMPTY="$(mktemp -d)"; RAILS="$(mktemp -d)"; REACT="$(mktemp -d)"
trap 'rm -rf "$EMPTY" "$RAILS" "$REACT"' EXIT

echo "== fixture-install (stack-agnostic) =="

# 1) empty project: a clean install has skills, rules, agents
bash "$ROOT/install.sh" --target "$EMPTY" >/dev/null 2>&1 || no "empty install failed"
[ -f "$EMPTY/.claude/skills/rite-define/SKILL.md" ] && ok "skills installed" || no "skills missing"
[ -f "$EMPTY/.agents/skills/rite-define/SKILL.md" ] && ok "Codex skills installed" || no "Codex skills missing"
[ -f "$EMPTY/.claude/skills/devrites-lib/reference/standards/README.md" ] && ok "rules installed" || no "rules missing"
[ -f "$EMPTY/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "security rule present" || no "security rule missing"
[ -f "$EMPTY/.agents/skills/devrites-lib/reference/standards/security.md" ] && ok "Codex rules mirror installed" || no "Codex rules mirror missing"
[ -d "$EMPTY/.claude/agents" ] && ok "agents installed" || no "agents missing"
[ -d "$EMPTY/.codex/agents" ] && ok "Codex agents installed" || no "Codex agents missing"
[ -f "$EMPTY/.codex/config.toml" ] && ok "Codex config installed" || no "Codex config missing"
[ -f "$EMPTY/.codex/hooks.json" ] && ok "Codex hooks installed" || no "Codex hooks missing"
[ -f "$EMPTY/.codex/mcp/devrites-mcp.mjs" ] && ok "Codex MCP server installed" || no "Codex MCP server missing"
[ -f "$EMPTY/AGENTS.md" ] && ok "Codex AGENTS bridge installed" || no "Codex AGENTS bridge missing"

# 2) different stacks: rails-shaped and react-shaped projects
printf "source 'https://rubygems.org'\ngem 'rails'\n" > "$RAILS/Gemfile"; mkdir -p "$RAILS/app/views"
cat > "$REACT/package.json" <<'JSON'
{ "name": "demo", "dependencies": { "react": "^18" }, "devDependencies": { "typescript": "^5" } }
JSON
printf '{}\n' > "$REACT/tsconfig.json"
bash "$ROOT/install.sh" --target "$RAILS" >/dev/null 2>&1 || no "rails install failed"
bash "$ROOT/install.sh" --target "$REACT" >/dev/null 2>&1 || no "react install failed"

# 3) the installed rule set is identical across all three (stack-agnostic)
er="$(cd "$EMPTY/.claude/skills/devrites-lib/reference/standards" && ls -1 | sort | tr '\n' ' ')"
rr="$(cd "$RAILS/.claude/skills/devrites-lib/reference/standards" && ls -1 | sort | tr '\n' ' ')"
kr="$(cd "$REACT/.claude/skills/devrites-lib/reference/standards" && ls -1 | sort | tr '\n' ' ')"
if [ "$er" = "$rr" ] && [ "$er" = "$kr" ]; then
  ok "rule set identical across empty/rails/react (stack-agnostic)"
else
  no "rule set varies by stack: empty=[$er] rails=[$rr] react=[$kr]"
fi

# 4) no language-specific rule subdirectories exist anywhere
for d in "$EMPTY" "$RAILS" "$REACT"; do
  for sub in ruby python typescript web golang rust ecc; do
    [ -e "$d/.claude/skills/devrites-lib/reference/standards/$sub" ] && no "unexpected stack-specific rules dir: $sub" || true
  done
done
ok "no language-specific rule packs anywhere"

echo ""
[ "$fail" -eq 0 ] && echo "fixture-install: PASS" || echo "fixture-install: FAIL"
exit "$fail"
