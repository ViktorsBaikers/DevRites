#!/usr/bin/env python3
"""Scan GitHub Actions workflows for supply-chain + permission risks.

DevRites' publish (release.yml) and auto-merge paths are high-value targets, so this
gate fails CI when a workflow:

  - uses a THIRD-PARTY action not pinned to a full 40-char commit SHA — a moving tag
    like `@v2` lets a compromised upstream inject code into the pipeline (GitHub-owned
    `actions/*` and `github/*` tags are tolerated);
  - declares no `permissions:` scope anywhere — the default token is broad;
  - uses `permissions: write-all` (over-broad);
  - uses `pull_request_target`, except for a Dependabot-only workflow that never
    checks out PR code.

Usage: validate-workflow-security.py [DIR]   (default: .github/workflows)
Exit: 0 clean; 1 on any finding.
"""
import os
import re
import sys

FIRST_PARTY = {"actions", "github"}   # GitHub-owned; major-tag refs tolerated
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
USES_RE = re.compile(r"^\s*-?\s*uses:\s*([^\s#]+)")
DEPENDABOT_ONLY_RE = re.compile(
    r"^\s*if:\s*(?:\$\{\{\s*)?"
    r"(?:github\.actor|github\.event\.pull_request\.user\.login)\s*==\s*"
    r"['\"]dependabot\[bot\]['\"]",
    re.MULTILINE,
)


def safe_dependabot_target(text):
    if re.search(r"^\s*-?\s*uses:\s*actions/checkout@", text, re.MULTILINE):
        return False
    jobs = text.split("\njobs:", 1)
    if len(jobs) != 2:
        return False
    blocks = re.split(r"(?m)^  [A-Za-z0-9_-]+:\s*(?:#.*)?$", jobs[1])[1:]
    return bool(blocks) and all(DEPENDABOT_ONLY_RE.search(block) for block in blocks)


def scan_text(path, text):
    findings = []
    lines = text.splitlines()
    dependabot_target_is_safe = safe_dependabot_target(text)
    if not re.search(r"^\s*permissions:", text, re.MULTILINE):
        findings.append("%s: no permissions: scope — the default GITHUB_TOKEN is broad; "
                        "add an explicit least-privilege permissions block" % path)
    for i, line in enumerate(lines, 1):
        if "write-all" in line:
            findings.append("%s:%d: permissions: write-all is over-broad — scope to the "
                            "minimum needed" % (path, i))
        if "pull_request_target" in line and not dependabot_target_is_safe:
            findings.append("%s:%d: pull_request_target runs with secrets on untrusted PR "
                            "code — only a Dependabot-only workflow without checkout is allowed"
                            % (path, i))
        m = USES_RE.match(line)
        if not m:
            continue
        ref = m.group(1)
        if ref.startswith("./") or ref.startswith("."):
            continue  # local action
        owner = ref.split("/", 1)[0]
        if owner in FIRST_PARTY:
            continue
        at = ref.rsplit("@", 1)
        pin = at[1] if len(at) == 2 else ""
        if not SHA_RE.match(pin):
            findings.append("%s:%d: third-party action '%s' not pinned to a full commit "
                            "SHA — pin it (a moving tag is a supply-chain risk)"
                            % (path, i, ref))
    return findings


def iter_workflows(d):
    if os.path.isfile(d):
        yield d
        return
    for root, _dirs, names in os.walk(d):
        for n in sorted(names):
            if n.endswith((".yml", ".yaml")):
                yield os.path.join(root, n)


def main(argv):
    target = argv[1] if len(argv) > 1 else ".github/workflows"
    if not os.path.exists(target):
        print("OK    no workflows at %s" % target)
        return 0
    findings = []
    for path in iter_workflows(target):
        with open(path, "r", encoding="utf-8") as fh:
            findings.extend(scan_text(path, fh.read()))
    if findings:
        for f in findings:
            print("FINDING " + f)
        print("\n%d workflow-security finding(s)." % len(findings))
        return 1
    print("OK    workflow security clean (%s)" % target)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
