#!/usr/bin/env python3
"""Invocation-integrity gate — every skill/rule an agent or skill NAMES must resolve, both harnesses.

`check-cross-refs.py` validates markdown links. This validates the thing it can't: a **bare
skill-name invocation** (`$devrites-frontend-craft`, `Skill(devrites-source-driven)`, "invoke the
`devrites-api-interface` skill") and a **rule read** (`.claude/skills/devrites-lib/reference/standards/security.md`) must each point
at something that actually exists — on Claude Code AND on the Codex-translated tree — or the agent
calls into the void at runtime. Also asserts the Claude→Codex skill/rule mirror is complete.

Self-contained: installs DevRites into a temp dir to get the Codex tree, sweeps, cleans up.
Run: python3 scripts/check-invocation-integrity.py    (exit 0 clean, 1 on any unresolved reference)
"""
import os, re, sys, glob, subprocess, tempfile, shutil

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
PACK = os.path.join(REPO, "pack/.claude")

# devrites-* tokens that are scripts/hooks/mcp, not skills — legitimate non-skill references.
NONSKILL = {
    "devrites-mcp", "devrites-lib", "devrites-reviewer-readonly", "devrites-wright-scope",
    "devrites-a1-guard", "devrites-allow", "devrites-redwatch", "devrites-orient",
    "devrites-statusline", "devrites-stop-gate", "devrites-cursor", "devrites-refresh-indexes",
    "devrites-source-cache", "devrites-source-cache-pre", "devrites-source-cache-post",
    "devrites-subagent-orient", "devrites-detect", "devrites-prose-craft-report",
    # the control-plane binary, invoked by name in skill/agent prose ("devrites-engine seal")
    "devrites-engine",
    # a git commit-trailer marker ("[devrites-context]" body in a WIP checkpoint), not a skill
    "devrites-context",
}
# names a skill deliberately mentions as NON-existent (self-documenting prose).
# rite-principles: principles authoring rides existing skills; no such lifecycle step (principles.md).
DOCUMENTED_NONEXISTENT = {"rite-use", "rite-principles"}

SKILL_TOK = re.compile(r"\$?\b(devrites-[a-z0-9-]+|rite-[a-z0-9-]+)\b")
RULE_REF = re.compile(r"(?:\.claude/skills/devrites-lib/reference/standards|\.agents/skills/devrites-lib/reference/standards)/([a-z0-9-]+)\.md")
CODEX_SLASH_RITE = re.compile(r"(^|[^A-Za-z0-9_./-])/(rite(?:-[a-z0-9-]+)?)([^A-Za-z0-9_-]|$)")


def names(pattern, strip):
    return {os.path.basename(p)[:-strip] if strip else os.path.basename(p) for p in glob.glob(pattern)}


def scan(path, valid_names, valid_rules, problems):
    try:
        txt = open(path, encoding="utf-8").read()
    except Exception:
        return
    for name in set(SKILL_TOK.findall(txt)):
        if name in valid_names or name in NONSKILL or name in DOCUMENTED_NONEXISTENT:
            continue
        problems.append(f"{os.path.relpath(path, REPO)}: unresolved skill/agent name '{name}'")
    for r in set(RULE_REF.findall(txt)):
        if r not in valid_rules:
            problems.append(f"{os.path.relpath(path, REPO)}: rule ref '{r}.md' not present")


def scan_codex_sigils(path, problems):
    try:
        txt = open(path, encoding="utf-8").read()
    except Exception:
        return
    for lineno, line in enumerate(txt.splitlines(), 1):
        if CODEX_SLASH_RITE.search(line):
            problems.append(f"{os.path.relpath(path, REPO)}:{lineno}: Codex mirror uses /rite invocation; use $rite")


def main():
    target = tempfile.mkdtemp()
    try:
        env = os.environ.copy()
        env["DEVRITES_NO_BINARY"] = "1"
        r = subprocess.run(["bash", os.path.join(REPO, "install.sh"), "--target", target, "--no-binary"],
                           capture_output=True, text=True, env=env)
        if r.returncode != 0:
            print("FAIL: install failed\n" + r.stderr[-2000:]); return 1

        claude_skills = names(f"{PACK}/skills/*", 0)
        claude_rules = names(f"{PACK}/skills/devrites-lib/reference/standards/*.md", 3)
        claude_agents = names(f"{PACK}/agents/*.md", 3)
        codex_skills = names(f"{target}/.agents/skills/*", 0)
        codex_rules = names(f"{target}/.agents/skills/devrites-lib/reference/standards/*.md", 3)
        codex_agents = names(f"{target}/.codex/agents/*.toml", 5)

        problems, mirror = [], []
        if claude_skills != codex_skills:
            mirror.append(f"skills not mirrored: {sorted(claude_skills ^ codex_skills)}")
        if claude_rules != codex_rules:
            mirror.append(f"rules not mirrored: {sorted(claude_rules ^ codex_rules)}")

        # Claude side
        for f in glob.glob(f"{PACK}/agents/*.md") + glob.glob(f"{PACK}/skills/**/*.md", recursive=True):
            scan(f, claude_skills | claude_agents, claude_rules, problems)
        # Codex side
        for f in glob.glob(f"{target}/.codex/agents/*.toml") + glob.glob(f"{target}/.agents/skills/**/SKILL.md", recursive=True):
            scan(f, codex_skills | codex_agents, codex_rules, problems)
            scan_codex_sigils(f, problems)

        print(f"invocation-integrity: {len(claude_skills)} skills / {len(claude_rules)} rules / "
              f"{len(claude_agents)} agents, mirrored to Codex; scanned agents + skills, both harnesses.")
        for m in mirror:
            print("  MIRROR: " + m)
        for p in sorted(set(problems)):
            print("  DEAD: " + p)
        if problems or mirror:
            print("INVOCATION-INTEGRITY: FAIL")
            return 1
        print("INVOCATION-INTEGRITY: PASS — every named skill/rule resolves on both harnesses.")
        return 0
    finally:
        shutil.rmtree(target, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main())
