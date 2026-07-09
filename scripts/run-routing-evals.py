#!/usr/bin/env python3
"""Deterministic DevRites routing/collision eval runner.

Zero-token lexical runner for evals/*.json. It scores skill names + descriptions
against positive/negative trigger prompts, reports rank-1/top-3 rates, direct
command misses, false-positive collisions, pairwise owner misses, host wording
confusion, and nearest skill-description collisions. Rank is a ratchet metric;
hard failures are schema, direct command misses, owner misses, and unallowlisted
severe description collisions.
"""
from __future__ import annotations

import argparse, json, math, re, sys
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SKILLS_DIR = ROOT / "pack" / ".claude" / "skills"
EVALS_DIR = ROOT / "evals"
BASELINE = EVALS_DIR / "routing-baseline.json"
TOKEN_RE = re.compile(r"[a-z0-9][a-z0-9-]*")
STOP = {"the","and","or","to","a","an","of","for","with","when","use","not","this","that","it","in","on","by","from","as","is","are","be","exactly","one"}
SIMILARITY_WARN = 0.72
SIMILARITY_FAIL = 0.88

@dataclass
class Skill:
    name: str
    description: str
    routing_text: str
    invocable: bool
    explicit_only: bool


def parse_frontmatter(text: str) -> dict[str, str]:
    if not text.startswith("---\n"):
        return {}
    end = text.find("\n---", 4)
    if end == -1:
        return {}
    fields = {}
    for line in text[4:end].splitlines():
        if not line.strip() or line.startswith((" ", "\t", "#")) or ":" not in line:
            continue
        k, v = line.split(":", 1)
        fields[k.strip()] = v.strip().strip('"\'')
    return fields


def tokens(s: str) -> list[str]:
    out = []
    for t in TOKEN_RE.findall(s.lower()):
        parts = t.split("-")
        out.append(t)
        out.extend(p for p in parts if p and p not in STOP)
    return [t for t in out if t not in STOP and len(t) > 1]


def load_skills(skills_dir: Path) -> list[Skill]:
    skills = []
    for d in sorted(skills_dir.iterdir()):
        f = d / "SKILL.md"
        if not f.is_file():
            continue
        text = f.read_text(encoding="utf-8")
        fm = parse_frontmatter(text)
        routing_lines = []
        in_trigger_section = False
        for line in text.splitlines():
            if line.startswith("## "):
                in_trigger_section = bool(re.search(r"trigger|when to use|use when", line, re.I))
                continue
            if in_trigger_section:
                if line.startswith("#"):
                    in_trigger_section = False
                else:
                    routing_lines.append(line)
        skills.append(Skill(
            fm.get("name", d.name),
            fm.get("description", ""),
            " ".join(routing_lines),
            fm.get("user-invocable") == "true",
            fm.get("disable-model-invocation") == "true",
        ))
    return skills


def load_eval_files(evals_dir: Path) -> list[Path]:
    return sorted(p for p in evals_dir.glob("*.json") if p.is_file() and p.name != "routing-baseline.json")


def corpus_vector(skill: Skill) -> Counter:
    # Explicit command forms are handled by direct_command_target. Repeating the
    # command verb here overweights generic lifecycle verbs such as plan/build.
    return Counter(tokens(f"{skill.name} {skill.name.replace('-', ' ')} {skill.description} {skill.routing_text}"))


def cosine(a: Counter, b: Counter) -> float:
    if not a or not b:
        return 0.0
    dot = sum(v * b.get(k, 0) for k, v in a.items())
    na = math.sqrt(sum(v * v for v in a.values()))
    nb = math.sqrt(sum(v * v for v in b.values()))
    return dot / (na * nb) if na and nb else 0.0


