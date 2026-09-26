#!/usr/bin/env bash
# /rite-fast contract: explicit-only lane, own fast-* agents, one writer, no per-slice gate.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT" <<'PY'
from pathlib import Path
import re
import sys

root = Path(sys.argv[1])
skill = (root / "pack/.claude/skills/rite-fast/SKILL.md").read_text()
agents = root / "pack/.claude/agents"
codex = root / "pack/generated/codex/agents"
roles = ["fast-planner", "fast-builder", "fast-checker", "fast-critic"]

def need(cond, msg):
    if not cond:
        raise SystemExit(f"FAIL: {msg}")

need("disable-model-invocation: true" in skill, "rite-fast must be explicit-only")
need("At most 10 slices" in skill, "slice cap of 10 must be stated")
need("No gate between slices" in skill, "no per-slice gate rule must be stated")
order = [skill.index(h) for h in ("## 1. Shape", "## 2. Build", "## 3. Check", "## 4. Refine", "## 5. Prove", "## 6. Report")]
need(order == sorted(order), "phases must run shape, build, check, refine, prove, report")
need("Only `fast-builder` writes source and tests" in skill, "fast-builder must be the only writer")
need("At most **2**" in skill, "fix rounds must be capped")

for role in roles:
    need(f"../../agents/{role}.md" in skill, f"skill must link {role}")
    text = (agents / f"{role}.md").read_text()
    need(re.search(r"^permissionMode: (plan|acceptEdits)$", text, re.M), f"{role} must set permissionMode")
    perm = re.search(r'default_permissions = "([^"]+)"', (codex / f"{role}.toml").read_text()).group(1)
    want = ":workspace" if role == "fast-builder" else ":read-only"
    need(perm == want, f"codex {role} permissions {perm}, want {want}")

builder = (agents / "fast-builder.md").read_text()
need("permissionMode: acceptEdits" in builder, "fast-builder must accept edits")
need("Do not run the test suite" in builder, "fast-builder must not verify per slice")
for role in ("fast-planner", "fast-checker", "fast-critic"):
    need("permissionMode: plan" in (agents / f"{role}.md").read_text(), f"{role} must be read-only")

print("rite-fast-contract: PASS")
PY
