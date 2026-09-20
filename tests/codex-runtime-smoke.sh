#!/usr/bin/env bash
# Check that Codex can see an installed DevRites pack.
#
# Default mode uses `codex debug prompt-input`, which does not call the model.
# DEVRITES_CODEX_MODEL_SMOKE=1 also runs a read-only `codex exec` session and
# requires Codex authentication, network access, and a token budget.
# DEVRITES_CODEX_SUBAGENT_SMOKE=1 runs a live end-to-end skill-triggered
# named-role observable check and uses more tokens.
# DEVRITES_CODEX_SUBAGENT_MODEL selects another authenticated model.
# DEVRITES_CODEX_SUBAGENT_ROLE selects any named role. The slice-wright case
# additionally checks the requested fixture edit.
# DEVRITES_CODEX_SUBAGENT_CONDITIONAL=1 selects a role after skill start.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
# shellcheck disable=SC1091
source "$ROOT/tests/runtime-smoke-lib.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
skip_or_fail() {
  if [ "${DEVRITES_CODEX_ACCEPTANCE:-0}" = "1" ]; then
    printf 'codex-runtime-smoke: FAIL (%s)\n' "$1" >&2
    exit 1
  fi
  printf 'codex-runtime-smoke: SKIP (%s)\n' "$1"
  exit 0
}
write_tree_manifest() {
  python3 - "$PROJECT" "$1" <<'PY'
import hashlib, json, os, pathlib, stat, sys
root = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])
manifest = {}
for path in sorted(root.rglob("*")):
    rel = path.relative_to(root)
    if ".git" in rel.parts:
        continue
    info = path.lstat()
    if stat.S_ISLNK(info.st_mode):
        manifest[rel.as_posix()] = {"type": "symlink", "target": os.readlink(path)}
    elif path.is_file():
        manifest[rel.as_posix()] = {
            "type": "file",
            "mode": stat.S_IMODE(info.st_mode),
            "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
        }
    elif path.is_dir():
        manifest[rel.as_posix()] = {"type": "dir", "mode": stat.S_IMODE(info.st_mode)}
out.write_text(json.dumps(manifest, sort_keys=True) + "\n")
PY
}

SUBAGENT_ROLE="${DEVRITES_CODEX_SUBAGENT_ROLE:-devrites-security-auditor}"

command -v codex >/dev/null 2>&1 || skip_or_fail "codex CLI not found"
command -v python3 >/dev/null 2>&1 || skip_or_fail "python3 not found"

T="$(mktemp -d)"
GEN=""
trap 'rm -rf "$T"; [ -n "$GEN" ] && rm -rf "$GEN"' EXIT
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
mkdir -p "$T/home" "$T/codex-home"
git -C "$PROJECT" init -q || no "could not initialize live Codex project"

case "$SUBAGENT_ROLE" in
  devrites-*)
    [ -f "$ROOT/pack/generated/codex/agents/$SUBAGENT_ROLE.toml" ] \
      || no "unknown DEVRITES_CODEX_SUBAGENT_ROLE=$SUBAGENT_ROLE"
    ;;
  *) no "unknown DEVRITES_CODEX_SUBAGENT_ROLE=$SUBAGENT_ROLE" ;;
esac
SKILL_DISPATCH_CHILD_LABEL="required"
SKILL_DISPATCH_BODY="Spawn the required specialist and return its result."
if [ "${DEVRITES_CODEX_SUBAGENT_CONDITIONAL:-0}" = "1" ]; then
  SKILL_DISPATCH_CHILD_LABEL="conditional"
  SKILL_DISPATCH_BODY="When asked to run the check, conditionally spawn $SUBAGENT_ROLE and return its result."
fi

echo "== codex-runtime-smoke (target: $PROJECT) =="
bash "$ROOT/install.sh" --target "$PROJECT" >/dev/null 2>&1 || no "install failed"
[ -f "$PROJECT/.agents/skills/rite-watch-pr/SKILL.md" ] \
  && ok "Codex read-only PR watcher installed" \
  || no "Codex read-only PR watcher missing"
[ -f "$PROJECT/.agents/skills/devrites-lib/reference/standards/loop-operations.md" ] \
  && ok "Codex loop operation policy installed" \
  || no "Codex loop operation policy missing"
