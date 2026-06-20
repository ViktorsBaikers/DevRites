#!/usr/bin/env bash
# doctor-test.sh — fixture tests for the DevRites health diagnose (doctor.sh).
# Healthy project → clean (exit 0); each corruption → detected (exit 1 + the named issue).
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
DOCTOR="$HERE/../pack/.claude/skills/devrites-lib/scripts/doctor.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

# Build a HEALTHY DevRites project at $1 (install scripts present, empty ACTIVE).
mkproj() {
  local d="$1"
  mkdir -p "$d/.devrites/work" "$d/.claude/skills/devrites-lib/scripts" "$d/.claude/hooks"
  for s in preamble.sh progress.sh readiness.sh conventions.py; do : > "$d/.claude/skills/devrites-lib/scripts/$s"; done
  : > "$d/.devrites/ACTIVE"
}

# run doctor at root $1; expect exit code $2; if $3 given, the output must contain it.
expect() { # label root want_rc [grep]
  local out rc
  out="$(bash "$DOCTOR" --root "$2" 2>&1)"; rc=$?
  if [ "$rc" -ne "$3" ]; then echo "FAIL [$1]: rc $rc != $3"; echo "$out"; fail=1; return; fi
  if [ -n "${4:-}" ] && ! printf '%s' "$out" | grep -q "$4"; then
    echo "FAIL [$1]: output missing /$4/"; echo "$out"; fail=1; return
  fi
  echo "ok   [$1]"
}

# healthy → exit 0
H="$TMP/healthy"; mkproj "$H"
expect "healthy → clean" "$H" 0

# not a DevRites project → silent exit 0
N="$TMP/plain"; mkdir -p "$N"
expect "non-devrites → silent" "$N" 0

# stale ACTIVE → exit 1
S="$TMP/stale"; mkproj "$S"; printf 'ghost' > "$S/.devrites/ACTIVE"
expect "stale ACTIVE detected" "$S" 1 "stale ACTIVE"

# corrupt workspace (no state.md) → exit 1
C="$TMP/corrupt"; mkproj "$C"; printf 'feat' > "$C/.devrites/ACTIVE"; mkdir -p "$C/.devrites/work/feat"
expect "corrupt workspace detected" "$C" 1 "corrupt workspace"

# orphaned gate (open validating + state done) → exit 1
O="$TMP/orphan"; mkproj "$O"; printf 'feat' > "$O/.devrites/ACTIVE"; mkdir -p "$O/.devrites/work/feat"
printf 'phase: done\n' > "$O/.devrites/work/feat/state.md"
printf 'status: open\ngate: validating\n' > "$O/.devrites/work/feat/questions.md"
expect "orphaned gate detected" "$O" 1 "orphaned gate"

# broken hook wiring (settings references a missing hook) → exit 1
B="$TMP/brokenhook"; mkproj "$B"
printf '{ "hooks": { "SessionStart": [ { "command": "bash .claude/hooks/devrites-orient.sh" } ] } }\n' > "$B/.claude/settings.json"
expect "broken hook wiring detected" "$B" 1 "missing hook"

# missing install script → exit 1
M="$TMP/incomplete"; mkproj "$M"; rm "$M/.claude/skills/devrites-lib/scripts/conventions.py"
expect "incomplete install detected" "$M" 1 "install incomplete"

# verbose prints ok: lines on a healthy project
if bash "$DOCTOR" --root "$H" --verbose | grep -q '^ok:'; then echo "ok   [verbose prints ok lines]"; else echo "FAIL [verbose prints ok lines]"; fail=1; fi

if [ "$fail" -ne 0 ]; then echo "DOCTOR TESTS: FAIL"; exit 1; fi
echo "DOCTOR TESTS: PASS"
exit 0
