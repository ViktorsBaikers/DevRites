#!/usr/bin/env bash
# load-state-questions.sh — assert the shared preamble tallies open questions correctly
# when question blocks are adjacent (back-to-back `## q-` headers with no blank
# section between them). The earlier awk dropped every open question except the
# last in a run of adjacent blocks; this test pins the regression.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
LOADER="$ROOT/pack/.claude/skills/devrites-lib/scripts/preamble.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

echo "== load-state-questions (adjacent-block tally) =="

# Build a workspace fixture with three ADJACENT open `gate: blocking` questions,
# plus one answered and one open advisory, to prove finalize-before-reset works.
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
slug="adjacent-blocks"
work="$TMP/.devrites/work/$slug"
mkdir -p "$work"
printf '%s\n' "$slug" > "$TMP/.devrites/ACTIVE"

cat > "$work/questions.md" <<'EOF'
## q-2026-06-17-001
status: open
slice: 01-a
gate: blocking
question: first blocking question
proposed: none
raised_at: 2026-06-17T00:00:00Z
## q-2026-06-17-002
status: open
slice: 02-b
gate: blocking
question: second blocking question
proposed: none
raised_at: 2026-06-17T00:01:00Z
## q-2026-06-17-003
status: open
slice: 03-c
gate: blocking
question: third blocking question
proposed: none
raised_at: 2026-06-17T00:02:00Z
## q-2026-06-17-004
status: answered
slice: 04-d
gate: validating
question: already answered, must not count
proposed: none
raised_at: 2026-06-17T00:03:00Z
answered_at: 2026-06-17T01:00:00Z
answer: done
## q-2026-06-17-005
status: open
slice: 05-e
gate: advisory
question: trailing open advisory at END
proposed: none
raised_at: 2026-06-17T00:04:00Z
EOF

# preamble.sh resolves paths relative to CWD (.devrites/ACTIVE), so run inside TMP.
out="$(cd "$TMP" && bash "$LOADER")"

line="$(printf '%s\n' "$out" | grep 'open:')"
if printf '%s' "$line" | grep -q '3 blocking'; then
  ok "three adjacent open blocking questions counted as 3"
else
  no "expected '3 blocking', got: $line"
fi

# The answered validating entry must NOT count; the trailing advisory MUST.
if printf '%s' "$line" | grep -q '0 validating'; then
  ok "answered question excluded from tally"
else
  no "expected '0 validating', got: $line"
fi
if printf '%s' "$line" | grep -q '1 advisory'; then
  ok "trailing open question counted at END"
else
  no "expected '1 advisory', got: $line"
fi

echo ""
[ "$fail" -eq 0 ] && echo "load-state-questions: PASS" || echo "load-state-questions: FAIL"
exit "$fail"
