#!/usr/bin/env python3
"""Check that slice path sets are pairwise path-disjoint.

Consumes JSON describing exact project-relative file paths per slice.
Exit 0 when every pair of slices has empty path intersection; exit 1 with
a clear reason on overlap, invalid paths, or malformed input.

Input JSON shape:
  {"slices": [{"id": "slice-1", "paths": ["src/a.go", "tests/a_test.go"]}, ...]}

Slice ids are optional; positional indices are used in error messages when omitted.
"""
import argparse
import json
import os
import re
import sys

WIN_ABS_RE = re.compile(r"^[A-Za-z]:[/\\]")


def normalize_path(raw: str) -> str:
    if not isinstance(raw, str):
        raise ValueError(f"path must be a string, got {type(raw).__name__}")
    path = raw.replace("\\", "/").strip()
    if not path:
        raise ValueError("empty path is not allowed")
    if path.startswith("/") or WIN_ABS_RE.match(path):
        raise ValueError(f"path must be project-relative, not absolute: {raw!r}")
    parts = [part for part in path.split("/") if part not in ("", ".")]
    if any(part == ".." for part in parts):
        raise ValueError(f"path must not contain '..': {raw!r}")
    return "/".join(parts)


def validate_slice(paths, slice_label, root=None):
    if not isinstance(paths, list):
        raise ValueError(f"{slice_label}: paths must be a list")
    normalized = []
    seen = set()
    for raw in paths:
        path = normalize_path(raw)
        if path in seen:
            raise ValueError(f"{slice_label}: duplicate path {path!r}")
        seen.add(path)
        normalized.append(path)
        if root is not None:
            full = os.path.join(root, path)
            if os.path.islink(full):
                raise ValueError(f"{slice_label}: symlink path is not allowed: {path!r}")
    return normalized


def check_disjoint(slices, root=None):
    if not isinstance(slices, list):
        raise ValueError("slices must be a list")
    if len(slices) < 2:
        raise ValueError("need at least two slices to check path-disjoint eligibility")

    owners = {}
    for index, item in enumerate(slices):
        if not isinstance(item, dict):
            raise ValueError(f"slice {index}: must be an object with a paths list")
        slice_id = item.get("id")
        slice_label = f"slice {slice_id!r}" if slice_id else f"slice {index}"
        paths = validate_slice(item.get("paths"), slice_label, root=root)
        for path in paths:
            owners.setdefault(path, []).append(slice_label)

    overlaps = {path: labels for path, labels in owners.items() if len(labels) > 1}
    if overlaps:
        details = "; ".join(
            f"{path!r} shared by {', '.join(labels)}" for path, labels in sorted(overlaps.items())
        )
        raise ValueError(f"path sets overlap: {details}")

    return [item.get("id") or str(index) for index, item in enumerate(slices)]


def read_input_text(path):
    if path == "-":
        try:
            return sys.stdin.read()
        except OSError as exc:
            raise ValueError(f"cannot read stdin: {exc}") from exc

    normalized = os.path.normpath(path)
    if normalized == ".." or normalized.startswith(f"..{os.sep}") or f"{os.sep}.." in normalized:
        raise ValueError(f"input path must not contain '..': {path!r}")

    try:
        with open(normalized, encoding="utf-8") as handle:
            return handle.read()
    except OSError as exc:
        raise ValueError(f"cannot read input {path!r}: {exc}") from exc


def load_payload(path):
    text = read_input_text(path)

    try:
        data = json.loads(text)
    except json.JSONDecodeError as exc:
        raise ValueError(f"invalid JSON: {exc}") from exc

    if isinstance(data, list):
        return data
    if isinstance(data, dict) and "slices" in data:
        return data["slices"]
    raise ValueError('input must be {"slices": [...]} or a top-level slices list')


def main():
    parser = argparse.ArgumentParser(description="Verify slice path sets are pairwise disjoint.")
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
    root = os.path.abspath(args.root) if args.root else None
    if root is not None and not os.path.isdir(root):
        raise ValueError(f"--root is not a directory: {root}")

    slice_ids = check_disjoint(load_payload(args.json), root=root)
    print(f"path-disjoint: ok ({len(slice_ids)} slices)")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"path-disjoint: {exc}", file=sys.stderr)
        raise SystemExit(1) from None
