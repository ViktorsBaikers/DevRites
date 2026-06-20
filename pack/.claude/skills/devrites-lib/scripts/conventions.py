#!/usr/bin/env python3
"""DevRites conventions ledger — store + deterministic corroboration scorer.

The ledger is a project-scoped, LOCAL record (`.devrites/conventions.md`, gitignored)
of conventions a *sealed slice proved* about this codebase: build/test commands, naming /
layering / error-model idioms, recurring gotchas, where things live. `/rite-seal` promotes
to it on a GO; orient reads it so the wright stops re-deriving the same idioms every slice.

Two things live here (issue B1 modules M1 + M2):

  M1 — corroboration scorer (`band`): a pure, deterministic function from corroboration
       and contradiction counts to a 0.3–0.9 confidence band. The number is EARNED by how
       many independent sealed slices proved the convention — never guessed by a model.

  M2 — ledger store (`read` / `promote` / `contradict`): read/update `.devrites/conventions.md`,
       a human-readable Markdown file with per-entry provenance.

Confidence never raises a convention's *authority*: a high-band entry is still untrusted
data, and a fresh observation of the live code always overrides it (enforced at orient,
issue B2 — not here).

Usage:
  conventions.py band --corroborations C --contradictions K
  conventions.py read [--root DIR]
  conventions.py promote   --key K --statement S --kind KIND --slug SLUG --evidence E [--date ISO] [--root DIR]
  conventions.py contradict --key K --slug SLUG --evidence E [--date ISO] [--root DIR]

Exit codes: 0 ok; 2 usage error; 3 not-found (contradict on an unknown key).
"""
import argparse
import datetime
import os
import sys

LEDGER_RELPATH = os.path.join(".devrites", "conventions.md")

HEADER = """# DevRites conventions ledger

Conventions proven by sealed slices on this project. **Local, gitignored, advisory.**
The confidence band is earned: it reflects how many independent sealed slices corroborated
the convention — it is not a model's guess. A high band is still untrusted data; a fresh
observation of the live code always wins over a stale entry here.
"""


# --- M1: deterministic corroboration scorer -------------------------------

def band(corroborations, contradictions):
    """Pure, total, deterministic. Return a 0.3–0.9 band, or None if retired.

    First proof seeds mid-band; each further independent corroboration raises it
    toward the 0.9 cap; each contradiction lowers it; net non-positive retires it.
    """
    c = int(corroborations)
    k = int(contradictions)
    if c - k <= 0:
        return None  # retired — contradictions caught up with the evidence
    raw = 0.4 + 0.1 * c - 0.2 * k
    return max(0.3, min(0.9, round(raw, 1)))


# --- M2: ledger store -----------------------------------------------------

def _ledger_path(root):
    return os.path.join(root, LEDGER_RELPATH)


def _parse(text):
    """Parse the ledger into {key: entry}. entry = dict with fields + proofs list."""
    entries = {}
    key = None
    for line in text.splitlines():
        if line.startswith("## "):
            key = line[3:].strip()
            entries[key] = {
                "statement": "", "kind": "", "corroborations": 0,
                "contradictions": 0, "status": "active", "proofs": [],
            }
            continue
        if key is None:
            continue
        s = line.strip()
        if s.startswith("- proof:"):
            entries[key]["proofs"].append(s[len("- proof:"):].strip())
        elif s.startswith("- ") and ":" in s:
            fld, val = s[2:].split(":", 1)
            fld, val = fld.strip(), val.strip()
            if fld in ("corroborations", "contradictions"):
                try:
                    entries[key][fld] = int(val)
                except ValueError:
                    pass
            elif fld in ("statement", "kind", "status"):
                entries[key][fld] = val
    return entries


def _serialize(entries):
    out = [HEADER]
    for key in sorted(entries):
        e = entries[key]
        b = band(e["corroborations"], e["contradictions"])
        status = "retired" if b is None else "active"
        e["status"] = status
        band_str = "retired" if b is None else "%.2f" % b
        out.append("## %s" % key)
        out.append("- statement: %s" % e["statement"])
        out.append("- kind: %s" % e["kind"])
        out.append("- band: %s" % band_str)
        out.append("- corroborations: %d" % e["corroborations"])
        out.append("- contradictions: %d" % e["contradictions"])
        out.append("- status: %s" % status)
        for p in e["proofs"]:
            out.append("- proof: %s" % p)
        out.append("")
    return "\n".join(out).rstrip() + "\n"


def _load(root):
    path = _ledger_path(root)
    if not os.path.isfile(path):
        return {}
    with open(path, "r", encoding="utf-8") as fh:
        return _parse(fh.read())


def _save(root, entries):
    path = _ledger_path(root)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(_serialize(entries))


def _today(date):
    return date or datetime.date.today().isoformat()


def cmd_read(root):
    path = _ledger_path(root)
    if not os.path.isfile(path):
        return 0
    with open(path, "r", encoding="utf-8") as fh:
        sys.stdout.write(fh.read())
    return 0


