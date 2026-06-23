#!/usr/bin/env bash
# Spec-grammar validator — deterministic, zero-token lint of the structured
# Requirement/Scenario acceptance grammar (rules/spec-grammar.md) in a spec.md.
# Turns "are the behavioral requirements well-formed?" into an exit code instead
# of prose the model walks by hand — the spec-side mirror of check-acceptance.sh
# (which grades the SAME [ACn] ids against seal.md at the end of the lifecycle).
#
# Grammar checked (only when the spec opts into the structured form):
#   ### Requirement: <name>     — must carry a SHALL/MUST statement (header or body)
#                               — must own >=1 #### Scenario; header names unique
#   #### Scenario: <name>       — must carry both a WHEN and a THEN line
# A spec with ZERO `### Requirement:` headers uses the simple flat-bullet form
# (- [ ] [AC1] ...) — valid, and this validator no-ops (exit 0). Absence is never
# a failure, the same discipline as the principles gate. The grammar is opt-in and
# recommended for behavioral / high-risk requirements (progressive rigor).
#
# Usage: spec-validate.sh <workspace-dir | spec.md path>
# Exit:  0 valid (or no structured requirements — nothing to lint) ·
#        1 grammar violation(s) · 2 usage · 5 missing spec.md
#
# [ACn] compatibility: tag each scenario's checkable line with an [ACn] id so the
# same criterion is graded at seal by check-acceptance.sh. This validator checks
# SHAPE, not proof; the two compose, they don't overlap.

set -euo pipefail

arg="${1:-}"
if [ -z "$arg" ]; then
  printf 'usage: spec-validate.sh <workspace-dir | spec.md path>\n' >&2
  exit 2
fi

if [ -d "$arg" ]; then
  spec="$arg/spec.md"
elif [ -f "$arg" ]; then
  spec="$arg"
else
  printf 'spec-validate: no such workspace or file: %s\n' "$arg" >&2
  exit 2
fi
[ -f "$spec" ] || { printf 'spec-validate: no spec.md at %s\n' "$spec" >&2; exit 5; }

# Parse the structured grammar in one awk pass. Emits ERROR:/WARN: finding lines
# and a trailing "__SUMMARY__ <requirements> <scenarios>" line. BSD/GNU awk safe:
# tolower() is POSIX; no IGNORECASE, no gensub. Keyword matches are case-sensitive
# on purpose — the grammar mandates uppercase SHALL / MUST / WHEN / THEN.
findings="$(awk -v FILE="$spec" '
  function close_scenario() {
    if (inScenario) {
      if (!scenWhen) { print "ERROR:" FILE ": Requirement \"" curReq "\" > Scenario \"" scenName "\" (line " scenLine ") has no WHEN line"; }
      if (!scenThen) { print "ERROR:" FILE ": Requirement \"" curReq "\" > Scenario \"" scenName "\" (line " scenLine ") has no THEN line"; }
      inScenario = 0
    }
  }
  function close_requirement() {
    close_scenario()
    if (curReq != "") {
      if (!sawShall) { print "ERROR:" FILE ": Requirement \"" curReq "\" (line " reqLine ") has no SHALL/MUST statement"; }
      if (scenInReq == 0) {
        if (looseWhenThen) { print "ERROR:" FILE ": Requirement \"" curReq "\" (line " reqLine ") has WHEN/THEN lines but no \"#### Scenario:\" header — wrap them in a scenario"; }
        else { print "ERROR:" FILE ": Requirement \"" curReq "\" (line " reqLine ") has no \"#### Scenario:\" block"; }
      }
    }
  }
  /^[[:space:]]*###[[:space:]]+[Rr]equirement:/ {
    close_requirement()
    reqCount++
    name = $0
    sub(/^[[:space:]]*###[[:space:]]+[Rr]equirement:[[:space:]]*/, "", name)
    sub(/[[:space:]]+$/, "", name)
    curReq = name; reqLine = NR
    sawShall = ($0 ~ /(^|[^A-Za-z])(SHALL|MUST)([^A-Za-z]|$)/) ? 1 : 0
    scenInReq = 0; looseWhenThen = 0
    key = tolower(name)
    if (key in seen) { print "ERROR:" FILE ": duplicate Requirement header \"" name "\" (lines " seen[key] " and " NR ") — headers must be unique"; }
    else { seen[key] = NR }
    next
  }
  /^[[:space:]]*####[[:space:]]+[Ss]cenario:/ {
    close_scenario()
    if (curReq == "") { print "ERROR:" FILE ": Scenario at line " NR " is not under any \"### Requirement:\""; }
    sName = $0
    sub(/^[[:space:]]*####[[:space:]]+[Ss]cenario:[[:space:]]*/, "", sName)
    sub(/[[:space:]]+$/, "", sName)
    inScenario = 1; scenCount++; scenInReq++
    scenName = sName; scenLine = NR; scenWhen = 0; scenThen = 0
    next
  }
  {
    if (curReq != "") {
      if ($0 ~ /(^|[^A-Za-z])(SHALL|MUST)([^A-Za-z]|$)/) { sawShall = 1 }
      hasWhen = ($0 ~ /(^|[^A-Za-z])WHEN([^A-Za-z]|$)/)
      hasThen = ($0 ~ /(^|[^A-Za-z])THEN([^A-Za-z]|$)/)
      if (inScenario) {
        if (hasWhen) { scenWhen = 1 }
        if (hasThen) { scenThen = 1 }
      } else if (hasWhen || hasThen) {
        looseWhenThen = 1
      }
    }
  }
  END { close_requirement(); print "__SUMMARY__ " reqCount+0 " " scenCount+0 }
' "$spec")"

summary="$(printf '%s\n' "$findings" | grep '^__SUMMARY__' || true)"
reqs="$(printf '%s' "$summary" | awk '{print $2+0}')"
scens="$(printf '%s' "$summary" | awk '{print $3+0}')"
problems="$(printf '%s\n' "$findings" | grep -E '^(ERROR|WARN):' || true)"
errs="$(printf '%s\n' "$findings" | grep -c '^ERROR:' || true)"

rel="${spec#"$PWD"/}"

if [ "$reqs" -eq 0 ]; then
  printf 'spec-validate: %s uses the simple acceptance form (no "### Requirement:" blocks) — nothing to lint\n' "$rel"
  exit 0
fi

if [ -n "$problems" ]; then
  printf '%s\n' "$problems" >&2
fi

if [ "$errs" -gt 0 ]; then
  printf 'spec-validate: %s — %d requirement(s) / %d scenario(s); %d grammar error(s) (see rules/spec-grammar.md)\n' \
    "$rel" "$reqs" "$scens" "$errs" >&2
  exit 1
fi

printf 'spec-validate: OK — %s: %d requirement(s) / %d scenario(s) well-formed (SHALL + WHEN/THEN, unique headers)\n' \
  "$rel" "$reqs" "$scens"
exit 0
