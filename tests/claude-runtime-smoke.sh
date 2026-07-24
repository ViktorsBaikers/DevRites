#!/usr/bin/env bash
# Check the installed Claude surface without invoking a model.
# Set DEVRITES_CLAUDE_MODEL_SMOKE=1 to run a budgeted, isolated live check.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
# shellcheck disable=SC1091
source "$ROOT/tests/runtime-smoke-lib.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v claude >/dev/null 2>&1 || {
  echo "claude-runtime-smoke: SKIP (claude CLI not found)"
  exit 0
}

T="$(mktemp -d)"
GEN=""
trap 'rm -rf "$T"; [ -n "$GEN" ] && rm -rf "$GEN"' EXIT
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi
PROJECT="$T/project"
mkdir -p "$PROJECT" "$T/home" "$T/claude-config"

echo "== claude-runtime-smoke (target: isolated temp project) =="
bash "$ROOT/install.sh" --target "$PROJECT" >/dev/null 2>&1 || no "install failed"

if [ -f "$PROJECT/.claude/agents/devrites-code-reviewer.md" ]; then
  ok "Claude reviewer role installed"
else
  no "Claude reviewer role missing"
fi
if [ -f "$PROJECT/.claude/agents/devrites-slice-wright.md" ]; then
  ok "Claude wright role installed"
else
  no "Claude wright role missing"
fi
if [ -f "$PROJECT/.claude/skills/rite-status/SKILL.md" ]; then
  ok "Claude skill installed"
else
  no "Claude skill missing"
fi

version="$(
  cd "$PROJECT" &&
    env -i \
      "PATH=$PATH" "LANG=C" "LC_ALL=C" \
      "HOME=$T/home" "CLAUDE_CONFIG_DIR=$T/claude-config" \
      claude --safe-mode --version 2>/dev/null
)"
if [ -n "$version" ]; then
  ok "Claude CLI starts with isolated config"
else
  no "Claude CLI isolated startup failed"
fi

if [ "${DEVRITES_CLAUDE_MODEL_SMOKE:-0}" = "1" ]; then
  AUTH_FILE="${DEVRITES_CLAUDE_API_KEY_FILE:-}"
  MODEL="${DEVRITES_CLAUDE_MODEL:-}"
  MAX_COST="${DEVRITES_CLAUDE_MAX_COST_USD:-}"
  if ! runtime_secure_auth_file_ready "$AUTH_FILE" || [ -z "$MODEL" ] || [ -z "$MAX_COST" ]; then
    no "live Claude smoke requires an explicit 0600 API-key file, pinned model, and max cost"
  else
    IFS= read -r API_KEY < "$AUTH_FILE"
    (
      cd "$PROJECT" || exit 1
      env -i \
        "PATH=$PATH" "LANG=C" "LC_ALL=C" \
        "HOME=$T/home" "CLAUDE_CONFIG_DIR=$T/claude-config" \
        "ANTHROPIC_API_KEY=$API_KEY" \
        claude --bare -p \
          --model "$MODEL" \
          --max-budget-usd "$MAX_COST" \
          --permission-mode plan \
          --tools "" \
          --no-session-persistence \
          'Reply exactly DEVRITES-CLAUDE-OK'
    ) > "$T/live.out" 2> "$T/live.err"
    rc=$?
    if [ "$rc" -eq 0 ] && grep -Fq "DEVRITES-CLAUDE-OK" "$T/live.out"; then
      ok "isolated live Claude smoke passed"
    else
      no "isolated live Claude smoke failed"
    fi
  fi
else
  ok "model-backed Claude smoke skipped (set DEVRITES_CLAUDE_MODEL_SMOKE=1 to run)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "claude-runtime-smoke: PASS" || echo "claude-runtime-smoke: FAIL"
exit "$fail"
