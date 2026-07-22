#!/usr/bin/env bash
# scripts/check-rule-uniqueness.sh: assert each canonical principle heading
# appears in exactly one canonical file under pack/.claude/.
#
# Drift catcher: if someone re-duplicates a principle's full treatment, this
# script fails and names the offenders. Cross-link summaries are fine: they
# don't use the canonical heading.
#
# Usage:
#   scripts/check-rule-uniqueness.sh
# Exits non-zero if any principle is duplicated or missing from its canonical
# home. Add new entries to PRINCIPLES below as principles get a canonical
# home.

set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
PACK="$ROOT/pack/.claude"

# Format: <grep-pattern>|<canonical-relative-path>
PRINCIPLES=(
  '^## Reuse before you write|pack/.claude/skills/devrites-lib/reference/standards/coding-style.md'
  '^## Trust boundary (three tiers)|pack/.claude/skills/devrites-lib/reference/standards/security.md'
  '^## Universal rationalizations|pack/.claude/skills/devrites-lib/reference/standards/anti-patterns.md'
)

fail=0

for entry in "${PRINCIPLES[@]}"; do
  pattern="${entry%%|*}"
  canonical="${entry##*|}"
  # Use literal-string grep (-F) to avoid regex meta-char surprises in headings.
  # Match anchored to line start by adding the '## ' prefix ourselves.
  needle="${pattern#^}"
  matches="$(grep -rln -F -- "$needle" "$PACK"/skills "$PACK"/agents 2>/dev/null | sort -u)"
  count="$(printf '%s\n' "$matches" | grep -c .)"
  if [ "$count" -eq 1 ] && [ "$matches" = "$ROOT/$canonical" ]; then
    printf 'ok: unique heading "%s" (canonical: %s)\n' "$needle" "$canonical"
  else
    printf 'FAIL: heading "%s" should appear only in %s; found in:\n' "$needle" "$canonical" >&2
    printf '%s\n' "$matches" | sed 's|^|    |' >&2
    fail=1
  fi
done

exit "$fail"
