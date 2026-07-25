#!/usr/bin/env bash
# Check that Codex can see an installed DevRites pack.
#
# Default mode uses `codex debug prompt-input`, which does not call the model.
# DEVRITES_CODEX_MODEL_SMOKE=1 also runs a read-only `codex exec` session and
# requires Codex authentication, network access, and a token budget.
# DEVRITES_CODEX_SUBAGENT_SMOKE=1 checks a live fresh-agent role-contract spawn
# and uses more tokens. It defaults to GPT-5.4's stable V1 surface;
# DEVRITES_CODEX_SUBAGENT_MODEL and
# DEVRITES_CODEX_SUBAGENT_SCHEMA select another authenticated model/schema pair.
# DEVRITES_CODEX_SUBAGENT_ROLE=devrites-slice-wright additionally proves the
# write-capable V2 receipt survives reconcile close.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
# shellcheck disable=SC1091
source "$ROOT/tests/runtime-smoke-lib.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v codex >/dev/null 2>&1 || { echo "codex-runtime-smoke: SKIP (codex CLI not found)"; exit 0; }
command -v python3 >/dev/null 2>&1 || { echo "codex-runtime-smoke: SKIP (python3 not found)"; exit 0; }

T="$(mktemp -d)"
GEN=""
RECEIPT_DIR=""
trap 'rm -rf "$T"; [ -n "$GEN" ] && rm -rf "$GEN"; [ -n "$RECEIPT_DIR" ] && rm -rf "$RECEIPT_DIR"' EXIT
live_engine_ready=1
if [ "${DEVRITES_CODEX_SUBAGENT_SMOKE:-0}" = "1" ]; then
  mkdir -p "$T/bin"
  if (cd "$ROOT/engine" && go build -o "$T/bin/devrites-engine" .); then
    export PATH="$T/bin:$PATH"
    ok "built current devrites-engine for live hook verification"
  else
    no "could not build current devrites-engine for live hook verification"
    live_engine_ready=0
  fi
fi
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi
# A unique basename prevents persistent-memory clients from reusing evidence
# from an earlier temp fixture also named "project".
PROJECT="$T/codex-runtime-smoke-$(basename "$T")"
mkdir -p "$PROJECT"
PROJECT="$(cd "$PROJECT" && pwd -P)"
RECEIPT_DIR="$(python3 - "$PROJECT/.devrites" <<'PY'
import hashlib, os, pathlib, sys, tempfile
root_hash = hashlib.sha256(os.path.normpath(sys.argv[1]).encode()).hexdigest()
print(pathlib.Path(tempfile.gettempdir()) / "devrites-agent-dispatch-v1" / root_hash)
PY
)"
mkdir -p "$T/home" "$T/codex-home"
git -C "$PROJECT" init -q || no "could not initialize live Codex project"

SUBAGENT_ROLE="${DEVRITES_CODEX_SUBAGENT_ROLE:-devrites-security-auditor}"
case "$SUBAGENT_ROLE" in
  devrites-security-auditor|devrites-slice-wright) ;;
  *) no "unknown DEVRITES_CODEX_SUBAGENT_ROLE=$SUBAGENT_ROLE" ;;
esac

echo "== codex-runtime-smoke (target: $PROJECT) =="
bash "$ROOT/install.sh" --target "$PROJECT" >/dev/null 2>&1 || no "install failed"
mkdir -p "$PROJECT/.agents/skills/devrites-runtime-smoke"
cat > "$PROJECT/.agents/skills/devrites-runtime-smoke/SKILL.md" <<EOF
---
name: devrites-runtime-smoke
description: Authenticated fixture for proving mandatory Codex role dispatch.
required-agent-roles: $SUBAGENT_ROLE
---

Spawn the required specialist and return its result.
EOF

(
  cd "$PROJECT" || exit 1
  HOME="$T/home" CODEX_HOME="$T/codex-home" codex debug prompt-input 'Use $rite-status to inspect DevRites.' > "$T/prompt.json" 2> "$T/prompt.err"
)
rc=$?
[ "$rc" -eq 0 ] && ok "codex debug prompt-input ran" || { no "codex debug prompt-input failed"; sed -n '1,80p' "$T/prompt.err"; }

