#!/usr/bin/env python3
"""Scan GitHub Actions workflows for supply-chain and permission risks.

Release and auto-merge workflows are especially sensitive, but these rules
apply to every workflow. The check rejects workflows that:

  - use a non-local action without a full 40-character commit SHA, because a
    moving tag can change upstream code;
  - omit an explicit `permissions:` scope or use `permissions: write-all`;
  - use `pull_request_target` (detected from the parsed `on` key) outside a workflow whose jobs (read from the parsed
    YAML, not line-matched) are each gated on the pull-request author at job
    level (exactly
    `github.event.pull_request.user.login == 'dependabot[bot]'`, never
    `github.actor`, which a re-trigger can change) and that does not check out
    pull-request code;
  - interpolate workflow_dispatch inputs directly into a shell `run:` block;
  - leave `: ` unquoted inside a workflow or step name, producing invalid YAML;
  - run commitlint without linting the pull-request title (via env) or without
    re-running when the title is edited, because the squash subject drives releases.

Usage: validate-workflow-security.py [DIR]   (default: .github/workflows)
Exit: 0 clean; 1 on any finding.
"""
import os
import re
import sys

try:
    import yaml  # type: ignore
except ImportError:
    sys.exit("PyYAML required: pip install -r scripts/requirements-ci.txt")

SHA_RE = re.compile(r"^[0-9a-f]{40}$")
USES_RE = re.compile(r"^\s*-?\s*uses\s*:\s*([^\s#]+)")
DEPENDABOT_GATE = "github.event.pull_request.user.login == 'dependabot[bot]'"
UNQUOTED_NAME_COLON_RE = re.compile(r"^\s*(?:-\s*)?name:\s+[^'\"].*:\s+\S")
RUN_RE = re.compile(r"^(\s*)(?:-\s*)?run\s*:\s*(.*)$")
DISPATCH_EXPRESSION_RE = re.compile(r"\$\{\{[^}]*\binputs\b", re.IGNORECASE)
COMMITLINT_RUN_RE = re.compile(r"^\s*(?:-\s*)?(?:run\s*:\s*)?.*\bnpx\b.*\bcommitlint\b", re.MULTILINE)
TITLE_ENV_RE = re.compile(r"^\s*\w+\s*:\s*\$\{\{\s*github\.event\.pull_request\.title\s*\}\}", re.MULTILINE)


def trigger_configs(doc):
    """Return the parsed `on` values. YAML 1.1 loads an unquoted `on` key as
    boolean True, so read both spellings."""
    return [doc[k] for k in ("on", True) if k in doc]


def trigger_names(doc):
    """Return the event names the workflow listens to (string, list or mapping)."""
    names = set()
    for on in trigger_configs(doc):
        if isinstance(on, str):
            names.add(on)
        elif isinstance(on, (list, dict)):
            names.update(k for k in on if isinstance(k, str))
    return names


def edited_pull_request(doc):
    """True when the pull_request trigger lists the 'edited' type."""
    for on in trigger_configs(doc):
        cfg = on.get("pull_request") if isinstance(on, dict) else None
        types = cfg.get("types") if isinstance(cfg, dict) else None
        if isinstance(types, str):
            types = [types]
        if isinstance(types, list) and "edited" in types:
            return True
    return False


def jobs_without_permissions(doc):
    """Return job IDs missing direct permissions, or None with a global scope."""
    if "permissions" in doc:
        return None
    jobs = doc.get("jobs")
    if not isinstance(jobs, dict) or not jobs:
        return ["<workflow>"]
    return [str(job_id) for job_id, job in jobs.items()
            if not isinstance(job, dict) or "permissions" not in job]


def safe_dependabot_target(doc):
    """True when every job is gated exactly on the Dependabot PR author and no
    step checks out code."""
    jobs = doc.get("jobs")
    if not isinstance(jobs, dict) or not jobs:
        return False
    for job in jobs.values():
        if not isinstance(job, dict):
            return False
        gate = job.get("if")
        if not isinstance(gate, str):
            return False
        gate = gate.strip()
        if gate.startswith("${{") and gate.endswith("}}"):
            gate = gate[3:-2].strip()
        if gate != DEPENDABOT_GATE:
            return False
        steps = job.get("steps")
        for step in steps if isinstance(steps, list) else []:
            uses = step.get("uses") if isinstance(step, dict) else None
            if isinstance(uses, str) and uses.strip().lower().startswith("actions/checkout"):
                return False
    return True


def scan_text(path, text):
    findings = []
    lines = text.splitlines()
    doc = None
    try:
        doc = yaml.safe_load(text)
    except yaml.YAMLError as err:
        findings.append("%s: not valid YAML (%s)" % (path, str(err).splitlines()[0]))
    if not isinstance(doc, dict):
        if not findings:
            findings.append("%s: workflow is not a YAML mapping" % path)
        doc = {}
    dependabot_target_is_safe = safe_dependabot_target(doc)
    unscoped_jobs = jobs_without_permissions(doc)
    # The parsed trigger set decides; the raw-text match stays as an extra
    # fail-closed signal for spellings the parser may resolve differently.
    target_trigger = "pull_request_target" in trigger_names(doc)
    target_lines = [i for i, line in enumerate(lines, 1) if "pull_request_target" in line]
    if target_trigger and not target_lines and not dependabot_target_is_safe:
        findings.append("%s: pull_request_target exposes secrets to untrusted PR "
                        "code. Only a Dependabot-only workflow without checkout is allowed"
                        % path)
    if unscoped_jobs:
        findings.append("%s: jobs without explicit permissions: %s. Add a global "
                        "least-privilege block or scope every job"
                        % (path, ", ".join(unscoped_jobs)))
    if COMMITLINT_RUN_RE.search(text) and not (
            edited_pull_request(doc) and TITLE_ENV_RE.search(text)):
        findings.append("%s: commitlint must lint github.event.pull_request.title (passed "
                        "through env) and trigger on pull_request type 'edited'. The PR "
                        "title is the squash subject that drives releases" % path)
    for i, line in enumerate(lines, 1):
        if UNQUOTED_NAME_COLON_RE.match(line):
            findings.append("%s:%d: name has an unquoted colon. Quote the complete "
                            "name so GitHub can parse the workflow" % (path, i))
        if "write-all" in line:
            findings.append("%s:%d: permissions: write-all grants too much access. "
                            "Limit it to the permissions this workflow needs" % (path, i))
        if i in target_lines and not dependabot_target_is_safe:
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
