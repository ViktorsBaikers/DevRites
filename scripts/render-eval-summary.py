#!/usr/bin/env python3
"""Render a GitHub-Actions-flavored markdown table from an eval summary JSONL.

Reads eval-summary.jsonl (one JSON object per line, written by
`scripts/eval-runner.py --summary-file`) and prints a markdown table to
stdout. Used by `.github/workflows/evals.yml` to populate the job summary.
"""
from __future__ import annotations

import json
import os
import pathlib
import sys


def main() -> int:
    path = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "eval-summary.jsonl")
    model = os.environ.get("DEVRITES_EVAL_MODEL", "claude-haiku-4-5-20251001")
    if not path.is_file():
        print(f"(no summary file found at {path})")
        return 0
    print(f"## DevRites trigger evals\n")
    print(f"Model: `{model}`\n")
    print("| Skill | Correct | Accuracy | FP | FN | Passed |")
    print("|---|---|---|---|---|---|")
    total_correct = total_n = total_fp = total_fn = 0
    any_failed = False
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        r = json.loads(line)
        tick = "✅" if r["passed"] else "❌"
        if not r["passed"]:
            any_failed = True
        total_correct += r["correct"]
        total_n += r["total"]
        total_fp += r["false_positives"]
        total_fn += r["false_negatives"]
        print(f"| `{r['skill']}` | {r['correct']}/{r['total']} | {r['accuracy']:.0%} | {r['false_positives']} | {r['false_negatives']} | {tick} |")
    if total_n:
        overall = total_correct / total_n
        print(f"| **overall** | **{total_correct}/{total_n}** | **{overall:.0%}** | **{total_fp}** | **{total_fn}** | {'✅' if not any_failed else '❌'} |")
    return 0


if __name__ == "__main__":
    sys.exit(main())
