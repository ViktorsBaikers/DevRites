#!/usr/bin/env python3
"""Validate YAML frontmatter of DevRites SKILL.md and agent files.

Usage: validate-frontmatter.py FILE [FILE ...]
Exits non-zero if any file fails, or if PyYAML is not installed.
"""
import re
import sys

try:
    import yaml  # type: ignore
except ImportError:
    sys.exit("PyYAML required: pip install -r scripts/requirements-ci.txt")

KNOWN_SKILL_FIELDS = {
    "name", "description", "argument-hint", "user-invocable",
    "disable-model-invocation",
}
# agentskills.io open standard: kebab-case name, no leading/trailing/
# consecutive hyphens; Anthropic additionally reserves these substrings.
NAME_PATTERN = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
RESERVED_NAME_SUBSTRINGS = ("anthropic", "claude")
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
AGENT_DESCRIPTION_WORD_LIMIT = 45
# The host generators read agent frontmatter line by line (awk), not as YAML, so
# agent files are accepted only in exactly the grammar those readers understand.
AGENT_LINE = re.compile(r"([A-Za-z][A-Za-z0-9_-]*): (\S.*)")
AGENT_TOOLS = re.compile(r"[A-Za-z][A-Za-z0-9_*-]*(?:, [A-Za-z][A-Za-z0-9_*-]*)*")
# Only these keys may take an empty value followed by a flat '  - scalar' block list.
AGENT_LIST_KEYS = ("skills", "disallowedTools")
AGENT_LIST_HEAD = re.compile(r"([A-Za-z][A-Za-z0-9_-]*):")
AGENT_LIST_ITEM = re.compile(r"  - [A-Za-z0-9][A-Za-z0-9_.*-]*")
# The awk readers split on LF only; these characters make other line models (splitlines,
# YAML) see a line break or fence they do not, so they are rejected anywhere in the frontmatter.
AGENT_FORBIDDEN_CHAR = re.compile("[\x00-\x08\x0b-\x1f\x7f-\x9f\u2028\u2029\ufeff]")
# A value that opens a quote must close it on its own line with only whitespace after;
# block scalars ('|', '>') take their content from later lines, so they are never allowed.
AGENT_QUOTED_VALUE = re.compile(r'"(?:[^"\\]|\\.)*"\s*|\'(?:[^\']|\'\')*\'\s*')
# A value may not open with one of these: YAML reads an anchor, tag, alias, directive or flow
# collection there where the line readers see plain text.
AGENT_VALUE_START = "&*!|>%@`{["
YAML_INDICATORS = tuple("> | \" ' [ { # & * ! % @ `".split())


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


def line_value(value):
    """A value as the generators read it: one pair of outer quotes removed, escapes undone."""
    if len(value) > 1 and value[0] == value[-1] == '"':
        return re.sub(r'\\(["\\])', r"\1", value[1:-1])
    if len(value) > 1 and value[0] == value[-1] == "'":
        return value[1:-1].replace("''", "'")
    return value


def agent_grammar_error(text, fields=None):
    """Return why an agent file's frontmatter is outside the allowlist, or None.

    When `fields` is a dict it receives the key -> value (or item list) mapping the line readers see.
    """
    if "\r" in text:
        return "carriage return found; agent files must use LF line endings only"
    lines = text.split("\n")
    if lines[0] != "---":
        return "first line must be exactly '---'"
    seen = set()
    n = 1
    while n < len(lines):
        n += 1
        line = lines[n - 1]
        if AGENT_FORBIDDEN_CHAR.search(line):
            return "line %d: control or Unicode line-separator character found" % n
        if "<!--" in line:
            return "line %d: HTML comment or include marker found in frontmatter" % n
        if line == "---":
            return None
        head = AGENT_LIST_HEAD.fullmatch(line)
        if head and head.group(1) in AGENT_LIST_KEYS:
            if head.group(1) in seen:
                return "line %d: repeated key %r" % (n, head.group(1))
            seen.add(head.group(1))
            items = 0
            while n < len(lines) and lines[n].startswith(" "):
                if not AGENT_LIST_ITEM.fullmatch(lines[n]):
                    return "line %d: list items must be flat '  - scalar' lines" % (n + 1)
                if fields is not None:
                    fields.setdefault(head.group(1), []).append(lines[n][4:])
                n += 1
                items += 1
            if not items:
                return "line %d: %r needs at least one '  - scalar' item" % (n, head.group(1))
            continue
        m = AGENT_LINE.fullmatch(line)
        if not m:
            return ("line %d must be a single 'key: value' line (letter-led key, one space, "
                    "non-empty value; no blank, comment, indented, continuation or complex-key lines)" % n)
        key, value = m.groups()
        if key in seen:
            return "line %d: repeated key %r" % (n, key)
        seen.add(key)
        if value[0] in AGENT_VALUE_START:
            return ("line %d: %r must not start with the YAML indicator %r; write it as one plain "
                    "or quoted line" % (n, key, value[0]))
        if value[0] in "\"'" and not AGENT_QUOTED_VALUE.fullmatch(value):
            return "line %d: a quoted value must close its quote on the same line with nothing after it" % n
        if re.search(r"\s#", value):
            return "line %d: %r must not carry a ' #' comment" % (n, key)
        if fields is not None:
            fields[key] = line_value(value)
        if key == "tools" and not AGENT_TOOLS.fullmatch(value):
            return "line %d: 'tools' must be names joined by ', ' on one line" % n
        if key == "description" and (value[0] in YAML_INDICATORS or " #" in value):
            return "line %d: 'description' must be one plain line (no leading YAML indicator, no ' #')" % n
    return "missing closing '---' frontmatter fence"


