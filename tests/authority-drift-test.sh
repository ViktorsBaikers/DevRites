#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK="$ROOT/scripts/check-authority-drift.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

files=(
  engine/internal/state/workflow_manifest.json
  engine/internal/lib/readiness_contract.json
  docs/quick-reference.md
  docs/engine/state-schema.md
  docs/cli.md
  SECURITY.md
  pack/.claude/skills/devrites-lib/reference/standards/core.md
)
for file in "${files[@]}"; do
  mkdir -p "$T/$(dirname "$file")"
  cp "$ROOT/$file" "$T/$file"
done

python3 "$CHECK" --root "$T" >/dev/null

python3 - "$T/engine/internal/state/workflow_manifest.json" <<'PY'
import json, sys
p = sys.argv[1]
d = json.load(open(p))
next(x for x in d["phases"] if x["id"] == "plan")["resumeVerb"] = "define"
open(p, "w").write(json.dumps(d))
PY
if python3 "$CHECK" --root "$T" >/tmp/devrites-authority-drift-bad.txt 2>&1; then
  echo "FAIL: ambiguous Plan resume passed authority validation"
  exit 1
fi
grep -q "violates ADR-0011" /tmp/devrites-authority-drift-bad.txt

cp "$ROOT/engine/internal/state/workflow_manifest.json" "$T/engine/internal/state/workflow_manifest.json"
perl -0pi -e 's/FRAME → SPEC/FRAME → OLD-SPEC/' "$T/docs/quick-reference.md"
if python3 "$CHECK" --root "$T" >/tmp/devrites-authority-drift-doc.txt 2>&1; then
  echo "FAIL: stale lifecycle docs passed authority validation"
  exit 1
fi
grep -q "lifecycle authority block is stale" /tmp/devrites-authority-drift-doc.txt

echo "ok: authority validator rejects lifecycle routing and generated-doc drift"
