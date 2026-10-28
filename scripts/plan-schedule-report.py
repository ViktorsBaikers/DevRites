#!/usr/bin/env python3
"""Plan-shape counts read straight from a plan's tasks.md.

Independent of the engine: it parses the artifacts and replays the same greedy
path-disjoint scheduler, so the numbers do not depend on the schedule code that
was added to devrites-engine. Prints per-round ready/picked/idle plus totals, and
compares DAG width against the cap actually used (--cap, default 5).

Usage: plan-schedule-report.py [--cap N] <tasks.md>...
"""
import re
import sys

SLICE_RE = re.compile(r"^##\s+(SLICE-\d+)", re.M)
FIELD_RE = re.compile(r"^(Writer allowlist|Dependencies)\s*:\s*(.+)$", re.M)


def norm_paths(raw):
    out = []
    for part in re.split(r"[,;]", raw):
        p = part.strip().strip("`").strip().rstrip(".")
        if p and p.lower() != "none":
            out.append(p)
    return out


def parse(path):
    try:
        with open(path, encoding="utf-8") as fh:
            text = fh.read()
    except OSError as exc:
        sys.exit(f"cannot read {path}: {exc}")
    marks = [(m.start(), m.group(1)) for m in SLICE_RE.finditer(text)]
    slices = []
    for i, (start, sid) in enumerate(marks):
        end = marks[i + 1][0] if i + 1 < len(marks) else len(text)
        body = text[start:end]
        fields = {name: value for name, value in FIELD_RE.findall(body)}
        # Plans annotate dependencies, e.g. "SLICE-001 (via `store.Open`)".
        deps = re.findall(r"SLICE-\d+", fields.get("Dependencies", ""))
        slices.append({
            "id": sid,
            "paths": norm_paths(fields.get("Writer allowlist", "")),
            "deps": deps,
        })
    return slices


def schedule(slices, cap):
    done, rounds, idle, width = set(), [], 0, 0
    while len(done) < len(slices):
        ready = [s for s in slices if s["id"] not in done and all(d in done for d in s["deps"])]
        if not ready:
            return None
        picked, used = [], set()
        for s in ready:
            if len(picked) >= cap:
                break
            if used & set(s["paths"]):
                continue
            picked.append(s["id"])
            used |= set(s["paths"])
        if not picked:
            return None
        done |= set(picked)
        idle += len(ready) - len(picked)
        width = max(width, len(ready))
        rounds.append({"round": len(rounds) + 1, "ready": len(ready), "picked": len(picked),
                       "idle": len(ready) - len(picked), "slices": picked})
    return rounds, idle, width


def main(argv):
    cap = 5
    if argv[:1] == ["--cap"]:
        try:
            cap = int(argv[1])
        except (IndexError, ValueError):
            sys.exit("--cap needs an integer")
        argv = argv[2:]
    for path in argv:
        slices = parse(path)
        result = schedule(slices, cap)
        name = path.rstrip("/").split("/")[-2] if path.endswith("tasks.md") else path
        if result is None:
            print(f"{name}: unschedulable (cycle or missing dependency)")
            continue
        rounds, idle, ready_width = result
        picked_width = max(r["picked"] for r in rounds)
        print(f"{name}: {len(slices)} slices, {len(rounds)} rounds at cap {cap}, "
              f"achieved width {picked_width}, dag width (max ready) {ready_width}, idle-eligible {idle}")
        for r in rounds:
            print(f"  round {r['round']}: ready {r['ready']}, picked {r['picked']}, idle {r['idle']} -> {', '.join(r['slices'])}")


if __name__ == "__main__":
    main(sys.argv[1:])
