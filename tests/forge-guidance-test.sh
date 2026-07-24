#!/usr/bin/env bash
# shellcheck disable=SC2016 # literal Markdown/backtick/$BASE assertions
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
FORGE="$ROOT/pack/.claude/skills/rite-build/reference/forge.md"
PHASE="$ROOT/pack/.claude/skills/rite-build/reference/phase-contract.md"
SCHEMA="$ROOT/pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md"
JUDGE="$ROOT/pack/.claude/agents/devrites-forge-judge.md"
ENGINE_FORGE="$ROOT/engine/internal/forge/forge.go"

require() {
  grep -Fq -- "$2" "$1" || {
    printf 'FAIL: %s missing %s\n' "$1" "$2" >&2
    exit 1
  }
}

for command in \
  "forge plan" \
  "forge process-token" \
  "forge record" \
  "forge extract" \
  "forge merge" \
  "forge cleanup" \
  "forge reap"; do
  require "$FORGE" "$command"
done

for name in \
  DEVRITES_FORGE_RUN_ID \
  DEVRITES_FORGE_CANDIDATE \
  DEVRITES_FORGE_WORKER_ID \
  DEVRITES_FORGE_WORKER_PID \
  DEVRITES_FORGE_PROCESS_START; do
  require "$FORGE" "$name"
done

for forbidden in "git worktree" "git branch" "cherry-pick" "|| true"; do
  if grep -Fq "$forbidden" "$FORGE"; then
    printf 'FAIL: Forge guidance owns Git mechanics directly: %s\n' "$forbidden" >&2
    exit 1
  fi
done

line_of() {
  awk -v needle="$2" 'index($0, needle) { print NR; exit }' "$1"
}

plan_line="$(line_of "$FORGE" 'devrites-engine forge plan')"
snapshot_line="$(line_of "$FORGE" 'devrites-engine reconcile snapshot')"
verify_line="$(line_of "$FORGE" 'verification verified')"
cleanup_line="$(line_of "$FORGE" 'devrites-engine forge cleanup')"
report_line="$(line_of "$FORGE" 'Now write `.devrites/work/<slug>/forge-report.md`')"

test "$plan_line" -lt "$snapshot_line"
test "$verify_line" -lt "$cleanup_line"
test "$cleanup_line" -lt "$report_line"

require "$PHASE" 'run `forge plan` before reconciliation'
require "$PHASE" 'manifest-only cleanup'
require "$FORGE" '--worker-binding=manifest-env-v1'
require "$FORGE" 'never reproduce the token algorithm in'
require "$ENGINE_FORGE" 'const WorkerBindingManifestEnvV1 = "manifest-env-v1"'
require "$ENGINE_FORGE" 'case "process-token":'
require "$SCHEMA" 'Forge strategies: <A=<complete approach>'
require "$SCHEMA" 'Forge scorecard: <acceptance=AC-### list'
require "$JUDGE" 'validated `devrites-forge/v1` manifest path'
require "$JUDGE" '"<candidate initial_base>" "<candidate commit>"'
require "$JUDGE" 'Runner-up notes:'

for inferred in 'forge/<slug>' '$BASE' 'Graft from runner-up'; do
  if grep -Fq "$inferred" "$JUDGE"; then
    printf 'FAIL: Forge judge still infers mutable candidate identity: %s\n' "$inferred" >&2
    exit 1
  fi
done

echo "ok: Forge guidance delegates Git ownership to the engine and preserves plan/snapshot/verify/cleanup/report order"
