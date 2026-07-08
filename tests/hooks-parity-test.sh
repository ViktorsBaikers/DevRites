#!/usr/bin/env bash
# hooks-parity-test.sh — DevRites hook coverage must stay in sync across all THREE configs.
#
# Post-cutover every hook invokes the global `devrites-engine` binary (behind the inline fail-open
# guard) as `devrites-engine hook <name> --harness=<h>`, so coverage is compared by hook NAME, not
# by script filename. DevRites registers hooks three ways, and a drift silently drops a guard:
#   1. plugin  `pack/.claude/hooks/hooks.json`   (Claude plugin install)
#   2. file    `pack/.claude/settings.json`      (install.sh seeds this)
#   3. Codex   `<target>/.codex/hooks.json`      (generated; Codex event names, per-harness wiring)
#
# Claude (1) and (2) must be IDENTICAL (same events + hook names). Codex (3) has the same CORE
# enforcement, minus the hooks that are Claude-only by design:
#   - source-cache-pre/-post : fire on Claude's WebFetch tool; Codex has no WebFetch (uses
#                              web_search, which self-caches), so there is nothing to revalidate.
#   - statusline            : Claude settings statusLine surface; Codex has no matching hook event.
# subagent-orient IS shared. reviewer-readonly + wright-scope live in Claude SUBAGENT FRONTMATTER
# but in the Codex hooks.json (Codex agent TOML can't carry frontmatter hooks) — same enforcement.
set -u
export DEVRITES_NO_BINARY=1   # only the pack config is under test; no engine binary needed
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
target="$(mktemp -d)"; trap 'rm -rf "$target"' EXIT
bash "$ROOT/install.sh" --target "$target" >/dev/null 2>&1 || { echo "FAIL: install failed"; exit 1; }

python3 - "$ROOT" "$target" <<'PY'
import json, re, sys, glob
root, target = sys.argv[1], sys.argv[2]
# Capture the hook name from every `devrites-engine hook <name>` invocation.
RE = r"devrites-engine hook ([a-z0-9-]+)"

def names(path):
    return set(re.findall(RE, json.dumps(json.load(open(path)))))

fail = []

# (1) vs (2): identical events + hook names
plugin   = json.load(open(f"{root}/pack/.claude/hooks/hooks.json"))["hooks"]
filebase = json.load(open(f"{root}/pack/.claude/settings.json"))["hooks"]
if set(plugin) != set(filebase):
    fail.append(f"Claude EVENTS differ (plugin vs file): {sorted(set(plugin) ^ set(filebase))}")
if names(f"{root}/pack/.claude/hooks/hooks.json") != names(f"{root}/pack/.claude/settings.json"):
    fail.append("Claude hook NAMES differ between hooks.json and settings.json")

# (3) Codex must carry every SHARED hook. Shared = all Claude hooks (settings.json + the two
# subagent-frontmatter guards) minus the documented Claude-only set.
CLAUDE_ONLY = {"source-cache-pre", "source-cache-post", "statusline"}
claude_all = names(f"{root}/pack/.claude/settings.json")
for f in glob.glob(f"{root}/pack/.claude/agents/*.md"):
    claude_all |= set(re.findall(RE, open(f).read()))
shared = claude_all - CLAUDE_ONLY
if not shared:
    fail.append("No shared hook names captured; parity regex or hook wiring is broken")
codex = names(f"{target}/.codex/hooks.json")
missing = shared - codex
if missing:
    fail.append(f"Codex hooks.json MISSING shared enforcement: {sorted(missing)}")
leaked = CLAUDE_ONLY & codex
if leaked:
    fail.append(f"Claude-only hook leaked into Codex: {sorted(leaked)}")

if fail:
    print("HOOKS-PARITY: FAIL")
    for f in fail: print("  " + f)
    sys.exit(1)
print(f"HOOKS-PARITY: PASS — Claude plugin==file ({len(names(f'{root}/pack/.claude/settings.json'))} hooks); "
      f"Codex carries all {len(shared)} shared enforcement hooks (incl. subagent-orient); "
      f"{len(CLAUDE_ONLY)} Claude-only by design (source-cache x2 + statusline).")
PY
