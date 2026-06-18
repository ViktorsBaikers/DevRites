#!/usr/bin/env bash
# Executable acceptance criteria — compile spec.md's acceptance criteria into
# machine-checkable items and verify each is proven (checked) in seal.md. Turns
# "walk acceptance one by one" (rite-seal/reference/final-evidence.md) into a
# deterministic check instead of prose the model walks by hand.
#
# Convention (optional but recommended): tag each criterion with a stable id —
#   spec.md  "## Acceptance criteria"   →  - [ ] [AC1] <criterion>
#   seal.md  "## Acceptance Criteria"   →  - [x] [AC1] <criterion> — evidence: <…>
# A criterion is "done" iff its [ACn] id is CHECKED ([x]) in seal.md. This is what
# makes the criterion executable: a stable handle the prove/seal phases grade against.
# Without ids it falls back to a count/unchecked heuristic and recommends adding them.
#
# Usage: check-acceptance.sh <workspace-dir>
# Exit:  0 every spec criterion proven in seal · 1 gap (uncovered/unproven) ·
#        2 usage · 5 missing spec.md or seal.md

set -euo pipefail

ws="${1:-}"
if [ -z "$ws" ] || [ ! -d "$ws" ]; then
  printf 'usage: check-acceptance.sh <workspace-dir>\n' >&2
  exit 2
fi
spec="$ws/spec.md"; seal="$ws/seal.md"
[ -f "$spec" ] || { printf 'check-acceptance: no spec.md in %s\n' "$ws" >&2; exit 5; }
[ -f "$seal" ] || { printf 'check-acceptance: no seal.md in %s — feature not sealed\n' "$ws" >&2; exit 5; }

# Print the body lines under an "## Acceptance Criteria" heading (case-insensitive
# on "criteria"), up to the next "## " heading. BSD/GNU awk safe (no IGNORECASE).
ac_section() { awk '/^##[[:space:]]+[Aa]cceptance [Cc]riteria/{s=1;next} /^##[[:space:]]/{s=0} s' "$1"; }

spec_body="$(ac_section "$spec")"
seal_body="$(ac_section "$seal")"

if [ -z "$spec_body" ]; then
  printf 'check-acceptance: spec.md has no "## Acceptance criteria" section — nothing to grade\n' >&2
  exit 1
fi

spec_ids="$(printf '%s\n' "$spec_body" | grep -oE '\[AC[0-9]+\]' | sort -u || true)"

if [ -n "$spec_ids" ]; then
  # ID mode — each spec [ACn] must appear on a checked ([x]) line in seal.md.
  seal_checked="$(printf '%s\n' "$seal_body" | grep -E '^[[:space:]]*-[[:space:]]*\[[xX]\]' | grep -oE '\[AC[0-9]+\]' | sort -u || true)"
  missing=""
  for id in $spec_ids; do
    printf '%s\n' "$seal_checked" | grep -qxF "$id" || missing="$missing $id"
  done
  total=$(printf '%s\n' "$spec_ids" | grep -c . || true)
  if [ -n "$missing" ]; then
    printf 'check-acceptance: %d/%d criteria proven; UNPROVEN/UNCOVERED:%s (must be checked [x] in seal.md)\n' \
      "$(( total - $(printf '%s' "$missing" | wc -w) ))" "$total" "$missing" >&2
    exit 1
  fi
  printf 'check-acceptance: OK — all %d acceptance criteria (%s) proven + checked in seal.md\n' "$total" "$(printf '%s' "$spec_ids" | tr '\n' ' ')"
  exit 0
fi

# Fallback (no ids) — count spec criteria vs seal checked/unchecked items.
spec_n=$(printf '%s\n' "$spec_body" | grep -cE '^[[:space:]]*-[[:space:]]' || true)
seal_total=$(printf '%s\n' "$seal_body" | grep -cE '^[[:space:]]*-[[:space:]]*\[[ xX]\]' || true)
seal_unchecked=$(printf '%s\n' "$seal_body" | grep -cE '^[[:space:]]*-[[:space:]]*\[[[:space:]]\]' || true)

if [ "$seal_total" -lt "$spec_n" ]; then
  printf 'check-acceptance: seal.md lists %d acceptance items but spec.md has %d criteria — %d dropped. (Tag criteria [AC1].. to grade by id.)\n' \
    "$seal_total" "$spec_n" "$(( spec_n - seal_total ))" >&2
  exit 1
fi
if [ "$seal_unchecked" -gt 0 ]; then
  printf 'check-acceptance: %d acceptance criterion(s) unchecked in seal.md (unproven). (Tag criteria [AC1].. to grade by id.)\n' "$seal_unchecked" >&2
  exit 1
fi
printf 'check-acceptance: OK — %d acceptance items all checked in seal.md (no ids; add [AC1].. for id-level grading)\n' "$seal_total"
exit 0