def cmd_orient(root, min_band):
    """Print a compact digest of ACTIVE conventions for the wright's ORIENT step.

    These are priors, not law: high-band entries are followed unless the slice contract
    says otherwise, and a fresh observation of the live code always overrides a stale one.
    Retired entries are omitted. Empty output (exit 0) means no priors to apply.
    """
    entries = _load(root)
    rows = []
    for key, e in entries.items():
        b = band(e["corroborations"], e["contradictions"])
        if b is None or b < min_band:
            continue
        rows.append((b, key, e))
    if not rows:
        return 0
    rows.sort(key=lambda r: (-r[0], r[1]))
    print("# Conventions ledger — proven priors (live code always overrides)")
    for b, key, e in rows:
        kind = (" (%s)" % e["kind"]) if e["kind"] else ""
        print("- [%.2f] %s%s: %s" % (b, key, kind, e["statement"]))
    return 0


def cmd_promote(root, key, statement, kind, slug, evidence, date):
    entries = _load(root)
    e = entries.get(key)
    if e is None:
        e = {"statement": statement, "kind": kind, "corroborations": 0,
             "contradictions": 0, "status": "active", "proofs": []}
        entries[key] = e
    else:
        # keep the latest non-empty statement/kind
        if statement:
            e["statement"] = statement
        if kind:
            e["kind"] = kind
    # Corroboration counts DISTINCT sealed slices — re-sealing the same slice is idempotent.
    already = any(p.split(" ", 1)[0] == slug for p in e["proofs"])
    if not already:
        e["corroborations"] += 1
        e["proofs"].append("%s %s — %s" % (slug, _today(date), evidence))
    _save(root, entries)
    b = band(e["corroborations"], e["contradictions"])
    print("promoted '%s' → band %s (%d corroboration(s))"
          % (key, "retired" if b is None else "%.2f" % b, e["corroborations"]))
    return 0


def cmd_contradict(root, key, slug, evidence, date, drift_file):
    entries = _load(root)
    e = entries.get(key)
    if e is None:
        print("no such convention: '%s'" % key, file=sys.stderr)
        return 3
    e["contradictions"] += 1
    e["proofs"].append("%s %s — CONTRADICTED: %s" % (slug, _today(date), evidence))
    _save(root, entries)
    b = band(e["corroborations"], e["contradictions"])
    state = "retired" if b is None else "band %.2f" % b
    when = _today(date)
    # The orchestrator (rite-build) is the single writer of .devrites bookkeeping; it passes
    # --drift-file so the contradiction is logged to the feature's drift.md atomically here.
    if drift_file:
        os.makedirs(os.path.dirname(os.path.abspath(drift_file)), exist_ok=True)
        with open(drift_file, "a", encoding="utf-8") as fh:
            fh.write(
                "\n## convention-drift: %s\n"
                "- slice: %s\n- contradicted_by: %s\n- now: %s\n- at: %s\n"
                % (key, slug, evidence, state, when))
    print("DRIFT: convention '%s' contradicted by %s — now %s" % (key, slug, state))
    return 0


def main(argv):
    p = argparse.ArgumentParser(prog="conventions.py", add_help=True)
    sub = p.add_subparsers(dest="cmd")

    pb = sub.add_parser("band")
    pb.add_argument("--corroborations", type=int, required=True)
    pb.add_argument("--contradictions", type=int, default=0)

    pr = sub.add_parser("read")
    pr.add_argument("--root", default=".")

    po = sub.add_parser("orient")
    po.add_argument("--root", default=".")
    po.add_argument("--min-band", type=float, default=0.0)

    pp = sub.add_parser("promote")
    pp.add_argument("--key", required=True)
    pp.add_argument("--statement", required=True)
    pp.add_argument("--kind", required=True)
    pp.add_argument("--slug", required=True)
    pp.add_argument("--evidence", required=True)
    pp.add_argument("--date", default=None)
    pp.add_argument("--root", default=".")

    pc = sub.add_parser("contradict")
    pc.add_argument("--key", required=True)
    pc.add_argument("--slug", required=True)
    pc.add_argument("--evidence", required=True)
    pc.add_argument("--date", default=None)
    pc.add_argument("--root", default=".")
    pc.add_argument("--drift-file", default=None)

    args = p.parse_args(argv[1:])
    if args.cmd == "band":
        b = band(args.corroborations, args.contradictions)
        print("retired" if b is None else "%.2f" % b)
        return 0
    if args.cmd == "read":
        return cmd_read(args.root)
    if args.cmd == "orient":
        return cmd_orient(args.root, args.min_band)
    if args.cmd == "promote":
        return cmd_promote(args.root, args.key, args.statement, args.kind,
                           args.slug, args.evidence, args.date)
    if args.cmd == "contradict":
        return cmd_contradict(args.root, args.key, args.slug, args.evidence, args.date,
                              args.drift_file)
    p.print_usage(sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
