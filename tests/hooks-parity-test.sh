#!/usr/bin/env bash
# hooks-parity-test.sh: DevRites hook coverage must stay in sync across Claude and Codex.
#
# Post-cutover every hook invokes the global `devrites-engine` binary (behind the inline fail-open
# guard) as `devrites-engine hook <name> --harness=<h>`, so coverage is compared by hook NAME, not
# by script filename. DevRites registers hooks in Claude settings and generated Codex hooks; drift
# between them silently drops a guard.
#
# Codex has the same core enforcement, minus the hooks that are Claude-only by design:
#   - source-cache-pre/-post : fire on Claude's WebFetch tool; Codex has no WebFetch (uses
#                              web_search, which self-caches), so there is nothing to revalidate.
#   - statusline            : Claude settings statusLine surface; Codex has no matching hook event.
#   - auq                   : fires on Claude's AskUserQuestion tool. Codex HAS an equivalent
#                             tool (request_user_input) but emits NO hook event for it: its
#                             PostToolUse matches only Bash/apply_patch/MCP calls, and the
#                             user-input-requested event was declined (openai/codex#12524,
#                             closed not-planned). Re-check if Codex hooks gain that event.
# subagent-orient IS shared. reviewer-readonly + wright-scope live in Claude SUBAGENT FRONTMATTER
# but in the Codex hooks.json (Codex agent TOML can't carry frontmatter hooks): same enforcement.
set -u
export DEVRITES_NO_BINARY=1   # only the pack config is under test; no engine binary needed
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
target="$(mktemp -d)"; gen=""; trap 'rm -rf "$target"; [ -n "$gen" ] && rm -rf "$gen"' EXIT
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  gen="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$gen" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$gen"
fi
bash "$ROOT/install.sh" --target "$target" >/dev/null 2>&1 || { echo "FAIL: install failed"; exit 1; }

python3 - "$ROOT" "$target" <<'PY'
import json, re, sys, glob
root, target = sys.argv[1], sys.argv[2]
# Capture the hook name from every `devrites-engine hook <name>` invocation.
RE = r"devrites-engine hook ([a-z0-9-]+)"

def names(path):
    return set(re.findall(RE, json.dumps(json.load(open(path)))))

fail = []

# Codex must carry every shared hook. Shared = all Claude hooks (settings.json + the two
# subagent-frontmatter guards) minus the documented Claude-only set.
CLAUDE_ONLY = {"source-cache-pre", "source-cache-post", "statusline", "auq"}
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
print(f"HOOKS-PARITY: PASS: Claude settings has {len(names(f'{root}/pack/.claude/settings.json'))} hooks; "
      f"Codex carries all {len(shared)} shared enforcement hooks (incl. subagent-orient); "
      f"{len(CLAUDE_ONLY)} Claude-only by design (source-cache x2 + statusline + auq).")
PY
