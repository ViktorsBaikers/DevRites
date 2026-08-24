#!/usr/bin/env bash
# Validate eval coverage against evals/coverage.json.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE="$ROOT/evals/coverage.json"

if [[ ! -f "$COVERAGE" ]]; then
  echo "missing evals/coverage.json" >&2
  exit 1
fi

python3 - "$ROOT" "$COVERAGE" <<'PY'
import json, pathlib, sys
root = pathlib.Path(sys.argv[1])
coverage = json.loads(pathlib.Path(sys.argv[2]).read_text())
failed = 0

behavioral_dir = root / "evals" / "behavioral"
behavioral = {}
for path in sorted(behavioral_dir.glob("*.json")):
    data = json.loads(path.read_text())
    behavioral[data["skill"]] = path.name

triggers = {}
for path in sorted((root / "evals").glob("*.json")):
    if path.name == "coverage.json":
        continue
    data = json.loads(path.read_text())
    triggers[data["skill"]] = path.name

for skill in coverage.get("require_behavioral", []):
    if skill not in behavioral:
        print(f"FAIL: gating skill {skill} missing behavioral eval")
        failed += 1
    else:
        print(f"OK: behavioral {skill} -> {behavioral[skill]}")

for skill in coverage.get("gating_skills", []):
    if skill not in triggers:
        print(f"WARN: gating skill {skill} missing trigger eval")
    else:
        print(f"OK: trigger {skill} -> {triggers[skill]}")

print(f"\nBehavioral gating skills covered: {len(coverage.get('require_behavioral', [])) - failed}/{len(coverage.get('require_behavioral', []))}")
sys.exit(1 if failed else 0)
PY
