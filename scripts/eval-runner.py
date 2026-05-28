#!/usr/bin/env python3
"""DevRites trigger-eval runner.

Wraps the Anthropic SDK to actually execute a trigger eval against a Claude
model. Off by default in CI (schema-only validation lives in
`scripts/run-evals.sh`); this runner is invoked explicitly when
`CLAUDE_API_KEY` is set.

Inputs
------
A trigger-eval JSON of the shape DevRites ships under `evals/`:

    {
      "skill": "rite-spec",
      "description": "...",
      "queries": [
        {"text": "...", "expected": "should_trigger"|"should_not_trigger", "rationale": "..."}
      ]
    }

Plus the pack at `pack/.claude/skills/` (so the runner can read the candidate
skills' descriptions and present them to the model).

What it does per query
----------------------
1. Loads each skill's `name` + `description` from `pack/.claude/skills/*/SKILL.md`.
2. Sends the query as a `user` message to Claude with a system prompt
   instructing it to pick exactly one skill name from the candidate list, or
   "none" if no skill applies. We do NOT ask the model to invoke a tool — we
   ask it to predict which skill the harness would have triggered. That keeps
   the eval cheap (no agentic loop, one round-trip per query) and faithful to
   what skill discovery actually decides.
3. Compares the predicted skill name to the expected verdict:
   - `should_trigger`  → predicted name must equal the eval's `skill` field.
   - `should_not_trigger` → predicted name must NOT equal the eval's `skill`.

Outputs
-------
A per-query line and a final accuracy summary. Exit code 1 if accuracy is
below `--min-accuracy` (default 0.85).

Usage
-----
    CLAUDE_API_KEY=sk-... ./scripts/eval-runner.py evals/rite-spec.json
    CLAUDE_API_KEY=sk-... ./scripts/eval-runner.py --min-accuracy 0.9 evals/*.json

The runner is intentionally thin (~200 lines). It is not the eval-viewer in
Anthropic's `skill-creator` — it just provides the missing "did the model
pick the right skill" signal so we can answer that question without firing
up the full Claude Code harness.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SKILLS_DIR = ROOT / "pack" / ".claude" / "skills"

SYSTEM_TEMPLATE = """\
You are the DevRites skill router. Your job is to predict which DevRites
skill (if any) should fire for a given user message.

Each candidate skill is described below. Read the descriptions carefully —
DevRites' triggers are encoded in them.

For the user message at the end, respond with exactly one line of the form:

    name: <skill-name>

…where <skill-name> is the name of the skill that should fire, or the
literal token `none` if no DevRites skill applies. Do not explain. Do not
suggest. Do not output anything except that one line.

# Candidate skills

