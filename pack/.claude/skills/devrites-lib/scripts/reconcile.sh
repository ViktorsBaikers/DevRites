#!/usr/bin/env bash
# Reconcile gate for /rite-build — proves the orchestrator never edited source
# itself. The slice-wright (a dispatched subagent) is the ONLY writer of code +
# tests; the orchestrator writes bookkeeping under .devrites/ and nothing else.
# This turns that rule (A1: "the orchestrator never edits source") into an exit
# code instead of a prose promise the model can rationalize past — same doctrine
# as readiness.sh / tick-afk.sh. Read-only w.r.t. the user's git index.
#
# Two phases, both called by /rite-build:
#   snapshot — step 3, right BEFORE dispatching the wright. Captures the working
#              tree (tracked + untracked, .gitignore honored) as a git tree
#              object, without touching the user's real index/staging. The base
#              file doubles as the "mid-build window" marker: while it exists the
#              optional pre-block hook (devrites-a1-guard.sh) is armed; a clean
#              check removes it (a violation keeps it hot until re-dispatch).
#   check    — step 6 (record), AFTER the wright returns and the orchestrator has
#              written the wright's reported `Files changed` paths (one per line)
#              to .devrites/work/<slug>/.reconcile-claimed. Diffs the tree against
#              the snapshot; every changed path that is NOT under .devrites/ and
#              NOT in the claimed manifest was edited outside the wright -> STOP.
#
# Usage:
#   reconcile.sh snapshot [slug]
#   reconcile.sh check    [slug]
#     slug — optional; defaults to .devrites/ACTIVE
#
# Exit codes:
#   0  ok — clean (check), snapshot written, or gate skipped (non-git repo /
#      inline fallback) with a printed notice
#   5  VIOLATION — source changed outside the wright's claimed set: STOP. A1
#      breach; re-dispatch the wright (continue it once) or revert the edits
#   6  setup error — bad args, missing snapshot, or missing claimed manifest

set -euo pipefail

mode="${1:-}"
slug="${2:-$(cat .devrites/ACTIVE 2>/dev/null || true)}"

# Non-git project: nothing to diff against. Skip rather than block the build.
root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$root" ]; then
  printf 'reconcile: not a git repo — gate skipped, verify the diff by hand.\n' >&2
  exit 0
fi

d=".devrites/work/$slug"
base="$d/.reconcile-base"
claimed="$d/.reconcile-claimed"
inline="$d/.reconcile-inline"

if [ -z "$slug" ] || [ ! -d "$d" ]; then
  printf 'reconcile: no active workspace (slug=%s) — nothing to reconcile.\n' "${slug:-<unset>}" >&2
  exit 6
fi

# Capture the whole working tree (tracked + untracked, .gitignore respected) as a
# tree object via a throwaway index, so the user's real index/staging is never
# touched. Prints the tree sha on stdout.
worktree_tree() {
  local idx="${TMPDIR:-/tmp}/devrites-reconcile-$$.idx"
  rm -f "$idx"
  GIT_INDEX_FILE="$idx" git -C "$root" add -A >/dev/null 2>&1
  local tree; tree="$(GIT_INDEX_FILE="$idx" git -C "$root" write-tree)"
  rm -f "$idx"
  printf '%s\n' "$tree"
}

# Close the mid-build window: drop the base (disarms the a1-guard hook) and the
# per-slice manifest/sentinel. Called only when the slice is accepted clean or
# was the inline fallback — a violation deliberately leaves the window hot.
close_window() { rm -f "$base" "$claimed" "$inline"; }

case "$mode" in
  snapshot)
    rm -f "$inline" "$claimed"   # clear any stale sentinel / prior-slice manifest
    worktree_tree > "$base"      # base present == mid-build window open
    printf 'reconcile: snapshot captured for %s.\n' "$slug"
    exit 0 ;;
  check)
    # Inline fallback: the orchestrator legitimately wrote the code itself
    # (no wright was dispatched), so the gate does not apply.
    if [ -f "$inline" ]; then
      printf 'reconcile: inline-fallback run (no wright dispatched) — gate skipped.\n' >&2
      close_window
      exit 0
    fi
    if [ ! -f "$base" ]; then
      printf 'reconcile: no snapshot (.reconcile-base) — run "reconcile.sh snapshot" before dispatch.\n' >&2
      exit 6
    fi
    if [ ! -f "$claimed" ]; then
      printf 'reconcile: no claimed manifest (.reconcile-claimed) — write the wright Files-changed paths first.\n' >&2
      exit 6
    fi
    base_tree="$(cat "$base")"
    now_tree="$(worktree_tree)"
    # Every path changed since the snapshot, minus bookkeeping (.devrites/ is the
    # orchestrator's own legit write target). Anything left that the wright did
    # not claim was edited by someone other than the wright.
    viol=0
    violist=""
    while IFS= read -r f; do
      [ -z "$f" ] && continue
      case "$f" in .devrites/*) continue ;; esac
      if ! grep -qxF "$f" "$claimed"; then
        viol=$(( viol + 1 ))
        violist="${violist}  - ${f}
"
      fi
    done < <(git -C "$root" diff --name-only "$base_tree" "$now_tree")
    if [ "$viol" -gt 0 ]; then
      printf 'reconcile: STOP — %d source file(s) changed OUTSIDE the wright (A1 breach):\n' "$viol" >&2
      printf '%s' "$violist" >&2
      printf 'The orchestrator must NOT edit source. Re-dispatch the wright (continue it once) or revert these.\n' >&2
      exit 5
    fi
    close_window
    printf "reconcile: OK — every source change is the wright's; orchestrator touched only bookkeeping.\n"
    exit 0 ;;
  *)
    printf 'reconcile: usage: reconcile.sh <snapshot|check> [slug]\n' >&2
    exit 6 ;;
esac
