#!/usr/bin/env bash
# scripts/check-rule-uniqueness.sh — assert each canonical principle heading
# appears in exactly one canonical file under pack/.claude/.
#
# Drift catcher: if someone re-duplicates a principle's full treatment, this
# script fails and names the offenders. Cross-link summaries are fine — they
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

# Anti-rationalization table: core.md keeps a minimal 5-row subset that MUST be
# byte-identical to the matching rows in anti-patterns.md (the single source of
# the full table). The unique-heading check above can't see this — the rows live
# under the same heading in both files — so assert the 5 shared excuse strings
# (the excuse-column text, stable across both files) appear verbatim in both.
# These cover the five rows core.md duplicates:
#   tests-later / small-refactor / special-case / user-will-tell / lint-and-build.
CORE="$PACK/skills/devrites-lib/reference/standards/core.md"
ANTI="$PACK/skills/devrites-lib/reference/standards/anti-patterns.md"
SHARED_EXCUSES=(
  '"I'\''ll add the tests later."'
  '"It'\''s only a small refactor while I'\''m in here."'
  '"This is a special case, the pattern doesn'\''t apply."'
  '"The user will tell me if something is wrong."'
  '"Lint and build pass — that proves quality."'
)
for excuse in "${SHARED_EXCUSES[@]}"; do
  in_core=0; in_anti=0
  grep -qF -- "$excuse" "$CORE" 2>/dev/null && in_core=1
  grep -qF -- "$excuse" "$ANTI" 2>/dev/null && in_anti=1
  if [ "$in_core" -eq 1 ] && [ "$in_anti" -eq 1 ]; then
    printf 'ok: shared excuse present in core.md + anti-patterns.md: %s\n' "$excuse"
  else
    printf 'FAIL: shared excuse must appear verbatim in BOTH core.md and anti-patterns.md: %s (core=%d anti=%d)\n' \
      "$excuse" "$in_core" "$in_anti" >&2
    fail=1
  fi
done

exit "$fail"
