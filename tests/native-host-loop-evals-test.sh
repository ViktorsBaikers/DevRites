#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
output="$(python3 "$ROOT/scripts/live-hosts/run-loop-evals.py" --validate-only)"

case "$output" in
  "native-host-loop-evals: PASS (16 scenarios)") ;;
  *) printf 'unexpected native-host eval validation: %s\n' "$output" >&2; exit 1 ;;
esac

python3 - "$ROOT/scripts/live-hosts/run-loop-evals.py" "$ROOT/evals/native-host/codex-acceptance.json" "$ROOT" <<'PY'
import argparse
import importlib.util
import json
import os
import pathlib
import re
import shlex
import shutil
import sys
import tempfile

path = pathlib.Path(sys.argv[1])
matrix_path = pathlib.Path(sys.argv[2])
root = pathlib.Path(sys.argv[3])
source = path.read_text()
for required in ('"--tools", ""', '"--ignore-user-config"', '"--disable", "shell_tool"', '"--disable", "unified_exec"'):
    if required not in source:
        raise SystemExit(f"model tool isolation missing: {required}")
matrix = json.loads(matrix_path.read_text())
expected_cases = {
    "installed-pack-visibility",
    "loop-policy-selection",
    "root-skill-loading",
    "named-readonly-role",
    "same-worktree-writer",
    "native-worktree-transfer",
    "time-event-activation",
}
if matrix.get("version") != 1 or matrix.get("suite") != "devrites-codex-native-acceptance" or matrix.get("host") != "codex-cli":
    raise SystemExit("invalid Codex acceptance matrix identity")
cases = matrix.get("cases")
if not isinstance(cases, list) or {case.get("id") for case in cases} != expected_cases or len(cases) != len(expected_cases):
    raise SystemExit("Codex acceptance matrix must contain every required case exactly once")
allowed_admission = {
    ("deterministic", "supported"),
    ("live-opt-in", "supported"),
    ("live-diagnostic", "cannot-verify"),
    ("capability-gated", "unavailable"),
}
assignment = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=.*$")
for case in cases:
    if set(case) != {"id", "mode", "availability", "runner", "evidence", "claim", "does_not_demonstrate"}:
        raise SystemExit(f"invalid Codex acceptance fields: {case.get('id')}")
    if (case["mode"], case["availability"]) not in allowed_admission:
        raise SystemExit(f"invalid Codex admission combination: {case['id']}")
    if case["availability"] == "unavailable":
        if case["runner"] is not None:
            raise SystemExit(f"unavailable Codex case must have no runner: {case['id']}")
        continue
    if not isinstance(case["runner"], str) or not case["runner"].strip():
        raise SystemExit(f"runnable Codex case lacks runner: {case['id']}")
    parts = shlex.split(case["runner"])
    while parts and assignment.match(parts[0]):
        parts.pop(0)
    if not parts:
        raise SystemExit(f"Codex runner has only environment assignments: {case['id']}")
    command = parts[0]
    executable = root / command if "/" in command else pathlib.Path(shutil.which(command) or "")
    if not executable.is_file() or not os.access(executable, os.X_OK):
        raise SystemExit(f"Codex runner command is not executable: {case['id']}: {command}")
    for token in parts[1:]:
        if token.startswith(("scripts/", "tests/")) and not (root / token).is_file():
            raise SystemExit(f"Codex runner input is missing: {case['id']}: {token}")
loop_policy = (root / "pack/.claude/skills/devrites-lib/reference/standards/loop-operations.md").read_text()
wright_policy = (root / "pack/.claude/skills/rite-build/reference/wright-dispatch.md").read_text()
runtime_smoke = (root / "tests/codex-runtime-smoke.sh").read_text()
for required in ("## Activation capability gate", "record `unavailable`", "do not prove a CLI schedule/event facility"):
    if required not in loop_policy:
        raise SystemExit(f"Codex activation gate missing: {required}")
for required in ("Codex CLI custom subagents", "use `same-worktree`"):
    if required not in wright_policy:
        raise SystemExit(f"Codex worktree capability gate missing: {required}")
for required in (
    "runtime_secure_auth_file_ready",
    "Codex acceptance challenge:",
    "structured child spawn and non-empty result observed",
    "write_tree_manifest",
    "Codex read-only PR watcher installed",
    "no fake Codex workflow mirror installed",
):
    if required not in runtime_smoke:
        raise SystemExit(f"Codex runtime acceptance check missing: {required}")