def agent_yaml_mismatch(fields, data):
    """Return the first key whose line-reader value differs from the YAML value, or None."""
    for key in sorted(set(fields) | set(data)):
        if key not in data:
            return "%r is read by the host generators but missing from the YAML view" % key
        if key not in fields:
            return "%r is in the YAML view but not read by the host generators" % key
        line, parsed = fields[key], data[key]
        if isinstance(line, list):
            same = isinstance(parsed, list) and [str(x) for x in parsed] == line
        elif isinstance(parsed, (list, dict)):
            same = False
        else:
            same = not isinstance(parsed, str) or parsed == line
        if not same:
            return "%r reads as %r on the host generators but %r in YAML" % (key, line, parsed)
    return None


def duplicate_top_level_keys(fm):
    """Return repeated top-level frontmatter keys in first-repeat order."""
    seen = set()
    duplicates = []
    for raw in fm.splitlines():
        if not raw or raw[0].isspace() or raw.lstrip().startswith("#") or ":" not in raw:
            continue
        key = raw.split(":", 1)[0].strip()
        if len(key) >= 2 and key[0] == key[-1] and key[0] in ("'", '"'):
            key = key[1:-1]
        if key in seen and key not in duplicates:
            duplicates.append(key)
        seen.add(key)
    return duplicates


def load_fm(fm):
    try:
        d = yaml.safe_load(fm)
    except yaml.YAMLError as e:
        return None, str(e)
    if isinstance(d, dict):
        return {str(k): d[k] for k in d}, None
    return None, "frontmatter is not a mapping"


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
            with open(path, "r", encoding="utf-8", newline="") as fh:
                text = fh.read()
        except OSError as e:
            print("ERROR %s: cannot read (%s)" % (path, e))
            errors += 1
            continue
        fields = {}
        if is_agent(path):
            err = agent_grammar_error(text, fields)
            if err:
                print("ERROR %s: %s" % (path, err))
                errors += 1
                continue
        fm, err = extract_frontmatter(text)
        if err:
            print("ERROR %s: %s" % (path, err))
            errors += 1
            continue
        duplicates = duplicate_top_level_keys(fm)
        if duplicates:
            print("ERROR %s: duplicate frontmatter field(s): %s"
                  % (path, ", ".join(duplicates)))
            errors += 1
            continue
        data, fm_error = load_fm(fm)
        if fm_error:
            print("ERROR %s: frontmatter is not valid YAML: %s" % (path, fm_error))
            errors += 1
            continue
        if is_agent(path):
            err = agent_yaml_mismatch(fields, data)
            if err:
                print("ERROR %s: frontmatter differs between line view and YAML: %s" % (path, err))
                errors += 1
                continue
        if is_agent(path) and "tools" in data and not (isinstance(data["tools"], str) and data["tools"].strip()):
            print("ERROR %s: 'tools' must be a non-empty list of tool names, not %r" % (path, data["tools"]))
            errors += 1
            continue
        if not isinstance(data, dict) or not data:
            print("ERROR %s: frontmatter did not parse to key/value fields" % path)
            errors += 1
            continue
        if "description" not in data or not str(data.get("description", "")).strip():
            print("ERROR %s: missing/empty 'description'" % path)
            errors += 1
            continue
        skill_name = str(data.get("name", "")).strip()
        if not skill_name:
            print("ERROR %s: missing/empty 'name'" % path)
            errors += 1
            continue
        lowered = skill_name.lower()
        if any(res in lowered for res in RESERVED_NAME_SUBSTRINGS):
            print("ERROR %s: name %r uses a reserved substring %s" % (path, skill_name, "/".join(RESERVED_NAME_SUBSTRINGS)))
            errors += 1
            continue
        if not NAME_PATTERN.fullmatch(skill_name):
            print("ERROR %s: name %r must be lower-case a-z/0-9 joined by single hyphens" % (path, skill_name))
            errors += 1
            continue
        # Real skills are <dir>/SKILL.md and the directory owns routing/id;
        # scratch validation fixtures may use any file name, so only enforce
        # the name-to-directory match in that canonical layout.
        normalized = path.replace("\\", "/")
        parts = normalized.rsplit("/", 2)
        if parts[-1] == "SKILL.md" and len(parts) >= 2:
            parent_dir = parts[-2] if len(parts) == 3 else "."
            if parent_dir != skill_name:
                print("ERROR %s: name %r does not match parent directory %r" % (path, skill_name, parent_dir))
                errors += 1
                continue
        agent = is_agent(path)
        known = KNOWN_AGENT_FIELDS if agent else KNOWN_SKILL_FIELDS
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
        desc_words = len(desc.strip().split())
        if agent and desc_words > AGENT_DESCRIPTION_WORD_LIMIT:
            print("ERROR %s: description %d words > %d agent budget (keep role-selection cues; move workflow into the body)"
                  % (path, desc_words, AGENT_DESCRIPTION_WORD_LIMIT))
            errors += 1
            continue
        explicit_only = str(data.get("disable-model-invocation", "")).lower() == "true"
        if not agent:
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
