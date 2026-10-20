#!/usr/bin/env python3
"""Grade reviewer replay runs: per-role recall on frozen cases, one arm or a paired A/B.

Usage:
  python3 -I scripts/grade-reviewer-replay.py --cases <dir> --run <arm-dir>
  python3 -I scripts/grade-reviewer-replay.py --cases <dir> --a <arm-dir> --b <arm-dir> [--aa-noise-floor F]

Recall only. An expected finding is hit when a returned `Finding:` row names the same
path (equal, or one is a whole-segment suffix of the other) with a line within +/-10,
or cites the expected ID. Returned rows that hit nothing are counted as unlabeled,
never as false positives: a frozen case labels only the findings that were later fixed.
Exit 0 when grading completed (whatever the score), 2 on bad arguments or a malformed case.
"""

import argparse
import json
import math
import pathlib
import re
import sys

LINE_WINDOW = 10
SEVERITIES = ("Critical", "Important", "Suggestion", "Nit", "FYI")
MIN_MARGIN = 0.05
Z95 = 1.96
# A path runs to the next whitespace, row separator, quote, or bracket that cannot sit inside one.
LOCATION = re.compile(r"([^\s|:,;`\"'()]+):(\d+)")


def fail(msg):
    print(f"grade-reviewer-replay: {msg}", file=sys.stderr)
    sys.exit(2)


def load_cases(cases_dir):
    """Return {case_id: expected.json} for every case directory; raise ValueError when malformed."""
    cases = {}
    dirs = sorted(p for p in pathlib.Path(cases_dir).iterdir() if p.is_dir()) if pathlib.Path(cases_dir).is_dir() else []
    if not dirs:
        raise ValueError(f"no case directories under {cases_dir}")
    for case_dir in dirs:
        name = case_dir.name
        for part in ("diff.patch", "slice.md"):
            if not (case_dir / part).is_file() or not (case_dir / part).read_text().strip():
                raise ValueError(f"{name}: {part} is missing or empty")
        try:
            case = json.loads((case_dir / "expected.json").read_text())
        except (OSError, json.JSONDecodeError) as err:
            raise ValueError(f"{name}: expected.json: {err}") from err
        if not isinstance(case, dict) or not all(isinstance(case.get(k), str) and case[k] for k in ("role", "cluster")):
            raise ValueError(f"{name}: expected.json needs non-empty role and cluster strings")
        findings = case.get("findings")
        if not isinstance(findings, list) or not findings:
            raise ValueError(f"{name}: expected.json needs a non-empty findings list")
        for want in findings:
            ok = (
                isinstance(want, dict)
                and isinstance(want.get("file"), str) and want["file"]
                and type(want.get("line")) is int and want["line"] > 0
                and want.get("severity") in SEVERITIES
                and isinstance(want.get("id", ""), str)
                and isinstance(want.get("security", False), bool)
            )
            if not ok:
                raise ValueError(f"{name}: each finding needs file, line > 0, severity in {SEVERITIES}; optional id, security")
        cases[name] = case
    return cases


def finding_rows(text):
    rows = []
    for line in text.splitlines():
        line = line.lstrip("-* \t")
        if line.startswith("Finding:"):
            rows.append(line[len("Finding:"):])
    return rows


def same_path(got, want):
    got, want = got.removeprefix("./"), want.removeprefix("./")
    return got == want or got.endswith("/" + want) or want.endswith("/" + got)


def matches(row, want):
    if want.get("id") and re.search(rf"(?<![\w-]){re.escape(want['id'])}(?![\w-])", row):
        return True
    return any(same_path(path, want["file"]) and abs(int(line) - want["line"]) <= LINE_WINDOW for path, line in LOCATION.findall(row))


def grade_trial(text, findings):
    rows = finding_rows(text)
    hit = [any(matches(row, want) for row in rows) for want in findings]
    unlabeled = sum(not any(matches(row, want) for want in findings) for row in rows)
    return {"recall": sum(hit) / len(findings), "hit": hit, "unlabeled": unlabeled}


def grade_arm(cases, run):
    """Per case: graded trials and mean recall (None when the arm has no trial for the case)."""
    graded = {}
    for case_id, case in cases.items():
        trials = [grade_trial(p.read_text(errors="replace"), case["findings"]) for p in sorted((run / case_id).glob("trial-*.md"))]
        graded[case_id] = {"trials": trials, "recall": mean([t["recall"] for t in trials]) if trials else None}
    return graded


def mean(values):
    return sum(values) / len(values)


def clustered_se(values, clusters):
    """Clustered standard error of the mean: sqrt(sum over clusters of (sum of residuals)^2) / n."""
    center, sums = mean(values), {}
    for value, cluster in zip(values, clusters):
        sums[cluster] = sums.get(cluster, 0.0) + value - center
    return math.sqrt(sum(s * s for s in sums.values())) / len(values)


def correlation(xs, ys):
    mx, my = mean(xs), mean(ys)
    sxy = sum((x - mx) * (y - my) for x, y in zip(xs, ys))
    sxx, syy = sum((x - mx) ** 2 for x in xs), sum((y - my) ** 2 for y in ys)
    return None if sxx == 0 or syy == 0 else sxy / math.sqrt(sxx * syy)


def r4(value):
    return None if value is None else round(value, 4)


