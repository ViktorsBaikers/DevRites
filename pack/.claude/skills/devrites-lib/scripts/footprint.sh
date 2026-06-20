#!/usr/bin/env bash
# footprint.sh — the fan-out footprint for a feature: how heavy the run was, in numbers
# DevRites can actually count. NO token or dollar figure is ever produced (DevRites can't
# truthfully source one — inventing it would violate the proof thesis). Deterministic:
# every number comes from the append-only dispatch log the orchestrator writes.
#
#   footprint.sh log    <slug> <kind> [label]   # append one dispatch record (orchestrator)
#   footprint.sh render <slug>                   # print the one-line footprint (read-only)
#
# kind ∈ wright | reviewer | audit | doubt. The log lives at
# .devrites/work/<slug>/footprint.log as "<epoch> <kind> <label>" lines.
set -u

cmd="${1:-}"; slug="${2:-}"
[ -n "$cmd" ] && [ -n "$slug" ] || { echo "usage: footprint.sh log|render <slug> [...]" >&2; exit 2; }

dir=".devrites/work/$slug"
log="$dir/footprint.log"

case "$cmd" in
  log)
    kind="${3:-}"; label="${4:-}"
    [ -n "$kind" ] || { echo "usage: footprint.sh log <slug> <kind> [label]" >&2; exit 2; }
    [ -d "$dir" ] || exit 0   # no workspace → nothing to record (never fail the caller)
    printf '%s %s %s\n' "$(date +%s)" "$kind" "$label" >> "$log"
    ;;
  render)
    [ -f "$log" ] || { echo "Footprint: n/a (no dispatch records)"; exit 0; }
    awk '
      { n++; c[$2]++; e=$1; if (min=="" || e<min) min=e; if (e>max) max=e }
      END {
        mins = (max>min) ? int((max-min)/60) : 0
        w=c["wright"]+0; r=c["reviewer"]+0; a=c["audit"]+0; d=c["doubt"]+0
        parts=""
        if (w) parts = parts sep w " wright"; sep=" · "
        if (r) { parts = parts sep r " reviewer" (r==1?"":"s"); sep=" · " }
        if (a) { parts = parts sep a " audit" (a==1?"":"s"); sep=" · " }
        if (d) { parts = parts sep d " doubt"; sep=" · " }
        printf "Footprint: %d subagent%s (%s) · %d slice%s · ~%dm active\n", \
               n, (n==1?"":"s"), parts, w, (w==1?"":"s"), mins
      }' "$log"
    ;;
  *)
    echo "usage: footprint.sh log|render <slug> [...]" >&2; exit 2 ;;
esac
