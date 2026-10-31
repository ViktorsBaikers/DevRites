#!/usr/bin/env python3
"""Grade finished lifecycle-benchmark runs: quality from hidden probes, cost from the ledgers.

Usage:
  python3 -I scripts/grade-lifecycle-benchmark.py --benchmark <dir> --run <project-dir>
      [--transcripts <dir> --host omp|claude]
  python3 -I scripts/grade-lifecycle-benchmark.py --benchmark <dir>
      --baseline <project-dir>... --candidate <project-dir>...
      [--baseline-label <host/model> --candidate-label <host/model>]

The run directory is never modified: its module is copied to a scratch directory,
the benchmark's hidden probes are added, and each probe runs on its own so one
crash cannot hide the others. --transcripts adds a tokens block to cost (see
scripts/transcript-usage.py). Compare mode grades at least MIN_RUNS distinct runs
per side and accepts the candidate only when every probe passed in every candidate
run and the median of a COST_FIELDS entry dropped. Exit 0 when grading completed
(whatever the score or verdict), 2 on bad arguments or a malformed benchmark.
"""

import argparse
import datetime as dt
import importlib.util
import json
import os
import pathlib
import shutil
import statistics
import subprocess
import sys
import tempfile
import time

PROBE_TIMEOUT_S = 180
MIN_RUNS = 3
COST_FIELDS = ("context_bytes", "agent_minutes", "root_only_minutes")
# The question gates that pause the run: blocking/escalating always take the checkpoint
# protocol; validating and advisory let the loop continue, so they never hold a run.
BLOCKING_GATES = ("blocking", "escalating")


def fail(msg):
    print(f"grade-lifecycle-benchmark: {msg}", file=sys.stderr)
    sys.exit(2)


def run_probes(bench, run, probes):
    with tempfile.TemporaryDirectory() as tmp:
        module = pathlib.Path(tmp) / "module"
        shutil.copytree(run, module, ignore=shutil.ignore_patterns(".git", ".devrites", "node_modules"))
        package = module / bench["probe_package"]
        if not package.is_dir():
            return {p["id"]: "fail: probe package missing" for p in probes}
        shutil.copy(bench["_dir"] / "probes" / bench["probe_file"], package / bench["probe_file"])
        # The race detector needs cgo; without a C compiler the concurrency probe
        # still runs, but only a crash or a wrong count can fail it.
        race = shutil.which("cc") is not None or shutil.which("gcc") is not None
        # Agent-written code runs here: pass only what the Go toolchain needs,
        # never the grader's tokens or keys.
        keep = ("PATH", "HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE", "GOROOT", "GOTOOLCHAIN")
        env = {k: os.environ[k] for k in keep if k in os.environ}
        env.update(GOWORK="off", CGO_ENABLED="1" if race else "0")

        def execute(cmd, cwd, timeout=PROBE_TIMEOUT_S):
            if timeout <= 0:
                return "fail: timeout"
            try:
                done = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True, timeout=timeout)
            except subprocess.TimeoutExpired:
                return "fail: timeout"
            output = done.stdout + done.stderr
            if done.returncode == 0 and "no tests to run" not in output:
                return "pass"
            lines = output.strip().splitlines()
            decisive = [l for l in lines if "DATA RACE" in l or "panic:" in l or "_test.go:" in l or "no tests to run" in l]
            return "fail: " + ((decisive or lines or ["no output"])[0].strip()[:160])

        builds, results = {}, {}
        for probe in probes:
            use_race = bool(race and probe.get("race"))
            binary = pathlib.Path(tmp) / ("probes-race.test" if use_race else "probes.test")
            if os.name == "nt":
                binary = binary.with_suffix(binary.suffix + ".exe")
            if use_race not in builds:
                cmd = ["go", "test", "-c", "-o", str(binary)] + (["-race"] if use_race else [])
                started = time.monotonic()
                outcome = execute(cmd + ["./" + bench["probe_package"]], module)
                builds[use_race] = (outcome, time.monotonic() - started)
            outcome, build_seconds = builds[use_race]
            if outcome == "pass":
                # go test runs its executable in the package directory, not the module root.
                outcome = execute(
                    [str(binary), "-test.run", f"^{probe['test']}$", "-test.count=1"],
                    package, timeout=PROBE_TIMEOUT_S - build_seconds,
                )
            results[probe["id"]] = outcome
        return results


def scope_checks(bench, run):
    results = {}
    for check in bench["unchanged_files"]:
        original = bench["_dir"] / "fixture" / check["path"]
        current = run / check["path"]
        same = current.is_file() and current.read_bytes() == original.read_bytes()
        results[check["id"]] = "pass" if same else f"fail: {check['path']} changed"
    return results


def parse_ts(value):
    stamp = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    return stamp if stamp.tzinfo else stamp.replace(tzinfo=dt.timezone.utc)


