#!/usr/bin/env bash
# Keep DevRites hook coverage in sync across Claude and Codex.
#
# Both hosts call `devrites-engine hook <name> --harness=<h>`, so the test
# compares hook names rather than script filenames. Root hooks live in host
# settings. Codex also routes built-in explorer/worker identities through the
# leaf guards; the engine keeps root work out of those policies.
#
# Codex provides the same core enforcement except for these Claude-only hooks:
#   - source-cache-pre/-post: Claude fires these for WebFetch. Codex uses
#     self-caching web_search and has nothing equivalent to revalidate.
#   - statusline: Codex has no hook event matching Claude's statusLine setting.
#   - auq: Codex has request_user_input but emits no hook event for it.
#     PostToolUse covers only Bash, apply_patch, and MCP calls. The proposed
#     user-input-requested event was closed as not planned (openai/codex#12524).
#     Revisit this exception if Codex adds that event.
# `agent-dispatch` is Codex-only because Claude natively enforces declared agent
# types and lifecycle hooks from canonical skill/agent frontmatter.
# `subagent-orient` is shared. Claude agent frontmatter and generated Codex TOML
# files both carry `reviewer-readonly` and `wright-scope`.
set -u
export DEVRITES_NO_BINARY=1   # This test covers pack configuration, not the engine binary.
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

def json_commands(path):
    commands = []
    def walk(value):
        if isinstance(value, dict):
            for key, child in value.items():
                if key == "command" and isinstance(child, str):
                    commands.append(child)
                else:
                    walk(child)
        elif isinstance(value, list):
            for child in value:
                walk(child)
    walk(json.load(open(path)))
    return commands

def hook_locations(path, name):
    out = []
    payload = json.load(open(path))
    for event, entries in payload.get("hooks", {}).items():
        for entry in entries:
            for hook in entry.get("hooks", []):
                if f"devrites-engine hook {name}" in hook.get("command", ""):
                    out.append((event, entry.get("matcher", ""), hook.get("command", "")))
    return out

fail = []
settings = json.load(open(f"{root}/pack/.claude/settings.json"))
source_post_locations = []
for event, entries in settings.get("hooks", {}).items():
    for entry in entries:
        if "devrites-engine hook source-cache-post" in json.dumps(entry):
            source_post_locations.append((event, entry.get("matcher", "")))
if source_post_locations != [("PostToolUse", "WebFetch")]:
    fail.append(
        "source-cache-post must remain Claude WebFetch PostToolUse-only; "
        f"got {source_post_locations}"
    )

policy = open(f"{root}/engine/hookpolicy.go").read()
BLOCKERS = set(re.findall(r'"([a-z0-9-]+)":\s*\{[^}]*canBlock:\s*true', policy))
if not BLOCKERS:
    fail.append("production hook registry exposes no canBlock hooks")

# Each canonical leaf declares its exact identity and fails closed before the
# host exposes mutation or nested dispatch.
agent_files = sorted(glob.glob(f"{root}/pack/.claude/agents/devrites-*.md"))
for path in agent_files:
    body = open(path).read()
    role = re.search(r"(?m)^name:\s*(devrites-[a-z0-9-]+)\s*$", body)
    if not role:
        fail.append(f"{path}: missing exact devrites role name")
        continue
    role = role.group(1)
    for required in (
        "matcher: Edit|Write|MultiEdit|NotebookEdit|Bash|Agent|Task",
        "DEVRITES_AGENT_RUN=1",
        f"DEVRITES_ACTIVE_AGENT={role}",
        "guard unavailable: install devrites-engine",
    ):
        if required not in body:
            fail.append(f"{path}: agent guard missing {required!r}")
    expected_hook = "wright-scope" if role == "devrites-slice-wright" else "reviewer-readonly"
    if f"devrites-engine hook {expected_hook} --harness=claude" not in body:
        fail.append(f"{path}: expected {expected_hook} guard")
    command = re.search(r"(?m)^\s*command:\s*'([^']+)'\s*$", body)
    if not command or "|| exit 0" in command.group(1):
        fail.append(f"{path}: declared leaf guard is missing or fail-open")

