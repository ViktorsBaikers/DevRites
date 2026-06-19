#!/usr/bin/env bash
# DevRites orientation preamble — print the active feature's workspace state.
# Run first (step 0) by every workspace-operating rite-* skill via the standard
# resolution snippet. Read-only; never mutates workspace files.
# Portable across harnesses (no Claude-Code-specific `!`-prefix syntax in SKILL.md).
#
# Usage: preamble.sh [slug]
#   slug — optional override; defaults to .devrites/ACTIVE
#
# Paths resolve relative to CWD (the project root), matching the rest of the
# pack; do not anchor to the git root here — callers/tests run it with CWD set
# to the workspace root, which is not always a git repo.
#
# Output: structured plain text the SKILL.md asks the model to orient from.

set -u
slug="${1:-$(cat .devrites/ACTIVE 2>/dev/null)}"
d=".devrites/work/$slug"

if [ -z "$slug" ] || [ ! -d "$d" ]; then
  echo "No active workspace. Run /rite-spec <feature> to start."
  exit 0
fi

echo "## $slug"
if [ -f "$d/state.md" ]; then
  echo "### state.md"
  cat "$d/state.md"
fi

echo
echo "### artifacts present"
for f in brief spec references strategy plan tasks eng-review test-plan questions decisions assumptions drift attempts touched-files evidence browser-evidence design-brief polish-report review seal ship handoff; do
  if [ -f "$d/$f.md" ]; then
    echo "  ✓ $f.md"
  fi
done

echo
echo "### run mode"
if [ -f ".devrites/AFK" ]; then
  echo "  AFK (sentinel present)"
  # surface optional config without parsing yaml — print non-comment lines verbatim
  grep -vE '^\s*(#|$)' .devrites/AFK 2>/dev/null | sed 's/^/    /'
else
  echo "  HITL (no sentinel)"
fi

echo
echo "### questions"
q="$d/questions.md"
if [ -f "$q" ]; then
  # Tally by gate using "status:" + "gate:" lines that follow each "## q-" header.
  awk '
    # Finalize the previous block BEFORE resetting — adjacent "## q-" headers
    # would otherwise drop every open question except the last.
    /^## q-/ {
      if (in_q && status == "open") counts[gate]++
      in_q=1; status=""; gate=""; next
    }
    in_q && /^status:/ { sub(/^status:[[:space:]]*/, "", $0); status=$0 }
    in_q && /^gate:/   { sub(/^gate:[[:space:]]*/,   "", $0); gate=$0 }
    in_q && /^##[[:space:]]/ {
      # a non-question header ends the block — finalize; only counts "open"
      if (status == "open") counts[gate]++
      in_q=0
    }
    END {
      if (in_q && status == "open") counts[gate]++
      printf "  open: %d blocking, %d validating, %d advisory, %d escalating\n", \
        counts["blocking"]+0, counts["validating"]+0, counts["advisory"]+0, counts["escalating"]+0
    }
  ' "$q"
else
  echo "  (no questions.md yet)"
fi
