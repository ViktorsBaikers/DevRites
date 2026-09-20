#!/usr/bin/env bash
# Native host permissions own the writer/read-only split. Exact paths are
# instruction-backed and no active agent profile retains an engine hook.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
target="$(mktemp -d)"; gen=""
trap 'rm -rf "$target"; [ -n "$gen" ] && rm -rf "$gen"' EXIT

if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  gen="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$gen" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$gen"
fi
bash "$ROOT/install.sh" --target "$target" >/dev/null 2>&1 \
  || { echo "FAIL: install failed"; exit 1; }

python3 - "$ROOT" "$target" <<'PY'
import json
import re
import sys
import tomllib
from pathlib import Path

root, target = map(Path, sys.argv[1:])
hook_re = re.compile(r"devrites-engine hook ([a-z0-9-]+)")
fail = []

claude_settings = json.loads((root / "pack/.claude/settings.json").read_text())
if claude_settings.get("permissions", {}).get("defaultMode") != "plan":
    fail.append("Claude root does not use native plan mode")
if claude_settings.get("hooks"):
    fail.append("Claude root retained project hooks")

codex_config = tomllib.loads((target / ".codex/config.toml").read_text())
if codex_config.get("default_permissions") != "devrites-orchestrator":
    fail.append('Codex root default_permissions is not "devrites-orchestrator"')
orchestrator = codex_config.get("permissions", {}).get("devrites-orchestrator", {})
if orchestrator.get("extends") != ":workspace":
    fail.append("Codex root profile does not extend :workspace")
if (target / ".codex/hooks.json").exists():
    fail.append("clean Codex install created a project hooks file")

claude_agents = sorted((root / "pack/.claude/agents").glob("devrites-*.md"))
codex_agents = sorted((target / ".codex/agents").glob("devrites-*.toml"))
if len(claude_agents) != 17 or len(codex_agents) != 17:
    fail.append(f"agent count mismatch: Claude={len(claude_agents)} Codex={len(codex_agents)}")

claude_hooks = set()
for path in claude_agents:
    body = path.read_text()
    role = path.stem
    names = set(hook_re.findall(body))
    claude_hooks |= names
    if role == "devrites-slice-wright":
        if "permissionMode: acceptEdits" not in body:
            fail.append(f"{path}: missing native writer permission")
    else:
        if "permissionMode: plan" not in body:
            fail.append(f"{path}: missing native plan permission mode")
    if "hooks:" in body or names:
        fail.append(f"{path}: retained an engine hook")

codex_hooks = set()
for path in codex_agents:
    body = path.read_text()
    profile = tomllib.loads(body)
    names = set(hook_re.findall(body))
    codex_hooks |= names
    expected = ":workspace" if path.stem == "devrites-slice-wright" else ":read-only"
    if profile.get("default_permissions") != expected:
        fail.append(f"{path}: missing exact {expected} permission profile")
    if "[[hooks." in body or names:
        fail.append(f"{path}: Codex profile retained an engine hook")
    if "sandbox_mode" in profile:
        fail.append(f"{path}: retained legacy sandbox_mode")

if claude_hooks:
    fail.append(f"Claude hook set differs: {sorted(claude_hooks)}")
if codex_hooks:
    fail.append(f"Codex hook set differs: {sorted(codex_hooks)}")

all_text = json.dumps(claude_settings) + "\n" + "\n".join(
    path.read_text() for path in claude_agents + codex_agents
)
for removed in ("git-guard", "a1-guard", "stop-gate", "wright-scope"):
    if removed in all_text:
        fail.append(f"removed hook survived: {removed}")

core = (root / "pack/.claude/skills/devrites-lib/reference/standards/core.md").read_text()
afk = (root / "pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md").read_text()
ship = (root / "pack/.claude/skills/rite-ship/SKILL.md").read_text()
git_ship = (root / "pack/.claude/skills/rite-ship/reference/git-ship.md").read_text()
autocomplete_paths = [
    root / "pack/.claude/skills/rite-autocomplete/SKILL.md",
    root / "pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md",
    root / "pack/.claude/skills/rite-autocomplete/reference/loop.md",
]

if "Before advancing a phase, run" not in core:
    fail.append("core guidance lost the readiness rest point")
if "devrites-engine check readiness <slug>" not in core:
    fail.append("core guidance lost the nested structural readiness check")
if "devrites-engine check seal <slug>" not in core:
    fail.append("core guidance lost rite-seal's final aggregate rest point")
if "three no-progress attempts per exact causal fingerprint" not in re.sub(r"\s+", " ", afk):
    fail.append("AFK guidance lost the per-fingerprint no-progress cap")
if "devrites-engine check seal <slug>" not in ship:
    fail.append("rite-ship preflight does not reuse the final seal aggregate")
if "A seal GO is never authorization for Git" not in git_ship:
    fail.append("rite-ship lost fresh exact Git approval")
for path in autocomplete_paths:
    if "never authorizes Git" not in path.read_text():
        fail.append(f"{path}: autocomplete flag still lacks the Git-approval boundary")

agents_bridge = (target / "AGENTS.md").read_text()
if "any changed or retried plan needs fresh approval" not in agents_bridge:
    fail.append("generated Codex AGENTS bridge lost fresh exact irreversible-action approval")
if list((target / ".codex").glob("*.rules")):
    fail.append("clean Codex install created preview exec-policy rules")

if fail:
    print("HOOKS-PARITY: FAIL")
    for item in fail:
        print("  " + item)
    raise SystemExit(1)
print("HOOKS-PARITY: PASS")
PY
