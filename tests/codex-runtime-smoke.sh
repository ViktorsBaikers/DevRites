#!/usr/bin/env bash
# codex-runtime-smoke.sh: verify a DevRites install is visible to Codex.
#
# Default mode uses `codex debug prompt-input`, which does not call the model.
# Set DEVRITES_CODEX_MODEL_SMOKE=1 to also run a real `codex exec` read-only
# session; that path requires Codex auth/network and may consume tokens.
# Set DEVRITES_CODEX_SUBAGENT_SMOKE=1 to run a real custom-subagent spawn
# assertion from the Codex JSON event stream; that path consumes more tokens.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v codex >/dev/null 2>&1 || { echo "codex-runtime-smoke: SKIP (codex CLI not found)"; exit 0; }
command -v python3 >/dev/null 2>&1 || { echo "codex-runtime-smoke: SKIP (python3 not found)"; exit 0; }

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
mkdir -p "$PROJECT" "$T/home" "$T/codex-home"

echo "== codex-runtime-smoke (target: $PROJECT) =="
bash "$ROOT/install.sh" --target "$PROJECT" >/dev/null 2>&1 || no "install failed"

(
  cd "$PROJECT" || exit 1
  HOME="$T/home" CODEX_HOME="$T/codex-home" codex debug prompt-input 'Use $rite-status to inspect DevRites.' > "$T/prompt.json" 2> "$T/prompt.err"
)
rc=$?
[ "$rc" -eq 0 ] && ok "codex debug prompt-input ran" || { no "codex debug prompt-input failed"; sed -n '1,80p' "$T/prompt.err"; }

python3 - "$T/prompt.json" <<'PY'
import pathlib, sys
s = pathlib.Path(sys.argv[1]).read_text() if pathlib.Path(sys.argv[1]).exists() else ""
checks = {
    "DevRites guidance visible": "DevRites" in s,
    "DevRites AGENTS block visible": "BEGIN DEVRITES CODEX" in s,
    "DevRites rules mirror path visible": ".agents/skills/devrites-lib/reference/standards/core.md" in s,
    "DevRites Codex agents path visible": ".codex/agents" in s,
}
for label, passed in checks.items():
    print(("ok:" if passed else "FAIL:") + " " + label)
if not all(checks.values()):
    sys.exit(1)
PY
if [ "$?" -eq 0 ]; then
  ok "Codex prompt input contains DevRites guidance"
else
  no "Codex prompt input missing DevRites guidance"
fi

[ -e "$PROJECT/.codex/mcp" ] && no "DevRites MCP directory installed" || ok "DevRites MCP directory not installed"
[ -e "$PROJECT/.codex/config.toml" ] && no "DevRites Codex MCP config installed" || ok "DevRites Codex MCP config not installed"

MODEL_HOME="${DEVRITES_CODEX_MODEL_HOME:-${HOME:-}}"
MODEL_CODEX_HOME="${DEVRITES_CODEX_MODEL_CODEX_HOME:-${CODEX_HOME:-}}"
if [ -z "$MODEL_CODEX_HOME" ] && [ -n "$MODEL_HOME" ]; then
  MODEL_CODEX_HOME="$MODEL_HOME/.codex"
fi
model_env_ready() {
  [ -n "$MODEL_HOME" ] && [ -n "$MODEL_CODEX_HOME" ] && [ -d "$MODEL_CODEX_HOME" ]
}

if [ "${DEVRITES_CODEX_MODEL_SMOKE:-0}" = "1" ]; then
  if ! model_env_ready; then
    no "model-backed codex exec requires real Codex auth/config (set DEVRITES_CODEX_MODEL_HOME and DEVRITES_CODEX_MODEL_CODEX_HOME if needed)"
  else
  (
    cd "$PROJECT" || exit 1
    HOME="$MODEL_HOME" CODEX_HOME="$MODEL_CODEX_HOME" codex exec \
      --ephemeral \
      --skip-git-repo-check \
      --dangerously-bypass-hook-trust \
      -s read-only \
      'Read AGENTS.md only. Reply with exactly: DEVRITES-CODEX-OK'
  ) > "$T/exec.out" 2> "$T/exec.err"
  rc=$?
  if [ "$rc" -eq 0 ] && grep -q 'DEVRITES-CODEX-OK' "$T/exec.out"; then
    ok "codex exec model smoke passed"
  else
    no "codex exec model smoke failed"
    sed -n '1,80p' "$T/exec.err"
    sed -n '1,80p' "$T/exec.out"
  fi
  fi
else
  ok "model-backed codex exec skipped (set DEVRITES_CODEX_MODEL_SMOKE=1 to run)"
fi

if [ "${DEVRITES_CODEX_SUBAGENT_SMOKE:-0}" = "1" ]; then
  if ! model_env_ready; then
    no "subagent smoke requires real Codex auth/config (set DEVRITES_CODEX_MODEL_HOME and DEVRITES_CODEX_MODEL_CODEX_HOME if needed)"
  else
    (
      cd "$PROJECT" || exit 1
      HOME="$MODEL_HOME" CODEX_HOME="$MODEL_CODEX_HOME" codex exec \
        --json \
        --ephemeral \
        --skip-git-repo-check \
        --dangerously-bypass-hook-trust \
        -s read-only \
        'Use the devrites-code-reviewer custom agent/subagent to inspect AGENTS.md only. Then reply exactly DEVRITES-SUBAGENT-OK if that subagent result was received. Do not edit files.'
    ) > "$T/subagent.jsonl" 2> "$T/subagent.err"
    rc=$?
    if [ "$rc" -eq 0 ] \
      && grep -q '"tool":"spawn_agent"' "$T/subagent.jsonl" \
      && grep -q '"tool":"wait"' "$T/subagent.jsonl" \
      && grep -q 'DEVRITES-SUBAGENT-OK' "$T/subagent.jsonl"; then
      ok "codex custom subagent smoke passed"
    else
      no "codex custom subagent smoke failed"
      sed -n '1,80p' "$T/subagent.err"
      sed -n '1,120p' "$T/subagent.jsonl"
    fi
  fi
else
  ok "custom-subagent codex exec skipped (set DEVRITES_CODEX_SUBAGENT_SMOKE=1 to run)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-runtime-smoke: PASS" || echo "codex-runtime-smoke: FAIL"
exit "$fail"