python3 - "$T/prompt.json" <<'PY'
import pathlib, sys
s = pathlib.Path(sys.argv[1]).read_text() if pathlib.Path(sys.argv[1]).exists() else ""
checks = {
    "DevRites guidance visible": "DevRites" in s,
    "DevRites AGENTS block visible": "BEGIN DEVRITES CODEX" in s,
    "DevRites rules mirror path visible": ".agents/skills/devrites-lib/reference/standards/core.md" in s,
    "DevRites Codex agents path visible": ".codex/agents" in s,
}
for label, passed in checks.items():
    print(("ok:" if passed else "FAIL:") + " " + label)
if not all(checks.values()):
    sys.exit(1)
PY
if [ "$?" -eq 0 ]; then
  ok "Codex prompt input contains DevRites guidance"
else
  no "Codex prompt input missing DevRites guidance"
fi

[ -e "$PROJECT/.codex/mcp" ] && no "DevRites MCP directory installed" || ok "DevRites MCP directory not installed"
[ -e "$PROJECT/.codex/config.toml" ] && no "DevRites Codex MCP config installed" || ok "DevRites Codex MCP config not installed"

MODEL_HOME="${DEVRITES_CODEX_MODEL_HOME:-}"
MODEL_CODEX_HOME="${DEVRITES_CODEX_MODEL_CODEX_HOME:-}"
LIVE_CODEX_HOME="$T/model-codex-home"
model_env_ready() {
  runtime_explicit_roots_ready "$MODEL_HOME" "$MODEL_CODEX_HOME"
}

prepare_live_codex_home() {
  [ -f "$MODEL_CODEX_HOME/auth.json" ] || return 1
  mkdir -p "$LIVE_CODEX_HOME" || return 1
  cp "$MODEL_CODEX_HOME/auth.json" "$LIVE_CODEX_HOME/auth.json" || return 1
  chmod 600 "$LIVE_CODEX_HOME/auth.json" || return 1
  python3 - "$PROJECT" "$LIVE_CODEX_HOME/config.toml" <<'PY'
import json, pathlib, sys
project = str(pathlib.Path(sys.argv[1]).resolve())
pathlib.Path(sys.argv[2]).write_text(
    "[features]\nhooks = true\nmulti_agent = true\n\n"
    "[agents]\nenabled = true\n\n"
    f"[projects.{json.dumps(project)}]\n"
    'trust_level = "trusted"\n'
)
PY
}

if [ "${DEVRITES_CODEX_MODEL_SMOKE:-0}" = "1" ]; then
  if ! model_env_ready; then
    no "model-backed codex exec requires real Codex auth/config (set DEVRITES_CODEX_MODEL_HOME and DEVRITES_CODEX_MODEL_CODEX_HOME if needed)"
  elif ! prepare_live_codex_home; then
    no "model-backed codex exec could not prepare isolated authenticated Codex home"
  else
  (
    cd "$PROJECT" || exit 1
    HOME="$MODEL_HOME" CODEX_HOME="$LIVE_CODEX_HOME" codex exec \
      --ephemeral \
      --dangerously-bypass-hook-trust \
      --enable hooks \
      -c shell_environment_policy.inherit=all \
      -s read-only \
      'Read AGENTS.md only. Reply with exactly: DEVRITES-CODEX-OK'
  ) > "$T/exec.out" 2> "$T/exec.err"
  rc=$?
  if [ "$rc" -eq 0 ] && grep -q 'DEVRITES-CODEX-OK' "$T/exec.out"; then
    ok "codex exec model smoke passed"
  else
    no "codex exec model smoke failed"
    sed -n '1,80p' "$T/exec.err"
    sed -n '1,80p' "$T/exec.out"
  fi
  fi
else
  ok "model-backed codex exec skipped (set DEVRITES_CODEX_MODEL_SMOKE=1 to run)"
fi

SUBAGENT_MODEL="${DEVRITES_CODEX_SUBAGENT_MODEL:-gpt-5.4}"
SUBAGENT_SCHEMA="${DEVRITES_CODEX_SUBAGENT_SCHEMA:-v1}"
SKILL_DISPATCH_TASK_NAME="$(python3 - "$PROJECT" "$SUBAGENT_ROLE" <<'PY'
import hashlib, sys
print(sys.argv[2].replace("-", "_") + "_" + hashlib.sha256(sys.argv[1].encode()).hexdigest()[:12])
PY
)"
SKILL_DISPATCH_ROLE="$SUBAGENT_ROLE"
SKILL_DISPATCH_AGENT_TYPE="explorer"
SKILL_DISPATCH_CHILD_REQUEST="inspect README.md without modifying files"
SKILL_DISPATCH_AFTER_WAIT=""
SUBAGENT_SANDBOX="read-only"
if [ "$SKILL_DISPATCH_ROLE" = "devrites-slice-wright" ]; then
  SKILL_DISPATCH_AGENT_TYPE="worker"
  SKILL_DISPATCH_CHILD_REQUEST="append exactly DEVRITES-WRIGHT-CHILD-OK to README.md"
  SKILL_DISPATCH_AFTER_WAIT=' Then run rtk devrites-engine reconcile check && rtk devrites-engine reconcile close.'
  SUBAGENT_SANDBOX="workspace-write"
