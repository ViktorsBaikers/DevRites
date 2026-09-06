#!/usr/bin/env bash
# Compaction reload contract: one owner, same artifacts in hygiene / checkpoint /
# autocomplete.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

HYGIENE="$ROOT/pack/.claude/skills/devrites-lib/reference/standards/context-hygiene.md"
CHECKPOINT="$ROOT/pack/.claude/skills/rite-build/reference/checkpoint.md"
AUTOCOMPLETE="$ROOT/pack/.claude/skills/rite-autocomplete/SKILL.md"

echo "== compaction-reload-contract-test =="

need() {
  file="$1"
  token="$2"
  grep -q "$token" "$file" && ok "$(basename "$file") contains $token" || no "$(basename "$file") missing $token"
}

for token in '.devrites/ACTIVE' 'state.md' 'questions.md' 'decisions.md' 'test-plan.md' 'evidence.md'; do
  need "$HYGIENE" "$token"
  need "$CHECKPOINT" "$token"
  need "$AUTOCOMPLETE" "$token"
done

grep -q 'context-hygiene.md' "$CHECKPOINT" && ok "checkpoint cites context-hygiene.md" || no "checkpoint must cite context-hygiene.md"

echo ""
[ "$fail" -eq 0 ] && echo "compaction-reload-contract-test: PASS" || echo "compaction-reload-contract-test: FAIL"
exit "$fail"
