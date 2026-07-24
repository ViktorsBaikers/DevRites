#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$DIR/common.sh"
CONTRACT_HELPER="$DIR/host-transport.py"

case "${1:-}" in
  version)
    contract_version claude
    ;;
  run)
    ;;
  *)
    contract_die "expected version or run"
    ;;
esac

contract_prepare_run
if [ -n "${DEVRITES_CONTRACT_FAKE_HOST:-}" ]; then
  contract_run_fake
fi
contract_read_auth
actual_version="$(env -i "PATH=$PATH" "LANG=C" "LC_ALL=C" claude --version 2>/dev/null)" \
  || contract_die "Claude version preflight failed"
[ "$actual_version" = "$DEVRITES_CONTRACT_HOST_VERSION_PIN" ] \
  || contract_die "Claude version pin changed"

simulate="$(python3 "$CONTRACT_HELPER" field simulate)"
role_class="$(python3 "$CONTRACT_HELPER" field role_class)"
tools="Agent,Read,Glob,Grep"
if [ "$role_class" = "wright" ]; then
  tools="$tools,Edit,Write,Bash"
fi
if [ "$simulate" = "inline-fallback" ]; then
  tools="Read,Glob,Grep"
fi
schema="$(cat "$DIR/agent-result.schema.json")"
mkdir -p "$XDG_CONFIG_HOME/claude"

trap 'contract_interrupt_result; exit 130' INT TERM
cd "$DEVRITES_ROOT"
set +e
env -i \
  "PATH=$PATH" "LANG=C" "LC_ALL=C" "PYTHONDONTWRITEBYTECODE=1" \
  "DEVRITES_CONTRACT_PACKET=$DEVRITES_CONTRACT_PACKET" \
  python3 "$CONTRACT_HELPER" render |
  env -i \
    "PATH=$PATH" "LANG=C" "LC_ALL=C" \
    "HOME=$HOME" "TMPDIR=$TMPDIR" \
    "XDG_CONFIG_HOME=$XDG_CONFIG_HOME" "XDG_STATE_HOME=$XDG_STATE_HOME" \
    "CLAUDE_CONFIG_DIR=$XDG_CONFIG_HOME/claude" \
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1" \
    "ANTHROPIC_API_KEY=$CONTRACT_AUTH_VALUE" \
    "DEVRITES_ROOT=$DEVRITES_ROOT" \
    "DEVRITES_AGENT_SCRATCH=$DEVRITES_AGENT_SCRATCH" \
    "DEVRITES_RUN_ID=$DEVRITES_RUN_ID" \
    claude -p \
      --output-format json \
      --json-schema "$schema" \
      --model "$DEVRITES_CONTRACT_MODEL" \
      --max-budget-usd "$DEVRITES_CONTRACT_MAX_COST_USD" \
      --permission-mode dontAsk \
      --tools "$tools" \
      --setting-sources project \
      --settings '{}' \
      --strict-mcp-config \
      --mcp-config '{"mcpServers":{}}' \
      --no-chrome \
      --no-session-persistence |
  env -i \
    "PATH=$PATH" "LANG=C" "LC_ALL=C" "PYTHONDONTWRITEBYTECODE=1" \
    "DEVRITES_CONTRACT_PACKET=$DEVRITES_CONTRACT_PACKET" \
    "DEVRITES_CONTRACT_RESULT=$DEVRITES_CONTRACT_RESULT" \
    "DEVRITES_CONTRACT_HOST_VERSION_PIN=$DEVRITES_CONTRACT_HOST_VERSION_PIN" \
    "DEVRITES_CONTRACT_MODEL=$DEVRITES_CONTRACT_MODEL" \
    python3 "$CONTRACT_HELPER" normalize
pipeline=("${PIPESTATUS[@]}")
set -e
host_rc="${pipeline[1]:-2}"
normalize_rc="${pipeline[2]:-2}"
[ "$normalize_rc" -eq 0 ] || exit 2
if [ "$host_rc" -eq 130 ]; then
  contract_interrupt_result
fi
exit "$host_rc"
