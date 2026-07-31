#!/usr/bin/env bash
# Install DevRites in a temporary project, then verify that uninstall removes
# managed files and empty directories while preserving runtime state. Binary
# lifecycle coverage lives in binary-lifecycle-test.sh, so this test always
# passes --keep-binary and cannot delete a developer's global binary.
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
export DEVRITES_ENGINE_CLI="$ENGINE_BIN"

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

no_node_config_case() {
  local fail=0
  local T
  T="$(mktemp -d)" || return 1
  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "no-Node install failed"
  DEVRITES_ENGINE_CLI="$ENGINE_BIN" PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 \
    && ok "default Codex permission config uninstalls without Node" \
    || no "default Codex permission config uninstall required Node"
  [ -e "$T/.codex/config.toml" ] && no "DevRites Codex permission config survived no-Node uninstall" || ok "DevRites Codex permission config removed without Node"
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
  grep -q '# BEGIN DEVRITES CODEX PERMISSIONS' "$T/.codex/config.toml" && ok "DevRites permission block merged into pre-existing .codex/config.toml" || no "DevRites permission block missing before uninstall"
  grep -q 'echo user-stop' "$T/.codex/hooks.json" && ok "installer leaves pre-existing Codex hooks untouched" || no "installer changed pre-existing Codex hooks"
  bash "$ROOT/uninstall.sh" --target "$T" --keep-binary >/dev/null 2>&1 || no "uninstall with merged AGENTS.md failed"
  [ -f "$T/AGENTS.md" ] && ok "pre-existing AGENTS.md preserved" || no "pre-existing AGENTS.md removed"
  grep -q 'Keep this guidance' "$T/AGENTS.md" && ok "pre-existing AGENTS.md content preserved" || no "pre-existing AGENTS.md content lost"
  grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T/AGENTS.md" && no "DevRites block survived uninstall" || ok "DevRites block removed from pre-existing AGENTS.md"
  [ -f "$T/.codex/config.toml" ] && ok "pre-existing .codex/config.toml preserved" || no "pre-existing .codex/config.toml removed"
  grep -q 'model = "gpt-5-codex"' "$T/.codex/config.toml" && ok "pre-existing .codex/config.toml content preserved" || no "pre-existing .codex/config.toml content lost"
  grep -q '# BEGIN DEVRITES CODEX PERMISSIONS' "$T/.codex/config.toml" && no "DevRites permission block survived uninstall" || ok "pre-existing .codex/config.toml has no DevRites permission block"
  [ -f "$T/.codex/hooks.json" ] && ok "pre-existing .codex/hooks.json preserved" || no "pre-existing .codex/hooks.json removed"
  grep -q 'echo user-stop' "$T/.codex/hooks.json" && ok "pre-existing .codex/hooks.json content preserved" || no "pre-existing .codex/hooks.json content lost"
  grep -q 'devrites-engine hook' "$T/.codex/hooks.json" && no "installer injected DevRites hooks into user file" || ok "pre-existing Codex hooks remain user-owned"
  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

customized_managed_case() {
  local fail=0
  local T out managed
  T="$(mktemp -d)" || return 1
  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "customized-file install failed"
  managed="$T/.claude/skills/rite/SKILL.md"
  printf 'local customization\n' > "$managed"
  out="$(bash "$ROOT/uninstall.sh" --target "$T" --keep-binary 2>&1)" \
    && no "default uninstall removed customized managed file" \
    || ok "default uninstall aborts on customized managed file"
  [ -f "$managed" ] && ok "default uninstall preserved customized managed file" || no "default uninstall removed customization"
  printf '%s' "$out" | grep -q -- 'rerun with --force' && ok "uninstall gives force remediation" || no "uninstall missing force remediation"
  out="$(bash "$ROOT/uninstall.sh" --target "$T" --force --dry-run --keep-binary 2>&1)" || no "forced uninstall dry-run failed"
  printf '%s' "$out" | grep -q '\[remove(force-customized)\] .claude/skills/rite/SKILL.md' \
    && ok "forced uninstall dry-run predicts customized removal" || no "forced uninstall dry-run output inaccurate"
  [ -f "$managed" ] && ok "forced uninstall dry-run wrote nothing" || no "forced uninstall dry-run removed customization"
  bash "$ROOT/uninstall.sh" --target "$T" --force --keep-binary >/dev/null 2>&1 || no "forced uninstall failed"
  [ ! -e "$managed" ] && ok "forced uninstall removed customized managed file" || no "forced uninstall kept customization"
  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

pids=()
main_uninstall_case & pids+=("$!")
no_manifest_case & pids+=("$!")
foreign_file_case & pids+=("$!")
no_node_config_case & pids+=("$!")
preexisting_merge_case & pids+=("$!")
customized_managed_case & pids+=("$!")

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "uninstall-smoke: PASS" || echo "uninstall-smoke: FAIL"
exit "$fail"
