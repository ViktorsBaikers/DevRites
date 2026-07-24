#!/usr/bin/env python3
"""Scan GitHub Actions workflows for supply-chain and permission risks.

Release and auto-merge workflows are especially sensitive, but these rules
apply to every workflow. The check rejects workflows that:

  - use a non-local action without a full 40-character commit SHA, because a
    moving tag can change upstream code;
  - omit an explicit `permissions:` scope or use `permissions: write-all`;
  - use `pull_request_target` outside a Dependabot-only workflow that does not
    check out pull-request code;
  - interpolate workflow_dispatch inputs directly into a shell `run:` block;
  - leave `: ` unquoted inside a workflow or step name, producing invalid YAML.

Usage: validate-workflow-security.py [DIR]   (default: .github/workflows)
Exit: 0 clean; 1 on any finding.
"""
import os
import re
import sys

SHA_RE = re.compile(r"^[0-9a-f]{40}$")
USES_RE = re.compile(r"^\s*-?\s*uses:\s*([^\s#]+)")
DEPENDABOT_ONLY_RE = re.compile(
    r"^\s*if:\s*(?:\$\{\{\s*)?"
    r"(?:github\.actor|github\.event\.pull_request\.user\.login)\s*==\s*"
    r"['\"]dependabot\[bot\]['\"]",
    re.MULTILINE,
)
UNQUOTED_NAME_COLON_RE = re.compile(r"^\s*(?:-\s*)?name:\s+[^'\"].*:\s+\S")
RUN_RE = re.compile(r"^(\s*)(?:-\s*)?run:\s*(.*)$")
DISPATCH_EXPRESSION_RE = re.compile(r"\$\{\{[^}]*\binputs\b", re.IGNORECASE)


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
        findings.append("%s: no permissions block. The default GITHUB_TOKEN is broad; "
                        "add an explicit least-privilege block" % path)
    for i, line in enumerate(lines, 1):
        if UNQUOTED_NAME_COLON_RE.match(line):
            findings.append("%s:%d: name has an unquoted colon. Quote the complete "
                            "name so GitHub can parse the workflow" % (path, i))
        if "write-all" in line:
            findings.append("%s:%d: permissions: write-all grants too much access. "
                            "Limit it to the permissions this workflow needs" % (path, i))
        if "pull_request_target" in line and not dependabot_target_is_safe:
            findings.append("%s:%d: pull_request_target exposes secrets to untrusted PR "
                            "code. Only a Dependabot-only workflow without checkout is allowed"
                            % (path, i))
        m = USES_RE.match(line)
        if not m:
            continue
        ref = m.group(1)
        if ref.startswith("./") or ref.startswith("."):
            continue  # local action
        at = ref.rsplit("@", 1)
        pin = at[1] if len(at) == 2 else ""
        if not SHA_RE.match(pin):
            findings.append("%s:%d: action '%s' is not pinned to a full commit "
                            "SHA. Pin it because a moving tag is a supply-chain risk"
                            % (path, i, ref))
    for i, line in enumerate(lines):
        match = RUN_RE.match(line)
        if not match:
            continue
        run_indent = len(match.group(1))
        run_lines = [(i + 1, match.group(2))]
        for j in range(i + 1, len(lines)):
            candidate = lines[j]
            if candidate.strip() and len(candidate) - len(candidate.lstrip()) <= run_indent:
                break
            run_lines.append((j + 1, candidate))
        for line_number, command in run_lines:
            if DISPATCH_EXPRESSION_RE.search(command):
                findings.append(
                    "%s:%d: workflow_dispatch input appears directly in run. "
                    "Pass it through env and quote the shell variable"
                    % (path, line_number)
                )
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
