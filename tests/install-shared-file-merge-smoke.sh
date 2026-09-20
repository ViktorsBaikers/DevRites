#!/usr/bin/env bash
# install-shared-file-merge-smoke.sh: shared Codex files merge, prune, and preserve user content.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

GEN=""
cleanup() { [ -n "$GEN" ] && rm -rf "$GEN"; }
trap cleanup EXIT
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi

echo "== install-shared-file-merge-smoke =="

shared_reinstall_and_prune_case() {
  local fail=0
  local T out
  T="$(mktemp -d)" || return 1

  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"
  printf '\nUser project guidance.\n' >> "$T/AGENTS.md"
  mkdir -p "$T/.codex"
  printf 'model = "gpt-5-codex"\n' > "$T/.codex/config.toml"
  printf '{ "hooks": { "Stop": [ { "hooks": [ { "type": "command", "command": "echo user-stop" } ] } ] } }\n' > "$T/.codex/hooks.json"
  bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "reinstall after shared Codex user edits failed"
  grep -q 'User project guidance' "$T/AGENTS.md" && ok "AGENTS user guidance survives reinstall" || no "AGENTS user guidance lost on reinstall"
  grep -q 'model = "gpt-5-codex"' "$T/.codex/config.toml" && ok "Codex config user setting survives reinstall" || no "Codex config user setting lost on reinstall"
  grep -q 'echo user-stop' "$T/.codex/hooks.json" && ok "Codex user hook survives reinstall" || no "Codex user hook lost on reinstall"
  [ "$(grep -c '<!-- BEGIN DEVRITES CODEX -->' "$T/AGENTS.md")" -eq 1 ] && ok "AGENTS block remains single after shared-file reinstall" || no "AGENTS block duplicated after shared-file reinstall"
  [ "$(grep -c '# BEGIN DEVRITES CODEX PERMISSIONS' "$T/.codex/config.toml")" -eq 1 ] && ok "Codex permission block remains single after shared-file reinstall" || no "Codex permission block missing or duplicated"
  grep -q 'devrites-engine hook' "$T/.codex/hooks.json" && no "installer injected DevRites hooks" || ok "Codex user hooks remain untouched"

  mkdir -p "$T/.claude/skills/_gone/reference"
  echo stale > "$T/.claude/skills/_gone/reference/dropped.md"
  printf '.claude/skills/_gone/reference/dropped.md\n' >> "$T/.claude/devrites.manifest"
  mkdir -p "$T/.agents/skills/_gone"
  echo stale > "$T/.agents/skills/_gone/SKILL.md"
  printf '.agents/skills/_gone/SKILL.md\n' >> "$T/.claude/devrites.manifest"
  echo keep > "$T/.devrites/should-survive"
  printf '.devrites/should-survive\n' >> "$T/.claude/devrites.manifest"
  out="$(bash "$ROOT/install.sh" --target "$T" --force 2>&1)" || no "force reinstall for prune failed"
  [ -f "$T/.claude/skills/_gone/reference/dropped.md" ] && no "dropped managed file not pruned" || ok "dropped managed file pruned"
  [ -d "$T/.claude/skills/_gone" ] && no "emptied dir left behind after prune" || ok "emptied dir tidied after prune"
  [ -f "$T/.agents/skills/_gone/SKILL.md" ] && no "dropped Codex mirror file not pruned" || ok "dropped Codex mirror file pruned"
  [ -d "$T/.agents/skills/_gone" ] && no "emptied Codex mirror dir left behind after prune" || ok "emptied Codex mirror dir tidied after prune"
  [ -f "$T/.devrites/should-survive" ] && ok ".devrites/ runtime state preserved across prune" || no ".devrites/ state pruned - DANGER"
  echo "$out" | grep -q 'pruned: 2' && ok "prune count reported (2 managed; .devrites entry skipped)" || no "prune count not reported"

  rm -rf "$T"
  [ "$fail" -eq 0 ]
}