{skills}
"""


@dataclass
class Skill:
    name: str
    description: str


@dataclass
class Query:
    text: str
    expected: str
    rationale: str


@dataclass
class Outcome:
    query: Query
    predicted: str
    correct: bool
    false_positive: bool  # expected should_not_trigger but skill fired
    false_negative: bool  # expected should_trigger but skill didn't fire


def load_skills() -> list[Skill]:
    skills: list[Skill] = []
    for skill_dir in sorted(SKILLS_DIR.iterdir()):
        skill_md = skill_dir / "SKILL.md"
        if not skill_md.is_file():
            continue
        body = skill_md.read_text(encoding="utf-8")
        # Lightweight YAML frontmatter parse — we only need name + description.
        m = re.match(r"^---\n(.*?)\n---", body, re.DOTALL)
        if not m:
            continue
        front = m.group(1)
        name_m = re.search(r"^name:\s*(\S+)", front, re.MULTILINE)
        desc_m = re.search(r"^description:\s*(.+)$", front, re.MULTILINE)
        if not name_m or not desc_m:
            continue
        skills.append(Skill(name=name_m.group(1), description=desc_m.group(1).strip()))
    return skills


def render_system(skills: list[Skill]) -> str:
    lines = []
    for s in skills:
        lines.append(f"- **{s.name}** — {s.description}")
    return SYSTEM_TEMPLATE.format(skills="\n".join(lines))


def load_eval(path: Path) -> tuple[str, list[Query]]:
    data = json.loads(path.read_text(encoding="utf-8"))
    queries = [
        Query(text=q["text"], expected=q["expected"], rationale=q.get("rationale", ""))
        for q in data["queries"]
    ]
    return data["skill"], queries


def predict(client, model: str, system: str, query_text: str) -> str:
    response = client.messages.create(
        model=model,
        max_tokens=64,
        system=system,
        messages=[{"role": "user", "content": query_text}],
    )
    text = "".join(block.text for block in response.content if getattr(block, "text", None))
    m = re.search(r"^\s*name:\s*([A-Za-z0-9_\-]+)", text, re.MULTILINE)
    if not m:
        return "none"
    name = m.group(1).strip().lower()
    return name


def score(target_skill: str, predicted: str, expected: str) -> bool:
    predicted = predicted.lower()
    target = target_skill.lower()
    if expected == "should_trigger":
        return predicted == target
    if expected == "should_not_trigger":
        return predicted != target
    raise ValueError(f"unknown expected verdict: {expected!r}")


def run_one(client, model: str, system: str, eval_path: Path) -> tuple[int, int, list[Outcome]]:
    target_skill, queries = load_eval(eval_path)
    outcomes: list[Outcome] = []
    correct = 0
    for q in queries:
        predicted = predict(client, model, system, q.text)
        ok = score(target_skill, predicted, q.expected)
        fp = (q.expected == "should_not_trigger" and not ok)
        fn = (q.expected == "should_trigger" and not ok)
        outcomes.append(Outcome(query=q, predicted=predicted, correct=ok, false_positive=fp, false_negative=fn))
        if ok:
            correct += 1
    return correct, len(queries), outcomes


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("eval_files", nargs="+", type=Path, help="Eval JSON files to execute")
    parser.add_argument("--model", default=os.environ.get("DEVRITES_EVAL_MODEL", "claude-haiku-4-5-20251001"))
    parser.add_argument("--min-accuracy", type=float, default=0.90,
                        help="Per-eval minimum correct/total (default: 0.90 ≈ 18/20)")
    parser.add_argument("--max-false-positives", type=int, default=2,
                        help="Per-eval max should_not_trigger queries that fired (default: 2)")
    parser.add_argument("--verbose", action="store_true", help="Print every per-query line")
    parser.add_argument("--summary-file", type=Path, default=None,
                        help="Write a machine-readable summary (one JSON line per eval) to this file")
    args = parser.parse_args()

    if not os.environ.get("CLAUDE_API_KEY"):
        sys.stderr.write("error: CLAUDE_API_KEY is not set; cannot run live eval.\n")
        return 2

    try:
        import anthropic
    except ImportError:
        sys.stderr.write(
            "error: anthropic SDK not installed. Run `pip install anthropic` first.\n"
        )
        return 2

    client = anthropic.Anthropic(api_key=os.environ["CLAUDE_API_KEY"])
    skills = load_skills()
    system = render_system(skills)

    total_correct = 0
    total_queries = 0
    total_fp = 0
    total_fn = 0
    per_file_failed = 0
    summary_lines: list[str] = []

    for eval_path in args.eval_files:
        if not eval_path.is_file():
            sys.stderr.write(f"skip: {eval_path} not a file\n")
            continue
        print(f"== {eval_path} ==")
        correct, n, outcomes = run_one(client, args.model, system, eval_path)
        fp = sum(1 for o in outcomes if o.false_positive)
        fn = sum(1 for o in outcomes if o.false_negative)
        total_correct += correct
        total_queries += n
        total_fp += fp
        total_fn += fn
        accuracy = correct / n if n else 0.0
        if args.verbose:
            for o in outcomes:
                marker = "✓" if o.correct else "✗"
                tag = "FP" if o.false_positive else ("FN" if o.false_negative else "  ")
                print(f"  {marker} {tag} [{o.query.expected:20s}] predicted={o.predicted:24s} : {o.query.text[:60]}")
        print(f"  accuracy: {correct}/{n} = {accuracy:.0%}  (FP={fp}, FN={fn})")
        failed_reasons = []
        if accuracy < args.min_accuracy:
            failed_reasons.append(f"accuracy {accuracy:.0%} < {args.min_accuracy:.0%}")
        if fp > args.max_false_positives:
            failed_reasons.append(f"false-positives {fp} > {args.max_false_positives}")
        if failed_reasons:
            per_file_failed += 1
            print(f"  FAIL: {'; '.join(failed_reasons)}")
        if args.summary_file:
            summary_lines.append(json.dumps({
                "file": str(eval_path),
                "skill": load_eval(eval_path)[0],
                "correct": correct,
                "total": n,
                "accuracy": round(accuracy, 4),
                "false_positives": fp,
                "false_negatives": fn,
                "passed": not failed_reasons,
            }))

    overall = total_correct / total_queries if total_queries else 0.0
    print()
    print(f"Overall: {total_correct}/{total_queries} = {overall:.0%}  (FP={total_fp}, FN={total_fn}, model={args.model})")
    if args.summary_file:
        args.summary_file.write_text("\n".join(summary_lines) + "\n", encoding="utf-8")
        print(f"Summary written to {args.summary_file}")
    if per_file_failed:
        print(f"{per_file_failed} eval file(s) below threshold")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
