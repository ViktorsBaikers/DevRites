#!/usr/bin/env bash
# Identity-file writes on the adopt path must go through /rite-customize,
# `devrites-engine check skill-trust`, and explicit human approval.
set -uo pipefail

ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
ADOPT="$ROOT/pack/.claude/skills/rite-adopt/SKILL.md"
CUSTOMIZE="$ROOT/pack/.claude/skills/rite-customize/SKILL.md"
SECURITY="$ROOT/pack/.claude/skills/devrites-lib/reference/standards/security.md"
fail=0

flat() { tr '\n' ' ' <"$1" | tr -s ' '; }
has() { # <label> <file> <ERE>
  if flat "$2" | grep -Eqi -- "$3"; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi
}

step5="$(awk '/^5\. For a verified/{p=1;next} /^6\. /{p=0} p' "$ADOPT" | tr '\n' ' ' | tr -s ' ')"
for pair in 'rite-customize:/rite-customize' 'skill-trust:check skill-trust' 'human approval:explicit human approval' \
            'identity files:AGENTS\.md.*CLAUDE\.md'; do
  if grep -Eqi -- "${pair#*:}" <<<"$step5"; then echo "ok   adopt step 5 names ${pair%%:*}"
  else echo "FAIL adopt step 5 names ${pair%%:*}"; fail=1; fi
done
if grep -Eqi -- 'Apply only after review' <<<"$step5"; then echo "FAIL adopt step 5 still self-applies after 'review'"; fail=1
else echo "ok   adopt step 5 does not self-apply"; fi

has "adopt loads manifest has security trigger" "$ADOPT" '"triggers":\{[^}]*"security":\["devrites-lib/reference/standards/security\.md"\]'

# the receiving rite applies skill-trust and the admission to instruction files, not only skill/agent Markdown
has "rite-customize runs skill-trust on AGENTS.md/CLAUDE.md" "$CUSTOMIZE" 'skill, agent, or instruction file.*AGENTS\.md.*CLAUDE\.md.*check skill-trust'
has "rite-customize forbids unadmitted identity-file writes of any origin" "$CUSTOMIZE" 'imported, observed, or agent-authored'

has "security.md names the admission" "$SECURITY" 'rite-customize.*skill-trust.*human approval'

exit "$fail"
