#!/usr/bin/env python3
"""Cross-reference linter for the DevRites pack.

Catches the maintenance-drift class the skills audit surfaced: a SKILL.md or
reference file pointing at another file that has moved, been renamed, or never
existed. Three checks, tuned for near-zero false positives:

  1. Markdown links to local .md targets  — `](path.md)` resolved relative to the
     containing file; the target must exist.
  2. Bare backtick .md filenames          — `` `name.md` `` must exist SOMEWHERE in
     the pack (by basename). Catches references to a file that exists nowhere.
  3. "canonical version: `path`" claims    — the named path (relative to the skills
     root) must exist. Catches a file disclaiming itself secondary to a missing one.

Scans pack/.claude/. Exit 1 on any dead reference. Read-only.
"""
import os
import re
import sys

ROOT = os.path.join("pack", ".claude")
SKILLS_ROOT = os.path.join(ROOT, "skills")

# Runtime workspace artifacts live in .devrites/work/<slug>/, not in the pack.
# References to them by bare filename are legitimate, not dead refs.
WORKSPACE_ARTIFACTS = {
    "spec.md", "plan.md", "tasks.md", "state.md", "decisions.md", "assumptions.md",
    "questions.md", "drift.md", "evidence.md", "browser-evidence.md", "touched-files.md",
    "review.md", "seal.md", "ship.md", "brief.md", "design-brief.md", "strategy.md",
    "polish-report.md", "handoff.md", "eng-review.md", "test-plan.md", "references.md",
}

# Files the skills legitimately tell Claude to read in the USER's project / the repo,
# which are not part of the shipped pack.
PROJECT_DOCS = {
    "CLAUDE.md", "AGENTS.md", "DESIGN.md", "PRODUCT.md", "CONTEXT.md", "NOTES.md",
    "README.md", "CHANGELOG.md", "CONTRIBUTING.md", "cli-mcp.md",
}


def resolve(here, link):
    """Resolve a link. `.claude/...`-rooted paths are install-root-relative (= pack/
    in this repo); everything else is relative to the containing file."""
    if link.startswith(".claude/"):
        return os.path.normpath(os.path.join("pack", link))
    return os.path.normpath(os.path.join(here, link))

md_files = []
all_basenames = set()
for base, _dirs, files in os.walk(ROOT):
    for f in files:
        if f.endswith(".md"):
            p = os.path.join(base, f)
            md_files.append(p)
            all_basenames.add(f)

LINK_RE = re.compile(r"\]\(([^)]+)\)")
BACKTICK_MD_RE = re.compile(r"`([A-Za-z0-9._/\-]+\.md)`")
SEE_RE = re.compile(r"\(see\s+`?([A-Za-z0-9._/\-]+\.md)`?[^)]*\)")
CANONICAL_RE = re.compile(r"[Cc]anonical version:\s*`?\s*`?([A-Za-z0-9._/\-]+\.md)`")

errors = []

for path in md_files:
    with open(path, encoding="utf-8") as fh:
        text = fh.read()
    here = os.path.dirname(path)

    # 1. markdown links to .md
    for link in LINK_RE.findall(text):
        link = link.strip()
        if link.startswith(("http://", "https://", "mailto:", "#")):
            continue
        target = link.split("#", 1)[0]
        if not target.endswith(".md"):
            continue
        resolved = resolve(here, target)
        if not os.path.isfile(resolved):
            errors.append(f"{path}: markdown link -> {link} (resolved {resolved}) does not exist")

    # 2. bare backtick filenames must exist somewhere in the pack (by basename),
    #    excluding runtime workspace artifacts and user-project docs.
    for tok in BACKTICK_MD_RE.findall(text):
        base = os.path.basename(tok)
        if base in WORKSPACE_ARTIFACTS or base in PROJECT_DOCS or base in all_basenames:
            continue
        if "/" in tok:  # a path inside the pack must actually resolve
            if not os.path.isfile(resolve(here, tok)):
                errors.append(f"{path}: backtick `{tok}` does not resolve")
            continue
        errors.append(f"{path}: backtick `{tok}` — no `{base}` exists anywhere in the pack")

    # 4. "(see X.md)" prose pointers: slash form resolves (install-aware);
    #    bare form must exist in the pack (or be a workspace artifact / project doc).
    for tok in SEE_RE.findall(text):
        base = os.path.basename(tok)
        if "/" in tok:
            resolved = resolve(here, tok)
            if not os.path.isfile(resolved):
                errors.append(f"{path}: (see {tok}) (resolved {resolved}) does not exist")
        elif base not in WORKSPACE_ARTIFACTS and base not in PROJECT_DOCS and base not in all_basenames:
            errors.append(f"{path}: (see {tok}) — no `{base}` exists anywhere in the pack")

    # 3. "canonical version: `path`" must resolve against the skills root
    for claim in CANONICAL_RE.findall(text):
        resolved = os.path.normpath(os.path.join(SKILLS_ROOT, claim))
        if not os.path.isfile(resolved):
            errors.append(f"{path}: 'canonical version' -> {claim} (resolved {resolved}) does not exist")

if errors:
    print(f"check-cross-refs: {len(errors)} dead reference(s):\n")
    for e in sorted(errors):
        print("  " + e)
    sys.exit(1)

print(f"check-cross-refs: OK — {len(md_files)} markdown files, no dead references.")