spec = importlib.util.spec_from_file_location("loop_evals", path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
for value in ("nan", "inf", "-inf", "0", "-1"):
    try:
        module.positive_float(value)
    except argparse.ArgumentTypeError:
        continue
    raise SystemExit(f"unsafe cost accepted: {value}")
if module.positive_float("0.25") != 0.25:
    raise SystemExit("positive finite cost rejected")
env = module.model_env(pathlib.Path("/tmp/home"), CODEX_HOME="/tmp/codex")
if set(env) != {"PATH", "HOME", "LANG", "LC_ALL", "CODEX_HOME"}:
    raise SystemExit(f"model environment leaked ambient keys: {sorted(env)}")
with tempfile.TemporaryDirectory() as raw:
    root = pathlib.Path(raw)
    source = root / "source"
    source.mkdir()
    auth = source / "auth.json"
    auth.write_text("{}\n")
    auth.chmod(0o600)
    target = module.isolated_codex_home(source, root / "isolated")
    copied = target / "auth.json"
    if copied.read_text() != "{}\n" or (copied.stat().st_mode & 0o777) != 0o600:
        raise SystemExit("Codex auth isolation failed")
PY

AUTH_T="$(mktemp -d)"
trap 'rm -rf "$AUTH_T"' EXIT
mkdir -p "$AUTH_T/bin" "$AUTH_T/model-home" "$AUTH_T/no-codex"
ln -s "$(command -v dirname)" "$AUTH_T/no-codex/dirname"
if PATH="$AUTH_T/no-codex" DEVRITES_CODEX_ACCEPTANCE=1 "$(command -v bash)" \
  "$ROOT/tests/codex-runtime-smoke.sh" >/dev/null 2>&1; then
  printf 'strict Codex acceptance skipped a missing CLI\n' >&2
  exit 1
fi

mkdir -p "$AUTH_T/wrapper-bin" "$AUTH_T/wrapper-home"
cat > "$AUTH_T/wrapper-bin/python3" <<'SH'
#!/usr/bin/env sh
printf '%s\n' "$*" > "$CODEX_WRAPPER_LOG"
SH
cat > "$AUTH_T/wrapper-bin/codex" <<'SH'
#!/usr/bin/env sh
exit 0
SH
chmod 700 "$AUTH_T/wrapper-bin/python3" "$AUTH_T/wrapper-bin/codex"
WRAPPER_LOG="$AUTH_T/wrapper.log"
if PATH="$AUTH_T/wrapper-bin:$PATH" CODEX_WRAPPER_LOG="$WRAPPER_LOG" \
  "$ROOT/scripts/live-hosts/run-codex-loop-acceptance.sh" >/dev/null 2>&1; then
  printf 'Codex loop wrapper accepted missing configuration\n' >&2
  exit 1
fi
[ ! -s "$WRAPPER_LOG" ] || { printf 'Codex loop runner started before configuration admission\n' >&2; exit 1; }
PATH="$AUTH_T/wrapper-bin:$PATH" \
CODEX_WRAPPER_LOG="$WRAPPER_LOG" \
DEVRITES_CODEX_ACCEPTANCE_MODEL='pinned-model' \
DEVRITES_CODEX_ACCEPTANCE_HOME="$AUTH_T/wrapper-home" \
DEVRITES_CODEX_ACCEPTANCE_REPORT="$AUTH_T/report.json" \
DEVRITES_CODEX_ACCEPTANCE_TRIALS=1 \
DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS=30 \
DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS=16 \
  "$ROOT/scripts/live-hosts/run-codex-loop-acceptance.sh"
for required in 'run-loop-evals.py' '--host codex' '--model pinned-model' '--trials 1' '--max-seconds 30' '--codex-home' '--report'; do
  grep -q -- "$required" "$WRAPPER_LOG" \
    || { printf 'configured Codex loop wrapper missed %s\n' "$required" >&2; exit 1; }
done

cat > "$AUTH_T/bin/codex" <<'SH'
#!/usr/bin/env sh
printf '%s\n' "$*" >> "$CODEX_FAKE_LOG"
case "${1:-}" in
  debug)
    printf '%s\n' 'DevRites BEGIN DEVRITES CODEX .agents/skills/devrites-lib/reference/standards/core.md .codex/agents rite-watch-pr'
    exit 0
    ;;
  exec) exit 99 ;;
  *) exit 0 ;;
esac
SH
chmod 700 "$AUTH_T/bin/codex"

reject_invalid_auth() {
  local kind="$1" source="$AUTH_T/auth-$1" log="$AUTH_T/$1.log"
  mkdir -p "$source"
  case "$kind" in
    missing) ;;
    empty) : > "$source/auth.json"; chmod 600 "$source/auth.json" ;;
    permissive) printf '%s\n' '{}' > "$source/auth.json"; chmod 644 "$source/auth.json" ;;
    symlink)
      printf '%s\n' '{}' > "$source/real-auth.json"
      chmod 600 "$source/real-auth.json"
      ln -s "$source/real-auth.json" "$source/auth.json"
      ;;
  esac
  if PATH="$AUTH_T/bin:$PATH" \
    CODEX_FAKE_LOG="$log" \
    DEVRITES_HOST_ARTIFACT_DIR="$ROOT/pack/generated" \
    DEVRITES_CODEX_ACCEPTANCE=1 \
    DEVRITES_CODEX_MODEL_SMOKE=1 \
    DEVRITES_CODEX_MODEL_HOME="$AUTH_T/model-home" \
    DEVRITES_CODEX_MODEL_CODEX_HOME="$source" \
    bash "$ROOT/tests/codex-runtime-smoke.sh" >/dev/null 2>&1; then
    printf 'invalid Codex auth was accepted: %s\n' "$kind" >&2
    exit 1
  fi
  if grep -q '^exec' "$log" 2>/dev/null; then
    printf 'Codex exec ran before invalid auth was rejected: %s\n' "$kind" >&2
    exit 1
  fi
}

for kind in missing empty permissive symlink; do
  reject_invalid_auth "$kind"
done

printf 'native-host-loop-evals-test: PASS\n'