def arm_report(cases, graded, run):
    roles = {}
    for case_id, case in cases.items():
        role = roles.setdefault(case["role"], {"cases": 0, "valid_trials": 0, "unlabeled": 0, "_recalls": []})
        role["cases"] += 1
        role["valid_trials"] += len(graded[case_id]["trials"])
        role["unlabeled"] += sum(t["unlabeled"] for t in graded[case_id]["trials"])
        if graded[case_id]["recall"] is not None:
            role["_recalls"].append(graded[case_id]["recall"])
    for role in roles.values():
        recalls = role.pop("_recalls")
        role["recall"] = r4(mean(recalls)) if recalls else None
    return {
        "mode": "arm",
        "run": str(run),
        "roles": roles,
        "cases": {
            case_id: {
                "recall": r4(g["recall"]),
                "trials": [{"recall": r4(t["recall"]), "unlabeled": t["unlabeled"]} for t in g["trials"]],
            }
            for case_id, g in graded.items()
        },
    }


def must_catch_violations(case_id, case, a_trials, b_trials):
    """A Critical or security finding A catches in >= 2/3 of its trials must not fall to zero in B."""
    out = []
    for index, want in enumerate(case["findings"]):
        if want["severity"] != "Critical" and not want.get("security"):
            continue
        a_hits = sum(t["hit"][index] for t in a_trials)
        b_hits = sum(t["hit"][index] for t in b_trials)
        if 3 * a_hits >= 2 * len(a_trials) and b_hits == 0:
            out.append({"case": case_id, "file": want["file"], "line": want["line"], "a_hits": a_hits, "a_trials": len(a_trials), "b_trials": len(b_trials)})
    return out


def paired_report(cases, graded_a, graded_b, aa_noise_floor):
    margin = None if aa_noise_floor is None else max(MIN_MARGIN, aa_noise_floor)
    roles, unpaired, unpaired_roles = {}, [], set()
    by_role = {}
    for case_id, case in cases.items():
        if graded_a[case_id]["recall"] is None or graded_b[case_id]["recall"] is None:
            # A case one arm never returned (all trials timed out or errored) must not let the role pass.
            unpaired.append(case_id)
            unpaired_roles.add(case["role"])
            continue
        by_role.setdefault(case["role"], []).append(case_id)
    for role, ids in sorted(by_role.items()):
        a = [graded_a[i]["recall"] for i in ids]
        b = [graded_b[i]["recall"] for i in ids]
        diffs = [y - x for x, y in zip(a, b)]
        clusters = [cases[i]["cluster"] for i in ids]
        diff, se = mean(diffs), clustered_se(diffs, clusters)
        half = Z95 * se
        violations = [v for i in ids for v in must_catch_violations(i, cases[i], graded_a[i]["trials"], graded_b[i]["trials"])]
        if margin is None:
            gate = "needs_aa"
        elif violations or diff - half < -margin:
            gate = "fail"
        elif len(set(clusters)) < 2:
            # One cluster's residuals sum to zero, so its SE is 0 and the interval carries no information.
            gate = "insufficient_clusters"
        elif role in unpaired_roles:
            gate = "incomplete"
        else:
            gate = "pass"
        roles[role] = {
            "cases": len(ids),
            "clusters": len(set(clusters)),
            "recall_a": r4(mean(a)),
            "recall_b": r4(mean(b)),
            "diff": r4(diff),
            "se": r4(se),
            "ci_half_width": r4(half),
            "ci_lower": r4(diff - half),
            "correlation": r4(correlation(a, b)),
            "must_catch_violations": violations,
            "gate": gate,
        }
    gates = {r["gate"] for r in roles.values()} | ({"incomplete"} if unpaired else set()) | (set() if roles else {"needs_aa"})
    overall = next((g for g in ("fail", "needs_aa", "insufficient_clusters", "incomplete") if g in gates), "pass")
    return {"mode": "paired", "aa_noise_floor": aa_noise_floor, "margin": margin, "roles": roles, "unpaired": unpaired, "gate": overall}


def run_dir(value):
    path = pathlib.Path(value).resolve()
    if not path.is_dir():
        fail(f"run directory not found: {path}")
    return path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cases", default=str(pathlib.Path(__file__).resolve().parents[1] / "evals" / "replay" / "cases"))
    parser.add_argument("--run", help="grade one arm")
    parser.add_argument("--a", help="control arm run directory (paired mode)")
    parser.add_argument("--b", help="treatment arm run directory (paired mode)")
    parser.add_argument("--aa-noise-floor", type=float, help="CI half-width of a same-build A/A paired grade, as a recall fraction")
    args = parser.parse_args()
    if bool(args.run) == bool(args.a or args.b) or bool(args.a) != bool(args.b):
        fail("pass either --run, or both --a and --b")
    if args.aa_noise_floor is not None and not (math.isfinite(args.aa_noise_floor) and 0 <= args.aa_noise_floor <= 1):
        fail("--aa-noise-floor must be a fraction between 0 and 1")
    try:
        cases = load_cases(args.cases)
    except (OSError, ValueError) as err:
        fail(str(err))
    if args.run:
        run = run_dir(args.run)
        report = arm_report(cases, grade_arm(cases, run), run)
    else:
        report = paired_report(cases, grade_arm(cases, run_dir(args.a)), grade_arm(cases, run_dir(args.b)), args.aa_noise_floor)
    print(json.dumps(report, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
