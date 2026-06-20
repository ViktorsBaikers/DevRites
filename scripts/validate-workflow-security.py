#!/usr/bin/env python3
"""Scan GitHub Actions workflows for supply-chain + permission risks.

DevRites' publish (release.yml) and auto-merge paths are high-value targets, so this
gate fails CI when a workflow:

  - uses a THIRD-PARTY action not pinned to a full 40-char commit SHA — a moving tag
    like `@v2` lets a compromised upstream inject code into the pipeline (GitHub-owned
    `actions/*` and `github/*` tags are tolerated);
  - declares no `permissions:` scope anywhere — the default token is broad;
  - uses `permissions: write-all` (over-broad);
  - uses `pull_request_target` (runs with secrets on untrusted PR code — review required).

Usage: validate-workflow-security.py [DIR]   (default: .github/workflows)
Exit: 0 clean; 1 on any finding.
"""
import os
import re
import sys

FIRST_PARTY = {"actions", "github"}   # GitHub-owned; major-tag refs tolerated
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
USES_RE = re.compile(r"^\s*-?\s*uses:\s*([^\s#]+)")


def scan_text(path, text):
    findings = []
    lines = text.splitlines()
    if not re.search(r"^\s*permissions:", text, re.MULTILINE):
        findings.append("%s: no permissions: scope — the default GITHUB_TOKEN is broad; "
                        "add an explicit least-privilege permissions block" % path)
    for i, line in enumerate(lines, 1):
        if "write-all" in line:
            findings.append("%s:%d: permissions: write-all is over-broad — scope to the "
                            "minimum needed" % (path, i))
        if "pull_request_target" in line:
            findings.append("%s:%d: pull_request_target runs with secrets on untrusted PR "
                            "code — review carefully or use pull_request" % (path, i))
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