# Codex must carry every shared hook: all Claude settings and scoped agent hooks
# except the documented Claude-only set.
CLAUDE_ONLY = {"source-cache-pre", "source-cache-post", "statusline", "auq"}
CODEX_ONLY = {"agent-dispatch"}
claude_all = names(f"{root}/pack/.claude/settings.json")
claude_commands = json_commands(f"{root}/pack/.claude/settings.json")
for f in glob.glob(f"{root}/pack/.claude/agents/*.md"):
    body = open(f).read()
    claude_all |= set(re.findall(RE, body))
    claude_commands += re.findall(r"(?m)^\s*command:\s*'([^']+)'\s*$", body)
shared = claude_all - CLAUDE_ONLY
if not shared:
    fail.append("No shared hook names captured; parity regex or hook wiring is broken")
codex_global = names(f"{target}/.codex/hooks.json")
codex_commands = json_commands(f"{target}/.codex/hooks.json")
for host, path, required_matchers in (
    ("Claude", f"{root}/pack/.claude/settings.json", {"Bash"}),
    ("Codex", f"{target}/.codex/hooks.json", {"Bash", "exec_command"}),
):
    locations = hook_locations(path, "git-guard")
    if len(locations) != 1 or locations[0][0] != "PreToolUse":
        fail.append(f"{host} git-guard must appear exactly once at PreToolUse: {locations}")
        continue
    matcher = set(locations[0][1].split("|"))
    missing_matchers = required_matchers - matcher
    if missing_matchers:
        fail.append(f"{host} git-guard matcher missing {sorted(missing_matchers)}")
codex_agents = set()
for f in glob.glob(f"{target}/.codex/agents/*.toml"):
    body = open(f).read()
    codex_agents |= set(re.findall(RE, body))
    codex_commands += re.findall(r"(?s)command\s*=\s*'''(.*?)'''", body)
for leaf in ("reviewer-readonly", "wright-scope"):
    locations = hook_locations(f"{target}/.codex/hooks.json", leaf)
    if len(locations) != 1 or locations[0][0] != "PreToolUse":
        fail.append(f"Codex {leaf} generic guard must appear exactly once at PreToolUse: {locations}")
        continue
    matcher = set(locations[0][1].split("|"))
    missing_matchers = {"Bash", "Edit", "apply_patch", "exec", "spawn_agent"} - matcher
    command = locations[0][2]
    if missing_matchers:
        fail.append(f"Codex {leaf} generic guard matcher missing {sorted(missing_matchers)}")
    if (
        "DEVRITES_CODEX_GENERIC_AGENT_COMPAT=1" not in command
        or "devrites-codex-generic-guard" not in command
    ):
        fail.append(f"Codex {leaf} is not scoped to the fail-closed generic compatibility path")
codex = codex_global | codex_agents
missing = shared - codex
if missing:
    fail.append(f"Codex hooks.json is missing shared enforcement: {sorted(missing)}")
leaked = CLAUDE_ONLY & codex
if leaked:
    fail.append(f"Claude-only hook leaked into Codex: {sorted(leaked)}")

# Both hosts must wire every production blocker. Hook commands inherit the
# operator's profile and kill-list environment, so generated commands may not
# clear or replace those control-plane variables before invoking the engine.
for host, wired, commands, host_only in (
    ("Claude", claude_all, claude_commands, CODEX_ONLY),
    ("Codex", codex, codex_commands, set()),
):
    missing_blockers = BLOCKERS - host_only - wired
    if missing_blockers:
        fail.append(f"{host} missing production blockers: {sorted(missing_blockers)}")
    for blocker in BLOCKERS:
        matches = [command for command in commands if f"devrites-engine hook {blocker}" in command]
        if not matches:
            continue
        for command in matches:
            if (
                "env -i" in command
                or re.search(r"\bunset\s+(?:DEVRITES_HOOK_PROFILE|DEVRITES_DISABLED_HOOKS)\b", command)
                or re.search(r"(?:^|[;\s])(?:DEVRITES_HOOK_PROFILE|DEVRITES_DISABLED_HOOKS)=", command)
            ):
                fail.append(f"{host} {blocker} command strips or overrides hook control-plane environment")

if fail:
    print("HOOKS-PARITY: FAIL")
    for f in fail: print("  " + f)
    sys.exit(1)
print(f"HOOKS-PARITY: PASS: Claude settings has {len(names(f'{root}/pack/.claude/settings.json'))} hooks; "
      f"Codex global + agent-scoped hooks carry all {len(shared)} shared enforcement hooks; "
      f"{len(CLAUDE_ONLY)} Claude-only by design (source-cache x2 + statusline + auq).")
PY
