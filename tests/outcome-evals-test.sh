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
grep -Fq "Outcome evals passed: native boundary + 15 isolated final-outcome negatives + candidate/readiness content binding + removed-command rejections + 2 adversarial fixtures." <<<"$output"
grep -Fq "PASS content_identity     unchanged-touch=pass restored-mtime-byte-change=blocked" <<<"$output"
grep -Fq "all 19 retired top-level commands are unknown (no aliases)" <<<"$output"
grep -Fq "wrong_ac_id" <<<"$output"
grep -Fq "PASS: unauthorized-spec-drift grader NO-GO + readiness AC map" <<<"$output"
grep -Fq "PASS: out-of-scope-writer-diff extra candidate path is not in tasks.md" <<<"$output"

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
