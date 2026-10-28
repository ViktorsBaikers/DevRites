#!/usr/bin/env python3
"""Render DSH skills and native presets from the expanded canonical pack."""

import json
from pathlib import Path
import re
import shutil
import sys


BRIDGE = """# DevRites for DSH

Use the installed skills under `.dsh/skills` and invoke `/rite` or `/rite-<verb>`.
Read `.dsh/skills/devrites-lib/reference/standards/core.md` before a workflow.
Dispatch means call `subagent` with `agent_preset` set to the exact named specialist,
a self-contained `prompt`, and a short `description`. Every dispatch uses the native
`spawn` provider, a fresh Session with zero parent conversation. Never use a generic
agent, inherit the orchestrator preset, or use `subagent_fork` as a substitute.
Include the exact project-relative allowed paths and required evidence in the prompt.
The child id returned by `subagent` is an admission receipt, not a completed result.
Collect the native parent settlement notice for each id, including final text and
terminal status. `list_agents` activity `inactive` does not prove success. Reconcile
every result and compare returned paths with the contract before continuing.
Missing preset, failed activation, interrupted execution, or missing result blocks
the dependent work. Report the failure; never impersonate the specialist inline.
The orchestrator writes workspace bookkeeping; only the named writer writes source
and tests within its path contract. No skill or result authorizes commits, pushes,
publishing, deployment, or other irreversible actions without explicit approval.
"""

DSH_DOCTOR = """   On DSH, require the loaded `.dsh/cordis.patch.yml` overlay, the exact named
   native presets and `devrites-orchestrator`, confining shell/filesystem
   providers, read-only reviewer and workspace-write writer ceilings, and the
   native `spawn` provider with exact preset selection. Missing required preset
   or wrong permissions is `FAIL`; files present without effective loading is
   `WARN`. Honor `--no-dsh`, `--no-skills`, and `--no-agents` install flags.
"""

TOOLS = {"Read": ["read", "read_image"], "Grep": ["grep"], "Glob": ["glob"],
         "Bash": ["bash", "job_output", "job_kill", "job_list"], "Edit": ["edit"], "Write": ["write"],
         "Skill": ["skill"], "WebFetch": ["web_fetch"], "WebSearch": ["web_search"]}


def tool_plugins(tools):
    packages = []
    for group, package in [({"read", "read_image", "edit", "write"}, "tool-fs"),
                           ({"glob", "grep"}, "tool-fs-search"),
                           ({"bash"}, "tool-bash"), ({"skill"}, "tool-skill"),
                           ({"job_output", "job_kill", "job_list"}, "tool-jobs"),
                           ({"web_fetch", "web_search"}, "tool-web"),
                           ({"list_agents", "send_message", "interrupt_agent"}, "tool-subagent-control"),
                           ({"ask_user_question"}, "tool-ask-user")]:
        if group & set(tools):
            row = {"id": package, "name": "@deepseek-ai/dsh-" + package}
            if package == "tool-fs-search":
                row["config"] = {"sampleOverCapGlobResults": False}
            packages.append(row)
    return packages


def rewrite(text):
    text = re.sub(r"(?:pack/)?\.(?:claude|agents|codex|omp|pi|devin)/skills\b", ".dsh/skills", text)
    text = re.sub(r"(?:pack/)?\.(?:claude|codex|omp|pi|devin)/agents\b", ".dsh/agents", text)
    text = re.sub(r"(\.dsh/agents/[A-Za-z0-9_-]+)\.toml", r"\1.md", text)
    text = re.sub(r"devrites-engine\s+context(?!\s+--skills-root)",
                  "devrites-engine context --skills-root .dsh/skills", text)
    text = re.sub(r"devrites-engine\s+check\s+anchors(?!\s+--skills-root)",
                  "devrites-engine check anchors --skills-root .dsh/skills", text)
    text = text.replace("`Agent` call", "`subagent` call").replace("`spawn_agent` call", "`subagent` call")
    text = text.replace("`AskUserQuestion`", "`ask_user_question`")
    return text


