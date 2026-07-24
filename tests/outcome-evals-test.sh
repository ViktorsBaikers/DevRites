#!/usr/bin/env bash

set -euo pipefail

if [ "${DEVRITES_OUTCOME_DISCOVERY_ONLY:-0}" = "1" ]; then
  echo "outcome-evals discovery sentinel"
  exit 0
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

output="$(bash "$ROOT/scripts/run-outcome-evals.sh")"
printf '%s\n' "$output"
grep -Fq "canonical parity + 15 isolated negative rows" <<<"$output"
grep -Fq "reason=DRV-GATE-SEAL-MISSING" <<<"$output"
grep -Fq "readiness=coverage-not-clear(code=6)" <<<"$output"
grep -Fq "readiness=engineering-not-ready(code=7)" <<<"$output"
if grep -Fq "stale PATH shim executed" <<<"$output"; then
  echo "outcome runner used a PATH engine instead of its exact build" >&2
  exit 1
fi

mkdir -p "$tmp/host-artifacts"
discovery="$(
  DEVRITES_OUTCOME_DISCOVERY_ONLY=1 \
  DEVRITES_HOST_ARTIFACT_DIR="$tmp/host-artifacts" \
  DEVRITES_ENGINE_CLI="$ROOT/scripts/run-outcome-evals.sh" \
    node "$ROOT/scripts/run-tests.mjs" --serial outcome-evals-test
)"
grep -Fq "outcome-evals discovery sentinel" <<<"$discovery"
grep -Fq "PASS: tests/outcome-evals-test.sh" <<<"$discovery"

echo "outcome eval regression: PASS"
