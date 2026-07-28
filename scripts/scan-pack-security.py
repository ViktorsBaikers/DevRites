#!/usr/bin/env python3
"""Scan the shipped DevRites pack for prompt-injection and hidden-unicode risks.

DevRites ships executable instruction files (skills, agents, rules, hooks) into
other people's projects. A poisoned or invisibly-altered file is a supply-chain
vulnerability, not a style nit (cf. the Snyk Feb-2026 "ToxicSkills" study: 36%
of public skills carried prompt injection). This is the BLOCKING gate that keeps
such a file from ever being released.

Usage: scan-pack-security.py PATH [PATH ...]
  PATH may be a file or a directory (directories are walked for text files).
Exits non-zero if any finding is reported.

Two finding classes:

  1. Hidden unicode (always a finding: no legitimate reason to exist in a
     Markdown/shell instruction file): bidi controls, zero-width characters,
     other invisibles, and homoglyph confusables (a word that mixes ASCII with
     look-alike Cyrillic/Greek letters).

  2. Prompt-injection patterns (a finding unless explicitly suppressed): the
     "ignore previous instructions" family, system-prompt overrides, tool /
     permission escalation, and data-exfiltration phrasing.

Suppression (for DevRites' own files that legitimately *discuss* injection
defensively: e.g. security.md, the reviewer agents): put a marker on the same
line OR anywhere in the file:

    ... ignore previous instructions ...   <!-- pack-scan-ignore: defensive doc -->

A whole file can opt a class out with a top-of-file marker:

    <!-- pack-scan-ignore-file: injection -->

Suppressions live in the file, so every exception is visible in the diff and
reviewable: consistent with DevRites' "no silent state" thesis. Hidden-unicode
findings can be suppressed the same way but should essentially never be.
"""
import os
import re
import sys
import unicodedata

# --- hidden unicode -------------------------------------------------------

# Invisible / formatting code points with no business in an instruction file.
ZERO_WIDTH = {
    0x200B,  # zero width space
    0x200C,  # zero width non-joiner
    0x200D,  # zero width joiner
    0x2060,  # word joiner
    0xFEFF,  # zero width no-break space / BOM (mid-file)
    0x180E,  # mongolian vowel separator
    0x00AD,  # soft hyphen
}
BIDI_CONTROLS = {
    0x202A, 0x202B, 0x202C, 0x202D, 0x202E,  # LRE RLE PDF LRO RLO
    0x2066, 0x2067, 0x2068, 0x2069,          # LRI RLI FSI PDI
    0x200E, 0x200F,                          # LRM RLM
}
HIDDEN = ZERO_WIDTH | BIDI_CONTROLS

# Scripts whose letters are commonly used as ASCII look-alikes.
CONFUSABLE_SCRIPTS = ("CYRILLIC", "GREEK")


def _char_script(ch):
    try:
        return unicodedata.name(ch).split(" ")[0]
    except ValueError:
        return ""


def find_hidden_unicode(text):
    """Yield (line_no, col, label) for hidden code points and homoglyph words."""
    for lineno, line in enumerate(text.splitlines(), start=1):
        for col, ch in enumerate(line, start=1):
            cp = ord(ch)
            if cp == 0xFEFF and lineno == 1 and col == 1:
                continue  # a leading BOM is legal; only mid-file ZWNBSP is suspect
            if cp in HIDDEN:
                yield lineno, col, "hidden char U+%04X (%s)" % (cp, _hidden_name(cp))
        # Homoglyph: a token mixing ASCII letters with confusable-script letters.
        for m in re.finditer(r"\S+", line):
            tok = m.group(0)
            has_ascii_alpha = any(("a" <= c.lower() <= "z") for c in tok)
            mixed = [c for c in tok if c.isalpha() and _char_script(c) in CONFUSABLE_SCRIPTS]
            if has_ascii_alpha and mixed:
                yield lineno, m.start() + 1, (
                    "homoglyph: ASCII word mixes %s look-alike(s) %r" % (
                        _char_script(mixed[0]).title(), "".join(sorted(set(mixed)))))


def _hidden_name(cp):
    try:
        return unicodedata.name(chr(cp))
    except ValueError:
        return "unnamed"


# --- prompt-injection patterns -------------------------------------------

