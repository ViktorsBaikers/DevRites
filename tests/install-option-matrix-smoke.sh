#!/usr/bin/env bash
# install-option-matrix-smoke.sh: independent install.sh option behavior.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

GEN=""
T="$(mktemp -d)"
cleanup() { rm -rf "$T"; [ -n "$GEN" ] && rm -rf "$GEN"; }
trap cleanup EXIT
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi

echo "== install-option-matrix-smoke =="

case_no_agents() {
  local t="$T/no-agents"; mkdir -p "$t"; fail=0
  bash "$ROOT/install.sh" --target "$t" --no-agents >/dev/null 2>&1 || no "--no-agents install failed"
  [ -d "$t/.claude/agents" ] && no "--no-agents still installed agents" || ok "--no-agents skipped agents"
  [ -d "$t/.codex/agents" ] && no "--no-agents still installed Codex agents" || ok "--no-agents skipped Codex agents"
  [ -f "$t/.claude/skills/rite-build/SKILL.md" ] && ok "--no-agents still installs skills" || no "--no-agents broke skills"
  [ -f "$t/.agents/skills/rite-build/SKILL.md" ] && ok "--no-agents still installs Codex skills" || no "--no-agents broke Codex skills"
  exit "$fail"
}

case_short_aliases() {
  local t="$T/short-aliases"; mkdir -p "$t"; fail=0
  bash "$ROOT/install.sh" --target "$t" --short-aliases=all >/dev/null 2>&1 || no "--short-aliases=all install failed"
  [ -f "$t/.claude/skills/define/SKILL.md" ] && ok "--short-aliases=all installs /define" || no "--short-aliases=all missing /define"
  [ -f "$t/.claude/skills/build/SKILL.md" ] && ok "--short-aliases=all installs /build" || no "--short-aliases=all missing /build"
  [ -f "$t/.agents/skills/define/SKILL.md" ] && ok "--short-aliases=all mirrors /define for Codex" || no "--short-aliases=all missing Codex /define"
  [ -f "$t/.agents/skills/build/SKILL.md" ] && ok "--short-aliases=all mirrors /build for Codex" || no "--short-aliases=all missing Codex /build"
  exit "$fail"
}

case_no_rules() {
  local t="$T/no-rules"; mkdir -p "$t"; fail=0
  bash "$ROOT/install.sh" --target "$t" --no-rules >/dev/null 2>&1 || no "--no-rules install failed"
  [ -d "$t/.claude/skills/devrites-lib/reference/standards" ] && ok "--no-rules is a no-op; standards ship with the devrites-lib skill" || no "--no-rules dropped the standards (should be a no-op now)"
  [ -f "$t/.claude/skills/rite-build/SKILL.md" ] && ok "--no-rules still installs skills" || no "--no-rules broke skills"
  exit "$fail"
}

case_no_codex() {
  local t="$T/no-codex"; mkdir -p "$t"; fail=0
  bash "$ROOT/install.sh" --target "$t" --no-codex >/dev/null 2>&1 || no "--no-codex install failed"
  [ -f "$t/.claude/skills/rite-build/SKILL.md" ] && ok "--no-codex still installs Claude skills" || no "--no-codex broke Claude skills"
  [ -d "$t/.agents" ] && no "--no-codex installed .agents" || ok "--no-codex skipped .agents"
  [ -d "$t/.codex" ] && no "--no-codex installed .codex" || ok "--no-codex skipped .codex"
  [ -f "$t/AGENTS.md" ] && no "--no-codex installed AGENTS.md" || ok "--no-codex skipped AGENTS.md"
  exit "$fail"
}

pids=()
case_no_agents & pids+=("$!")
case_short_aliases & pids+=("$!")
case_no_rules & pids+=("$!")
case_no_codex & pids+=("$!")
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "install-option-matrix-smoke: PASS" || echo "install-option-matrix-smoke: FAIL"
exit "$fail"
