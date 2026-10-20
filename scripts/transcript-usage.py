#!/usr/bin/env python3
"""Sum per-dispatch model, turns, tokens, and cost from host transcripts, offline and read-only.

Usage: python3 -I scripts/transcript-usage.py --host omp|claude --transcripts <dir> <workspace>...

<workspace> is a feature workspace (.devrites/work/<slug>) holding dispatch.json.
<dir> is the host's transcript directory for the project, always given explicitly:
  omp     <dir>/<session>.jsonl is the root session, <dir>/<session>/<Handle>.jsonl a subagent
  claude  <dir>/<session>.jsonl is the root session, <dir>/<session>/subagents/agent-*.jsonl a
          subagent whose sibling agent-*.meta.json names its agentType
Each dispatch.json role joins the unused subagent transcript (same handle on omp, same agent
type on claude) whose first timestamp is nearest the role's start, within MAX_SKEW_S. Root
totals cover the root sessions that launched a joined subagent. The JSON report holds no
paths. When no dispatch joins a transcript the report says "usage": "unavailable" and the
exit is still 0. Exit 2 on bad arguments.
"""

import argparse
import datetime as dt
import json
import pathlib
import sys

# A role is recorded started right after the host launches it; a transcript that began
# further away belongs to another run.
MAX_SKEW_S = 1800
FIELDS = ("input", "cache_read", "cache_write", "output")
HOST_KEYS = {
    "omp": ("input", "cacheRead", "cacheWrite", "output"),
    "claude": ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "output_tokens"),
}


def fail(msg):
    print(f"transcript-usage: {msg}", file=sys.stderr)
    sys.exit(2)


def parse_ts(value):
    try:
        stamp = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except (AttributeError, ValueError):
        return None
    return stamp if stamp.tzinfo else stamp.replace(tzinfo=dt.timezone.utc)


def number(value):
    return value if isinstance(value, (int, float)) and not isinstance(value, bool) else 0


def records(path):
    with open(path, errors="replace") as handle:
        for line in handle:
            try:
                record = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(record, dict):
                yield record


def first_stamp(path):
    for record in records(path):
        stamp = parse_ts(record.get("timestamp"))
        if stamp:
            return stamp
    return None


def turns(path, host):
    """(model, token counts, cost) per assistant message, zero-usage messages included."""
    found = {}
    for record in records(path):
        message = record.get("message")
        if not isinstance(message, dict) or message.get("role") != "assistant":
            continue
        if record.get("type") != ("message" if host == "omp" else "assistant"):
            continue
        usage = message.get("usage") if isinstance(message.get("usage"), dict) else {}
        counts = [number(usage.get(key)) for key in HOST_KEYS[host]]
        cost = None
        if host == "omp" and isinstance(usage.get("cost"), dict):
            cost = number(usage["cost"].get("total"))
        # Claude Code writes one record per content block, each repeating the message usage.
        key = f"#{len(found)}" if host == "omp" else message.get("id") or f"#{len(found)}"
        found[key] = (str(message.get("model") or "unknown"), counts, cost)
    return list(found.values())


def empty():
    return {"runs": 0, "turns": 0, "models": {}, "input": 0, "cache_read": 0, "cache_write": 0, "output": 0, "cost": None}


def add(total, file_turns):
    total["runs"] += 1
    for model, counts, cost in file_turns:
        total["turns"] += 1
        total["models"][model] = total["models"].get(model, 0) + 1
        for field, value in zip(FIELDS, counts):
            total[field] += value
        if cost is not None:
            total["cost"] = (total["cost"] or 0) + cost


def finish(total):
    if total["cost"] is not None:
        total["cost"] = round(total["cost"], 4)
    return total


def role_key(role):
    return role[len("devrites-"):] if role.startswith("devrites-") else role


def dispatched(workspace, host):
    """(started, wave, role, join key) for every started role in the workspace's dispatch.json."""
    try:
        waves = json.loads((workspace / "dispatch.json").read_text())["waves"]
        found = []
        for wave_name, wave in waves.items():
            for role, entry in wave["roles"].items():
                started, handle = parse_ts(entry.get("started")), entry.get("handle")
                if started and handle:
                    key = handle if host == "omp" else role_key(role)
                    found.append((started, f"{workspace.name}/{wave_name}", role, key))
        return found
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        return []


def candidates(transcripts, host):
    """Every subagent transcript as (path, join key, first timestamp)."""
    if host == "omp":
        return [(path, path.stem, first_stamp(path)) for path in sorted(transcripts.glob("*/*.jsonl"))]
    found = []
    for path in sorted(transcripts.glob("*/subagents/agent-*.jsonl")):
        try:
            key = role_key(json.loads(path.with_suffix(".meta.json").read_text())["agentType"])
        except (OSError, ValueError, KeyError, TypeError, AttributeError):
            continue
        found.append((path, key, first_stamp(path)))
    return found


def join(roles, found):
    used, joined, missing = set(), [], 0
    for started, wave, role, key in sorted(roles):
        best = None
        for index, (_, cand_key, stamp) in enumerate(found):
            if index in used or cand_key != key or stamp is None:
                continue
            skew = abs((stamp - started).total_seconds())
            if skew <= MAX_SKEW_S and (best is None or skew < best[0]):
                best = (skew, index)
        if best is None:
            missing += 1
            continue
        used.add(best[1])
        joined.append((wave, role, found[best[1]][0]))
    return joined, missing


def usage(workspaces, host, transcripts):
    roles = [role for workspace in workspaces for role in dispatched(workspace, host)]
    joined, missing = join(roles, candidates(transcripts, host))
    report = {"host": host, "dispatches": {"joined": len(joined), "missing": missing}}
    if not joined:
        report.update(usage="unavailable", reason="no dispatched role joined a transcript in this layout")
        return report
    by_role, by_wave, sessions = {}, {}, set()
    for wave, role, path in joined:
        file_turns = turns(path, host)
        add(by_role.setdefault(role, empty()), file_turns)
        add(by_wave.setdefault(wave, empty()), file_turns)
        sessions.add(path.parent if host == "omp" else path.parent.parent)
    root = empty()
    for session in sorted(sessions):
        log = session.parent / (session.name + ".jsonl")
        if log.is_file():
            add(root, turns(log, host))
    report.update(
        usage="available",
        roles={role: finish(total) for role, total in by_role.items()},
        waves={wave: finish(total) for wave, total in by_wave.items()},
        root=finish(root),
    )
    return report


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", required=True, choices=("omp", "claude"))
    parser.add_argument("--transcripts", required=True)
    parser.add_argument("workspace", nargs="+")
    args = parser.parse_args()
    transcripts = pathlib.Path(args.transcripts).resolve()
    if not transcripts.is_dir():
        fail(f"transcript directory not found: {args.transcripts}")
    workspaces = [pathlib.Path(w).resolve() for w in args.workspace]
    for given, workspace in zip(args.workspace, workspaces):
        if not workspace.is_dir():
            fail(f"workspace not found: {given}")
    print(json.dumps(usage(workspaces, args.host, transcripts), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
