#!/usr/bin/env bash
# Resolve an open question in the active DevRites workspace.
# Used by /rite-resolve. Keeps questions.md and state.md consistent.
#
# Usage:
#   resolve.sh <qid> "<answer>"
#   resolve.sh --drop <qid> ["<reason>"]
#   resolve.sh --batch <path-to-yaml>
#   resolve.sh next-qid <questions.md path>
#
# Exit codes:
#   0  resolved
#   2  no active workspace
#   3  qid not found
#   4  qid not open (already answered/dropped)
#   5  bad arguments
#   6  qid collision (next-qid: computed id already has a header)

set -u

die() { printf '%s\n' "$*" >&2; exit "${2:-1}"; }

# next-qid: print the next sequential qid for TODAY. Counts existing
# `## q-YYYY-MM-DD-` headers for today's date, prints q-YYYY-MM-DD-NNN with the
# next zero-padded id, and refuses (exit 6) if that header already exists.
# Operates on an explicit questions.md path — no active workspace required.
if [ "${1:-}" = "next-qid" ]; then
  qpath="${2:-}"
  [ -n "$qpath" ] || die "Usage: resolve.sh next-qid <questions.md path>" 5
  today="$(date +%F)"
  # A fresh questions.md may not exist yet — treat absent as zero headers.
  if [ -f "$qpath" ]; then
    used="$(grep -c "^## q-${today}-" "$qpath" 2>/dev/null || true)"
  else
    used=0
  fi
  next=$(( used + 1 ))
  qid="$(printf 'q-%s-%03d' "$today" "$next")"
  # Defend against gaps/manual edits: never hand out an id that already exists.
  if [ -f "$qpath" ] && grep -q "^## ${qid}\([[:space:]]\|$\)" "$qpath"; then
    die "qid already present: $qid (questions.md edited out of sequence)" 6
  fi
  printf '%s\n' "$qid"
  exit 0
fi

slug="$(cat .devrites/ACTIVE 2>/dev/null || true)"
[ -n "$slug" ] || die "No active workspace. Run /rite-spec <feature> first." 2
work=".devrites/work/$slug"
qfile="$work/questions.md"
sfile="$work/state.md"
[ -f "$qfile" ] || die "questions.md missing at $qfile" 2
[ -f "$sfile" ] || die "state.md missing at $sfile" 2

mode=""
qid=""
payload=""
case "${1:-}" in
  --drop)
    mode="drop"
    qid="${2:-}"
    payload="${3:-dropped}"
    ;;
  --batch)
    mode="batch"
    payload="${2:-}"
    [ -f "$payload" ] || die "Batch file not found: $payload" 5
    ;;
  "")
    die "Usage: resolve.sh <qid> \"<answer>\"  |  --drop <qid> [\"<reason>\"]  |  --batch <file>" 5
    ;;
  *)
    mode="answer"
    qid="$1"
    payload="${2:-}"
    [ -n "$payload" ] || die "Answer text required for $qid" 5
    ;;
esac

now() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }

# Apply a single resolution to questions.md. Awk rewrites the file atomically.
apply_one() {
  local _qid="$1" _status="$2" _answer="$3"
  local _now; _now="$(now)"

  awk -v qid="$_qid" -v st="$_status" -v ans="$_answer" -v ts="$_now" '
    BEGIN { in_q=0; touched=0; found=0; status_seen=0 }
    function flush_if_target() {
      # called when leaving the target block
      if (in_q && !status_seen) {
        # malformed entry without status — append the closing fields
        print "status: " st
        print "answered_at: " ts
        print "answer: " ans
      }
    }
    /^## q-/ {
      flush_if_target()
      in_q = ($0 ~ ("^## " qid "([[:space:]]|$)"))
      status_seen = 0
      if (in_q) { found = 1 }
      print
      next
    }
    in_q && /^status:/ {
      status_seen = 1
      cur = $0; sub(/^status:[[:space:]]*/, "", cur)
      if (cur != "open") {
        # signal non-open with sentinel to stderr-equivalent: write file unchanged
        # we cannot abort from awk cleanly; mark and continue
        print "status: " cur
        in_q = 0
        bad_state = 1
        next
      }
      print "status: " st
      next
    }
    in_q && /^answered_at:/ {
      print "answered_at: " ts
      next
    }
    in_q && /^answer:/ {
      print "answer: " ans
      next
    }
    { print }
    END {
      flush_if_target()
      if (!found) exit 10
      if (bad_state) exit 11
    }
  ' "$qfile" > "$qfile.tmp"
  rc=$?

  if [ "$rc" -eq 10 ]; then
    rm -f "$qfile.tmp"
    die "qid not found: $_qid" 3
  fi
  if [ "$rc" -eq 11 ]; then
    rm -f "$qfile.tmp"
    die "qid not open (already answered/dropped): $_qid" 4
  fi

  # If the awk output is missing answered_at / answer (entry without those keys),
  # append them at the end of the block by re-running with insertion mode.
  mv "$qfile.tmp" "$qfile"

  # Ensure answered_at + answer keys exist for the target qid (insert after `status:` if missing)
  awk -v qid="$_qid" -v ts="$_now" -v ans="$_answer" '
    BEGIN { in_q=0; saw_at=0; saw_ans=0; saw_status_line=0 }
    /^## q-/ {
      if (in_q) {
        if (!saw_at)  print "answered_at: " ts
        if (!saw_ans) print "answer: " ans
      }
      in_q = ($0 ~ ("^## " qid "([[:space:]]|$)"))
      saw_at = 0; saw_ans = 0; saw_status_line = 0
      print; next
    }
    in_q && /^answered_at:/ { saw_at = 1 }
    in_q && /^answer:/      { saw_ans = 1 }
    in_q && /^status:/      { saw_status_line = 1 }
    { print }
    END {
      if (in_q) {
        if (!saw_at)  print "answered_at: " ts
        if (!saw_ans) print "answer: " ans
      }
    }
  ' "$qfile" > "$qfile.tmp" && mv "$qfile.tmp" "$qfile"
}

