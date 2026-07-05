#!/usr/bin/env bash
# uninstall-smoke.sh — install into a temp project, then uninstall, asserting that
# DevRites files are removed, empty dirs pruned, and runtime state preserved.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== uninstall-smoke (target: $T) =="
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"
# simulate in-progress feature data that must be preserved
mkdir -p "$T/.devrites/work/demo"; echo "phase: build" > "$T/.devrites/work/demo/state.md"
printf 'demo\n' > "$T/.devrites/ACTIVE"

# dry-run uninstall changes nothing
bash "$ROOT/uninstall.sh" --target "$T" --dry-run >/dev/null 2>&1
[ -f "$T/.claude/devrites.manifest" ] && ok "dry-run uninstall kept manifest" || no "dry-run removed files"

# real uninstall
bash "$ROOT/uninstall.sh" --target "$T" >/dev/null 2>&1 || no "uninstall exited non-zero"
[ -e "$T/.claude/devrites.manifest" ] && no "manifest not removed" || ok "manifest removed"
[ -e "$T/AGENTS.md" ] && no "AGENTS bridge not removed" || ok "AGENTS bridge removed"
# .claude is pruned of all manifest-managed DevRites content. The seeded
# settings.json is intentionally NOT manifested (install.sh seeds it once and
# preserves it on uninstall/update so the user's hooks survive), so tolerate it
# as the only legitimate survivor — anything else left is a prune leak.
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
# runtime state preserved
[ -f "$T/.devrites/ACTIVE" ] && ok "ACTIVE preserved" || no "ACTIVE wrongly removed"
[ -f "$T/.devrites/work/demo/state.md" ] && ok "work/ feature data preserved" || no "work/ wrongly removed"
# scaffolding README removed (manifest-managed)
[ -f "$T/.devrites/README.md" ] && no ".devrites/README not removed" || ok ".devrites/README removed"

# uninstall with no manifest must error cleanly
T2="$(mktemp -d)"
bash "$ROOT/uninstall.sh" --target "$T2" >/dev/null 2>&1 && no "uninstall succeeded with no manifest" || ok "uninstall errors without manifest"
rm -rf "$T2"

# foreign file is never removed
T3="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T3" >/dev/null 2>&1
echo "mine" > "$T3/.claude/skills/rite/USER_NOTE.txt"
echo "mine" > "$T3/.agents/skills/rite/USER_NOTE.txt"
bash "$ROOT/uninstall.sh" --target "$T3" >/dev/null 2>&1
[ -f "$T3/.claude/skills/rite/USER_NOTE.txt" ] && ok "foreign file preserved (dir not nuked)" || no "foreign file removed!"
[ -f "$T3/.agents/skills/rite/USER_NOTE.txt" ] && ok "foreign Codex skill file preserved (dir not nuked)" || no "foreign Codex skill file removed!"
rm -rf "$T3"

# default DevRites-owned Codex hooks uninstall without Node
T5="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T5" >/dev/null 2>&1
PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" bash "$ROOT/uninstall.sh" --target "$T5" >/dev/null 2>&1 \
  && ok "default Codex hooks uninstall without Node" \
  || no "default Codex hooks uninstall required Node"
[ -e "$T5/.codex/hooks.json" ] && no "DevRites-owned Codex hooks survived no-Node uninstall" || ok "DevRites-owned Codex hooks removed without Node"
rm -rf "$T5"

# pre-existing AGENTS.md keeps user content; uninstall removes only the DevRites block
T4="$(mktemp -d)"
printf '# Existing AGENTS\n\nKeep this guidance.\n' > "$T4/AGENTS.md"
mkdir -p "$T4/.codex"
printf '# Existing Codex config\nmodel = "gpt-5-codex"\n' > "$T4/.codex/config.toml"
printf '{ "hooks": { "Stop": [ { "hooks": [ { "type": "command", "command": "echo user-stop" } ] } ] } }\n' > "$T4/.codex/hooks.json"
bash "$ROOT/install.sh" --target "$T4" >/dev/null 2>&1 || no "install with pre-existing AGENTS.md failed"
grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T4/AGENTS.md" && ok "DevRites block merged into pre-existing AGENTS.md" || no "DevRites block missing before uninstall"
grep -q '# BEGIN DEVRITES CODEX MCP' "$T4/.codex/config.toml" && ok "DevRites MCP block merged into pre-existing .codex/config.toml" || no "DevRites MCP block missing before uninstall"
grep -q 'devrites-stop-gate.sh' "$T4/.codex/hooks.json" && ok "DevRites hooks merged into pre-existing .codex/hooks.json" || no "DevRites hooks missing before uninstall"
bash "$ROOT/uninstall.sh" --target "$T4" >/dev/null 2>&1 || no "uninstall with merged AGENTS.md failed"
[ -f "$T4/AGENTS.md" ] && ok "pre-existing AGENTS.md preserved" || no "pre-existing AGENTS.md removed"
grep -q 'Keep this guidance' "$T4/AGENTS.md" && ok "pre-existing AGENTS.md content preserved" || no "pre-existing AGENTS.md content lost"
grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T4/AGENTS.md" && no "DevRites block survived uninstall" || ok "DevRites block removed from pre-existing AGENTS.md"
[ -f "$T4/.codex/config.toml" ] && ok "pre-existing .codex/config.toml preserved" || no "pre-existing .codex/config.toml removed"
grep -q 'model = "gpt-5-codex"' "$T4/.codex/config.toml" && ok "pre-existing .codex/config.toml content preserved" || no "pre-existing .codex/config.toml content lost"
grep -q '# BEGIN DEVRITES CODEX MCP' "$T4/.codex/config.toml" && no "DevRites MCP block survived uninstall" || ok "DevRites MCP block removed from pre-existing .codex/config.toml"
[ -f "$T4/.codex/hooks.json" ] && ok "pre-existing .codex/hooks.json preserved" || no "pre-existing .codex/hooks.json removed"
grep -q 'echo user-stop' "$T4/.codex/hooks.json" && ok "pre-existing .codex/hooks.json content preserved" || no "pre-existing .codex/hooks.json content lost"
grep -q 'devrites-' "$T4/.codex/hooks.json" && no "DevRites hooks survived uninstall" || ok "DevRites hooks removed from pre-existing .codex/hooks.json"
rm -rf "$T4"

echo ""
[ "$fail" -eq 0 ] && echo "uninstall-smoke: PASS" || echo "uninstall-smoke: FAIL"
exit "$fail"
