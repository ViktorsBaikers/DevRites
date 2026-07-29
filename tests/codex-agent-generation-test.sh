#!/usr/bin/env bash
# Verify that every Claude DevRites agent becomes a Codex custom agent with
# Codex runtime paths.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
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
    report("You are the Codex custom-agent version" in instructions, f"{name}: Codex wrapper present")
    report("agent-result/v1" in instructions, f"{name}: returns the universal typed result envelope")
    report(".claude/agents" not in instructions, f"{name}: no .claude/agents runtime path")
    report(".claude/skills/devrites-lib/reference/standards" not in instructions, f"{name}: no .claude/skills/devrites-lib/reference/standards runtime path")
    report(".agents/skills/devrites-lib/reference/standards/" in instructions or name == "devrites-slice-wright", f"{name}: uses mirrored rules path when rules are referenced")

    if name == "devrites-slice-wright":
        report(data.get("sandbox_mode") != "read-only", f"{name}: write-capable")
    else:
        report(data.get("sandbox_mode") == "read-only", f"{name}: read-only sandbox")
    if name == "devrites-proof-runner":
        report(
            "Do not execute shell, browser, build, test" in instructions
            and "Run only packet-approved" not in instructions,
            f"{name}: validates immutable root-owned proof instead of executing gates",
        )
    if name == "devrites-forge-judge":
        report(
            'cd "<validated manifest primary_root>" && git diff' in instructions
            and "git -C" not in instructions,
            f"{name}: immutable diff command matches the read-only parser",
        )

    expected_subcommand = "wright-scope" if name == "devrites-slice-wright" else "reviewer-readonly"
    expected_required = (
        "DEVRITES_WRIGHT_AGENT_REQUIRED=1"
        if name == "devrites-slice-wright"
        else "DEVRITES_REVIEWER_AGENT_REQUIRED=1"
    )
    if tomllib is not None:
        groups = data.get("hooks", {}).get("PreToolUse", [])
        matcher = groups[0].get("matcher", "") if len(groups) == 1 else ""
        handlers = groups[0].get("hooks", []) if len(groups) == 1 else []
        command = handlers[0].get("command", "") if len(handlers) == 1 else ""
    else:
        matcher_match = re.search(
            r'^\[\[hooks\.PreToolUse\]\]\nmatcher = "([^"]+)"',
            text,
            re.M,
        )
        command_match = re.search(r"^command = '''(.*)'''$", text, re.M)
        matcher = matcher_match.group(1) if matcher_match else ""
        command = command_match.group(1) if command_match else ""
        groups = [True] if matcher_match else []
        handlers = [True] if command_match else []

    required_surfaces = {
        "Bash",
        "Edit",
        "Write",
        "apply_patch",
        "exec",
        "Task",
        "spawn_agent",
        "delegate",
        "dispatch_agent",
        "create_agent",
    }
    report(len(groups) == 1 and len(handlers) == 1, f"{name}: one agent-scoped leaf hook")
    report(
        required_surfaces <= set(matcher.split("|")),
        f"{name}: leaf hook covers mutation and nested dispatch",
    )
    report(
        f"{expected_required} devrites-engine hook {expected_subcommand} --harness=codex"
        in command,
        f"{name}: leaf hook routes to the exact engine guard",
    )
    report(
        "DEVRITES_AGENT_RUN=1" in command
        and f"DEVRITES_ACTIVE_AGENT={name}" in command,
        f"{name}: leaf hook declares its exact scoped identity",
    )
    report(
        "command -v devrites-engine" not in command
        and "node " not in command
        and 'case "$rc" in' in command
        and "exit 2" in command,
        f"{name}: leaf hook has no optional runtime and normalizes failures to deny",
    )

skill_files = sorted(skills_dir.rglob("*.md"))
report(bool(skill_files), f"installed Codex skill mirror is present ({len(skill_files)} markdown files)")
skill_text = "\n".join(path.read_text() for path in skill_files)

for forbidden in (
    ".reconcile-inline",
    "labeled fallback only at the final capability-ladder rung",
    "run the scout work **inline**",
    "no per-agent model control → run the scout inline",
    "MUST call that worker",
):
    report(forbidden not in skill_text, f"Codex skills exclude obsolete fallback {forbidden!r}")

for skill in sorted(skills_dir.glob("*/SKILL.md")):
    frontmatter = skill.read_text().split("\n---\n", 1)[0]
    match = re.search(r"(?m)^required-agent-roles:\s*(.+)$", frontmatter)
    report(bool(match), f"{skill.parent.name}: required-agent-roles declared")
    if not match or match.group(1).strip() == "none":
        continue
    roles = [role.strip() for role in match.group(1).split(",")]
    report(
        all(role in dst_names for role in roles),
        f"{skill.parent.name}: required agent roles resolve ({sorted(set(roles) - dst_names)})",
    )

for conditional_skill in ("devrites-source-driven", "rite-temper", "rite-upgrade"):
    frontmatter = (skills_dir / conditional_skill / "SKILL.md").read_text().split("\n---\n", 1)[0]
    report(
        re.search(r"(?m)^required-agent-roles:\s*none$", frontmatter) is not None,
        f"{conditional_skill}: conditional dispatch does not arm an unconditional receipt",
    )

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

for required in (
    "invoke means run a skill in this context",
    "Only after the runtime explicitly identifies MultiAgent V1, use generic `explorer`",
    'fork_turns="none"',
    "injects that contract's exact `developer_instructions`",
    "On explicitly identified MultiAgent V1, `devrites-slice-wright` uses generic `worker`",
    "`.wright-allowlist`",
    "On MultiAgent V2",
    "`agent_type=devrites-<role>`",
    "a unique `task_name`",
    "missing visible `agent_type` field is still V2",
    "stop before any generic/default spawn",
    "durable rollout",
    "Codex loads the role TOML's `developer_instructions` natively",
    "`required-agent-roles` frontmatter arms the fail-closed Stop receipt",
    "If the required dispatch for the explicitly identified runtime is unavailable or rejected, stop for HITL",
    "Never switch runtime lanes",
    "Never execute a DevRites specialist role in the root context",
):
    report(required in skill_text, f"Codex dispatch ladder includes {required!r}")

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
