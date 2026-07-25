#!/usr/bin/env python3
"""Validate YAML frontmatter of DevRites SKILL.md and agent files.

Usage: validate-frontmatter.py FILE [FILE ...]
Exits non-zero if any file fails. Uses PyYAML if present, else a minimal parser
(frontmatter here is simple key: value, no nested structures needed).
"""
import sys
import re
from pathlib import Path

KNOWN_SKILL_FIELDS = {
    "name", "description", "argument-hint", "user-invocable",
    "disable-model-invocation", "required-agent-roles",
}
KNOWN_AGENT_FIELDS = {
    "name", "description", "tools", "disallowedTools", "model", "permissionMode",
    "mcpServers", "hooks", "maxTurns", "skills", "initialPrompt", "memory",
    "effort", "background", "isolation", "color",
}
DESCRIPTION_WORD_LIMITS = {
    "public": 90,
    "internal": 75,
    "library": 60,
    "explicit": 30,
}
AGENT_ROLE_RE = re.compile(r"^devrites-[a-z0-9-]+$")


def extract_frontmatter(text):
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        return None, "missing opening '---' frontmatter fence"
    body = []
    for i in range(1, len(lines)):
        if lines[i].strip() == "---":
            return "\n".join(body), None
        body.append(lines[i])
    return None, "missing closing '---' frontmatter fence"


def parse_simple(fm):
    """Minimal top-level 'key: value' parser. Good enough for these files.

    Handles YAML block scalars ('|', '>') by collecting indented continuation
    lines so multi-line values are preserved (and can be detected/rejected).
    """
    data = {}
    lines = fm.splitlines()
    i = 0
    while i < len(lines):
        raw = lines[i]
        if not raw.strip() or raw.lstrip().startswith("#"):
            i += 1
            continue
        if raw[0] in (" ", "\t"):
            i += 1
            continue
        if ":" not in raw:
            i += 1
            continue
        key, val = raw.split(":", 1)
        v = val.strip()
        # Block scalar indicators: gather indented following lines.
        if v and v[0] in ("|", ">"):
            collected = []
            j = i + 1
            while j < len(lines) and (lines[j] == "" or lines[j].startswith((" ", "\t"))):
                collected.append(lines[j].lstrip())
                j += 1
            data[key.strip()] = "\n".join(collected).rstrip()
            i = j
            continue
        data[key.strip()] = v.strip('"').strip("'")
        i += 1
    return data


def load_fm(fm):
    try:
        import yaml  # type: ignore
        d = yaml.safe_load(fm)
        if isinstance(d, dict):
            return {str(k): d[k] for k in d}
    except Exception:
        pass
    return parse_simple(fm)


def is_agent(path):
    return "/agents/" in path.replace("\\", "/")


def required_agent_roles(path, value):
    raw = str(value).strip()
    if raw == "none":
        return [], None
    if not raw:
        return [], "required-agent-roles must be 'none' or a comma-separated role list"
    roles = [role.strip() for role in raw.split(",")]
    if any(not AGENT_ROLE_RE.fullmatch(role) for role in roles):
        return [], "required-agent-roles contains an invalid role"
    if len(set(roles)) != len(roles):
        return [], "required-agent-roles contains a duplicate role"
    skill_path = Path(path).resolve()
    agents_dir = skill_path.parent.parent.parent / "agents"
    if agents_dir.is_dir():
        missing = [role for role in roles if not (agents_dir / (role + ".md")).is_file()]
        if missing:
            return [], "required-agent-roles references missing agent(s): " + ", ".join(missing)
    return roles, None


def main(argv):
    files = argv[1:]
    if not files:
        print("usage: validate-frontmatter.py FILE [FILE ...]", file=sys.stderr)
        return 2
    errors = 0
    for path in files:
        try:
            with open(path, "r", encoding="utf-8") as fh:
                text = fh.read()
        except OSError as e:
            print("ERROR %s: cannot read (%s)" % (path, e))
            errors += 1
            continue
        fm, err = extract_frontmatter(text)
        if err:
            print("ERROR %s: %s" % (path, err))
            errors += 1
            continue
        data = load_fm(fm)
        if not isinstance(data, dict) or not data:
            print("ERROR %s: frontmatter did not parse to key/value fields" % path)
            errors += 1
            continue
        if "description" not in data or not str(data.get("description", "")).strip():
            print("ERROR %s: missing/empty 'description'" % path)
            errors += 1
            continue
        known = KNOWN_AGENT_FIELDS if is_agent(path) else KNOWN_SKILL_FIELDS
        unknown = [k for k in data if k not in known]
        if unknown:
            print("ERROR %s: unknown field(s) not in canonical SKILL.md spec: %s"
                  % (path, ", ".join(unknown)))
            errors += 1
            continue
        if not is_agent(path):
            if "required-agent-roles" not in data:
                print("ERROR %s: missing 'required-agent-roles' (use 'none' when no fresh agent is mandatory)" % path)
                errors += 1
                continue
            _, role_err = required_agent_roles(path, data.get("required-agent-roles"))
            if role_err:
                print("ERROR %s: %s" % (path, role_err))
                errors += 1
                continue
        # description length cap is 1024 chars per Anthropic SKILL.md spec
        warn = ""
        desc = str(data.get("description", ""))
        dlen = len(desc)
        if dlen > 1024:
            print("ERROR %s: description %d chars > 1024 cap (Anthropic SKILL.md spec)"
                  % (path, dlen))
            errors += 1
            continue
        if "\n" in desc:
            print("ERROR %s: description must be a single line (no newlines)" % path)
            errors += 1
            continue
        desc_words = len(desc.strip().split())
        explicit_only = str(data.get("disable-model-invocation", "")).lower() == "true"
        if not is_agent(path):
            if str(data.get("name", "")) == "devrites-lib":
                budget = DESCRIPTION_WORD_LIMITS["library"]
            elif explicit_only:
                budget = DESCRIPTION_WORD_LIMITS["explicit"]
            elif str(data.get("user-invocable", "")) == "true":
                budget = DESCRIPTION_WORD_LIMITS["public"]
            else:
                budget = DESCRIPTION_WORD_LIMITS["internal"]
            if desc_words > budget:
                print("ERROR %s: description %d words > %d budget (keep triggers tight; move workflow into the body)"
                      % (path, desc_words, budget))
                errors += 1
                continue
            failed = False
            for phrase in ("Use when", "Not for"):
                count = desc.count(phrase)
                if count > 1:
                    print("ERROR %s: description repeats %r %d times (collapse duplicate trigger branches)"
                          % (path, phrase, count))
                    errors += 1
                    failed = True
                    break
            if failed:
                continue
            if explicit_only and str(data.get("name", "")) != "devrites-lib":
                for phrase in ("Use when", "Not for"):
                    if phrase in desc:
                        print("ERROR %s: explicit-only description contains %r; keep it a human summary" % (path, phrase))
                        errors += 1
                        failed = True
                        break
                if failed:
                    continue
        ui = data.get("user-invocable", "(default)")
        print("OK    %s  (user-invocable=%s)%s" % (path, ui, warn))
    if errors:
        print("\n%d file(s) failed frontmatter validation." % errors)
        return 1
    print("\nAll %d file(s) passed frontmatter validation." % len(files))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