def workspaces(run):
    return [p for p in sorted(run.glob(".devrites/work/*")) + sorted(run.glob(".devrites/archive/*")) if p.is_dir()]


def covered_minutes(intervals, start, end):
    """Minutes of [start, end] inside the union of the intervals."""
    seconds, reach = 0.0, start
    for begin, finish in sorted((max(a, start), min(b, end)) for a, b in intervals):
        begin = max(begin, reach)
        if finish > begin:
            seconds += (finish - begin).total_seconds()
            reach = finish
    return seconds / 60


def dispatch_intervals(workspace):
    """(started, returned_at) of every returned role in dispatch.json; an unreturned role has no end."""
    try:
        waves = json.loads((workspace / "dispatch.json").read_text())["waves"]
        return [
            (parse_ts(role["started"]), parse_ts(role["returned_at"]))
            for wave in waves.values()
            for role in wave["roles"].values()
            if role.get("started") and role.get("returned_at")
        ]
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        return []


def question_waits(text, end):
    """(raised_at, answered_at) of every blocking question; one still open waits until end."""
    waits, block = [], None
    for line in text.splitlines() + ["## end"]:
        if line.startswith("## "):
            if block and block.get("raised_at") and block.get("gate") in BLOCKING_GATES:
                waits.append((block["raised_at"], block.get("answered_at") or end))
            block = {} if line.lower().startswith("## q-") else None
        elif block is not None and ":" in line:
            key, value = (part.strip() for part in line.split(":", 1))
            if key.lower() == "gate":
                block["gate"] = value.lower()
            elif key.lower() in ("raised_at", "answered_at"):
                try:
                    block[key.lower()] = parse_ts(value)
                except ValueError:
                    pass
    return waits


def cost(run):
    """Summarize the run's ledgers; None when DevRites never ran.

    agent_minutes sums dispatch.json role intervals; active_minutes is wall time
    outside blocking question waits; root_only_minutes is active time with no
    dispatched agent running.
    """
    dirs = workspaces(run)
    ledgers = [d / "metrics.jsonl" for d in dirs if (d / "metrics.jsonl").is_file()]
    if not ledgers:
        return None
    events = []
    for ledger in ledgers:
        for line in ledger.read_text(errors="replace").splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    events = [e for e in events if e.get("ts")]
    dispatches, wright, outcomes, abandoned, context_bytes = 0, 0, {}, 0, 0
    for event in events:
        role = event.get("role", "")
        if event.get("event") == "dispatch":
            dispatches += 1
            wright += "slice-wright" in role
        elif event.get("event") == "abandon":
            abandoned += 1
        elif event.get("event") == "context":
            context_bytes += event.get("bytes", 0)
        if event.get("event") == "return" and event.get("note", "").startswith("outcome="):
            outcome = event["note"].split()[0].split("=", 1)[1]
            outcomes[outcome] = outcomes.get(outcome, 0) + 1
    stamps = [parse_ts(e["ts"]) for e in events]
    now = dt.datetime.now(dt.timezone.utc)
    start, end = min(stamps, default=now), max(stamps, default=now)
    intervals = [i for d in dirs for i in dispatch_intervals(d)]
    texts = [(d / "questions.md").read_text(errors="replace") for d in dirs if (d / "questions.md").is_file()]
    waits = [w for text in texts for w in question_waits(text, end)]
    wall = (end - start).total_seconds() / 60
    active = wall - covered_minutes(waits, start, end)
    root_only = wall - covered_minutes(waits + intervals, start, end)
    return {
        "wall_hours": round(wall / 60, 2),
        "agent_minutes": round(sum((b - a).total_seconds() for a, b in intervals) / 60, 1),
        "active_minutes": round(active, 1),
        "root_only_minutes": round(root_only, 1),
        "root_only_fraction": round(root_only / active, 2) if active > 0 else None,
        "context_bytes": context_bytes,
        "dispatches": dispatches,
        "wright_dispatches": wright,
        "questions": sum(text.count("\n## q-") for text in texts),
        "return_outcomes": outcomes,
        "abandoned_waves": abandoned,
    }


