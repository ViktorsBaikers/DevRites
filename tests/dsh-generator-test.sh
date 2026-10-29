#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
OUT="${DEVRITES_HOST_ARTIFACT_DIR:-$ROOT/pack/generated}"
python3 - "$ROOT" "$OUT" <<'PY'
import json
import importlib.util
from pathlib import Path
import sys

root, out = map(Path, sys.argv[1:])
spec = importlib.util.spec_from_file_location("dsh_generate", root / "scripts/dsh-generate.py")
generator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(generator)
assert generator.rewrite(".codex/agents/reviewer.toml config.toml Cargo.toml") == ".dsh/agents/reviewer.md config.toml Cargo.toml"
assert generator.rewrite("devrites-engine context --skills-root .claude/skills x") == "devrites-engine context --skills-root .dsh/skills x"
prefix = "mcp__codebase-memory-mcp__"
queries = {prefix + name for name in (
    "index_status", "list_projects", "get_graph_schema", "get_architecture",
    "search_graph", "search_code", "trace_path", "trace_call_path", "get_code_snippet",
    "get_file_outline", "query_graph", "detect_changes", "check_index_coverage", "compare_graphs",
)}
index_tool = prefix + "index_repository"
assert set(generator.mapped_tools("Read, " + prefix + "*")) == queries | {"read", "read_image"}
assert generator.mapped_tools(prefix + "search_graph") == [prefix + "search_graph"]
assert generator.mapped_tools("Read") == ["read", "read_image"]
for grant in ("mcp__other__*", prefix + "spoof__*", "mcp__codebase-memory-mcp-evil__*",
              prefix + "delete_project", prefix + "manage_adr", prefix + "ingest_traces",
              prefix + "future_tool", index_tool):
    assert generator.mapped_tools(grant) == [], grant
patch = json.loads((out / "dsh/runtime/cordis.patch.yml").read_text())
presets = patch[0]["insert"][0]["config"]["presets"]
by_id = {preset["id"]: preset for preset in presets}
canonical = {p.stem for p in (out / "claude/agents").glob("*.md")}
assert set(by_id) == canonical | {"devrites-orchestrator"}
assert len(by_id) == len(presets)
for preset in presets:
    config = next(row["config"] for row in preset["plugins"] if row["name"] == "devrites:role")
    skill_plugins = [row["name"] for row in preset["plugins"] if row["name"] in (
        "@deepseek-ai/dsh-skill-filesystem", "@deepseek-ai/dsh-tool-skill")]
    expected = ["@deepseek-ai/dsh-skill-filesystem", "@deepseek-ai/dsh-tool-skill"] if "skill" in config["allowedTools"] else []
    assert skill_plugins == expected, preset["id"]
for role in canonical:
    config = next(row["config"] for row in by_id[role]["plugins"] if row["name"] == "devrites:role")
    canonical_text = (out / "claude/agents" / (role + ".md")).read_text()
    fields, _ = generator.frontmatter(canonical_text)
    grants = {tool.strip() for tool in fields.get("tools", "").split(",")}
    expected_queries = queries if prefix + "*" in grants else queries & grants
    assert {tool for tool in config["allowedTools"] if tool.startswith("mcp__")} == expected_queries
    assert index_tool not in config["allowedTools"]
    assert "indexTool" not in config
    assert config["role"] == role
    assert config["instructions"].strip()
    assert "<!-- include:" not in config["instructions"]
    assert config["instructions"] in (out / "dsh/agents" / (role + ".md")).read_text()
    writable = "permissionMode: acceptEdits" in canonical_text.split("---", 2)[1]
    assert config["permission"] == ("workspace-write" if writable else "read-only")
    if not writable:
        assert not {"edit", "write"} & set(config["allowedTools"])
    assert "subagent" not in config["allowedTools"]
    if "bash" in config["allowedTools"]:
        assert {"job_output", "job_kill", "job_list"} <= set(config["allowedTools"])
        assert any(row["name"] == "@deepseek-ai/dsh-tool-jobs" for row in by_id[role]["plugins"])
orchestrator = [row for row in by_id["devrites-orchestrator"]["plugins"] if row["id"].startswith("devrites-")]
assert orchestrator[0]["config"]["permission"] == "inherit"
assert orchestrator[1]["config"]["provider"] == "spawn"
assert set(orchestrator[1]["config"]["allowedAgentPresets"]) == canonical
assert set(orchestrator[0]["config"]["allowedAgentPresets"]) == canonical
assert {tool for tool in orchestrator[0]["config"]["allowedTools"] if tool.startswith("mcp__")} == queries | {index_tool}
assert orchestrator[0]["config"]["indexTool"] == index_tool
mcp_patch = json.loads((out / "dsh/runtime/codebase-memory.patch.yml").read_text())
assert mcp_patch == [{"insert": [{"id": "devrites-codebase-memory", "name": "@deepseek-ai/dsh-mcp-client", "config": {
    "serverName": "codebase-memory-mcp", "transport": "stdio", "command": "codebase-memory-mcp", "args": [],
}}]}]
assert all(row["name"] != "@deepseek-ai/dsh-mcp-client" for row in patch[0]["insert"])
for source in (out / "claude/skills").glob("*/SKILL.md"):
    mirror = out / "dsh/skills" / source.parent.name / "SKILL.md"
    assert mirror.is_file(), mirror
    for flag in ("disable-model-invocation", "user-invocable"):
        values = [line for line in source.read_text().splitlines() if line.startswith(flag + ":")]
        for value in values:
            assert value in mirror.read_text()
assert (out / "dsh/runtime/specialist.mjs").read_bytes() == (root / "scripts/dsh-specialist.mjs").read_bytes()
doctor = (out / "dsh/skills/rite-doctor/SKILL.md").read_text()
assert "On DSH, require the loaded `.dsh/cordis.patch.yml` overlay" in doctor
assert "native `spawn` provider with exact preset selection" in doctor
print(f"DSH generation passed: {len(canonical)} named specialists and all skill mirrors")
PY
node "$ROOT/tests/dsh-specialist-test.mjs"
