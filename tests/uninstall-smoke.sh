#!/usr/bin/env bash
# uninstall-smoke.sh — install into a temp project, then uninstall, asserting that
# DevRites files are removed, empty dirs pruned, and runtime state preserved.
# Binary lifecycle coverage lives in binary-lifecycle-test.sh; this smoke test
# always passes --keep-binary so it cannot delete a developer's global binary.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
GEN="$(mktemp -d)"
trap 'rm -rf "$GEN"' EXIT
if [ -n "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  cp -R "$DEVRITES_HOST_ARTIFACT_DIR"/. "$GEN"/ \
    || { echo "  FAIL: could not copy host artifacts"; exit 1; }
else
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
fi
export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
ENGINE_BIN="$GEN/devrites-engine"
( cd "$ROOT/engine" && GOCACHE="$GEN/go-cache" CGO_ENABLED=0 go build -trimpath -o "$ENGINE_BIN" . ) >/dev/null 2>&1 \
  || { echo "  FAIL: could not build test engine"; exit 1; }

echo "== uninstall-smoke =="

main_uninstall_case() {
  local fail=0
  local T leftover
  T="$(mktemp -d)" || return 1

  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"
  mkdir -p "$T/.devrites/work/demo"; echo "phase: build" > "$T/.devrites/work/demo/state.md"
  printf 'demo\n' > "$T/.devrites/ACTIVE"

  bash "$ROOT/uninstall.sh" --target "$T" --dry-run --keep-binary >/dev/null 2>&1
  [ -f "$T/.claude/devrites.manifest" ] && ok "dry-run uninstall kept manifest" || no "dry-run removed files"

  bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 || no "uninstall exited non-zero"
  [ -e "$T/.claude/devrites.manifest" ] && no "manifest not removed" || ok "manifest removed"
  [ -e "$T/AGENTS.md" ] && no "AGENTS bridge not removed" || ok "AGENTS bridge removed"
  if [ -d "$T/.claude" ]; then
    leftover="$(find "$T/.claude" -mindepth 1 ! -path "$T/.claude/settings.json")"
    [ -z "$leftover" ] && ok ".claude pruned (seeded settings.json preserved)" \
      || no ".claude has leftover DevRites content: $leftover"
  else
    ok ".claude fully pruned"
  fi
  if [ -d "$T/.agents" ]; then
    leftover="$(find "$T/.agents" -mindepth 1)"
    [ -z "$leftover" ] && ok ".agents pruned" || no ".agents has leftover DevRites content: $leftover"
  else
    ok ".agents fully pruned"
  fi
  if [ -d "$T/.codex" ]; then
    leftover="$(find "$T/.codex" -mindepth 1)"
    [ -z "$leftover" ] && ok ".codex pruned" || no ".codex has leftover DevRites content: $leftover"
  else
    ok ".codex fully pruned"
  fi
  [ -f "$T/.devrites/ACTIVE" ] && ok "ACTIVE preserved" || no "ACTIVE wrongly removed"
  [ -f "$T/.devrites/work/demo/state.md" ] && ok "work/ feature data preserved" || no "work/ wrongly removed"
  [ -f "$T/.devrites/README.md" ] && no ".devrites/README not removed" || ok ".devrites/README removed"

  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

no_manifest_case() {
  local fail=0
  local T
  T="$(mktemp -d)" || return 1
  bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 && no "uninstall succeeded with no manifest" || ok "uninstall errors without manifest"
  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

foreign_file_case() {
  local fail=0
  local T
  T="$(mktemp -d)" || return 1
  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "foreign-file install failed"
  echo "mine" > "$T/.claude/skills/rite/USER_NOTE.txt"
  echo "mine" > "$T/.agents/skills/rite/USER_NOTE.txt"
  bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 || no "foreign-file uninstall failed"
  [ -f "$T/.claude/skills/rite/USER_NOTE.txt" ] && ok "foreign file preserved (dir not nuked)" || no "foreign file removed!"
  [ -f "$T/.agents/skills/rite/USER_NOTE.txt" ] && ok "foreign Codex skill file preserved (dir not nuked)" || no "foreign Codex skill file removed!"
  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

no_node_hooks_case() {
  local fail=0
  local T
  T="$(mktemp -d)" || return 1
  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "no-Node install failed"
  DEVRITES_ENGINE_CLI="$ENGINE_BIN" PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 \
    && ok "default Codex hooks uninstall without Node" \
    || no "default Codex hooks uninstall required Node"
  [ -e "$T/.codex/hooks.json" ] && no "DevRites-owned Codex hooks survived no-Node uninstall" || ok "DevRites-owned Codex hooks removed without Node"
  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

preexisting_merge_case() {
  local fail=0
  local T
  T="$(mktemp -d)" || return 1
  printf '# Existing AGENTS\n\nKeep this guidance.\n' > "$T/AGENTS.md"
  mkdir -p "$T/.codex"
  printf '# Existing Codex config\nmodel = "gpt-5-codex"\n' > "$T/.codex/config.toml"
  printf '{ "hooks": { "Stop": [ { "hooks": [ { "type": "command", "command": "echo user-stop" } ] } ] } }\n' > "$T/.codex/hooks.json"
  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install with pre-existing AGENTS.md failed"
  grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T/AGENTS.md" && ok "DevRites block merged into pre-existing AGENTS.md" || no "DevRites block missing before uninstall"
  grep -q '# BEGIN DEVRITES CODEX MCP' "$T/.codex/config.toml" && no "DevRites MCP block merged into pre-existing .codex/config.toml" || ok "DevRites MCP block not merged into pre-existing .codex/config.toml"
  grep -q 'devrites-engine hook stop-gate' "$T/.codex/hooks.json" && ok "DevRites hooks merged into pre-existing .codex/hooks.json" || no "DevRites hooks missing before uninstall"
  bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 || no "uninstall with merged AGENTS.md failed"
  [ -f "$T/AGENTS.md" ] && ok "pre-existing AGENTS.md preserved" || no "pre-existing AGENTS.md removed"
  grep -q 'Keep this guidance' "$T/AGENTS.md" && ok "pre-existing AGENTS.md content preserved" || no "pre-existing AGENTS.md content lost"
  grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T/AGENTS.md" && no "DevRites block survived uninstall" || ok "DevRites block removed from pre-existing AGENTS.md"
  [ -f "$T/.codex/config.toml" ] && ok "pre-existing .codex/config.toml preserved" || no "pre-existing .codex/config.toml removed"
  grep -q 'model = "gpt-5-codex"' "$T/.codex/config.toml" && ok "pre-existing .codex/config.toml content preserved" || no "pre-existing .codex/config.toml content lost"
  grep -q '# BEGIN DEVRITES CODEX MCP' "$T/.codex/config.toml" && no "DevRites MCP block survived uninstall" || ok "pre-existing .codex/config.toml has no DevRites MCP block"
  [ -f "$T/.codex/hooks.json" ] && ok "pre-existing .codex/hooks.json preserved" || no "pre-existing .codex/hooks.json removed"
  grep -q 'echo user-stop' "$T/.codex/hooks.json" && ok "pre-existing .codex/hooks.json content preserved" || no "pre-existing .codex/hooks.json content lost"
  grep -qE 'devrites-engine hook |devrites-' "$T/.codex/hooks.json" && no "DevRites hooks survived uninstall" || ok "DevRites hooks removed from pre-existing .codex/hooks.json"
  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

pids=()
main_uninstall_case & pids+=("$!")
no_manifest_case & pids+=("$!")
foreign_file_case & pids+=("$!")
no_node_hooks_case & pids+=("$!")
preexisting_merge_case & pids+=("$!")

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "uninstall-smoke: PASS" || echo "uninstall-smoke: FAIL"
exit "$fail"
