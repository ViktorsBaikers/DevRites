#!/usr/bin/env bash
set -euo pipefail

contract_die() {
  printf '%s\n' "agent-contract host wrapper: $1" >&2
  exit 2
}

contract_require() {
  local name="$1"
  [ -n "${!name:-}" ] || contract_die "missing isolated run input"
}

contract_path_within() {
  local child="$1" parent="$2"
  case "$child/" in
    "$parent"/*) return 0 ;;
    *) return 1 ;;
  esac
}

contract_prepare_run() {
  local name
  for name in \
    HOME TMPDIR XDG_CONFIG_HOME XDG_STATE_HOME \
    DEVRITES_ROOT DEVRITES_AGENT_SCRATCH DEVRITES_RUN_ID \
    DEVRITES_CONTRACT_RUN_ROOT DEVRITES_CONTRACT_PACKET \
    DEVRITES_CONTRACT_RESULT DEVRITES_CONTRACT_HOST \
    DEVRITES_CONTRACT_HOST_VERSION_PIN DEVRITES_CONTRACT_MODEL \
    DEVRITES_CONTRACT_AUTH_FILE DEVRITES_CONTRACT_MAX_COST_USD
  do
    contract_require "$name"
  done
  for name in \
    "$HOME" "$TMPDIR" "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" \
    "$DEVRITES_ROOT" "$DEVRITES_AGENT_SCRATCH"
  do
    contract_path_within "$name" "$DEVRITES_CONTRACT_RUN_ROOT" \
      || contract_die "isolated root escaped the run directory"
  done
  contract_path_within "$DEVRITES_CONTRACT_PACKET" "$DEVRITES_AGENT_SCRATCH" \
    || contract_die "packet escaped the scratch root"
  contract_path_within "$DEVRITES_CONTRACT_RESULT" "$DEVRITES_AGENT_SCRATCH" \
    || contract_die "result escaped the scratch root"
  [ -f "$DEVRITES_CONTRACT_PACKET" ] || contract_die "packet is unavailable"
  [ -f "$DEVRITES_CONTRACT_AUTH_FILE" ] || contract_die "scoped auth is unavailable"
}

contract_read_auth() {
  local mode
  if mode="$(stat -f '%Lp' "$DEVRITES_CONTRACT_AUTH_FILE" 2>/dev/null)"; then
    :
  else
    mode="$(stat -c '%a' "$DEVRITES_CONTRACT_AUTH_FILE" 2>/dev/null)" \
      || contract_die "cannot inspect scoped auth"
  fi
  [ "$mode" = "600" ] || contract_die "scoped auth must be mode 0600"
  IFS= read -r CONTRACT_AUTH_VALUE < "$DEVRITES_CONTRACT_AUTH_FILE" \
    || contract_die "scoped auth is empty"
  [ -n "$CONTRACT_AUTH_VALUE" ] || contract_die "scoped auth is empty"
}

contract_run_fake() {
  local -a env_args=(
    "PATH=$PATH"
    "LANG=C"
    "LC_ALL=C"
    "HOME=$HOME"
    "TMPDIR=$TMPDIR"
    "XDG_CONFIG_HOME=$XDG_CONFIG_HOME"
    "XDG_STATE_HOME=$XDG_STATE_HOME"
    "PYTHONDONTWRITEBYTECODE=1"
    "DEVRITES_ROOT=$DEVRITES_ROOT"
    "DEVRITES_AGENT_SCRATCH=$DEVRITES_AGENT_SCRATCH"
    "DEVRITES_RUN_ID=$DEVRITES_RUN_ID"
    "DEVRITES_CONTRACT_RUN_ROOT=$DEVRITES_CONTRACT_RUN_ROOT"
    "DEVRITES_CONTRACT_PACKET=$DEVRITES_CONTRACT_PACKET"
    "DEVRITES_CONTRACT_RESULT=$DEVRITES_CONTRACT_RESULT"
    "DEVRITES_CONTRACT_HOST=$DEVRITES_CONTRACT_HOST"
    "DEVRITES_CONTRACT_HOST_VERSION_PIN=$DEVRITES_CONTRACT_HOST_VERSION_PIN"
    "DEVRITES_CONTRACT_MODEL=$DEVRITES_CONTRACT_MODEL"
    "DEVRITES_CONTRACT_AUTH_FILE=$DEVRITES_CONTRACT_AUTH_FILE"
    "DEVRITES_CONTRACT_MAX_COST_USD=$DEVRITES_CONTRACT_MAX_COST_USD"
    "DEVRITES_CONTRACT_FAKE_VERSION=$DEVRITES_CONTRACT_FAKE_VERSION"
  )
  if [ -n "${DEVRITES_CONTRACT_FAKE_FAULT:-}" ]; then
    env_args+=("DEVRITES_CONTRACT_FAKE_FAULT=$DEVRITES_CONTRACT_FAKE_FAULT")
  fi
  exec env -i "${env_args[@]}" python3 "$DEVRITES_CONTRACT_FAKE_HOST" run
}

contract_version() {
  local binary="$1"
  if [ -n "${DEVRITES_CONTRACT_FAKE_HOST:-}" ]; then
    contract_require DEVRITES_CONTRACT_FAKE_VERSION
    exec env -i \
      "PATH=$PATH" "LANG=C" "LC_ALL=C" \
      "DEVRITES_CONTRACT_FAKE_VERSION=$DEVRITES_CONTRACT_FAKE_VERSION" \
      python3 "$DEVRITES_CONTRACT_FAKE_HOST" version
  fi
  command -v "$binary" >/dev/null 2>&1 || contract_die "host CLI is unavailable"
  exec env -i "PATH=$PATH" "LANG=C" "LC_ALL=C" "$binary" --version
}

contract_interrupt_result() {
  env -i \
    "PATH=$PATH" "LANG=C" "LC_ALL=C" "PYTHONDONTWRITEBYTECODE=1" \
    "DEVRITES_CONTRACT_PACKET=$DEVRITES_CONTRACT_PACKET" \
    "DEVRITES_CONTRACT_RESULT=$DEVRITES_CONTRACT_RESULT" \
    "DEVRITES_CONTRACT_HOST_VERSION_PIN=$DEVRITES_CONTRACT_HOST_VERSION_PIN" \
    "DEVRITES_CONTRACT_MODEL=$DEVRITES_CONTRACT_MODEL" \
    python3 "$CONTRACT_HELPER" interrupted >/dev/null 2>&1 || true
}