[ -f "$PROJECT/.codex/agents/devrites-slice-wright.toml" ] \
  && ok "Codex exact writer installed" \
  || no "Codex exact writer missing"
[ ! -e "$PROJECT/.codex/workflows" ] \
  && ok "no fake Codex workflow mirror installed" \
  || no "unexpected Codex workflow mirror installed"
MODEL_SKILL_CHALLENGE="DEVRITES-CODEX-GUIDANCE-$(basename "$T")-$$"
mkdir -p "$PROJECT/.agents/skills/devrites-runtime-smoke"
cat > "$PROJECT/.agents/skills/devrites-runtime-smoke/SKILL.md" <<EOF
---
name: devrites-runtime-smoke
description: Authenticated fixture for exercising Codex named-role dispatch.
---

Codex acceptance challenge: $MODEL_SKILL_CHALLENGE
When asked for the acceptance challenge, return that value without spawning an agent.
For the named-role check: $SKILL_DISPATCH_BODY
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
    "Codex watcher skill visible": "rite-watch-pr" in s,
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
[ -f "$PROJECT/.codex/config.toml" ] && ok "native Codex permission config installed" || no "native Codex permission config missing"
grep -q 'mcp_servers\.devrites' "$PROJECT/.codex/config.toml" 2>/dev/null \
  && no "DevRites MCP registration installed" \
  || ok "DevRites MCP registration not installed"

MODEL_HOME="${DEVRITES_CODEX_MODEL_HOME:-}"
MODEL_CODEX_HOME="${DEVRITES_CODEX_MODEL_CODEX_HOME:-}"
LIVE_CODEX_HOME="$T/model-codex-home"
model_env_ready() {
  runtime_explicit_roots_ready "$MODEL_HOME" "$MODEL_CODEX_HOME"
}

prepare_live_codex_home() {
  runtime_secure_auth_file_ready "$MODEL_CODEX_HOME/auth.json" || return 1
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
      '$devrites-runtime-smoke Return only the value after "Codex acceptance challenge:" in this skill.'
  ) > "$T/exec.out" 2> "$T/exec.err"
  rc=$?
  if [ "$rc" -eq 0 ] && grep -Fq "$MODEL_SKILL_CHALLENGE" "$T/exec.out"; then
    ok "codex exec loaded the requested skill challenge"
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
SKILL_DISPATCH_ROLE="$SUBAGENT_ROLE"
SKILL_DISPATCH_CHILD_REQUEST="inspect README.md without modifying files"
SKILL_DISPATCH_ORIGINAL_README="# Codex skill dispatch smoke"
SKILL_DISPATCH_EXPECTED_README="$SKILL_DISPATCH_ORIGINAL_README"
if [ "$SKILL_DISPATCH_ROLE" = "devrites-slice-wright" ]; then
  SKILL_DISPATCH_CHILD_REQUEST="replace README.md with exactly '# Codex slice-wright smoke' and write no other path"
  SKILL_DISPATCH_EXPECTED_README="# Codex slice-wright smoke"
fi
SKILL_DISPATCH_PROMPT="\$devrites-runtime-smoke Authenticated DevRites named-role smoke. Do not work in the root. Use Codex's exact named custom agent $SKILL_DISPATCH_ROLE. Ask the $SKILL_DISPATCH_CHILD_LABEL child to $SKILL_DISPATCH_CHILD_REQUEST, wait for it, and use its non-empty result. Never substitute a generic/default child. Then reply exactly DEVRITES-SKILL-DISPATCH-OK."

if [ "${DEVRITES_CODEX_SUBAGENT_SMOKE:-0}" = "1" ]; then
  if ! model_env_ready; then
    no "subagent smoke requires real Codex auth/config (set DEVRITES_CODEX_MODEL_HOME and DEVRITES_CODEX_MODEL_CODEX_HOME if needed)"
  elif ! prepare_live_codex_home; then
    no "subagent smoke could not prepare isolated authenticated Codex home"
  else
    mkdir -p "$PROJECT/.devrites/work/codex-skill-smoke"
    printf '%s\n' "$SKILL_DISPATCH_ORIGINAL_README" > "$PROJECT/README.md"
    printf '%s\n' 'codex-skill-smoke' > "$PROJECT/.devrites/ACTIVE"
    cat > "$PROJECT/.devrites/work/codex-skill-smoke/spec.md" <<'EOF'
