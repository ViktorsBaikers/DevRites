#!/usr/bin/env bash
# codex-agent-generation-test.sh — verify every Claude DevRites agent is converted
# to a Codex-native custom agent with Codex-native runtime paths.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v python3 >/dev/null 2>&1 || { echo "codex-agent-generation-test: SKIP (python3 not found)"; exit 0; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== codex-agent-generation-test (target: $T) =="
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"

python3 - "$ROOT" "$T" <<'PY'
import pathlib, sys, tomllib

root = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
src_dir = root / "pack/.claude/agents"
dst_dir = target / ".codex/agents"

failed = False

src_agents = sorted(src_dir.glob("devrites-*.md"))
dst_agents = sorted(dst_dir.glob("devrites-*.toml"))

def report(ok, msg):
    global failed
    print(("ok: " if ok else "FAIL: ") + msg)
    if not ok:
        failed = True

report(len(src_agents) == 14, f"source has 14 DevRites agents ({len(src_agents)})")
report(len(dst_agents) == len(src_agents), f"Codex generated one TOML per agent ({len(dst_agents)})")

for src in src_agents:
    name = src.stem
    dst = dst_dir / f"{name}.toml"
    report(dst.exists(), f"{name}.toml exists")
    if not dst.exists():
        continue
    try:
        data = tomllib.loads(dst.read_text())
    except Exception as exc:
        report(False, f"{name}.toml parses as TOML: {exc}")
        continue

    report(data.get("name") == name, f"{name}: name preserved")
    report(bool(data.get("description")), f"{name}: description present")
    instructions = data.get("developer_instructions", "")
    report("You are the Codex custom-agent version" in instructions, f"{name}: Codex wrapper present")
    report(".claude/agents" not in instructions, f"{name}: no .claude/agents runtime path")
    report(".claude/rules" not in instructions, f"{name}: no .claude/rules runtime path")
    report(".agents/devrites/rules/" in instructions or name == "devrites-slice-wright", f"{name}: uses mirrored rules path when rules are referenced")

    if name == "devrites-slice-wright":
        report(data.get("sandbox_mode") != "read-only", f"{name}: write-capable")
    else:
        report(data.get("sandbox_mode") == "read-only", f"{name}: read-only sandbox")

if failed:
    sys.exit(1)
PY
if [ "$?" -eq 0 ]; then
  ok "all Codex agent TOML files are complete and Codex-native"
else
  no "Codex agent generation validation failed"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-agent-generation-test: PASS" || echo "codex-agent-generation-test: FAIL"
exit "$fail"
