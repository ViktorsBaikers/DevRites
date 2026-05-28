#!/usr/bin/env bash
# Print the active DevRites feature's state — used by /rite-status.
# Portable across harnesses (no Claude-Code-specific `!`-prefix syntax in SKILL.md).
#
# Usage: load-state.sh [slug]
#   slug — optional override; defaults to .devrites/ACTIVE
#
# Output: structured plain text the SKILL.md asks the model to summarize.

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
for f in brief spec plan tasks questions decisions assumptions drift touched-files evidence browser-evidence design-brief polish-report review seal; do
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
    /^## q-/ { in_q=1; status=""; gate=""; next }
    in_q && /^status:/ { sub(/^status:[[:space:]]*/, "", $0); status=$0 }
    in_q && /^gate:/   { sub(/^gate:[[:space:]]*/,   "", $0); gate=$0 }
    in_q && /^##[[:space:]]/ {
      # next header — finalize previous; only counts "open"
      if (status == "open") counts[gate]++
      in_q=0
    }
    END {
      if (status == "open") counts[gate]++
      printf "  open: %d blocking, %d validating, %d advisory, %d escalating\n", \
        counts["blocking"]+0, counts["validating"]+0, counts["advisory"]+0, counts["escalating"]+0
    }
  ' "$q"
else
  echo "  (no questions.md yet)"
fi
