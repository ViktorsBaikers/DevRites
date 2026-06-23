#!/usr/bin/env bash
# stuck.sh — loop detector for the unattended build. The slice-wright or orchestrator
# appends one record per tool action; `check` flags the semantic-repeat patterns that
# mean the agent is stuck burning tokens (the #1 silent AFK failure mode): the same
# action repeating, an action<->error ping-pong, or one action dominating the window.
# A detected loop pauses the build even under AFK (rules/afk-hitl.md). Deterministic;
# matching is by (tool,target) hash, timestamps ignored.
#
#   stuck.sh log   <slug> <tool> [target]   # append one action record (caller)
#   stuck.sh check <slug> [window]          # non-zero if the recent window looks stuck
#
# The log lives at .devrites/work/<slug>/action.log as "<epoch> <tool> <sha1>" lines.
#
# Exit codes:
#   0  ok — not stuck (or logging, or no workspace: never fail the caller on log)
#   2  bad args
#   3  STUCK — the recent window is a loop: stop and escalate to a blocking question
set -u

cmd="${1:-}"; slug="${2:-}"
[ -n "$cmd" ] && [ -n "$slug" ] || { echo "usage: stuck.sh log|check <slug> [...]" >&2; exit 2; }

dir=".devrites/work/$slug"
log="$dir/action.log"

hash_of() {
  if command -v sha1sum >/dev/null 2>&1; then
    printf '%s' "$1" | sha1sum | awk '{print $1}'
  else
    printf '%s' "$1" | shasum | awk '{print $1}'
  fi
}

case "$cmd" in
  log)
    tool="${3:-}"; target="${4:-}"
    [ -n "$tool" ] || { echo "usage: stuck.sh log <slug> <tool> [target]" >&2; exit 2; }
    [ -d "$dir" ] || exit 0
    printf '%s %s %s\n' "$(date +%s)" "$tool" "$(hash_of "$tool|$target")" >> "$log"
    ;;
  check)
    win="${3:-4}"
    [ -f "$log" ] || { echo "stuck: no actions logged — not stuck."; exit 0; }
    awk -v win="$win" '
      { h[NR]=$3 }
      END {
        if (NR < win) { print "stuck: only " NR " action(s) — not stuck."; exit 0 }
        same=1
        for (i=NR-win+1; i<=NR; i++) if (h[i]!=h[NR]) { same=0; break }
        if (same) { print "stuck: last " win " actions identical — STUCK."; exit 3 }
        if (NR>=6) {
          pp=1
          for (i=NR-5; i<=NR; i++) { e=(((NR-i)%2)==0)?h[NR]:h[NR-1]; if (h[i]!=e) { pp=0; break } }
          if (pp && h[NR]!=h[NR-1]) { print "stuck: A<->B ping-pong over last 6 — STUCK."; exit 3 }
        }
        if (NR>=8) {
          for (i=NR-7;i<=NR;i++) c[h[i]]++
          top=0; for (k in c) if (c[k]>top) top=c[k]
          if (top>=6) { print "stuck: one action " top "/8 of recent window — STUCK."; exit 3 }
        }
        print "stuck: recent window varied — not stuck."
      }
    ' "$log"
    ;;
  *)
    echo "usage: stuck.sh log|check <slug> [...]" >&2; exit 2 ;;
esac
