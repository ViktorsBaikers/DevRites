#!/usr/bin/env bash
# DevRites PreToolUse hook — A1 pre-block: keep the /rite-build ORCHESTRATOR from editing
# source while a slice is mid-build. The slice-wright (a Task subagent) is the only sanctioned
# writer of code + tests; the orchestrator writes only .devrites/ bookkeeping. This hook is the
# optional pre-block companion to the reconcile.sh post-hoc gate (which always runs).
#
# Safety model: FAIL-OPEN, and OBSERVE by default.
#   - It NEVER blocks unless ALL hold: a build window is open, the edit is to a source file,
#     it came from the MAIN thread (no agent_id), and enforce mode is on.
#   - On any doubt — no active workspace, no open window, no node, unparseable payload, a
#     subagent caller, a .devrites/ path, the inline fallback — it exits 0 and stays silent.
#   - Default mode is "observe": it LOGS what it would block (to .a1-guard.log) and allows.
#     Flip to enforce only AFTER confirming the log never flags the wright's own edits — that
#     confirms this Claude Code build populates agent_id for subagents (older builds may not;
#     if the log flags wright edits, agent_id is unavailable here — stay in observe).
#
# Enable enforce (either one):
#   export DEVRITES_A1_HOOK=enforce          # session-wide
#   touch .devrites/work/<slug>/.a1-enforce  # per-feature
#
# Window: opened by `reconcile.sh snapshot` (writes .reconcile-base) before dispatch, closed by
# a clean `reconcile.sh check`. A violation leaves it open so inline patching stays blocked
# until the wright is re-dispatched. A stale window (abandoned build) → clear with
# `rm .devrites/work/<slug>/.reconcile-base` or re-run the check.
set -u

input="$(cat)"

# Fast path: must look like a tool call at all.
case "$input" in *'"tool_name"'*) : ;; *) exit 0 ;; esac

root="${CLAUDE_PROJECT_DIR:-$PWD}"

# No active DevRites feature → not our concern (keeps non-DevRites projects untouched).
[ -f "$root/.devrites/ACTIVE" ] || exit 0
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null)"
[ -n "$slug" ] || exit 0
d="$root/.devrites/work/$slug"

# Mid-build window open? Only armed between a reconcile snapshot and its check. This scoping is
# what keeps the hook off /rite-polish, /rite-quick, and ordinary manual edits — they edit
# source from the main thread too, legitimately, and no window is open then.
[ -f "$d/.reconcile-base" ] || exit 0

# Inline fallback: the orchestrator is legitimately the writer (no wright was dispatched).
[ -f "$d/.reconcile-inline" ] && exit 0

# Parse with node (ships with Claude Code). No node or a parse failure → fail open.
command -v node >/dev/null 2>&1 || exit 0
parsed="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const j=JSON.parse(s);const ti=j.tool_input||{};const fp=ti.file_path||ti.path||"";process.stdout.write([j.tool_name||"",fp,j.agent_id||""].join("\t"))}catch(e){}})' 2>/dev/null)"
[ -n "$parsed" ] || exit 0
IFS=$'\t' read -r tool fpath agent <<EOF
$parsed
EOF

# Only Edit-family writes can touch source.
case "$tool" in Edit|Write|MultiEdit|NotebookEdit) : ;; *) exit 0 ;; esac

# Subagent caller (the wright) → allow. agent_id is present only inside a subagent.
[ -n "$agent" ] && exit 0

# Can't tell the path → fail open.
[ -n "$fpath" ] || exit 0
case "$fpath" in /*) abs="$fpath" ;; *) abs="$root/$fpath" ;; esac

# Bookkeeping under .devrites/ is the orchestrator's own legit target → allow.
case "$abs" in "$root/.devrites/"*) exit 0 ;; esac

# Reaching here = the MAIN thread is about to edit a SOURCE file mid-build. That is the A1
# breach. Observe (default) logs and allows; enforce denies.
mode="${DEVRITES_A1_HOOK:-observe}"
[ -f "$d/.a1-enforce" ] && mode="enforce"

if [ "$mode" = "enforce" ]; then
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"DevRites A1: the orchestrator must not edit source mid-build — the slice-wright is the only writer. Re-dispatch the wright (continue it once) or stop + escalate; do not patch the code yourself. (devrites-a1-guard)"}}'
  exit 0
fi

# Observe mode — record what enforce WOULD have blocked, then allow.
printf '%s\tWOULD-BLOCK\t%s\t%s\n' "$(date '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || echo '?')" "$tool" "$abs" >> "$d/.a1-guard.log" 2>/dev/null || true
exit 0
