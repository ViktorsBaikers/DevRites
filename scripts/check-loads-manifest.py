#!/usr/bin/env python3
"""loads: manifest integrity gate.

Static checks over every <!-- loads: {...} --> manifest in pack/.claude/skills
so dead or mistyped declarations fail at validate time instead of silently
producing incomplete context bundles at runtime:

  1. The manifest is valid JSON and uses only engine keys
     (always, triggers, workspace, workspaceByRole, agents).
  2. Every `always`/`triggers` path exists under pack/.claude/skills/.
  3. Every trigger name has a fire path: an engine `triggerSignals` regexp
     matches it (extracted from engine/internal/lib/triggers.go), a
     `(trigger `name`)` annotation exists in pack prose, the reserved
     `agents` name (auto-fired by `context --role`), or the name appears in
     the declaring SKILL.md's own prose — the model needs a stated
     condition, not just a manifest key.
  4. Every workspaceByRole role key and every agents-map entry resolves to a
     real file under pack/.claude/agents/ (default `devrites-<role>.md`).
  5. workspace/workspaceByRole artifact names are declared in
     devrites-lib/reference/workspace-artifact-schema.md.

Run: python3 scripts/check-loads-manifest.py    (exit 0 clean, 1 on any violation)
"""

import json
import os
import re
import sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def repo_paths(root):
    skills = os.path.join(root, "pack/.claude/skills")
    return (root, skills, os.path.join(root, "pack/.claude/agents"),
            os.path.join(root, "engine/internal/lib/triggers.go"),
            os.path.join(skills,
                         "devrites-lib/reference/workspace-artifact-schema.md"))

KNOWN_KEYS = {"always", "triggers", "workspace", "workspaceByRole", "agents"}
RESERVED_TRIGGERS = {"agents"}  # auto-fired by `context --role`


def engine_signal_regexes(triggers_go):
    """Extract the nameRe literals from the triggerSignals block."""
    src = open(triggers_go, encoding="utf-8").read()
    parts = src.split("var triggerSignals", 1)
    if len(parts) < 2 or "func suggestTriggers" not in parts[1]:
        raise SystemExit(
            f"FAIL: {triggers_go}: cannot locate var triggerSignals / "
            "func suggestTriggers — refactor the extractor with the source")
    block = parts[1].split("func suggestTriggers", 1)[0]
    return [re.compile(p, re.IGNORECASE) for p in
            re.findall(r"regexp\.MustCompile\(`([^`]+)`\)", block)]


def prose_annotated_triggers(skills_root):
    """Trigger names annotated `(trigger `name`)` anywhere in pack prose."""
    found = set()
    for root, _dirs, files in os.walk(skills_root):
        for name in files:
            if not name.endswith(".md"):
                continue
            text = open(os.path.join(root, name), encoding="utf-8").read()
            found.update(re.findall(r"\(trigger `([a-z0-9_-]+)`\)", text))
    return found


def schema_artifacts(schema_doc):
    return set(re.findall(r"\b[a-z][a-z0-9_-]*\.md\b",
                          open(schema_doc, encoding="utf-8").read()))


def main():
    root = REPO
    if "--root" in sys.argv:
        i = sys.argv.index("--root")
        if i + 1 >= len(sys.argv):
            print("FAIL: --root requires a path")
            return 1
        root = os.path.abspath(sys.argv[i + 1])
    _root, skills, agents_dir, triggers_go, schema_doc = repo_paths(root)

    problems = []
    signals = engine_signal_regexes(triggers_go)
    annotated = prose_annotated_triggers(skills)
    artifacts = schema_artifacts(schema_doc)
    manifests = 0

    for entry in sorted(os.listdir(skills)):
        skill_md = os.path.join(skills, entry, "SKILL.md")
        if not os.path.isfile(skill_md):
            continue
        text = open(skill_md, encoding="utf-8").read()
        m = re.search(r"<!--\s*loads:\s*(\{.*?\})\s*-->", text, re.DOTALL)
        if not m:
            continue
        manifests += 1
        rel = os.path.relpath(skill_md, root)
        try:
            manifest = json.loads(m.group(1))
        except json.JSONDecodeError as e:
            problems.append(f"{rel}: loads manifest is not valid JSON: {e}")
            continue

        unknown = set(manifest) - KNOWN_KEYS
        if unknown:
            problems.append(f"{rel}: unknown loads keys {sorted(unknown)}")

        # always/triggers paths must exist under the skills tree
        for path in manifest.get("always", []):
            if not os.path.isfile(os.path.join(skills, path)):
                problems.append(f"{rel}: always path missing: {path}")
        for trigger, paths in manifest.get("triggers", {}).items():
            for path in paths:
                if not os.path.isfile(os.path.join(skills, path)):
                    problems.append(
                        f"{rel}: trigger {trigger!r} path missing: {path}")

        # every trigger name needs a fire path: engine signal, prose
        # annotation, reserved auto-fire name, or a mention in the skill's
        # own prose outside the loads: manifest comment
        body = re.sub(r"<!--\s*loads:.*?-->", "", text, flags=re.DOTALL)
        for trigger in manifest.get("triggers", {}):
            if trigger in RESERVED_TRIGGERS:
                continue
            fired = (any(sig.search(trigger) for sig in signals)
                     or trigger in annotated
                     or trigger.lower() in body.lower())
            if not fired:
                problems.append(
                    f"{rel}: trigger {trigger!r} is dead — no engine signal, "
                    "no `(trigger `name`)` annotation, and no prose mention "
                    "tells the model when to pass it")

        # agents map: values are agent filenames under pack/.claude/agents/
        agent_overrides = manifest.get("agents", {})
        for role, rel_path in agent_overrides.items():
            if not os.path.isfile(os.path.join(agents_dir, rel_path)):
                problems.append(
                    f"{rel}: agents[{role!r}] -> {rel_path} missing under "
                    "pack/.claude/agents/")

        # workspaceByRole keys resolve to devrites-<role>.md or an override
        for role in manifest.get("workspaceByRole", {}):
            agent_file = agent_overrides.get(role, f"devrites-{role}.md")
            if not os.path.isfile(os.path.join(agents_dir, agent_file)):
                problems.append(
                    f"{rel}: workspaceByRole role {role!r} maps to no agent "
                    f"file ({agent_file})")

        # workspace artifact names must be schema-declared
        ws_names = set(manifest.get("workspace", []))
        for names in manifest.get("workspaceByRole", {}).values():
            ws_names.update(names)
        for name in sorted(ws_names):
            if name not in artifacts:
                problems.append(
                    f"{rel}: workspace artifact {name!r} is not declared in "
                    "workspace-artifact-schema.md")

    if manifests == 0:
        problems.append("no loads: manifests found under pack/.claude/skills")

    if problems:
        for p in problems:
            print(f"FAIL: {p}")
        print(f"loads-manifest: {len(problems)} problem(s) across {manifests} manifests")
        return 1
    print(f"loads-manifest: OK ({manifests} manifests)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