fi
case "$SUBAGENT_SCHEMA" in
  v1)
    SKILL_DISPATCH_PROMPT="\$devrites-runtime-smoke Authenticated DevRites dispatch smoke: do not work in the root. Call spawn_agent with agent_type $SKILL_DISPATCH_AGENT_TYPE and fork_turns none, and send a message naming .codex/agents/$SKILL_DISPATCH_ROLE.toml that asks the child to $SKILL_DISPATCH_CHILD_REQUEST. Wait for the returned child and use its result.$SKILL_DISPATCH_AFTER_WAIT Then reply exactly DEVRITES-SKILL-DISPATCH-OK."
    printf '%s\n' "$SKILL_DISPATCH_PROMPT" | grep -q "spawn_agent.*agent_type $SKILL_DISPATCH_AGENT_TYPE.*fork_turns none" \
      && ok "skill dispatch smoke explicitly requests the MultiAgent V1 compatibility path" \
      || no "skill dispatch smoke does not request the MultiAgent V1 compatibility path"
    ;;
  v2)
    SKILL_DISPATCH_AGENT_TYPE="$SKILL_DISPATCH_ROLE"
    SKILL_DISPATCH_PROMPT="\$devrites-runtime-smoke Authenticated DevRites dispatch smoke: do not work in the root. This smoke targets Codex MultiAgent V2. The visible V2 schema may omit agent_type even though the runtime accepts it: call spawn_agent with agent_type $SKILL_DISPATCH_ROLE, task_name $SKILL_DISPATCH_TASK_NAME, and fork_turns none anyway. Send the exact named child a message to $SKILL_DISPATCH_CHILD_REQUEST, wait for it, and use its non-empty result.$SKILL_DISPATCH_AFTER_WAIT Then reply exactly DEVRITES-SKILL-DISPATCH-OK."
    printf '%s\n' "$SKILL_DISPATCH_PROMPT" | grep -q "spawn_agent.*agent_type $SKILL_DISPATCH_ROLE.*task_name $SKILL_DISPATCH_TASK_NAME.*fork_turns none" \
      && ok "skill dispatch smoke explicitly requests the MultiAgent V2 named-role path" \
      || no "skill dispatch smoke does not request the MultiAgent V2 named-role path"
    ;;
  *)
    no "unknown DEVRITES_CODEX_SUBAGENT_SCHEMA=$SUBAGENT_SCHEMA (expected v1 or v2)"
    SKILL_DISPATCH_AGENT_TYPE=""
    SKILL_DISPATCH_PROMPT=""
    ;;
esac

if [ "${DEVRITES_CODEX_SUBAGENT_SMOKE:-0}" = "1" ]; then
  if [ "$live_engine_ready" -ne 1 ]; then
    :
  elif ! model_env_ready; then
    no "subagent smoke requires real Codex auth/config (set DEVRITES_CODEX_MODEL_HOME and DEVRITES_CODEX_MODEL_CODEX_HOME if needed)"
  elif ! prepare_live_codex_home; then
    no "subagent smoke could not prepare isolated authenticated Codex home"
  else
    mkdir -p "$PROJECT/.devrites/work/codex-skill-smoke"
    printf '%s\n' '# Codex skill dispatch smoke' > "$PROJECT/README.md"
    printf '%s\n' 'codex-skill-smoke' > "$PROJECT/.devrites/ACTIVE"
    cat > "$PROJECT/.devrites/work/codex-skill-smoke/spec.md" <<'EOF'
# Spec

Review the installed smoke README without modifying the project.
EOF
    printf '%s\n' 'README.md' > "$PROJECT/.devrites/work/codex-skill-smoke/touched-files.md"
    if [ "$SKILL_DISPATCH_ROLE" = "devrites-slice-wright" ]; then
      printf '%s\n' 'README.md' > "$PROJECT/.devrites/work/codex-skill-smoke/.wright-allowlist"
      cat > "$PROJECT/.devrites/work/codex-skill-smoke/spec.md" <<'EOF'
