#!/usr/bin/env bash
# install-smoke.sh — dry-run, temp install, idempotent reinstall, flag behavior,
# and the no-global-write guard. Exits non-zero on any failure.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok()   { printf '  ok: %s\n' "$*"; }
no()   { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
cleanup() { rm -rf "$T"; }
trap cleanup EXIT

echo "== install-smoke (target: $T) =="

# 1) dry-run writes nothing
bash "$ROOT/install.sh" --target "$T" --dry-run >/dev/null 2>&1
[ -e "$T/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"
[ -e "$T/.agents" ] && no "dry-run created .agents" || true
[ -e "$T/.codex" ] && no "dry-run created .codex" || true
[ -e "$T/AGENTS.md" ] && no "dry-run created AGENTS.md" || true

# 2) real install
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && ok "manifest written" || no "no manifest"
for f in \
  ".claude/skills/rite/SKILL.md" \
  ".agents/skills/rite/SKILL.md" \
  ".claude/skills/rite-define/SKILL.md" \
  ".agents/skills/rite-define/SKILL.md" \
  ".agents/skills/devrites-lib/reference/standards/core.md" \
  ".agents/skills/devrites-lib/reference/standards/security.md" \
  ".claude/skills/rite-seal/SKILL.md" \
  ".claude/skills/rite-polish/SKILL.md" \
  ".claude/skills/rite-polish/reference/code.md" \
  ".claude/skills/rite-polish/reference/ui.md" \
  ".claude/skills/rite-pressure-test/SKILL.md" \
  ".claude/skills/devrites-doubt/SKILL.md" \
  ".claude/skills/devrites-lib/reference/parallel-dispatch.md" \
  ".claude/skills/devrites-frontend-craft/reference/shape.md" \
  ".claude/skills/devrites-debug-recovery/reference/build-the-loop.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".codex/agents/devrites-code-reviewer.toml" \
  ".codex/config.toml" \
  ".codex/hooks.json" \
  ".codex/mcp/devrites-mcp.mjs" \
  ".claude/agents/devrites-spec-reviewer.md" \
  ".claude/skills/devrites-lib/reference/standards/security.md" \
  ".claude/skills/devrites-lib/reference/standards/anti-patterns.md" \
  ".claude/skills/devrites-lib/reference/standards/README.md" \
  "AGENTS.md" \
  ".devrites/README.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$T/$f" ] && ok "present: $f" || no "missing: $f"
done

# 2b) deleted/merged/renamed skills must NOT be present
for f in \
  ".claude/skills/rite-polish-code/SKILL.md" \
  ".claude/skills/rite-polish-ui/SKILL.md" \
  ".claude/skills/devrites-context-pack/SKILL.md" \
  ".claude/skills/devrites-selector/SKILL.md" \
  ".claude/skills/devrites-idea-refine/SKILL.md" \
  ".claude/skills/idea-pressure-test/SKILL.md" \
  ".claude/skills/devrites-prototype/SKILL.md" \
  ".claude/skills/devrites-handoff/SKILL.md" \
  ".claude/skills/devrites-zoom-out/SKILL.md" \
  ".claude/skills/devrites-code-simplifier/SKILL.md" \
  ".claude/skills/devrites-security-hardening/SKILL.md" \
  ".claude/skills/devrites-performance-check/SKILL.md" \
  ".claude/skills/devrites-parallel-review/SKILL.md" \
  ".claude/skills/devrites-rules/SKILL.md" ; do
  [ -f "$T/$f" ] && no "stale skill still installed: $f" || ok "removed: $f"
done

# /polish and /normalize aliases are removed in favor of /rite-polish modes.
[ -d "$T/.claude/skills/polish" ]    && no "stale /polish alias still installed"    || ok "no /polish alias (removed)"
[ -d "$T/.claude/skills/normalize" ] && no "stale /normalize alias still installed" || ok "no /normalize alias (removed)"

# 3) manifest count sanity (>= 80 files) and ACTIVE not manifest-managed
n="$(grep -vc '^#' "$T/.claude/devrites.manifest")"
[ "$n" -ge 80 ] && ok "manifest lists $n files" || no "manifest too small ($n)"
grep -q '\.devrites/ACTIVE' "$T/.claude/devrites.manifest" && no "ACTIVE should NOT be manifest-managed" || ok "ACTIVE excluded from manifest"
grep -q '^\.agents/skills/rite/SKILL.md$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex skill mirror" || no "manifest missing Codex skill mirror"
grep -q '^\.agents/skills/devrites-lib/reference/standards/core.md$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex rules mirror" || no "manifest missing Codex rules mirror"
grep -q '^\.codex/agents/devrites-code-reviewer.toml$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex custom agent" || no "manifest missing Codex custom agent"
grep -q '^AGENTS.md$' "$T/.claude/devrites.manifest" && no "AGENTS.md should be marker-managed, not file-managed" || ok "AGENTS.md not file-managed"
grep -q '^\.claude/devrites\.agents-merge$' "$T/.claude/devrites.manifest" && ok "AGENTS merge marker managed" || no "AGENTS merge marker missing"
grep -q '^\.codex/config.toml$' "$T/.claude/devrites.manifest" && no ".codex/config.toml should be marker-managed, not file-managed" || ok ".codex/config.toml not file-managed"
grep -q '^\.claude/devrites\.codex-config-merge$' "$T/.claude/devrites.manifest" && ok "Codex config merge marker managed" || no "Codex config merge marker missing"
grep -q '^\.codex/hooks.json$' "$T/.claude/devrites.manifest" && no ".codex/hooks.json should be marker-managed, not file-managed" || ok ".codex/hooks.json not file-managed"
grep -q '^\.claude/devrites\.codex-hooks-merge$' "$T/.claude/devrites.manifest" && ok "Codex hooks merge marker managed" || no "Codex hooks merge marker missing"
grep -q '^\.codex/hooks/devrites-.*\.sh$' "$T/.claude/devrites.manifest" && no "post-cutover: Codex hook scripts should no longer be shipped" || ok "Codex hook scripts no longer shipped (engine binary is the control plane)"
grep -q '^\.codex/mcp/devrites-mcp.mjs$' "$T/.claude/devrites.manifest" && ok "manifest tracks Codex MCP server" || no "manifest missing Codex MCP server"
grep -q '## Codex compatibility' "$T/.agents/skills/rite-build/SKILL.md" && ok "Codex skill mirror has compatibility block" || no "Codex skill mirror missing compatibility block"
grep -q 'spawn the matching Codex custom agent' "$T/.agents/skills/rite-build/SKILL.md" && ok "Codex skill mirror maps subagent dispatch" || no "Codex skill mirror missing subagent mapping"
grep -q 'Read `.agents/skills/devrites-lib/reference/standards/core.md`' "$T/.agents/skills/rite-build/SKILL.md" && ok "Codex skill mirror loads DevRites rules mirror" || no "Codex skill mirror missing rules instruction"
grep -q '\.claude/agents' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at .claude/agents" || ok "Codex skill mirror does not point at .claude/agents"
grep -q '\.claude/skills/devrites-lib/reference/standards' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at .claude/skills/devrites-lib/reference/standards" || ok "Codex skill mirror does not point at .claude/skills/devrites-lib/reference/standards"
grep -q '\.claude/skills' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at .claude/skills" || ok "Codex skill mirror does not point at .claude/skills"
grep -q 'pack/\.claude' "$T/.agents/skills/rite-build/SKILL.md" && no "Codex skill mirror still points at pack/.claude" || ok "Codex skill mirror does not point at pack/.claude"
grep -q 'F=.agents/skills/rite-$V/SKILL.md' "$T/.agents/skills/rite/SKILL.md" && ok "Codex rite router dispatches inside .agents" || no "Codex rite router not rewritten to .agents"
grep -q '\.claude/skills/rite-$V' "$T/.agents/skills/rite/SKILL.md" && no "Codex rite router still points at .claude/skills" || ok "Codex rite router does not point at .claude/skills"
grep -R -q '\.\./.*agents/devrites-.*\.md' "$T/.agents/skills" && no "Codex skill mirrors still contain relative agent links" || ok "Codex skill mirrors rewrite relative agent links"
grep -q '\.codex/agents/devrites-slice-wright.toml' "$T/.agents/skills/rite-build/SKILL.md" && ok "Codex skill root points at Codex agent TOML" || no "Codex skill root missing Codex agent TOML path"
grep -q '\.claude/agents' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" && no "Codex reference still points at .claude/agents" || ok "Codex reference rewrites .claude/agents"
grep -q '\.claude/skills/devrites-lib/reference/standards' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" && no "Codex reference still points at .claude/skills/devrites-lib/reference/standards" || ok "Codex reference rewrites .claude/skills/devrites-lib/reference/standards"
grep -q '\.codex/agents' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" && ok "Codex reference points at Codex agents" || no "Codex reference missing Codex agents path"
grep -q '\.agents/skills/devrites-lib/reference/standards' "$T/.agents/skills/rite-build/reference/wright-dispatch.md" && ok "Codex reference points at mirrored rules" || no "Codex reference missing mirrored rules path"
grep -q '\.claude/skills/devrites-lib/reference/standards' "$T/.codex/agents/devrites-code-reviewer.toml" && no "Codex agent still points at .claude/skills/devrites-lib/reference/standards" || ok "Codex agent uses mirrored rules paths"
grep -q '\$rite-seal' "$T/.agents/skills/devrites-audit/SKILL.md" && ok "Codex skill mirror rewrites slash rite invocations" || no "Codex skill mirror missing dollar rite invocation rewrite"
grep -q '\$rite-build' "$T/.codex/agents/devrites-slice-wright.toml" && ok "Codex agent descriptions rewrite slash rite invocations" || no "Codex agent descriptions missing dollar rite invocation rewrite"
if grep -R -nE '(^|[^A-Za-z0-9_./-])/rite(-[a-z0-9-]+)?([^A-Za-z0-9_-]|$)' "$T/.agents/skills" "$T/.codex/agents" >/tmp/dr_codex_slash_rite 2>/dev/null; then
  no "Codex mirrors still contain slash rite invocations"
  sed -n '1,20p' /tmp/dr_codex_slash_rite
else
  ok "Codex mirrors contain no slash rite invocations"
fi
python3 - <<PY
import json, pathlib, tomllib
tomllib.loads(pathlib.Path("$T/.codex/agents/devrites-code-reviewer.toml").read_text())
tomllib.loads(pathlib.Path("$T/.codex/agents/devrites-slice-wright.toml").read_text())
json.loads(pathlib.Path("$T/.codex/hooks.json").read_text())
tomllib.loads(pathlib.Path("$T/.codex/config.toml").read_text())
PY
[ "$?" -eq 0 ] && ok "generated Codex TOML/JSON config parses" || no "generated Codex TOML/JSON config invalid"
grep -q '"\$comment"' "$T/.codex/hooks.json" && no "generated Codex hooks include unsupported top-level comment" || ok "generated Codex hooks omit unsupported top-level comment"
grep -q 'DEVRITES_WRIGHT_AGENT_REQUIRED=1 devrites-engine hook wright-scope' "$T/.codex/hooks.json" \
  && ok "Codex wright-scope hook is scoped to slice-wright agent" \
  || no "Codex wright-scope hook missing agent-required guard"

# 4) no global write
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"
[ -e "$HOME/.codex/agents/devrites-code-reviewer.toml" ] && no "wrote to ~/.codex !!" || ok "~/.codex untouched"

# 5) idempotent reinstall
out="$(bash "$ROOT/install.sh" --target "$T" 2>&1)"
echo "$out" | grep -q 'installed: 0' && ok "reinstall installs 0 new (idempotent)" || no "reinstall not idempotent"
echo "$out" | grep -q 'skipped(conflict): 0' && ok "reinstall skips 0 (all managed)" || no "reinstall reported conflicts"

# 5a) User edits to shared Codex config surfaces survive reinstall/update.
printf '\nUser project guidance.\n' >> "$T/AGENTS.md"
printf '\nmodel = "gpt-5-codex"\n' >> "$T/.codex/config.toml"
node -e 'const fs=require("fs"); const p=process.argv[1]; const j=JSON.parse(fs.readFileSync(p,"utf8")); (j.hooks ||= {}).Stop ||= []; j.hooks.Stop.push({hooks:[{type:"command",command:"echo user-stop"}]}); fs.writeFileSync(p, JSON.stringify(j,null,2)+"\n")' "$T/.codex/hooks.json"
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "reinstall after shared Codex user edits failed"
grep -q 'User project guidance' "$T/AGENTS.md" && ok "AGENTS user guidance survives reinstall" || no "AGENTS user guidance lost on reinstall"
grep -q 'model = "gpt-5-codex"' "$T/.codex/config.toml" && ok "Codex config user setting survives reinstall" || no "Codex config user setting lost on reinstall"
grep -q 'echo user-stop' "$T/.codex/hooks.json" && ok "Codex user hook survives reinstall" || no "Codex user hook lost on reinstall"
[ "$(grep -c '<!-- BEGIN DEVRITES CODEX -->' "$T/AGENTS.md")" -eq 1 ] && ok "AGENTS block remains single after shared-file reinstall" || no "AGENTS block duplicated after shared-file reinstall"
[ "$(grep -c '# BEGIN DEVRITES CODEX MCP' "$T/.codex/config.toml")" -eq 1 ] && ok "Codex MCP block remains single after shared-file reinstall" || no "Codex MCP block duplicated after shared-file reinstall"
[ "$(grep -c 'devrites-engine hook stop-gate' "$T/.codex/hooks.json")" -eq 1 ] && ok "DevRites hooks remain single after shared-file reinstall" || no "DevRites hooks duplicated after shared-file reinstall"

# 5b) prune: a managed file dropped from the pack is removed on reinstall/update,
#     while .devrites/ runtime state is never touched (this is the path update.sh runs).
mkdir -p "$T/.claude/skills/_gone/reference"
echo stale > "$T/.claude/skills/_gone/reference/dropped.md"
printf '.claude/skills/_gone/reference/dropped.md\n' >> "$T/.claude/devrites.manifest"
mkdir -p "$T/.agents/skills/_gone"
echo stale > "$T/.agents/skills/_gone/SKILL.md"
printf '.agents/skills/_gone/SKILL.md\n' >> "$T/.claude/devrites.manifest"
echo keep > "$T/.devrites/should-survive"
printf '.devrites/should-survive\n' >> "$T/.claude/devrites.manifest"   # even if mis-listed, must NOT be pruned
out="$(bash "$ROOT/install.sh" --target "$T" --force 2>&1)"
[ -f "$T/.claude/skills/_gone/reference/dropped.md" ] && no "dropped managed file not pruned" || ok "dropped managed file pruned"
[ -d "$T/.claude/skills/_gone" ] && no "emptied dir left behind after prune" || ok "emptied dir tidied after prune"
[ -f "$T/.agents/skills/_gone/SKILL.md" ] && no "dropped Codex mirror file not pruned" || ok "dropped Codex mirror file pruned"
[ -d "$T/.agents/skills/_gone" ] && no "emptied Codex mirror dir left behind after prune" || ok "emptied Codex mirror dir tidied after prune"
[ -f "$T/.devrites/should-survive" ] && ok ".devrites/ runtime state preserved across prune" || no ".devrites/ state pruned — DANGER"
echo "$out" | grep -q 'pruned: 2' && ok "prune count reported (2 managed; .devrites entry skipped)" || no "prune count not reported"
rm -f "$T/.devrites/should-survive"

# 6) --no-agents
T2="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T2" --no-agents >/dev/null 2>&1
[ -d "$T2/.claude/agents" ] && no "--no-agents still installed agents" || ok "--no-agents skipped agents"
[ -d "$T2/.codex/agents" ] && no "--no-agents still installed Codex agents" || ok "--no-agents skipped Codex agents"
[ -f "$T2/.claude/skills/rite-build/SKILL.md" ] && ok "--no-agents still installs skills" || no "--no-agents broke skills"
[ -f "$T2/.agents/skills/rite-build/SKILL.md" ] && ok "--no-agents still installs Codex skills" || no "--no-agents broke Codex skills"
rm -rf "$T2"

# 7) --short-aliases=all installs /define /build /prove /seal wrappers
T3="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T3" --short-aliases=all >/dev/null 2>&1
[ -f "$T3/.claude/skills/define/SKILL.md" ] && ok "--short-aliases=all installs /define" || no "--short-aliases=all missing /define"
[ -f "$T3/.claude/skills/build/SKILL.md" ]  && ok "--short-aliases=all installs /build"  || no "--short-aliases=all missing /build"
[ -f "$T3/.agents/skills/define/SKILL.md" ] && ok "--short-aliases=all mirrors /define for Codex" || no "--short-aliases=all missing Codex /define"
[ -f "$T3/.agents/skills/build/SKILL.md" ]  && ok "--short-aliases=all mirrors /build for Codex"  || no "--short-aliases=all missing Codex /build"
rm -rf "$T3"

# 7b) --no-rules is a deprecated no-op — standards now ship inside the devrites-lib skill
T4="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T4" --no-rules >/dev/null 2>&1
[ -d "$T4/.claude/skills/devrites-lib/reference/standards" ] && ok "--no-rules is a no-op; standards ship with the devrites-lib skill" || no "--no-rules dropped the standards (should be a no-op now)"
[ -f "$T4/.claude/skills/rite-build/SKILL.md" ] && ok "--no-rules still installs skills" || no "--no-rules broke skills"
# default install DOES include rules
T5="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T5" >/dev/null 2>&1
[ -f "$T5/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "default install includes DevRites rules" || no "default missing rules"
rm -rf "$T4" "$T5"

# 7c) --no-codex keeps the legacy Claude-only footprint
T6="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T6" --no-codex >/dev/null 2>&1
[ -f "$T6/.claude/skills/rite-build/SKILL.md" ] && ok "--no-codex still installs Claude skills" || no "--no-codex broke Claude skills"
[ -d "$T6/.agents" ] && no "--no-codex installed .agents" || ok "--no-codex skipped .agents"
[ -d "$T6/.codex" ] && no "--no-codex installed .codex" || ok "--no-codex skipped .codex"
[ -f "$T6/AGENTS.md" ] && no "--no-codex installed AGENTS.md" || ok "--no-codex skipped AGENTS.md"
rm -rf "$T6"

# 7d) runtime pinned aliases mirror to Codex when Codex support is installed
T7="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T7" >/dev/null 2>&1
bash "$ROOT/scripts/pin.sh" --target "$T7" add b rite-build >/dev/null 2>&1 || no "pin add failed"
[ -f "$T7/.claude/skills/b/SKILL.md" ] && ok "pin writes Claude alias" || no "pin missing Claude alias"
[ -f "$T7/.agents/skills/b/SKILL.md" ] && ok "pin mirrors Codex alias" || no "pin missing Codex alias"
grep -q '^\.agents/skills/b/SKILL.md$' "$T7/.claude/devrites.manifest" && ok "pin manifests Codex alias" || no "pin missing Codex manifest entry"
bash "$ROOT/scripts/pin.sh" --target "$T7" remove b >/dev/null 2>&1 || no "pin remove failed"
[ -e "$T7/.claude/skills/b/SKILL.md" ] && no "pin remove left Claude alias" || ok "pin removes Claude alias"
[ -e "$T7/.agents/skills/b/SKILL.md" ] && no "pin remove left Codex alias" || ok "pin removes Codex alias"
rm -rf "$T7"

# 7e) existing AGENTS.md is merged, not overwritten or skipped
T8="$(mktemp -d)"
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
grep -q '# BEGIN DEVRITES CODEX MCP' "$T8/.codex/config.toml" && ok "DevRites MCP block merged into .codex/config.toml" || no "DevRites MCP block not merged"
grep -q '^\.codex/config.toml$' "$T8/.claude/devrites.manifest" && no "existing .codex/config.toml wrongly file-managed" || ok "existing .codex/config.toml not file-managed"
grep -q '^\.claude/devrites\.codex-config-merge$' "$T8/.claude/devrites.manifest" && ok "Codex config merge marker managed" || no "Codex config merge marker missing"
grep -q 'echo user-stop' "$T8/.codex/hooks.json" && ok "existing .codex/hooks.json content preserved" || no "existing .codex/hooks.json content lost"
grep -q 'devrites-engine hook stop-gate' "$T8/.codex/hooks.json" && ok "DevRites hooks merged into .codex/hooks.json" || no "DevRites hooks not merged"
grep -q '"\$comment"' "$T8/.codex/hooks.json" && no "merged Codex hooks include unsupported top-level comment" || ok "merged Codex hooks omit unsupported top-level comment"
grep -q '^\.codex/hooks.json$' "$T8/.claude/devrites.manifest" && no "existing .codex/hooks.json wrongly file-managed" || ok "existing .codex/hooks.json not file-managed"
grep -q '^\.claude/devrites\.codex-hooks-merge$' "$T8/.claude/devrites.manifest" && ok "Codex hooks merge marker managed" || no "Codex hooks merge marker missing"
bash "$ROOT/install.sh" --target "$T8" >/dev/null 2>&1 || no "reinstall with existing AGENTS.md failed"
[ "$(grep -c '<!-- BEGIN DEVRITES CODEX -->' "$T8/AGENTS.md")" -eq 1 ] && ok "AGENTS merge is idempotent" || no "AGENTS merge duplicated block"
[ "$(grep -c '# BEGIN DEVRITES CODEX MCP' "$T8/.codex/config.toml")" -eq 1 ] && ok "Codex config merge is idempotent" || no "Codex config merge duplicated block"
[ "$(grep -c 'devrites-engine hook stop-gate' "$T8/.codex/hooks.json")" -eq 1 ] && ok "Codex hooks merge is idempotent" || no "Codex hooks merge duplicated entries"
bash "$ROOT/install.sh" --target "$T8" --no-codex --force >/dev/null 2>&1 || no "reinstall --no-codex with merged Codex blocks failed"
grep -q 'Keep this user guidance' "$T8/AGENTS.md" && ok "--no-codex prune preserved existing AGENTS.md content" || no "--no-codex prune lost existing AGENTS.md content"
grep -q '<!-- BEGIN DEVRITES CODEX -->' "$T8/AGENTS.md" && no "--no-codex prune left DevRites AGENTS block" || ok "--no-codex prune stripped DevRites AGENTS block"
grep -q 'model = "gpt-5-codex"' "$T8/.codex/config.toml" && ok "--no-codex prune preserved existing Codex config content" || no "--no-codex prune lost existing Codex config content"
grep -q '# BEGIN DEVRITES CODEX MCP' "$T8/.codex/config.toml" && no "--no-codex prune left DevRites MCP block" || ok "--no-codex prune stripped DevRites MCP block"
grep -q 'echo user-stop' "$T8/.codex/hooks.json" && ok "--no-codex prune preserved existing Codex hooks content" || no "--no-codex prune lost existing Codex hooks content"
grep -q 'devrites-' "$T8/.codex/hooks.json" && no "--no-codex prune left DevRites Codex hooks" || ok "--no-codex prune stripped DevRites Codex hooks"
[ -e "$T8/.claude/devrites.agents-merge" ] && no "--no-codex prune left AGENTS merge marker" || ok "--no-codex prune removed AGENTS merge marker"
[ -e "$T8/.claude/devrites.codex-config-merge" ] && no "--no-codex prune left Codex config merge marker" || ok "--no-codex prune removed Codex config merge marker"
[ -e "$T8/.claude/devrites.codex-hooks-merge" ] && no "--no-codex prune left Codex hooks merge marker" || ok "--no-codex prune removed Codex hooks merge marker"
[ -e "$T8/.agents/skills/rite/SKILL.md" ] && no "--no-codex prune left Codex skill mirror" || ok "--no-codex prune removed Codex skill mirror"
[ -e "$T8/.codex/mcp/devrites-mcp.mjs" ] && no "--no-codex prune left DevRites MCP server" || ok "--no-codex prune removed DevRites MCP server"
rm -rf "$T8"

# 8) static guard script
bash "$ROOT/scripts/check-no-global-writes.sh" >/dev/null 2>&1 && ok "check-no-global-writes passed" || no "check-no-global-writes failed"

echo ""
[ "$fail" -eq 0 ] && echo "install-smoke: PASS" || echo "install-smoke: FAIL"
exit "$fail"
