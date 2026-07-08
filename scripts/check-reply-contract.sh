#!/usr/bin/env bash
# check-reply-contract.sh — keep user-facing rite-* completion replies normalized.

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SKILLS="$ROOT/pack/.claude/skills"
CONTRACT="devrites-lib/reference/reply-contract.md"
fail=0

bad() {
  printf 'FAIL: %s\n' "$*"
  fail=1
}

for f in "$SKILLS"/rite*/SKILL.md; do
  [ -f "$f" ] || continue
  rel="${f#"$ROOT"/}"

  if grep -qE '^## Output' "$f"; then
    if ! grep -q "$CONTRACT" "$f" && ! grep -q 'Reply-contract exception:' "$f"; then
      bad "$rel has ## Output but does not reference $CONTRACT or declare Reply-contract exception"
    fi
  fi

  case "$rel" in
    pack/.claude/skills/rite/SKILL.md)
      continue
      ;;
  esac

  # Obvious menu-style next actions make lifecycle replies hard to scan. The
  # contract allows one command only; alternates belong in Open/Alternative.
  if grep -nE '^Next:[^`]*(/rite-[^[:space:]]+)[^`]*([[:space:]]or[[:space:]]|[[:space:]]/[[:alnum:]_-]+| · |→)' "$f" >/tmp/dr_reply_next 2>/dev/null; then
    bad "$rel has ambiguous Next: wording:"
    sed "s|$ROOT/||" /tmp/dr_reply_next
  fi
done

if [ "$fail" -eq 0 ]; then
  printf 'reply-contract: PASS\n'
else
  printf 'reply-contract: FAIL\n'
fi

exit "$fail"
