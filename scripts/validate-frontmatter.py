#!/usr/bin/env python3
"""Validate YAML frontmatter of DevRites SKILL.md and agent files.

Usage: validate-frontmatter.py FILE [FILE ...]
Exits non-zero if any file fails. Uses PyYAML if present, else a minimal parser
(frontmatter here is simple key: value, no nested structures needed).
"""
import sys

KNOWN_SKILL_FIELDS = {
    "name", "description", "argument-hint", "user-invocable",
    "disable-model-invocation",
}
KNOWN_AGENT_FIELDS = {
    "name", "description", "tools", "disallowedTools", "model", "permissionMode",
    "mcpServers", "hooks", "maxTurns", "skills", "initialPrompt", "memory",
    "effort", "background", "isolation", "color",
}


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
        if "when_to_use" in data:
            dlen += len(str(data.get("when_to_use", "")))
            if dlen > 1536:
                warn += " [warn: description+when_to_use %d>1536 chars]" % dlen
        ui = data.get("user-invocable", "(default)")
        ctx = data.get("context", "")
        flag = (" context=%s" % ctx) if ctx else ""
        print("OK    %s  (user-invocable=%s%s)%s" % (path, ui, flag, warn))
    if errors:
        print("\n%d file(s) failed frontmatter validation." % errors)
        return 1
    print("\nAll %d file(s) passed frontmatter validation." % len(files))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
