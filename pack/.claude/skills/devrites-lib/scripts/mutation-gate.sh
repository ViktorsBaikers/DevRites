#!/usr/bin/env bash
# mutation-gate.sh — certify the tests are strong enough to back "done": run the project's
# mutation runner scoped to the changed files and surface survived mutants (a behaviour no
# test actually checks). Coverage says a line RAN; mutation says it is CHECKED. Advisory by
# default (a survivor is a finding for the test reviewer, not an auto-fail) — set
# DEVRITES_MUTATION=enforce to make a sub-threshold score a hard NO-GO. No runner installed
# -> exit 0 with a note (never block a project that has no mutation tooling). Best-effort:
# wraps the runner, does not reimplement it.
#
# Usage: mutation-gate.sh [slug]   (slug defaults to .devrites/ACTIVE)
# Env:   DEVRITES_MUTATION=enforce   DEVRITES_MUTATION_MIN=<percent, default 60>
# Exit:  0 advisory/clean/no-tool · 2 no workspace · 3 enforce + below threshold
set -u

slug="${1:-}"
[ -n "$slug" ] || slug="$(cat .devrites/ACTIVE 2>/dev/null || true)"
[ -n "$slug" ] || { echo "mutation-gate: no active workspace." >&2; exit 2; }
d=".devrites/work/$slug"
[ -d "$d" ] || { echo "mutation-gate: no workspace at $d." >&2; exit 2; }

root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
enforce="${DEVRITES_MUTATION:-advisory}"
min="${DEVRITES_MUTATION_MIN:-60}"

tool=""
if [ -f "${root:-.}/stryker.conf.json" ] || [ -f "${root:-.}/stryker.config.js" ] || [ -f "${root:-.}/stryker.config.mjs" ]; then
  command -v npx >/dev/null 2>&1 && tool="stryker"
elif command -v mutmut >/dev/null 2>&1; then tool="mutmut"
elif command -v cosmic-ray >/dev/null 2>&1; then tool="cosmic-ray"
elif command -v pitest >/dev/null 2>&1 || [ -f "${root:-.}/pom.xml" ]; then tool="pitest"
fi

if [ -z "$tool" ]; then
  echo "mutation-gate: no mutation runner detected (stryker / mutmut / cosmic-ray / pitest) — advisory skipped."
  echo "  Strengthen the criticals by hand-mutating (flip a comparison, drop a guard) and confirming a test goes red (rules/testing.md)."
  exit 0
fi

echo "mutation-gate: detected '$tool'. Run it scoped to the slice's changed files and read the mutation score:"
case "$tool" in
  stryker)    echo "  npx stryker run --mutate \"\$(git diff --name-only HEAD | paste -sd, -)\"" ;;
  mutmut)     echo "  mutmut run --paths-to-mutate \"\$(git diff --name-only HEAD | tr '\\n' ' ')\" && mutmut results" ;;
  cosmic-ray) echo "  (configure session over the changed files, then: cosmic-ray exec ... && cr-report ...)" ;;
  pitest)     echo "  mvn org.pitest:pitest-maven:mutationCoverage -DtargetClasses=<changed>" ;;
esac
echo "  Record the score + any survived mutants in evidence.md; a survivor is a behaviour no test checks (a finding for the test reviewer)."
echo "  Mode: ${enforce} (DEVRITES_MUTATION=enforce makes a score < ${min}% a NO-GO)."

# This wrapper does not execute the (potentially very slow) runner itself — the score is
# read by the caller and banded into the seal verdict. Enforce only short-circuits the
# advisory text; the deterministic NO-GO is applied by /rite-seal against the recorded score.
exit 0
