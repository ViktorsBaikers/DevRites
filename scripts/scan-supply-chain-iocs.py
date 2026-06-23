#!/usr/bin/env python3
"""Scan an npm lockfile against known supply-chain indicators of compromise.

Dependabot manages version *updates*; this gate catches a *known-malicious* installed
version (a compromised release that slipped in). It checks every name@version in the
lockfile against the bundled IOC dataset (scripts/supply-chain-iocs.json) — an offline,
always-available advisory source, so CI is deterministic and needs no network.

`--online` additionally queries the OSV API per package and is best-effort: if the
advisory source is unreachable it prints a notice and falls back to the bundled dataset
(it never fails the build on a network error — only on a real IOC match).

Usage: scan-supply-chain-iocs.py [LOCKFILE] [--iocs FILE] [--online]
  LOCKFILE defaults to package-lock.json.
Exit: 0 clean; 1 on an IOC match; 2 on a usage/parse error.
"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_IOCS = os.path.join(HERE, "supply-chain-iocs.json")


def load_iocs(path):
    with open(path, "r", encoding="utf-8") as fh:
        data = json.load(fh)
    index = {}
    for e in data.get("iocs", []):
        index[e["name"]] = e
    return index


def installed_packages(lockfile):
    """Yield (name, version) for every package in an npm lockfile (v2/v3 or v1)."""
    with open(lockfile, "r", encoding="utf-8") as fh:
        data = json.load(fh)
    pkgs = data.get("packages")
    if isinstance(pkgs, dict):  # lockfile v2/v3
        for key, meta in pkgs.items():
            if not key or "node_modules/" not in key:
                continue
            name = key.split("node_modules/")[-1]
            ver = (meta or {}).get("version")
            if name and ver:
                yield name, ver
    deps = data.get("dependencies")
    if isinstance(deps, dict):  # lockfile v1 (recursive)
        stack = list(deps.items())
        while stack:
            name, meta = stack.pop()
            meta = meta or {}
            ver = meta.get("version")
            if name and ver:
                yield name, ver
            nested = meta.get("dependencies")
            if isinstance(nested, dict):
                stack.extend(nested.items())


def query_osv(name, version):
    """Best-effort OSV lookup. Returns a finding note or None; never raises."""
    try:
        import json as _json
        import urllib.request
        body = _json.dumps({"package": {"name": name, "ecosystem": "npm"},
                            "version": version}).encode()
        req = urllib.request.Request("https://api.osv.dev/v1/query", data=body,
                                     headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=8) as r:
            vulns = _json.loads(r.read()).get("vulns", [])
        ids = [v.get("id") for v in vulns if v.get("id")]
        return ("OSV: " + ", ".join(ids)) if ids else None
    except Exception:
        return "__unreachable__"


def main(argv):
    args = [a for a in argv[1:]]
    online = "--online" in args
    args = [a for a in args if a != "--online"]
    iocs_path = DEFAULT_IOCS
    if "--iocs" in args:
        i = args.index("--iocs")
        iocs_path = args[i + 1]
        del args[i:i + 2]
    lockfile = args[0] if args else "package-lock.json"

    if not os.path.isfile(lockfile):
        print("OK    no lockfile at %s — nothing to scan" % lockfile)
        return 0
    try:
        index = load_iocs(iocs_path)
        pkgs = list(installed_packages(lockfile))
    except (OSError, ValueError, KeyError) as e:
        print("ERROR cannot scan (%s)" % e, file=sys.stderr)
        return 2

    findings = []
    for name, ver in pkgs:
        e = index.get(name)
        if e and ver in e.get("versions", []):
            findings.append("FINDING %s@%s — %s (%s)" % (name, ver, e["note"], e["source"]))

    if online:
        offline_notice = False
        for name, ver in pkgs:
            res = query_osv(name, ver)
            if res == "__unreachable__":
                offline_notice = True
                break
            if res:
                findings.append("FINDING %s@%s — %s" % (name, ver, res))
        if offline_notice:
            print("note: OSV unreachable — fell back to the bundled IOC dataset only")

    if findings:
        for f in sorted(set(findings)):
            print(f)
        print("\n%d supply-chain finding(s) across %d packages." % (len(set(findings)), len(pkgs)))
        return 1
    print("OK    supply-chain scan clean (%d packages, %d IOCs)" % (len(pkgs), len(index)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
