#!/usr/bin/env bash
# Verify that every Claude DevRites agent becomes a Codex custom agent with
# Codex runtime paths.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
# Binary installation has its own lifecycle test; keep this pack smoke isolated.
export DEVRITES_NO_BINARY=1
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v python3 >/dev/null 2>&1 || { echo "codex-agent-generation-test: SKIP (python3 not found)"; exit 0; }

T="$(mktemp -d)"
GEN=""
trap 'rm -rf "$T"; [ -n "$GEN" ] && rm -rf "$GEN"' EXIT
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "codex-agent-generation-test: FAIL (could not build host artifacts)"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi

echo "== codex-agent-generation-test (target: $T) =="
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"

python3 - "$ROOT" "$T" <<'PY'
import pathlib, re, sys

try:
    import tomllib
except ModuleNotFoundError:
    tomllib = None

root = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
src_dir = root / "pack/.claude/agents"
dst_dir = target / ".codex/agents"
skills_dir = target / ".agents/skills"

failed = False

src_agents = sorted(src_dir.glob("devrites-*.md"))
dst_agents = sorted(dst_dir.glob("devrites-*.toml"))

def report(ok, msg):
    global failed
    print(("ok: " if ok else "FAIL: ") + msg)
    if not ok:
        failed = True

def parse_generated_toml(text):
    """Parse generated TOML with tomllib when it is available.

    Python before 3.11 has no tomllib, so those runs use the narrow compatibility
    parser. Supported interpreters use the standard library parser.
    """
    if tomllib is not None:
        return tomllib.loads(text)

    data = {}
    m = re.search(
        r"^developer_instructions = '''\n(.*?)\n'''"
        r"(?=\Z|\n(?:[ \t]*\n)*(?:\[\[hooks\.|\Z))",
        text,
        re.M | re.S,
    )
    if m:
        data["developer_instructions"] = m.group(1)
        text = text[:m.start()] + text[m.end():]
    text = re.split(r"^\[\[hooks\.", text, maxsplit=1, flags=re.M)[0]
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if "=" not in line:
            raise ValueError(f"unsupported TOML line: {line!r}")
        key, value = [part.strip() for part in line.split("=", 1)]
        if value.startswith('"') and value.endswith('"'):
            data[key] = value[1:-1].replace('\\"', '"').replace('\\\\', '\\')
        else:
            raise ValueError(f"unsupported TOML value for {key}: {value!r}")
    return data

src_names = {path.stem for path in src_agents}
dst_names = {path.stem for path in dst_agents}
report(bool(src_agents), f"source has DevRites agents ({len(src_agents)})")
report(
    dst_names == src_names,
    f"Codex generated the exact dynamic source roster ({len(dst_agents)}/{len(src_agents)})",
)

for src in src_agents:
    name = src.stem
    dst = dst_dir / f"{name}.toml"
    report(dst.exists(), f"{name}.toml exists")
    if not dst.exists():
        continue
    text = dst.read_text()
    try:
        data = parse_generated_toml(text)
    except Exception as exc:
        report(False, f"{name}.toml parses as TOML: {exc}")
        continue

    report(data.get("name") == name, f"{name}: name preserved")
    report(bool(data.get("description")), f"{name}: description present")
    instructions = data.get("developer_instructions", "")
    report("You are the Codex custom-agent version" not in instructions, f"{name}: no duplicate Codex wrapper")
    report("agent-result/v1" not in instructions, f"{name}: uses native result delivery")
    report(".claude/agents" not in instructions, f"{name}: no .claude/agents runtime path")
    report(".claude/skills/devrites-lib/reference/standards" not in instructions, f"{name}: no .claude/skills/devrites-lib/reference/standards runtime path")
    source_text = src.read_text()
    if ".claude/skills/devrites-lib/reference/standards/" in source_text:
        report(
            ".agents/skills/devrites-lib/reference/standards/" in instructions,
            f"{name}: uses mirrored rules path when rules are referenced",
        )

    expected_permissions = ":workspace" if name == "devrites-slice-wright" else ":read-only"
    report(
        data.get("default_permissions") == expected_permissions,
        f"{name}: uses exact {expected_permissions} permission profile",
    )
    report("sandbox_mode" not in data, f"{name}: omits legacy sandbox_mode")
    report(
        "hooks" not in data and "[[hooks." not in text,
        f"{name}: native permission profile needs no engine hook",
    )
    if name == "devrites-proof-runner":
        report(
            "Execute no command or external write." in instructions
            and "Run only packet-approved" not in instructions,
            f"{name}: validates immutable root-owned proof instead of executing gates",
        )

