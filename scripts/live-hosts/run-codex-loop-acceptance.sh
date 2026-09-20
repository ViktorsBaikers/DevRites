#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd -P)"

fail() {
  printf 'codex-loop-acceptance: FAIL: %s\n' "$1" >&2
  exit 1
}

require_env() {
  local name="$1"
  [ -n "${!name:-}" ] || fail "set $name explicitly"
}

for name in \
  DEVRITES_CODEX_ACCEPTANCE_MODEL \
  DEVRITES_CODEX_ACCEPTANCE_HOME \
  DEVRITES_CODEX_ACCEPTANCE_REPORT \
  DEVRITES_CODEX_ACCEPTANCE_TRIALS \
  DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS \
  DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS; do
  require_env "$name"
done

case "$DEVRITES_CODEX_ACCEPTANCE_TRIALS" in
  *[!0-9]*|'') fail "DEVRITES_CODEX_ACCEPTANCE_TRIALS must be a positive integer" ;;
esac
case "$DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS" in
  *[!0-9]*|'') fail "DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS must be a positive integer" ;;
esac
case "$DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS" in
  *[!0-9]*|'') fail "DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS must be a positive integer" ;;
esac
[ "$DEVRITES_CODEX_ACCEPTANCE_TRIALS" -gt 0 ] || fail "DEVRITES_CODEX_ACCEPTANCE_TRIALS must be positive"
[ "$DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS" -gt 0 ] || fail "DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS must be positive"
[ "$DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS" -gt 0 ] || fail "DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS must be positive"

calls=$((16 * DEVRITES_CODEX_ACCEPTANCE_TRIALS))
[ "$calls" -le "$DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS" ] \
  || fail "configured $calls calls exceed DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS=$DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS"
[ -d "$DEVRITES_CODEX_ACCEPTANCE_HOME" ] \
  || fail "DEVRITES_CODEX_ACCEPTANCE_HOME must be an authenticated Codex home directory"
command -v python3 >/dev/null 2>&1 || fail "python3 is unavailable"
command -v codex >/dev/null 2>&1 || fail "codex CLI is unavailable"

exec python3 "$ROOT/scripts/live-hosts/run-loop-evals.py" \
  --host codex \
  --model "$DEVRITES_CODEX_ACCEPTANCE_MODEL" \
  --arm "${DEVRITES_CODEX_ACCEPTANCE_ARM:-candidate}" \
  --trials "$DEVRITES_CODEX_ACCEPTANCE_TRIALS" \
  --max-seconds "$DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS" \
  --codex-home "$DEVRITES_CODEX_ACCEPTANCE_HOME" \
  --report "$DEVRITES_CODEX_ACCEPTANCE_REPORT"