# Spec

Append exactly one line containing DEVRITES-WRIGHT-CHILD-OK to README.md.
EOF
      git -C "$PROJECT" add . >/dev/null 2>&1
      git -C "$PROJECT" -c user.name=DevRites -c user.email=devrites@example.invalid commit -qm 'fixture baseline'
      (cd "$PROJECT" && devrites-engine reconcile snapshot codex-skill-smoke) \
        || no "wright smoke could not create reconcile window"
    fi
    (
      cd "$PROJECT" || exit 1
      ephemeral_args=(--ephemeral)
      [ "$SUBAGENT_SCHEMA" = "v2" ] && ephemeral_args=()
      HOME="$MODEL_HOME" CODEX_HOME="$LIVE_CODEX_HOME" codex exec \
        --json \
        "${ephemeral_args[@]}" \
        -m "$SUBAGENT_MODEL" \
        --dangerously-bypass-hook-trust \
        --enable hooks \
        -c shell_environment_policy.inherit=all \
        -s "$SUBAGENT_SANDBOX" \
        "$SKILL_DISPATCH_PROMPT"
    ) > "$T/subagent.jsonl" 2> "$T/subagent.err"
    rc=$?
    if [ "$SKILL_DISPATCH_ROLE" = "devrites-slice-wright" ]; then
      [ ! -e "$PROJECT/.devrites/work/codex-skill-smoke/.reconcile-base" ] \
        || no "wright smoke did not close reconcile window"
      grep -q "DEVRITES-WRIGHT-CHILD-OK" "$PROJECT/README.md" \
        || no "wright smoke child did not write the allowed file"
    fi
    python3 - "$RECEIPT_DIR" "$SKILL_DISPATCH_ROLE" <<'PY'
import json, pathlib, sys

state_dir, role = map(str, sys.argv[1:])
events = []
for path in pathlib.Path(state_dir).glob("*.jsonl"):
    events.extend(json.loads(line) for line in path.read_text().splitlines() if line.strip())
assert any(
    event.get("event") == "armed" and event.get("role") == role
    for event in events
), "missing required-agent-roles armed receipt"
PY
    armed_rc=$?
    receipt_rc=1
    if [ "$SUBAGENT_SCHEMA" = "v1" ]; then
      python3 - "$PROJECT/.devrites" "$SKILL_DISPATCH_ROLE" "$SKILL_DISPATCH_AGENT_TYPE" <<'PY'
import hashlib, json, os, pathlib, sys, tempfile

root, role, agent_type = map(str, sys.argv[1:])
root_hash = hashlib.sha256(os.path.normpath(root).encode()).hexdigest()
state_dir = pathlib.Path(tempfile.gettempdir()) / "devrites-agent-dispatch-v1" / root_hash
events = []
for path in state_dir.glob("*.jsonl"):
    events.extend(json.loads(line) for line in path.read_text().splitlines() if line.strip())

pending = [e for e in events if e.get("event") == "pending" and e.get("role") == role and e.get("agent_type") == agent_type]
assert pending, "missing pending spawn receipt"
tool_ids = {e["tool_use_id"] for e in pending}
started = [e for e in events if e.get("event") == "started" and e.get("tool_use_id") in tool_ids and e.get("agent_id")]
assert started, "missing SubagentStart receipt"
agent_ids = {e["agent_id"] for e in started}
assert any(e.get("event") == "stopped" and e.get("agent_id") in agent_ids and e.get("result_sha256") for e in events), "missing non-empty result receipt"
PY
    else
      python3 - "$LIVE_CODEX_HOME" "$T/subagent.jsonl" "$PROJECT" "$SKILL_DISPATCH_ROLE" "$SKILL_DISPATCH_TASK_NAME" <<'PY'
import json, pathlib, sys

code_home, event_path, project, role, task_name = map(str, sys.argv[1:])
events = [json.loads(line) for line in pathlib.Path(event_path).read_text().splitlines() if line.strip()]
thread_ids = [event.get("thread_id") for event in events if event.get("type") == "thread.started"]
assert thread_ids and thread_ids[-1], "missing parent thread id"
parent_id = thread_ids[-1]
rollouts = list((pathlib.Path(code_home) / "sessions").rglob("*.jsonl"))
parent_paths = [path for path in rollouts if path.name.endswith(f"-{parent_id}.jsonl")]
assert len(parent_paths) == 1, f"expected one parent rollout, got {len(parent_paths)}"
parent = [json.loads(line) for line in parent_paths[0].read_text().splitlines() if line.strip()]
meta = parent[0]
assert meta.get("type") == "session_meta"
assert pathlib.Path(meta["payload"]["cwd"]).resolve() == pathlib.Path(project).resolve()