# Each: (label, compiled regex). Kept high-signal to limit false positives on
# legitimate prose; defensive discussion is handled via suppression markers.
# Gaps use '.' (any non-newline char): scanning is per-line, so a match can't
# cross lines, and excluding periods would miss tokens like ".env" / ".ssh".
_INJECTION = [
    ("ignore-previous-instructions",
     r"\b(ignore|disregard|forget)\b.{0,40}\b(previous|prior|above|earlier|all)\b"
     r".{0,20}\b(instruction|instructions|prompt|prompts|direction|directions|context)\b"),
    ("system-prompt-override",
     r"\b(you are now|act as|pretend to be|new instructions\s*:|override (your|the) (system|prompt|rules))"),
    ("permission-escalation",
     r"\b(bypass|ignore|disable|skip)\b.{0,30}\b(approval|permission|permissions|guardrail|guardrails|safety|sandbox|restriction|restrictions)\b"),
    ("data-exfiltration",
     r"\b(exfiltrat\w+|send|post|upload|leak|curl|wget)\b.{0,40}\b(env|environment|secret|secrets|credential|credentials|token|tokens|api[_-]?key|\.ssh|private key)\b"),
]
INJECTION = [(label, re.compile(rx, re.IGNORECASE)) for label, rx in _INJECTION]


def find_injection(text, file_suppressed):
    """Yield (line_no, label, excerpt) for injection patterns, honoring suppression."""
    for lineno, line in enumerate(text.splitlines(), start=1):
        if _line_suppressed(line):
            continue
        for label, rx in INJECTION:
            if "injection" in file_suppressed:
                break
            m = rx.search(line)
            if m:
                yield lineno, label, line.strip()[:120]


_LINE_SUPPRESS = re.compile(r"pack-scan-ignore\b(?!-file)", re.IGNORECASE)
# Class list is word/comma/space only: a trailing "-->" comment closer must not
# get swallowed into the captured class name.
_FILE_SUPPRESS = re.compile(r"pack-scan-ignore-file\s*:\s*([\w, ]+)", re.IGNORECASE)


def _line_suppressed(line):
    return bool(_LINE_SUPPRESS.search(line))


def _file_suppressions(text):
    classes = set()
    for m in _FILE_SUPPRESS.finditer(text):
        for c in m.group(1).split(","):
            classes.add(c.strip().lower())
    return classes


# --- driver ---------------------------------------------------------------

TEXT_EXTS = {".md", ".sh", ".json", ".toml", ".txt", ".py", ".js", ".yaml", ".yml", ""}


def iter_files(paths):
    for p in paths:
        if os.path.isdir(p):
            for root, _dirs, names in os.walk(p):
                for n in sorted(names):
                    fp = os.path.join(root, n)
                    ext = os.path.splitext(n)[1].lower()
                    if ext in TEXT_EXTS:
                        yield fp
        else:
            yield p


def scan(paths):
    """Return a list of finding strings. Empty list == clean."""
    findings = []
    for path in iter_files(paths):
        try:
            with open(path, "r", encoding="utf-8") as fh:
                text = fh.read()
        except (OSError, UnicodeDecodeError) as e:
            findings.append("ERROR %s: cannot read as utf-8 (%s)" % (path, e))
            continue
        file_sup = _file_suppressions(text)
        hidden_off = ("unicode" in file_sup) or ("hidden" in file_sup)
        if not hidden_off:
            for lineno, col, label in find_hidden_unicode(text):
                if _line_suppressed(text.splitlines()[lineno - 1]):
                    continue
                findings.append("FINDING %s:%d:%d: %s" % (path, lineno, col, label))
        for lineno, label, excerpt in find_injection(text, file_sup):
            findings.append("FINDING %s:%d: injection/%s: %s" % (path, lineno, label, excerpt))
    return findings


def main(argv):
    paths = argv[1:]
    if not paths:
        print("usage: scan-pack-security.py PATH [PATH ...]", file=sys.stderr)
        return 2
    findings = scan(paths)
    if findings:
        for f in findings:
            print(f)
        print("\n%d security finding(s). Fix, or add an auditable "
              "pack-scan-ignore marker if this is defensive content." % len(findings))
        return 1
    print("OK    pack security scan clean (%s)" % ", ".join(paths))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
