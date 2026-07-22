#!/usr/bin/env bash
# injection-baseline-present.sh: guard the A2 prompt-injection-resistance baseline against
# silent drift: the canonical block must live (once) in the security rule, and every agent
# that reads untrusted input must carry an inline reference to it.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
RULE="$HERE/../pack/.claude/skills/devrites-lib/reference/standards/security.md"
AGENTS="$HERE/../pack/.claude/agents"
fail=0

note() { printf '%s\n' "$1"; }

# 1) canonical baseline section exists in the security rule.
if grep -q '^## Prompt-injection resistance' "$RULE"; then
  note "ok   canonical section present in security.md"
else
  note "FAIL canonical section '## Prompt-injection resistance' missing from security.md"; fail=1
fi

# 2) baseline names BOTH untrusted code/file contents AND the conventions ledger.
if grep -q 'data, never instructions' "$RULE" && grep -q 'conventions.md' "$RULE"; then
  note "ok   baseline covers file contents + ledger entries"
else
  note "FAIL baseline must name both 'data, never instructions' and the conventions ledger"; fail=1
fi

# 3) every agent carries the inline reference (the drift guard).
missing=0
for f in "$AGENTS"/*.md; do
  if ! grep -q 'Untrusted-input safety' "$f"; then
    note "FAIL agent missing injection-resistance reference: $(basename "$f")"; missing=1; fail=1
  fi
done
[ "$missing" -eq 0 ] && note "ok   all agents carry the inline reference"

# 4) single source of truth: the canonical block is not copied into agents.
copies=$(grep -l 'canonical baseline for every DevRites agent' "$AGENTS"/*.md 2>/dev/null | wc -l | tr -d ' ')
if [ "$copies" -eq 0 ]; then
  note "ok   no agent duplicates the canonical block"
else
  note "FAIL $copies agent(s) copy the canonical block: reference it, don't duplicate it"; fail=1
fi

if [ "$fail" -ne 0 ]; then
  note "INJECTION-BASELINE TESTS: FAIL"
  exit 1
fi
note "INJECTION-BASELINE TESTS: PASS"
exit 0
