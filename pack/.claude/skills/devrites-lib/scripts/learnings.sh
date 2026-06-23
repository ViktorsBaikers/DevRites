#!/usr/bin/env bash
# learnings.sh — the cross-feature learning ledger for /rite-learn. Recurring corrections,
# dismissed review findings, and dead-ends are durable signal that today dies with each
# archived feature; this captures them in .devrites/learnings.md (loaded by the review
# skills before a fan-out) and mines archived features for repeated patterns worth
# promoting to a project rule. `add`/`mine` only; the promotion decision stays human.
#
#   learnings.sh add  <slug> "<text>" [tag]   # append a dated entry to the ledger
#   learnings.sh list                          # print the ledger
#   learnings.sh mine [archive-dir]            # cluster repeated finding phrases from archived features
#   learnings.sh nudge [archive-dir]           # silent unless a class recurs >=3x AND new signal since last review
#
# Ledger: .devrites/learnings.md (created on first add). `list`/`mine`/`nudge` are read-only.
# `nudge` snoozes on .devrites/.learnings-reviewed (touched by /rite-learn) so it never nags.
#
# Exit codes: 0 ok · 2 bad args
set -u

ledger=".devrites/learnings.md"
cmd="${1:-}"

case "$cmd" in
  add)
    slug="${2:-}"; text="${3:-}"; tag="${4:-note}"
    [ -n "$text" ] || { echo "usage: learnings.sh add <slug> \"<text>\" [tag]" >&2; exit 2; }
    if [ ! -f "$ledger" ]; then
      mkdir -p "$(dirname "$ledger")"
      printf '# DevRites learnings ledger\n\nProject-local lessons mined from shipped features — recurring corrections,\ndismissed-finding classes, and dead-ends. Loaded by the review skills before a fan-out so\nthe same false positive or the same mistake does not recur. Untrusted prior: live code\nalways overrides a ledger entry (see rules/security.md).\n\n' > "$ledger"
    fi
    printf -- '- [%s] (%s · %s) %s\n' "$(date +%Y-%m-%d)" "$tag" "${slug:-?}" "$text" >> "$ledger"
    echo "learnings: recorded."
    ;;
  list)
    [ -f "$ledger" ] && cat "$ledger" || echo "learnings: ledger empty ($ledger not present)."
    ;;
  mine)
    arch="${2:-.devrites/archive}"
    [ -d "$arch" ] || { echo "learnings: no archive at $arch — nothing to mine."; exit 0; }
    echo "learnings: repeated finding/decision phrases across archived features (count >= 2):"
    # Pull short finding/dead-end lines, normalize (lowercase, strip ids/paths/numbers), cluster.
    grep -rhiE '^[-*] |finding|dead end|drift|dismiss' \
      "$arch"/*/decisions.md "$arch"/*/drift.md "$arch"/*/review.md 2>/dev/null \
      | sed -E 's/^[-*[:space:]]+//; s/`[^`]*`//g; s/[0-9]+//g; s/[[:space:]]+/ /g' \
      | tr '[:upper:]' '[:lower:]' \
      | awk '{ $1=$1; if (length($0) > 12) print substr($0,1,80) }' \
      | sort | uniq -c | sort -rn \
      | awk '$1 >= 2 { printf "  %2d×  %s\n", $1, substr($0, index($0,$2)) }' \
      | head -25
    echo "(review these — a stable recurring class is a candidate for a project rule or a ledger entry.)"
    ;;
  nudge)
    arch="${2:-.devrites/archive}"
    [ -d "$arch" ] || exit 0
    # Cross-feature only: need >=2 shipped features before a "recurring across features" nudge.
    [ "$(ls -d "$arch"/*/ 2>/dev/null | wc -l | tr -d ' ')" -ge 2 ] || exit 0
    # Snooze: stay silent if reviewed since the newest archived file changed.
    marker=".devrites/.learnings-reviewed"
    if [ -f "$marker" ] && [ -z "$(find "$arch" -type f -newer "$marker" 2>/dev/null | head -1)" ]; then
      exit 0
    fi
    top="$(grep -rhiE '^[-*] |finding|dead end|drift|dismiss' \
            "$arch"/*/decisions.md "$arch"/*/drift.md "$arch"/*/review.md 2>/dev/null \
          | sed -E 's/^[-*[:space:]]+//; s/`[^`]*`//g; s/[0-9]+//g; s/[[:space:]]+/ /g' \
          | tr '[:upper:]' '[:lower:]' \
          | awk '{ if (length($0) > 12) print substr($0,1,60) }' \
          | sort | uniq -c | sort -rn | awk '$1 >= 3 { print; exit }')"
    [ -z "$top" ] && exit 0
    n="$(printf '%s' "$top" | awk '{print $1}')"
    phrase="$(printf '%s' "$top" | sed -E 's/^[[:space:]]*[0-9]+[[:space:]]+//' | cut -c1-48)"
    printf 'learnings: a pattern recurs %sx across shipped features ("%s…") — review + maybe promote it to a rule with /rite-learn.\n' "$n" "$phrase"
    ;;
  *)
    echo "usage: learnings.sh add|list|mine|nudge [...]" >&2; exit 2 ;;
esac