# Spec

Review the installed smoke README without modifying the project.
EOF
    printf '%s\n' 'README.md' > "$PROJECT/.devrites/work/codex-skill-smoke/touched-files.md"
    write_tree_manifest "$T/tree.before.json"
    (
      cd "$PROJECT" || exit 1
      HOME="$MODEL_HOME" CODEX_HOME="$LIVE_CODEX_HOME" codex exec \
        --json \
        -m "$SUBAGENT_MODEL" \
        --dangerously-bypass-hook-trust \
        --enable hooks \
        -c shell_environment_policy.inherit=all \
        "$SKILL_DISPATCH_PROMPT"
    ) > "$T/subagent.jsonl" 2> "$T/subagent.err"
    rc=$?
    subagent_ok=1
    if [ "$rc" -eq 0 ]; then
      ok "codex exec --json exited successfully"
    else
      no "codex exec --json failed"
      subagent_ok=0
    fi
    if grep -q 'DEVRITES-SKILL-DISPATCH-OK' "$T/subagent.jsonl"; then
      ok "requested final marker present in public JSON output"
    else
      no "requested final marker missing from public JSON output"
      subagent_ok=0
    fi
    if python3 - "$T/subagent.jsonl" <<'PY'
import json, pathlib, sys
spawned = False
returned = False
for line in pathlib.Path(sys.argv[1]).read_text().splitlines():
    try:
        event = json.loads(line)
    except json.JSONDecodeError:
        continue
    if event.get("type") != "item.completed" or not isinstance(event.get("item"), dict):
        continue
    item = event["item"]
    if item.get("type") != "collab_tool_call" or item.get("status") != "completed":
        continue
    if item.get("tool") == "spawn_agent" and item.get("receiver_thread_ids"):
        spawned = True
    if item.get("tool") == "wait" and isinstance(item.get("agents_states"), dict):
        returned = any(
            isinstance(state, dict)
            and state.get("status") == "completed"
            and isinstance(state.get("message"), str)
            and state["message"].strip()
            for state in item["agents_states"].values()
        )
if not (spawned and returned):
    raise SystemExit("missing structured child spawn/result evidence")
PY
    then
      ok "structured child spawn and non-empty result observed"
    else
      no "structured child spawn/result evidence missing"
      subagent_ok=0
    fi
    write_tree_manifest "$T/tree.after.json"
    if python3 - "$T/tree.before.json" "$T/tree.after.json" "$SKILL_DISPATCH_ROLE" <<'PY'
import json, pathlib, sys
before = json.loads(pathlib.Path(sys.argv[1]).read_text())
after = json.loads(pathlib.Path(sys.argv[2]).read_text())
changed = {path for path in set(before) | set(after) if before.get(path) != after.get(path)}
expected = {"README.md"} if sys.argv[3] == "devrites-slice-wright" else set()
if changed != expected:
    raise SystemExit(f"unexpected project tree delta: {sorted(changed)}; expected {sorted(expected)}")
PY
    then
      ok "$SKILL_DISPATCH_ROLE stayed inside the complete fixture tree boundary"
    else
      no "$SKILL_DISPATCH_ROLE changed an unexpected fixture path"
      subagent_ok=0
    fi
    if printf '%s\n' "$SKILL_DISPATCH_EXPECTED_README" | cmp -s - "$PROJECT/README.md"; then
      ok "$SKILL_DISPATCH_ROLE produced the expected project content"
    else
      no "$SKILL_DISPATCH_ROLE produced the wrong project content"
      subagent_ok=0
    fi
    if [ "$subagent_ok" -eq 1 ]; then
      ok "codex skill-triggered child diagnostic passed; exact custom role not exposed"
    else
      sed -n '1,80p' "$T/subagent.err"
      sed -n '1,120p' "$T/subagent.jsonl"
    fi
  fi
else
  ok "named-role codex exec skipped (set DEVRITES_CODEX_SUBAGENT_SMOKE=1 to run)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-runtime-smoke: PASS" || echo "codex-runtime-smoke: FAIL"
exit "$fail"