skill_files = sorted(skills_dir.rglob("*.md"))
report(bool(skill_files), f"installed Codex skill mirror is present ({len(skill_files)} markdown files)")
skill_text = "\n".join(path.read_text() for path in skill_files)
execution_text = "\n".join(
    path.read_text()
    for path in skill_files
    if not path.as_posix().endswith("devrites-lib/reference/standards/agents.md")
)
report(
    all(name in execution_text for name in src_names),
    "every generated agent is named outside the catalog by executable workflow guidance",
)

for forbidden in (
    ".reconcile-inline",
    "labeled fallback only at the final capability-ladder rung",
    "run the scout work **inline**",
    "no per-agent model control → run the scout inline",
    "MUST call that worker",
    "MultiAgent V1",
    "MultiAgent V2",
    "required-agent-roles",
    "agent-dispatch",
):
    report(forbidden not in skill_text, f"Codex skills exclude obsolete fallback {forbidden!r}")

task_wording = re.findall(r"\bTask\b", skill_text)
report(not task_wording, "Codex skills contain no legacy Task orchestration wording")

agent_md_refs = re.findall(r"\.codex/agents/[^\s`\"'<>()\]]+\.md\b", skill_text)
report(not agent_md_refs, f"Codex skills contain no .codex agent markdown references ({agent_md_refs[:3]})")

brace_agent_refs = re.findall(r"\.codex/agents/[^\s`\"'<>()\]]*\{[^}\n]+\}", skill_text)
report(not brace_agent_refs, f"Codex skills contain no brace-compressed agent paths ({brace_agent_refs[:3]})")

agent_path_refs = set(re.findall(r"\.codex/agents/(devrites-[a-z0-9-]+)\.toml\b", skill_text))
report(
    agent_path_refs <= dst_names,
    f"all file-backed agent references resolve ({sorted(agent_path_refs - dst_names)})",
)

dispatch_refs = set(
    re.findall(
        r"(?im)\bdispatch(?:ed|es|ing)?\b[^\n`]{0,100}`(devrites-[a-z0-9-]+)`",
        skill_text,
    )
)
report(bool(dispatch_refs), "fresh-context dispatch references were discovered")
report(
    dispatch_refs <= dst_names,
    f"dispatch references resolve only to agents ({sorted(dispatch_refs - dst_names)})",
)

skill_names = {path.parent.name for path in skills_dir.glob("*/SKILL.md")}
invoke_refs = set(
    re.findall(
        r"(?im)\binvok(?:e|ed|es|ing)\b[^\n`]{0,100}`(devrites-[a-z0-9-]+)`",
        skill_text,
    )
)
report(bool(invoke_refs), "inline invocation references were discovered")
report(
    invoke_refs <= skill_names,
    f"invoke references resolve only to skills ({sorted(invoke_refs - skill_names)})",
)

bridge_text = (target / "AGENTS.md").read_text()
report(
    bool(re.search(r"Dispatch every exact named `devrites-<role>`.*fresh subagent thread.*wait", bridge_text, re.S)),
    "AGENTS bridge owns native exact-role fresh-context orchestration",
)
report("generic/default child" in bridge_text, "AGENTS bridge forbids generic role substitution")
report("spawn_agent" in bridge_text, "AGENTS bridge uses Codex native subagent lifecycle")
report(
    "Dispatch every exact named `devrites-<role>`" in bridge_text
    and "never skip it" in bridge_text,
    "AGENTS bridge requires every workflow agent to execute",
)
report(
    'devrites-slice-wright` alone uses `default_permissions = ":workspace"' in bridge_text
    and 'every other specialist uses `default_permissions = ":read-only"' in bridge_text
    and "root must never edit source or tests itself" in bridge_text,
    "AGENTS bridge keeps source writing in the exact native wright",
)
report(
    "`git diff --name-only`" in bridge_text
    and "reject any extra path" in bridge_text
    and "Never bypass the wright" in bridge_text
    and "never recreate an engine dispatch bridge" in bridge_text,
    "AGENTS bridge requires instruction-backed exact-path review",
)
for removed_syntax in ("agent_type=devrites-", 'fork_turns="none"', "a unique `task_name`"):
    report(
        removed_syntax not in bridge_text + skill_text,
        f"Codex guidance omits host-internal syntax {removed_syntax!r}",
    )

if failed:
    sys.exit(1)
PY
if [ "$?" -eq 0 ]; then
  ok "all Codex agent TOML files and dispatch references are complete and Codex-native"
else
  no "Codex agent generation validation failed"
fi

echo ""
[ "$fail" -eq 0 ] && echo "codex-agent-generation-test: PASS" || echo "codex-agent-generation-test: FAIL"
exit "$fail"