# Update state.md: if the Awaiting human block references the qid, remove the block,
# flip Status: awaiting_human → running, clear Next step, and append a Log entry.
# Two-pass: scan to detect a match, then rewrite the file with the side-effects applied.
clear_state_if_matches() {
  local _qid="$1"
  local _now; _now="$(now)"

  # Pass 1: does the Awaiting block reference this qid?
  local matched
  matched="$(awk -v qid="$_qid" '
    BEGIN { in_aw=0; m=0 }
    /^## Awaiting human/  { in_aw=1; next }
    in_aw && /^##[[:space:]]/ { in_aw=0 }
    in_aw {
      if ($0 ~ ("qid:[[:space:]]*" qid "([[:space:]]|$)")) m=1
    }
    END { print m }
  ' "$sfile")"

  if [ "$matched" != "1" ]; then
    return 0
  fi

  # Pass 2: rewrite. Drop the Awaiting human block (header + body until the next `## `),
  # flip Status, clear Next step, append a Log entry under ## Log.
  awk -v qid="$_qid" -v ts="$_now" '
    BEGIN { in_aw=0; in_log=0; log_appended=0 }
    /^## Awaiting human/ { in_aw=1; next }
    in_aw && /^##[[:space:]]/ { in_aw=0 }     # next ## header ends the block
    in_aw { next }                              # swallow lines inside the awaiting block

    /^- Status:/ {
      print "- Status: running"
      next
    }
    /^- Next step:/ {
      print "- Next step: (resume — `/rite-build` to continue the workflow)"
      next
    }
    /^## Log/ {
      print
      in_log = 1
      next
    }
    in_log && /^##[[:space:]]/ {
      # leaving the log section — append our entry before this header if we have not yet
      if (!log_appended) {
        printf "- %s build: resolved %s\n", ts, qid
        log_appended = 1
      }
      in_log = 0
      print
      next
    }
    { print }

    END {
      if (in_log && !log_appended) {
        printf "- %s build: resolved %s\n", ts, qid
      }
    }
  ' "$sfile" > "$sfile.tmp" && mv "$sfile.tmp" "$sfile"
}

case "$mode" in
  answer)
    apply_one "$qid" "answered" "$payload"
    clear_state_if_matches "$qid"
    printf 'Resolved: %s\n' "$qid"
    printf 'Status:   answered\n'
    printf 'Workspace: questions.md + state.md updated.\n'
    ;;
  drop)
    apply_one "$qid" "dropped" "$payload"
    clear_state_if_matches "$qid"
    printf 'Dropped:  %s\n' "$qid"
    printf 'Reason:   %s\n' "$payload"
    printf 'Workspace: questions.md + state.md updated.\n'
    ;;
  batch)
    # Batch format: each line is `qid: <answer>` or `--drop qid: <reason>`. See answer-protocol.md.
    while IFS= read -r line; do
      case "$line" in
        ''|\#*) continue ;;
        --drop*)
          rest="${line#--drop }"
          bid="${rest%%:*}"
          reason="${rest#*:}"
          reason="${reason# }"
          apply_one "$bid" "dropped" "$reason"
          clear_state_if_matches "$bid"
          printf 'Dropped:  %s — %s\n' "$bid" "$reason"
          ;;
        *)
          bid="${line%%:*}"
          ans="${line#*:}"
          ans="${ans# }"
          apply_one "$bid" "answered" "$ans"
          clear_state_if_matches "$bid"
          printf 'Resolved: %s\n' "$bid"
          ;;
      esac
    done < "$payload"
    ;;
esac
