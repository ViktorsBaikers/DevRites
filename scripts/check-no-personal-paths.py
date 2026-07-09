#!/usr/bin/env python3
"""Fail if shipped DevRites artifacts contain maintainer-local absolute paths."""
import os
import re
import sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
TARGETS = ["README.md", "docs", "pack", "scripts", "install.sh", "update.sh", "uninstall.sh", "package.json"]
PATTERNS = [
    re.compile(r"/Users/(?!runner|homebrew|shared/)[A-Za-z0-9._-]+"),
    re.compile(r"/home/(?!runner|circleci|github|vscode|node)[A-Za-z0-9._-]+"),
    re.compile(r"C:\\\\Users\\\\(?!runner|Public)[A-Za-z0-9._-]+", re.I),
]
TEXT_EXTS = {"", ".md", ".sh", ".py", ".js", ".mjs", ".json", ".yaml", ".yml", ".toml", ".txt"}
ALLOW = {"scripts/check-no-personal-paths.py"}

findings = []
for target in TARGETS:
    path = os.path.join(ROOT, target)
    if not os.path.exists(path):
        continue
    files = []
    if os.path.isdir(path):
        for base, dirs, names in os.walk(path):
            dirs[:] = [d for d in dirs if d not in {"node_modules", ".git", "__pycache__"}]
            files += [os.path.join(base, n) for n in names]
    else:
        files = [path]
    for file in files:
        rel = os.path.relpath(file, ROOT)
        if rel in ALLOW:
            continue
        if os.path.splitext(file)[1].lower() not in TEXT_EXTS:
            continue
        try:
            text = open(file, encoding="utf-8").read()
        except (OSError, UnicodeDecodeError):
            continue
        for lineno, line in enumerate(text.splitlines(), 1):
            if "no-personal-paths-ignore" in line:
                continue
            for rx in PATTERNS:
                if rx.search(line):
                    findings.append(f"{rel}:{lineno}: {line.strip()[:140]}")

if findings:
    print("personal-path findings:")
    print("\n".join(findings))
    sys.exit(1)
print("no-personal-paths: PASS")