spawns = []
wait_indices = []
delivered_indices = []
for index, record in enumerate(parent):
    payload = record.get("payload", {})
    if payload.get("type") == "function_call" and payload.get("name") == "spawn_agent":
        args = json.loads(payload.get("arguments", "{}"))
        if args.get("task_name") == task_name:
            spawns.append((index, args))
    if payload.get("type") == "function_call" and payload.get("name") == "wait_agent":
        wait_indices.append(index)
    if payload.get("type") == "agent_message" and payload.get("author") == f"/root/{task_name}" and payload.get("recipient") == "/root":
        text = "\n".join(
            item.get("text", "")
            for item in payload.get("content", [])
            if item.get("type") == "input_text"
        ).strip()
        if text:
            delivered_indices.append(index)

assert len(spawns) == 1, f"expected one named spawn, got {len(spawns)}"
spawn_index, spawn_args = spawns[0]
assert spawn_args.get("agent_type") == role, spawn_args
assert spawn_args.get("fork_turns") == "none", spawn_args
assert delivered_indices, "missing non-empty child-to-root result"
assert any(spawn_index < wait < delivered_indices[0] for wait in wait_indices), "missing wait between spawn and delivered result"

children = []
for path in rollouts:
    if path == parent_paths[0]:
        continue
    records = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
    if not records or records[0].get("type") != "session_meta":
        continue
    child_meta = records[0].get("payload", {})
    if (
        child_meta.get("parent_thread_id") == parent_id
        and child_meta.get("agent_path") == f"/root/{task_name}"
    ):
        children.append(records)
assert len(children) == 1, f"expected one child rollout, got {len(children)}"
child = children[0]
child_meta = child[0]["payload"]
assert child_meta.get("agent_role") == role, child_meta
developer_text = "\n".join(
    item.get("text", "")
    for record in child
    for item in record.get("payload", {}).get("content", [])
    if record.get("payload", {}).get("type") == "message"
    and record.get("payload", {}).get("role") == "developer"
    and item.get("type") == "input_text"
)
assert f"Codex custom-agent version of DevRites `{role}`" in developer_text, "named role instructions were not loaded"
assert any(
    record.get("payload", {}).get("type") == "task_complete"
    and record.get("payload", {}).get("last_agent_message", "").strip()
    for record in child
), "missing non-empty child completion"
PY
    fi
    receipt_rc=$?
    stream_rc=0
    if [ "$SUBAGENT_SCHEMA" = "v1" ]; then
      grep -q '"tool":"spawn_agent"' "$T/subagent.jsonl" \
        && grep -q '"tool":"wait"' "$T/subagent.jsonl" \
        && grep -q "$SKILL_DISPATCH_AGENT_TYPE" "$T/subagent.jsonl" \
        && grep -q "$SKILL_DISPATCH_ROLE" "$T/subagent.jsonl" \
        || stream_rc=1
    fi
    if [ "$rc" -eq 0 ] \
      && [ "$armed_rc" -eq 0 ] \
      && [ "$receipt_rc" -eq 0 ] \
      && [ "$stream_rc" -eq 0 ] \
      && ! grep -q 'unknown agent_type' "$T/subagent.err" "$T/subagent.jsonl" \
      && grep -q 'DEVRITES-SKILL-DISPATCH-OK' "$T/subagent.jsonl"; then
      ok "codex skill-triggered $SUBAGENT_SCHEMA role-contract dispatch smoke passed"
    else
      no "codex skill-triggered $SUBAGENT_SCHEMA role-contract dispatch smoke failed"
      [ "$armed_rc" -eq 0 ] || printf '%s\n' "  required-agent-roles arming verification failed"
      [ "$receipt_rc" -eq 0 ] || printf '%s\n' "  receipt verification failed"
      sed -n '1,80p' "$T/subagent.err"
      sed -n '1,120p' "$T/subagent.jsonl"
    fi
  fi
else
  ok "fresh-agent codex exec skipped (set DEVRITES_CODEX_SUBAGENT_SMOKE=1 to run)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-runtime-smoke: PASS" || echo "codex-runtime-smoke: FAIL"
exit "$fail"