def frontmatter(text):
    if not text.startswith("---\n") or "\n---" not in text[4:]:
        raise ValueError("canonical agent is missing frontmatter")
    header, body = text[4:].split("\n---", 1)
    fields = {}
    for line in header.splitlines():
        if ":" in line and not line.startswith((" ", "\t")):
            key, value = line.split(":", 1)
            fields[key] = value.strip().strip("\"'")
    return fields, body.lstrip("\n")


def generate(source, target):
    target.mkdir(parents=True, exist_ok=True)
    shutil.copytree(source / "skills", target / "skills", dirs_exist_ok=True,
                    ignore=shutil.ignore_patterns(".impeccable"))
    for path in (target / "skills").rglob("*.md"):
        text = rewrite(path.read_text())
        if path == target / "skills/rite-doctor/SKILL.md":
            text = text.replace("   File presence without effective loading is `WARN`;", DSH_DOCTOR + "   File presence without effective loading is `WARN`;")
        text = re.sub(r"(?<=\S) {2,}$", r"\\", text, flags=re.MULTILINE)
        path.write_text("\n".join(line.rstrip() for line in text.splitlines()).rstrip() + "\n")
    agents = target / "agents"
    agents.mkdir()
    presets = []
    for path in sorted((source / "agents").glob("*.md")):
        text = rewrite(path.read_text())
        fields, body = frontmatter(text)
        role = fields["name"]
        permission = "workspace-write" if fields.get("permissionMode") == "acceptEdits" else "read-only"
        tools = []
        raw = fields.get("tools", "Read, Grep, Glob, Bash, Skill" + (", Edit, Write" if permission == "workspace-write" else ""))
        for tool in raw.split(","):
            tools.extend(TOOLS.get(tool.strip(), []))
        if permission == "read-only":
            tools = [tool for tool in tools if tool not in ("edit", "write")]
        tools = sorted(set(tools))
        agents.joinpath(path.name).write_text(text)
        presets.append({"id": role, "name": role, "description": fields["description"],
                        "plugins": tool_plugins(tools) + [{"id": "devrites-role", "name": "devrites:role", "config": {
                            "role": role, "permission": permission, "allowedTools": tools,
                            "instructions": body}}]})
    roles = [preset["id"] for preset in presets]
    root_tools = sorted({tool for names in TOOLS.values() for tool in names}
                        | {"subagent", "list_agents", "send_message", "interrupt_agent", "ask_user_question"})
    presets.insert(0, {"id": "devrites-orchestrator", "name": "DevRites", "description": "DevRites workflow orchestrator with named fresh specialists.",
                       "plugins": tool_plugins(root_tools) + [
                           {"id": "devrites-role", "name": "devrites:role", "config": {
                               "role": "devrites-orchestrator", "permission": "workspace-write",
                               "allowedTools": root_tools, "allowedAgentPresets": roles, "instructions": BRIDGE}},
                           {"id": "devrites-subagent", "name": "@deepseek-ai/dsh-tool-subagent", "config": {
                               "provider": "spawn", "toolName": "subagent", "allowedAgentPresets": roles}},
                       ]})
    patch = [{"insert": [{"id": "devrites-presets", "name": "./specialist.mjs", "config": {"presets": presets}}]}]
    target.joinpath("runtime").mkdir()
    target.joinpath("runtime/cordis.patch.yml").write_text(json.dumps(patch, indent=2, ensure_ascii=False) + "\n")
    shutil.copyfile(Path(__file__).with_name("dsh-specialist.mjs"), target / "runtime/specialist.mjs")
    target.joinpath("AGENTS.md").write_text("<!-- BEGIN DEVRITES DSH -->\n" + BRIDGE + "<!-- END DEVRITES DSH -->\n")


if __name__ == "__main__":
    generate(Path(sys.argv[1]), Path(sys.argv[2]))
