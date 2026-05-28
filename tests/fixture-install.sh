#!/usr/bin/env bash
# fixture-install.sh — DevRites installs identically regardless of project stack.
# Rules are common / stack-agnostic, so there is one fixture, not one per language:
# it proves a clean install and that the installed rule set does not vary by stack.
set -u
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
[ -f "$EMPTY/.claude/rules/README.md" ] && ok "rules installed" || no "rules missing"
[ -f "$EMPTY/.claude/rules/security.md" ] && ok "security rule present" || no "security rule missing"
[ -d "$EMPTY/.claude/agents" ] && ok "agents installed" || no "agents missing"

# 2) different stacks: rails-shaped and react-shaped projects
printf "source 'https://rubygems.org'\ngem 'rails'\n" > "$RAILS/Gemfile"; mkdir -p "$RAILS/app/views"
cat > "$REACT/package.json" <<'JSON'
{ "name": "demo", "dependencies": { "react": "^18" }, "devDependencies": { "typescript": "^5" } }
JSON
printf '{}\n' > "$REACT/tsconfig.json"
bash "$ROOT/install.sh" --target "$RAILS" >/dev/null 2>&1 || no "rails install failed"
bash "$ROOT/install.sh" --target "$REACT" >/dev/null 2>&1 || no "react install failed"

# 3) the installed rule set is identical across all three (stack-agnostic)
er="$(cd "$EMPTY/.claude/rules" && ls -1 | sort | tr '\n' ' ')"
rr="$(cd "$RAILS/.claude/rules" && ls -1 | sort | tr '\n' ' ')"
kr="$(cd "$REACT/.claude/rules" && ls -1 | sort | tr '\n' ' ')"
if [ "$er" = "$rr" ] && [ "$er" = "$kr" ]; then
  ok "rule set identical across empty/rails/react (stack-agnostic)"
else
  no "rule set varies by stack: empty=[$er] rails=[$rr] react=[$kr]"
fi

# 4) no language-specific rule subdirectories exist anywhere
for d in "$EMPTY" "$RAILS" "$REACT"; do
  for sub in ruby python typescript web golang rust ecc; do
    [ -e "$d/.claude/rules/$sub" ] && no "unexpected stack-specific rules dir: $sub" || true
  done
done
ok "no language-specific rule packs anywhere"

echo ""
[ "$fail" -eq 0 ] && echo "fixture-install: PASS" || echo "fixture-install: FAIL"
exit "$fail"
