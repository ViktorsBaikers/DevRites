#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
# shellcheck disable=SC1091
source "$DIR/common.sh"
CONTRACT_HELPER="$DIR/host-transport.py"

case "${1:-}" in
  version)
    contract_version codex
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
actual_version="$(env -i "PATH=$PATH" "LANG=C" "LC_ALL=C" codex --version 2>/dev/null)" \
  || contract_die "Codex version preflight failed"
[ "$actual_version" = "$DEVRITES_CONTRACT_HOST_VERSION_PIN" ] \
  || contract_die "Codex version pin changed"

role_class="$(python3 "$CONTRACT_HELPER" field role_class)"
sandbox="read-only"
if [ "$role_class" = "wright" ]; then
  sandbox="workspace-write"
fi
mkdir -p "$XDG_CONFIG_HOME/codex"

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
    "CODEX_HOME=$XDG_CONFIG_HOME/codex" \
    "OPENAI_API_KEY=$CONTRACT_AUTH_VALUE" \
    "DEVRITES_ROOT=$DEVRITES_ROOT" \
    "DEVRITES_AGENT_SCRATCH=$DEVRITES_AGENT_SCRATCH" \
    "DEVRITES_RUN_ID=$DEVRITES_RUN_ID" \
    codex exec \
      --json \
      --ephemeral \
      --strict-config \
      --ignore-user-config \
      --skip-git-repo-check \
      --dangerously-bypass-hook-trust \
      --sandbox "$sandbox" \
      --model "$DEVRITES_CONTRACT_MODEL" \
      --output-schema "$DIR/agent-result.schema.json" \
      -c shell_environment_policy.inherit=none \
      -C "$DEVRITES_ROOT" \
      - |
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