def transcript_tokens(run, host, transcripts):
    path = pathlib.Path(__file__).resolve().with_name("transcript-usage.py")
    spec = importlib.util.spec_from_file_location("transcript_usage", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.usage(workspaces(run), host, transcripts)


def grade(bench, run, host=None, transcripts=None):
    results = run_probes(bench, run, bench["probes"])
    results.update(scope_checks(bench, run))
    kinds = {p["id"]: p["kind"] for p in bench["probes"]}
    kinds.update({c["id"]: "scope" for c in bench["unchanged_files"]})
    score = {}
    for check_id, outcome in results.items():
        kind = score.setdefault(kinds[check_id], {"passed": 0, "total": 0})
        kind["total"] += 1
        kind["passed"] += outcome == "pass"
    summary = cost(run)
    if summary is not None and transcripts:
        summary["tokens"] = transcript_tokens(run, host, transcripts)
    return {"benchmark": bench["name"], "results": results, "score": score, "cost": summary}


def spread(values):
    return {"median": statistics.median(values), "min": min(values), "max": max(values)} if values else None


def compare(bench, sides, label):
    reports = {side: [grade(bench, run) for run in runs] for side, runs in sides.items()}
    passes, costs = {}, {}
    for side, side_reports in reports.items():
        for report in side_reports:
            for check_id, outcome in report["results"].items():
                passes.setdefault(check_id, {"baseline": 0, "candidate": 0})[side] += outcome == "pass"
            for field, value in (report["cost"] or {}).items():
                if isinstance(value, (int, float)) and not isinstance(value, bool):
                    costs.setdefault(field, {"baseline": [], "candidate": []})[side].append(value)
    runs = {side: len(side_reports) for side, side_reports in reports.items()}
    summary = {field: {side: spread(values) for side, values in by_side.items()} for field, by_side in costs.items()}
    reasons = [
        f"{check_id} passed in {count['candidate']} of {runs['candidate']} candidate runs"
        for check_id, count in sorted(passes.items())
        if count["candidate"] < runs["candidate"]
    ]
    # A run without a ledger would leave the cost medians resting on fewer than MIN_RUNS values.
    reasons += [
        f"{side} run {i + 1} has no metrics ledger"
        for side, side_reports in reports.items()
        for i, report in enumerate(side_reports)
        if report["cost"] is None
    ]
    drops = [
        field
        for field in COST_FIELDS
        if summary.get(field, {}).get("baseline")
        and summary[field].get("candidate")
        and summary[field]["candidate"]["median"] < summary[field]["baseline"]["median"]
    ]
    if not drops:
        reasons.append("no median drop in " + ", ".join(COST_FIELDS))
    return {
        "benchmark": bench["name"],
        "label": label,
        "runs": runs,
        "passes": passes,
        "cost": summary,
        "cost_drops": drops,
        "verdict": "reject" if reasons else "accept",
        "reasons": reasons,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--benchmark", required=True)
    parser.add_argument("--run")
    parser.add_argument("--transcripts")
    parser.add_argument("--host", choices=("omp", "claude"))
    parser.add_argument("--baseline", nargs="+")
    parser.add_argument("--candidate", nargs="+")
    parser.add_argument("--baseline-label")
    parser.add_argument("--candidate-label")
    args = parser.parse_args()
    comparing = args.baseline is not None or args.candidate is not None
    if comparing == (args.run is not None):
        fail("give --run, or --baseline and --candidate")
    if bool(args.transcripts) != bool(args.host):
        fail("--transcripts and --host go together")
    if comparing:
        if args.transcripts:
            fail("--transcripts grades one --run")
        for side in ("baseline", "candidate"):
            if len(getattr(args, side) or []) < MIN_RUNS:
                fail(f"--{side} needs at least {MIN_RUNS} runs")
        if (args.baseline_label or args.candidate_label) and args.baseline_label != args.candidate_label:
            fail(f"labels differ: baseline {args.baseline_label!r}, candidate {args.candidate_label!r}")
        sides = {side: [pathlib.Path(r).resolve() for r in getattr(args, side)] for side in ("baseline", "candidate")}
        runs = sides["baseline"] + sides["candidate"]
        if len(set(runs)) != len(runs):
            fail("a run directory is listed more than once")
    else:
        if args.baseline_label or args.candidate_label:
            fail("labels apply to --baseline and --candidate")
        runs = [pathlib.Path(args.run).resolve()]
    for run in runs:
        if not run.is_dir():
            fail(f"run directory not found: {run}")
    transcripts = pathlib.Path(args.transcripts).resolve() if args.transcripts else None
    if transcripts and not transcripts.is_dir():
        fail(f"transcript directory not found: {transcripts}")

    bench_dir = pathlib.Path(args.benchmark).resolve()
    try:
        bench = json.loads((bench_dir / "benchmark.json").read_text())
    except (OSError, json.JSONDecodeError) as err:
        fail(f"benchmark.json: {err}")
    try:
        for key in ("name", "probe_package", "probe_file", "probes", "unchanged_files"):
            bench[key]
        for probe in bench["probes"]:
            probe["id"], probe["kind"], probe["test"]
        for check in bench["unchanged_files"]:
            check["id"], check["path"]
    except (KeyError, TypeError) as err:
        fail(f"benchmark.json is missing {err}")
    if not (bench_dir / "probes" / bench["probe_file"]).is_file():
        fail(f"probe file missing: probes/{bench['probe_file']}")
    bench["_dir"] = bench_dir

    if comparing:
        report = compare(bench, sides, args.baseline_label)
    else:
        report = grade(bench, runs[0], args.host, transcripts)
    print(json.dumps(report, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
