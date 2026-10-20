#!/usr/bin/env python3
"""Grade one finished lifecycle-benchmark run: quality from hidden probes, cost from metrics.

Usage: python3 -I scripts/grade-lifecycle-benchmark.py --benchmark <dir> --run <project-dir>

The run directory is never modified: its module is copied to a scratch directory,
the benchmark's hidden probes are added, and each probe runs on its own so one
crash cannot hide the others. Exit 0 when grading completed (whatever the score),
2 on bad arguments or a malformed benchmark.
"""

import argparse
import datetime as dt
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

PROBE_TIMEOUT_S = 180


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
        results = {}
        for probe in probes:
            cmd = ["go", "test", "-count=1"] + (["-race"] if race and probe.get("race") else [])
            cmd += ["-run", f"^{probe['test']}$", "./" + bench["probe_package"]]
            try:
                done = subprocess.run(cmd, cwd=module, env=env, capture_output=True, text=True, timeout=PROBE_TIMEOUT_S)
            except subprocess.TimeoutExpired:
                results[probe["id"]] = "fail: timeout"
                continue
            if done.returncode == 0 and "no tests to run" not in done.stdout:
                results[probe["id"]] = "pass"
            else:
                lines = (done.stdout + done.stderr).strip().splitlines()
                decisive = [l for l in lines if "DATA RACE" in l or "panic:" in l or "_test.go:" in l]
                results[probe["id"]] = "fail: " + ((decisive or lines or ["no output"])[0].strip()[:160])
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
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))


def cost(run):
    """Summarize every metrics.jsonl the run produced; None when DevRites never ran."""
    ledgers = sorted(run.glob(".devrites/work/*/metrics.jsonl")) + sorted(run.glob(".devrites/archive/*/metrics.jsonl"))
    if not ledgers:
        return None
    events = []
    for ledger in ledgers:
        for line in ledger.read_text(errors="replace").splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    events.sort(key=lambda e: e.get("ts", ""))
    open_by_role, agent_minutes, dispatches, wright, outcomes, abandoned = {}, 0.0, 0, 0, {}, 0
    events = [e for e in events if e.get("ts")]
    for event in events:
        role = event.get("role", "")
        if event.get("event") == "dispatch":
            dispatches += 1
            wright += "slice-wright" in role
            open_by_role.setdefault(role, []).append(parse_ts(event["ts"]))
        elif event.get("event") == "abandon":
            abandoned += 1
        if event.get("event") == "return" and event.get("note", "").startswith("outcome="):
            outcome = event["note"].split("=", 1)[1]
            outcomes[outcome] = outcomes.get(outcome, 0) + 1
        if event.get("event") == "return" and open_by_role.get(role):
            minutes = (parse_ts(event["ts"]) - open_by_role[role].pop()).total_seconds() / 60
            agent_minutes += minutes if minutes <= 480 else 0
    questions = sum(
        p.read_text(errors="replace").count("\n## q-")
        for p in list(run.glob(".devrites/work/*/questions.md")) + list(run.glob(".devrites/archive/*/questions.md"))
    )
    stamps = [parse_ts(e["ts"]) for e in events if e.get("ts")]
    return {
        "wall_hours": round((max(stamps) - min(stamps)).total_seconds() / 3600, 2) if stamps else 0,
        "agent_minutes": round(agent_minutes, 1),
        "dispatches": dispatches,
        "wright_dispatches": wright,
        "questions": questions,
        "return_outcomes": outcomes,
        "abandoned_waves": abandoned,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--benchmark", required=True)
    parser.add_argument("--run", required=True)
    args = parser.parse_args()
    bench_dir, run = pathlib.Path(args.benchmark).resolve(), pathlib.Path(args.run).resolve()
    try:
        bench = json.loads((bench_dir / "benchmark.json").read_text())
    except (OSError, json.JSONDecodeError) as err:
        fail(f"benchmark.json: {err}")
    if not run.is_dir():
        fail(f"run directory not found: {run}")
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

    results = run_probes(bench, run, bench["probes"])
    results.update(scope_checks(bench, run))
    kinds = {p["id"]: p["kind"] for p in bench["probes"]}
    kinds.update({c["id"]: "scope" for c in bench["unchanged_files"]})
    score = {}
    for check_id, outcome in results.items():
        kind = score.setdefault(kinds[check_id], {"passed": 0, "total": 0})
        kind["total"] += 1
        kind["passed"] += outcome == "pass"
    print(json.dumps({"benchmark": bench["name"], "results": results, "score": score, "cost": cost(run)}, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
