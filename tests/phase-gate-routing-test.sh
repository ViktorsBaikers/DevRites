#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SPEC="$ROOT/pack/.claude/skills/rite-spec/SKILL.md"
DEFINE="$ROOT/pack/.claude/skills/rite-define/SKILL.md"
VET="$ROOT/pack/.claude/skills/rite-vet/SKILL.md"
fail=0

ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

echo "== phase-gate-routing-test =="

if grep -q 'Do not run `devrites-engine analyze`' "$SPEC"; then
  ok "rite-spec defers analyze until tasks exist"
else
  no "rite-spec does not explicitly defer analyze"
fi

write_line="$(grep -n '^6\. \*\*Write\*\*' "$DEFINE" | cut -d: -f1)"
analyze_line="$(grep -n '^[[:space:]]*devrites-engine analyze' "$DEFINE" | head -1 | cut -d: -f1)"
readiness_line="$(grep -n '^7\. \*\*Readiness gate\*\*' "$DEFINE" | cut -d: -f1)"
if [ -n "$write_line" ] && [ -n "$analyze_line" ] && [ -n "$readiness_line" ] \
   && [ "$analyze_line" -gt "$write_line" ] && [ "$analyze_line" -lt "$readiness_line" ]; then
  ok "rite-define runs analyze after writing tasks and before readiness"
else
  no "rite-define does not run analyze after tasks are written"
fi

vet_analyze_count="$(grep -c '^[[:space:]]*devrites-engine analyze' "$VET" || true)"
if [ "$vet_analyze_count" -ge 2 ]; then
  ok "rite-vet re-runs analyze after plan hardening"
else
  no "rite-vet can mutate tasks after its only analyze pass"
fi

echo ""
[ "$fail" -eq 0 ] && echo "phase-gate-routing-test: PASS" || echo "phase-gate-routing-test: FAIL"
exit "$fail"
