#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK="$ROOT/scripts/check-authority-drift.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

files=(
  engine/internal/state/workflow_manifest.json
  docs/quick-reference.md
  docs/engine/state-schema.md
  docs/engine/workspace-schema.md
  CONTEXT.md
  docs/architecture.md
  docs/flow.md
  SECURITY.md
  pack/.claude/skills/devrites-lib/reference/standards/core.md
  pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md
  pack/.claude/skills/rite-vet/reference/depth.md
  pack/.claude/skills/rite-temper/reference/significance.md
  pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md
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
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-bad.txt 2>&1; then
  echo "FAIL: ambiguous Plan resume passed authority validation"
  exit 1
fi
grep -q "violates ADR-0011" $T/devrites-authority-drift-bad.txt

cp "$ROOT/engine/internal/state/workflow_manifest.json" "$T/engine/internal/state/workflow_manifest.json"
perl -0pi -e 's/FRAME → SPEC/FRAME → OLD-SPEC/' "$T/docs/quick-reference.md"
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-doc.txt 2>&1; then
  echo "FAIL: stale lifecycle docs passed authority validation"
  exit 1
fi
grep -q "lifecycle authority block is stale" $T/devrites-authority-drift-doc.txt
grep -q "authority block is stale; run python3 scripts/check-authority-drift.py --write$" $T/devrites-authority-drift-doc.txt

cp "$ROOT/docs/quick-reference.md" "$T/docs/quick-reference.md"
python3 - "$T/docs/engine/state-schema.md" <<'PY'
import re, sys
p = sys.argv[1]
text = open(p).read()
mutated, count = re.subn(r"state schema \(v?\d+\)", "state schema (v3)", text, count=1)
assert count == 1, "state-schema title claim not found"
open(p, "w").write(mutated)
PY
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-title.txt 2>&1; then
  echo "FAIL: stale (v3) schema title passed authority validation"
  exit 1
fi
grep -q "state-schema.md" $T/devrites-authority-drift-title.txt
grep -q "claims schema v3" $T/devrites-authority-drift-title.txt

cp "$ROOT/docs/engine/state-schema.md" "$T/docs/engine/state-schema.md"
printf '\nRefused: a workspace claiming schema v9 is not readable.\n' >> "$T/CONTEXT.md"
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-schema-vn.txt 2>&1; then
  echo "FAIL: 'schema v9' prose passed authority validation"
  exit 1
fi
grep -q "CONTEXT.md" $T/devrites-authority-drift-schema-vn.txt
grep -q "claims schema v9" $T/devrites-authority-drift-schema-vn.txt

cp "$ROOT/CONTEXT.md" "$T/CONTEXT.md"
python3 - "$T/CONTEXT.md" <<'PY'
import re, sys
p = sys.argv[1]
text = open(p).read()
mutated, count = re.subn(r"schema\s+is\s+v?\d+", "schema\n  is v3", text, count=1)
assert count == 1, "CONTEXT.md schema claim not found"
open(p, "w").write(mutated)
PY
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-context.txt 2>&1; then
  echo "FAIL: stale CONTEXT.md schema prose passed authority validation"
  exit 1
fi
grep -q "CONTEXT.md" $T/devrites-authority-drift-context.txt
grep -q "claims schema v3" $T/devrites-authority-drift-context.txt

cp "$ROOT/CONTEXT.md" "$T/CONTEXT.md"
printf '\nHistorical note: workspaces written before v5 resolve to schema 2.\n' >> "$T/docs/engine/workspace-schema.md"
if ! python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-history.txt 2>&1; then
  echo "FAIL: a historical schema reference tripped authority validation"
  cat $T/devrites-authority-drift-history.txt
  exit 1
fi

HITL=pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md
perl -0pi -e 's/(- Filesystem destruction outside the workspace\.\n)/$1- Production credential rotation.\n/' "$T/$HITL"
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-risk.txt 2>&1; then
  echo "FAIL: extra afk-hitl risk item passed authority validation"
  exit 1
fi
grep -q "rite-vet/reference/depth.md" $T/devrites-authority-drift-risk.txt
grep -q "rite-temper/reference/significance.md" $T/devrites-authority-drift-risk.txt
grep -q "rite-autocomplete/reference/stop-conditions.md" $T/devrites-authority-drift-risk.txt

cp "$ROOT/$HITL" "$T/$HITL"
STOP=pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md
perl -0pi -e 's/- auth\/authz boundary change;\n/- auth\/authz boundary change;\n- secret rotation;\n/' "$T/$STOP"
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-risk-copy.txt 2>&1; then
  echo "FAIL: extra stop-conditions risk item passed authority validation"
  exit 1
fi
grep -q "stop-conditions.md" $T/devrites-authority-drift-risk-copy.txt
grep -q "secret rotation" $T/devrites-authority-drift-risk-copy.txt

cp "$ROOT/$STOP" "$T/$STOP"
perl -0pi -e 's/- Filesystem destruction outside the workspace\.\n/- Filesystem destruction outside\n  the workspace.\n/' "$T/$HITL"
if ! python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-wrap.txt 2>&1; then
  echo "FAIL: a wrapped afk-hitl risk bullet tripped authority validation"
  cat $T/devrites-authority-drift-wrap.txt
  exit 1
fi
perl -0pi -e 's/  the workspace\.\n/  the workspace.\n  (see the checkpoint protocol)\n- Production credential rotation.\n/' "$T/$HITL"
if python3 "$CHECK" --root "$T" >$T/devrites-authority-drift-wrap-extra.txt 2>&1; then
  echo "FAIL: risk item added after a wrapped bullet passed authority validation"
  exit 1
fi
grep -q "production credential rotation" $T/devrites-authority-drift-wrap-extra.txt

grep -qF "drifted from canonical sources (run: bash scripts/build-host-artifacts.sh)\"" "$ROOT/scripts/validate.sh"

echo "ok: authority validator rejects lifecycle routing, stale schema claims, generated-doc drift, and risk-list drift"
