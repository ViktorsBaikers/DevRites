#!/usr/bin/env bash
# The lifecycle benchmark is only a quality guard if its grader is right: every
# hidden probe must fail on its own seeded defect and pass on a correct fix.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
BENCH="$ROOT/evals/benchmark/lifecycle-stock"
GRADER="$ROOT/scripts/grade-lifecycle-benchmark.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

if ! command -v go >/dev/null 2>&1; then
  echo "SKIP: go not on PATH"
  exit 0
fi

variant() {
  rm -rf "${T:?}/$1"
  cp -R "$BENCH/fixture" "$T/$1"
  if [ -n "${2:-}" ]; then
    cp "$2" "$T/$1/stock/stock.go"
  fi
}

# expect <variant> <id=pass|fail>... asserts each named result.
expect() {
  local name="$1"
  shift
  python3 -I "$GRADER" --benchmark "$BENCH" --run "$T/$name" >"$T/$name.json"
  python3 -I - "$T/$name.json" "$name" "$@" <<'PY'
import json, sys
report = json.load(open(sys.argv[1]))
name = sys.argv[2]
for want in sys.argv[3:]:
    check_id, outcome = want.split("=")
    got = report["results"][check_id]
    if (got == "pass") != (outcome == "pass"):
        sys.exit(f"FAIL: {name}: {check_id} = {got!r}, want {outcome}")
PY
}

variant pristine
expect pristine A1=fail A2=fail D1=fail D2=fail D3=fail S1=pass S2=pass

variant fixed "$BENCH/reference/fixed/stock.go"
expect fixed A1=pass A2=pass D1=pass D2=pass D3=pass S1=pass S2=pass

variant naive "$BENCH/reference/naive/stock.go"
expect naive A1=pass A2=pass D1=fail D2=fail

# Each seeded probe fails alone when only its defect returns.
variant boundary "$BENCH/reference/fixed/stock.go"
sed -i.bak 's/if s.available\[item\] < qty {/if s.available[item] <= qty {/' "$T/boundary/stock/stock.go"
expect boundary A1=pass A2=pass D1=fail D2=pass D3=pass

variant validation "$BENCH/reference/fixed/stock.go"
sed -i.bak '/if qty <= 0 {/,/}/d' "$T/validation/stock/stock.go"
expect validation A1=pass A2=pass D1=pass D2=fail D3=pass

if command -v cc >/dev/null 2>&1 || command -v gcc >/dev/null 2>&1; then
  variant unlocked "$BENCH/reference/fixed/stock.go"
  sed -i.bak '/s\.mu\./d' "$T/unlocked/stock/stock.go"
  expect unlocked A1=pass A2=pass D1=pass D2=pass D3=fail
  expect naive D3=fail
else
  echo "note: no C compiler, race-only variant skipped"
fi

variant drive-by "$BENCH/reference/fixed/stock.go"
printf '\n// tidied\n' >>"$T/drive-by/stock/loader.go"
expect drive-by S1=fail S2=pass

# Acceptance and dependency probes must fail on their own breakage too.
variant release-noop "$BENCH/reference/fixed/stock.go"
sed -i.bak '/s\.available\[item\] += qty/d' "$T/release-noop/stock/stock.go"
expect release-noop A1=fail A2=pass D1=pass D2=pass

variant release-unchecked "$BENCH/reference/fixed/stock.go"
sed -i.bak 's/if s\.reserved\[item\] < qty {/if false {/' "$T/release-unchecked/stock/stock.go"
expect release-unchecked A1=pass A2=fail D1=pass

variant go-mod "$BENCH/reference/fixed/stock.go"
printf '\n// tidied\n' >>"$T/go-mod/go.mod"
expect go-mod S1=pass S2=fail

# Cost is read from the run's metrics ledger and question log.
mkdir -p "$T/fixed/.devrites/work/feat"
cat >"$T/fixed/.devrites/work/feat/metrics.jsonl" <<'EOF'
{"ts":"2026-10-20T10:00:00Z","phase":"build","event":"dispatch","role":"devrites-slice-wright"}
{"ts":"2026-10-20T10:12:00Z","phase":"build","event":"return","role":"devrites-slice-wright"}
{"ts":"2026-10-20T10:13:00Z","phase":"build","event":"dispatch","role":"devrites-code-reviewer"}
{"ts":"2026-10-20T10:18:00Z","phase":"build","event":"return","role":"devrites-code-reviewer","note":"outcome=findings"}
{"ts":"2026-10-20T10:19:00Z","phase":"build","event":"abandon","note":"host session died"}
EOF
printf '# Questions\n\n## q-2026-10-20-001\nstatus: answered\n' >"$T/fixed/.devrites/work/feat/questions.md"
python3 -I "$GRADER" --benchmark "$BENCH" --run "$T/fixed" | python3 -I -c '
import json, sys
cost = json.load(sys.stdin)["cost"]
want = {"agent_minutes": 17.0, "dispatches": 2, "wright_dispatches": 1, "questions": 1, "wall_hours": 0.32,
        "return_outcomes": {"findings": 1}, "abandoned_waves": 1}
if cost != want:
    sys.exit(f"FAIL: cost = {cost}, want {want}")
'

echo "lifecycle-benchmark-grader: ok"