def direct_command_target(prompt: str, skill_names: set[str]) -> str | None:
    p = prompt.strip().lower()
    if p in {"/rite", "$rite", "rite"} and "rite" in skill_names:
        return "rite"
    first = p.split(maxsplit=1)[0]
    if first in skill_names:
        return first
    m = re.match(r"^[$/]rite-([a-z0-9-]+)\b", p)
    if m:
        name = f"rite-{m.group(1)}"
        return name if name in skill_names else None
    m = re.match(r"^[$/]rite\s+([a-z0-9-]+)\b", p)
    if m:
        name = f"rite-{m.group(1)}"
        return name if name in skill_names else None
    return None


def rank_prompt(prompt: str, skills: list[Skill], vectors: dict[str, Counter]) -> list[tuple[str, float]]:
    names = {s.name for s in skills}
    direct = direct_command_target(prompt, names)
    q = Counter(tokens(prompt))
    scored = []
    for s in skills:
        if s.explicit_only and direct != s.name:
            score = 0.0
        else:
            score = cosine(q, vectors[s.name])
        if direct == s.name:
            score += 10.0
        # Nudge command surface above internals for public lifecycle wording.
        if s.invocable and re.search(r"[/$]?rite|ship|seal|prove|build|review|spec|plan|vet|quick", prompt, re.I):
            score += 0.02
        scored.append((s.name, score))
    return sorted(scored, key=lambda x: (-x[1], x[0]))


def description_collisions(skills: list[Skill], vectors: dict[str, Counter]) -> list[dict]:
    rows = []
    for i, a in enumerate(skills):
        for b in skills[i+1:]:
            sim = cosine(vectors[a.name], vectors[b.name])
            if sim >= SIMILARITY_WARN:
                rows.append({"a": a.name, "b": b.name, "similarity": round(sim, 4), "hard": sim >= SIMILARITY_FAIL})
    return sorted(rows, key=lambda r: -r["similarity"])


def display_path(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)


def ratchet_failures(report: dict, baseline_path: Path | None) -> list[str]:
    if not baseline_path or not baseline_path.is_file():
        return []
    baseline = json.loads(baseline_path.read_text(encoding="utf-8"))
    failures = []
    for key in ("skills", "queries", "positive_queries", "negative_queries"):
        if key not in baseline:
            failures.append(f"baseline metadata: missing {key}")
        elif baseline[key] != report[key]:
            failures.append(f"baseline metadata: {key} is {baseline[key]}, current corpus is {report[key]}")
    recorded_at = baseline.get("recorded_at", "")
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", recorded_at):
        failures.append("baseline metadata: recorded_at must be YYYY-MM-DD")
    for key, label in (("rank1_rate", "rank-1"), ("rank_top3_rate", "top-3")):
        floor = baseline.get(f"{key}_min")
        if floor is not None and report[key] < floor:
            failures.append(f"routing ratchet: {label} {report[key]:.4f} below baseline {floor:.4f}")
    for key, label in (
        ("false_positive_collisions", "false-positive collisions"),
        ("public_vs_internal_confusion", "public/internal confusion"),
        ("host_wording_confusion", "host wording confusion"),
    ):
        ceiling = baseline.get(f"{key}_max")
        if ceiling is not None and report[key] > ceiling:
            failures.append(f"routing ratchet: {label} {report[key]} above baseline {ceiling}")
    return failures


