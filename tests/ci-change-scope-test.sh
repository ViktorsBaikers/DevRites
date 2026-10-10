#!/usr/bin/env bash
# Path-scope outputs for required-check-safe CI filtering.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SH="$ROOT/scripts/ci-change-scope.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

# GitHub Actions exports GITHUB_OUTPUT for every step. The scope script
# appends key=value to that file when it is set, so capturing stdout would
# miss the real contract and would write into the runner's output file.
# Point the script at a private OUTPUT so this test never mutates runner output.
OUTPUT="$(mktemp)"
NOREMOTE="$(mktemp -d)"
git -C "$NOREMOTE" init -q
trap 'rm -rf "$OUTPUT" "$NOREMOTE"' EXIT

scope() {
  : >"$OUTPUT"
  GITHUB_OUTPUT="$OUTPUT" GITHUB_EVENT_NAME=pull_request DEVRITES_CI_CHANGED_PATHS="$1" bash "$SH" || return
  cat "$OUTPUT"
}

expect() {
  local paths="$1" key="$2" want="$3"
  local out
  out="$(scope "$paths")" || { no "script failed for [$paths]"; return; }
  printf '%s\n' "$out" | grep -qx "${key}=${want}" && ok "$key=$want for [$paths]" || {
    no "expected $key=$want for [$paths]; got:"
    printf '%s\n' "$out"
  }
}

echo "== ci-change-scope-test =="

# Push/default (no PR): full matrix.
: >"$OUTPUT"
GITHUB_OUTPUT="$OUTPUT" GITHUB_EVENT_NAME=push bash "$SH" || no "script failed for push"
out="$(cat "$OUTPUT")"
printf '%s\n' "$out" | grep -qx 'run_tests=true' && ok "push keeps run_tests=true" || no "push should run tests"

expect $'engine/main.go\nengine/go.mod' run_tests true
expect $'engine/main.go\nengine/go.mod' run_engine true
expect $'engine/main.go\nengine/go.mod' run_full true
expect $'engine/main.go\ntests/eval-coverage-ledger-test.sh' run_tests true
expect $'README.md\ndocs/release.md' run_full false
expect $'README.md\ndocs/release.md' run_tests false
# Docs-only PRs skip the shell test shards but still run the validate job
# (run_docs=true), which checks README, CHANGELOG and docs/.
expect $'README.md' run_docs true
expect $'CHANGELOG.md' run_docs true
expect $'CONTRIBUTING.md' run_docs true
expect $'docs/release.md' run_docs true
expect $'engine/main.go\nREADME.md' run_docs true
expect $'engine/main.go' run_docs false
expect $'pack/.claude/skills/rite-build/SKILL.md' run_docs false
# Engine tests read docs/engine/, so those edits must run the engine job and
# the shell suite even though other docs/ edits skip both.
expect $'docs/engine/commands.md' run_engine true
expect $'docs/engine/commands.md' run_full true
expect $'docs/engine/commands.md' run_tests true
expect $'docs/release.md' run_engine false
expect $'pack/.claude/skills/rite-build/SKILL.md' run_tests true
expect $'pack/.claude/skills/rite-build/SKILL.md' run_engine false
expect $'pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md' run_engine true
expect $'pack/.claude/skills/rite-spec/reference/spec-template.md' run_engine true
expect $'bin/devrites.mjs' run_engine true
expect $'pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md' run_pack_evals true
expect $'.github/workflows/ci.yml' run_tests true
expect $'engine/main.go\n.github/workflows/ci.yml' run_tests true

# Dependency advisory gates run only when a PR touches dependency inputs, so a
# newly published advisory never fails an unrelated PR (deps-scan.yml covers it).
printf '%s\n' "$out" | grep -qx 'run_deps=true' && ok "push keeps run_deps=true" || no "push should run dependency gates"
expect $'pack/.claude/skills/rite-build/SKILL.md' run_deps false
expect $'engine/main.go' run_deps false
expect $'package-lock.json' run_deps true
expect $'engine/go.sum' run_deps true
expect $'osv-scanner.toml' run_deps true
expect $'scripts/npm-audit-exceptions.json' run_deps true
expect $'.github/workflows/ci.yml' run_deps false

# The validate job must run whenever run_docs is true, so README/CHANGELOG/
# CONTRIBUTING/docs/ edits get every validate.sh check, not a reduced subset.
CI_YML="$ROOT/.github/workflows/ci.yml"
validate_if="$(awk '/^  validate:$/{j=1;next} j&&/^  [a-z-]+:$/{exit} j&&/^    if:/{print}' "$CI_YML")"
case "$validate_if" in
  *outputs.run_docs*) ok "validate job runs when run_docs is true" ;;
  *) no "validate job should run when run_docs is true; got: $validate_if" ;;
esac
grep -q '^  docs:$' "$CI_YML" && no "a separate docs job would skip validate.sh checks" || ok "no separate docs job"

# Uncomputable PR file list keeps the full matrix. Missing merge-base used
# to look like "no engine files" and skip gosec until after merge to main.
: >"$OUTPUT"
# Run in a throwaway repo with no origin so the fetch fails locally and never
# touches this checkout's .git or the network.
(cd "$NOREMOTE" && GITHUB_OUTPUT="$OUTPUT" GITHUB_EVENT_NAME=pull_request GITHUB_BASE_REF=no-such-base \
  env -u DEVRITES_CI_CHANGED_PATHS bash "$SH") || no "script failed when the PR file list is unknown"
out="$(cat "$OUTPUT")"
printf '%s\n' "$out" | grep -qx 'run_engine=true' && ok "unknown PR file list keeps run_engine=true" || {
  no "unknown PR file list should keep run_engine=true; got:"
  printf '%s\n' "$out"
}
printf '%s\n' "$out" | grep -qx 'run_full=true' && ok "unknown PR file list keeps run_full=true" || {
  no "unknown PR file list should keep run_full=true; got:"
  printf '%s\n' "$out"
}
printf '%s\n' "$out" | grep -qx 'run_docs=true' && ok "unknown PR file list keeps run_docs=true" || {
  no "unknown PR file list should keep run_docs=true; got:"
  printf '%s\n' "$out"
}
printf '%s\n' "$out" | grep -qx 'run_deps=true' && ok "unknown PR file list keeps run_deps=true" || {
  no "unknown PR file list should keep run_deps=true; got:"
  printf '%s\n' "$out"
}

echo ""
[ "$fail" -eq 0 ] && echo "ci-change-scope-test: PASS" || echo "ci-change-scope-test: FAIL"
exit "$fail"
