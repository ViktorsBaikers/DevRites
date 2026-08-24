#!/usr/bin/env python3
"""Thin wrapper: path-disjoint eligibility via devrites-engine (Go is SSOT).

Consumes JSON describing exact project-relative file paths per slice.
Exit 0 when every pair of slices has empty path intersection; nonzero with
a clear reason on overlap, invalid paths (incl. `.devrites/**`), or malformed
input.

Input JSON shape:
  {"slices": [{"id": "slice-1", "paths": ["src/a.go", "tests/a_test.go"]}, ...]}

Slice ids are optional; positional indices are used in error messages when omitted.
"""
from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path


def resolve_engine() -> str:
    for key in ("DEVRITES_ENGINE_CLI", "DEVRITES_ENGINE"):
        env = os.environ.get(key)
        if env and os.path.isfile(env) and os.access(env, os.X_OK):
            return env
    which = shutil.which("devrites-engine")
    if which:
        return which
    root = Path(__file__).resolve().parents[1]
    for cand in (root / "engine" / "devrites", root / "bin" / "devrites-engine"):
        if cand.is_file() and os.access(cand, os.X_OK):
            return str(cand)
    raise FileNotFoundError(
        "devrites-engine not found; set DEVRITES_ENGINE_CLI or build engine/"
    )


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Verify slice path sets are pairwise disjoint (engine SSOT)."
    )
    parser.add_argument(
        "json",
        nargs="?",
        default="-",
        help='JSON file path, or "-" for stdin (default)',
    )
    parser.add_argument(
        "--root",
        default=None,
        help="Optional repository root for symlink rejection on existing paths",
    )
    args = parser.parse_args()

    cmd = [resolve_engine(), "check", "path-disjoint"]
    if args.root:
        cmd.extend(["--root", os.path.abspath(args.root)])
    cmd.append(args.json)

    # Preserve stdin for "-" so the engine reads the same payload.
    stdin = sys.stdin if args.json == "-" else None
    try:
        proc = subprocess.run(cmd, stdin=stdin, check=False)
    except OSError as exc:
        print(f"path-disjoint: cannot execute engine: {exc}", file=sys.stderr)
        return 1
    return proc.returncode


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except FileNotFoundError as exc:
        print(f"path-disjoint: {exc}", file=sys.stderr)
        raise SystemExit(1) from None