existing_file_merge_no_codex_case() {
  local fail=0
  local T8
  T8="$(mktemp -d)" || return 1

  printf '# Project Agent Notes\n\nKeep this user guidance.\n' > "$T8/AGENTS.md"
  mkdir -p "$T8/.codex"
  printf '# Existing Codex config\nmodel = "gpt-5-codex"\n' > "$T8/.codex/config.toml"
  printf '{ "hooks": { "Stop": [ { "hooks": [ { "type": "command", "command": "echo user-stop" } ] } ] } }\n' > "$T8/.codex/hooks.json"
  bash "$ROOT/install.sh" --target "$T8" >/dev/null 2>&1 || no "install with existing AGENTS.md failed"
  grep -q 'Keep this user guidance' "$T8/AGENTS.md" && ok "existing AGENTS.md content preserved" || no "existing AGENTS.md content lost"
  grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T8/AGENTS.md" && ok "DevRites block merged into existing AGENTS.md" || no "DevRites block not merged"
  grep -q '^AGENTS.md$' "$T8/.claude/devrites.manifest" && no "existing AGENTS.md wrongly file-managed" || ok "existing AGENTS.md not file-managed"
  grep -q '^\.claude/devrites\.agents-merge$' "$T8/.claude/devrites.manifest" && ok "AGENTS merge marker managed" || no "AGENTS merge marker missing"
  grep -q 'model = "gpt-5-codex"' "$T8/.codex/config.toml" && ok "existing .codex/config.toml content preserved" || no "existing .codex/config.toml content lost"
  grep -q '# BEGIN DEVRITES CODEX PERMISSIONS' "$T8/.codex/config.toml" && ok "DevRites permission block merged into .codex/config.toml" || no "DevRites permission block missing"
  grep -q '^\.codex/config.toml$' "$T8/.claude/devrites.manifest" && no "existing .codex/config.toml wrongly file-managed" || ok "existing .codex/config.toml not file-managed"
  grep -q '^\.claude/devrites\.codex-config-merge$' "$T8/.claude/devrites.manifest" && ok "Codex permission config merge marker managed" || no "Codex permission config merge marker missing"
  grep -q 'echo user-stop' "$T8/.codex/hooks.json" && ok "existing .codex/hooks.json content preserved" || no "existing .codex/hooks.json content lost"
  grep -q 'devrites-engine hook' "$T8/.codex/hooks.json" && no "installer injected DevRites hooks into .codex/hooks.json" || ok "existing .codex/hooks.json remains user-owned"
  grep -q '^\.codex/hooks.json$' "$T8/.claude/devrites.manifest" && no "existing .codex/hooks.json wrongly file-managed" || ok "existing .codex/hooks.json not file-managed"
  grep -q '^\.claude/devrites\.codex-hooks-merge$' "$T8/.claude/devrites.manifest" && no "obsolete Codex hooks merge marker managed" || ok "obsolete Codex hooks merge marker absent"
  bash "$ROOT/install.sh" --target "$T8" >/dev/null 2>&1 || no "reinstall with existing AGENTS.md failed"
  [ "$(grep -c '<!-- BEGIN DEVRITES CODEX -->' "$T8/AGENTS.md")" -eq 1 ] && ok "AGENTS merge is idempotent" || no "AGENTS merge duplicated block"
  [ "$(grep -c '# BEGIN DEVRITES CODEX PERMISSIONS' "$T8/.codex/config.toml")" -eq 1 ] && ok "Codex permission merge is idempotent" || no "Codex permission block duplicated"
  bash "$ROOT/install.sh" --target "$T8" --no-codex --force >/dev/null 2>&1 || no "reinstall --no-codex with merged Codex blocks failed"
  grep -q 'Keep this user guidance' "$T8/AGENTS.md" && ok "--no-codex prune preserved existing AGENTS.md content" || no "--no-codex prune lost existing AGENTS.md content"
  grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T8/AGENTS.md" && no "--no-codex prune left DevRites AGENTS block" || ok "--no-codex prune stripped DevRites AGENTS block"
  grep -q 'model = "gpt-5-codex"' "$T8/.codex/config.toml" && ok "--no-codex prune preserved existing Codex config content" || no "--no-codex prune lost existing Codex config content"
  grep -q '# BEGIN DEVRITES CODEX PERMISSIONS' "$T8/.codex/config.toml" && no "--no-codex prune left DevRites permission block" || ok "--no-codex prune removed DevRites permission block"
  grep -q 'echo user-stop' "$T8/.codex/hooks.json" && ok "--no-codex prune preserved existing Codex hooks content" || no "--no-codex prune lost existing Codex hooks content"
  grep -q 'devrites-engine hook' "$T8/.codex/hooks.json" && no "--no-codex prune found injected DevRites hooks" || ok "--no-codex prune leaves user hooks untouched"
  [ -e "$T8/.claude/devrites.agents-merge" ] && no "--no-codex prune left AGENTS merge marker" || ok "--no-codex prune removed AGENTS merge marker"
  [ -e "$T8/.claude/devrites.codex-config-merge" ] && no "--no-codex prune left Codex permission config merge marker" || ok "--no-codex prune removed Codex permission config merge marker"
  [ -e "$T8/.claude/devrites.codex-hooks-merge" ] && no "--no-codex prune found obsolete Codex hooks marker" || ok "--no-codex prune has no Codex hooks marker"
  [ -e "$T8/.agents/skills/rite/SKILL.md" ] && no "--no-codex prune left Codex skill mirror" || ok "--no-codex prune removed Codex skill mirror"
  [ -e "$T8/.codex/mcp" ] && no "--no-codex prune left DevRites MCP directory" || ok "--no-codex prune has no DevRites MCP directory"

  rm -rf "$T8"
  [ "$fail" -eq 0 ]
}

pids=()
shared_reinstall_and_prune_case & pids+=("$!")
existing_file_merge_no_codex_case & pids+=("$!")

for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "install-shared-file-merge-smoke: PASS" || echo "install-shared-file-merge-smoke: FAIL"
exit "$fail"