def run(args) -> tuple[int, dict]:
    skills = load_skills(args.skills_dir)
    by_name = {s.name: s for s in skills}
    vectors = {s.name: corpus_vector(s) for s in skills}
    results = []
    hard_failures = []
    rank1 = top3 = positives = negatives = false_positive = owner_miss = host_confusion = direct_miss = internal_over_public = 0
    for path in load_eval_files(args.evals_dir):
        data = json.loads(path.read_text(encoding="utf-8"))
        target = data.get("skill")
        if target not in by_name:
            hard_failures.append(f"{path}: unknown skill {target!r}")
            continue
        for i, q in enumerate(data.get("queries", [])):
            text = q.get("text", "")
            expected = q.get("expected")
            ranked = rank_prompt(text, skills, vectors)
            names = [n for n, _ in ranked]
            rank = names.index(target) + 1 if target in names else None
            top = names[0] if names else "none"
            if expected == "should_trigger":
                positives += 1
                if rank == 1: rank1 += 1
                if rank and rank <= 3: top3 += 1
                direct = direct_command_target(text, set(by_name))
                if direct == target and rank != 1:
                    direct_miss += 1
                    hard_failures.append(f"{path}: query[{i}] direct command {text!r} ranked {rank}")
                if (text.strip().startswith("$") or "codex" in text.lower()) and rank != 1:
                    host_confusion += 1
                if top.startswith("devrites-") and target.startswith("rite-"):
                    internal_over_public += 1
            elif expected == "should_not_trigger":
                negatives += 1
                if top == target:
                    false_positive += 1
                owner = q.get("owner")
                if owner:
                    if owner not in by_name:
                        owner_miss += 1
                        hard_failures.append(f"{display_path(path)}: query[{i}] declares unknown owner {owner!r}")
                    else:
                        owner_rank = names.index(owner) + 1
                        owner_score = ranked[owner_rank - 1][1]
                        target_score = ranked[rank - 1][1] if rank else 0
                        if owner_score <= 0 or owner_rank > rank:
                            owner_miss += 1
                            hard_failures.append(
                                f"{display_path(path)}: query[{i}] declared owner {owner} does not outrank {target} "
                                f"(owner #{owner_rank} @ {owner_score:.2f}, target #{rank} @ {target_score:.2f})"
                            )
            results.append({"file": display_path(path), "skill": target, "query": text, "expected": expected, "rank": rank, "top": top, "top3": names[:3]})
    collisions = description_collisions(skills, vectors)
    for c in collisions:
        if c["hard"]:
            hard_failures.append(f"description collision {c['a']} <-> {c['b']} similarity={c['similarity']}")
    report = {
        "skills": len(skills),
        "queries": len(results),
        "positive_queries": positives,
        "negative_queries": negatives,
        "rank1_rate": round(rank1 / positives, 4) if positives else 0,
        "rank_top3_rate": round(top3 / positives, 4) if positives else 0,
        "false_positive_collisions": false_positive,
        "owner_misses": owner_miss,
        "nearest_neighbor_collisions": collisions[:25],
        "public_vs_internal_confusion": internal_over_public,
        "host_wording_confusion": host_confusion,
        "direct_command_misses": direct_miss,
        "hard_failures": hard_failures,
    }
    ratchet = ratchet_failures(report, args.baseline)
    report["routing_ratchet_failures"] = ratchet
    hard_failures.extend(ratchet)
    if args.json_out:
        args.json_out.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    if not args.quiet:
        print(f"routing evals: {rank1}/{positives} rank-1 ({report['rank1_rate']:.0%}); {top3}/{positives} top-3 ({report['rank_top3_rate']:.0%})")
        print(f"false-positive collisions: {false_positive}; owner misses: {owner_miss}; host wording confusion: {host_confusion}; public/internal confusion: {internal_over_public}")
        if collisions:
            print("nearest description collisions:")
            for c in collisions[:10]:
                print(f"  {c['similarity']:.2f} {c['a']} <-> {c['b']}{' HARD' if c['hard'] else ''}")
        if hard_failures:
            print("FAIL:")
            for f in hard_failures:
                print(f"  {f}")
        else:
            print("run-routing-evals: PASS")
    return (1 if hard_failures else 0), report


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--skills-dir", type=Path, default=SKILLS_DIR)
    p.add_argument("--evals-dir", type=Path, default=EVALS_DIR)
    p.add_argument("--json-out", type=Path)
    p.add_argument("--baseline", type=Path, default=BASELINE, help="Routing ratchet baseline JSON (default: evals/routing-baseline.json)")
    p.add_argument("--quiet", action="store_true")
    args = p.parse_args()
    code, _ = run(args)
    return code

if __name__ == "__main__":
    sys.exit(main())
