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

# Real executables must isolate crashes/state, keep package cwd and secrets out,
# reject missing tests, and scope build failures to the affected race mode.
python3 -I - "$GRADER" "$T/fixed" "$T/probe paths" <<'PY'
import importlib.util, os, pathlib, shutil, sys, tempfile

spec = importlib.util.spec_from_file_location("grader", sys.argv[1])
grader = importlib.util.module_from_spec(spec)
spec.loader.exec_module(grader)
run, root = map(pathlib.Path, sys.argv[2:])
before = {p.relative_to(run): p.read_bytes() for p in run.rglob("*") if p.is_file()}
(root / "probes").mkdir(parents=True)
# Exercise compiler and executable paths containing spaces.
tempfile.tempdir = str(root)
(root / "probes/probe_test.go").write_text('''package stock

import (
    "os"
    "testing"
)

var probeCalls int

func TestProbePanic(t *testing.T) { panic("isolated probe panic") }

func TestProbeFreshProcess(t *testing.T) {
    probeCalls++
    if probeCalls != 1 { t.Fatal("probe process reused") }
    if os.Getenv("LIFECYCLE_PROBE_SECRET") != "" { t.Fatal("grader secret leaked") }
    if _, err := os.ReadFile("stock.go"); err != nil { t.Fatal(err) }
}

func TestProbeFreshProcessExtra(t *testing.T) { panic("probe name was not anchored") }
''')
bench = {"_dir": root, "probe_package": "stock", "probe_file": "probe_test.go"}
probes = [
    {"id": "panic", "test": "TestProbePanic"},
    {"id": "first", "test": "TestProbeFreshProcess"},
    {"id": "second", "test": "TestProbeFreshProcess"},
    {"id": "missing", "test": "TestProbeMissing"},
]
os.environ["LIFECYCLE_PROBE_SECRET"] = "must-not-leak"
results = grader.run_probes(bench, run, probes)
assert results["panic"].startswith("fail: panic: isolated probe panic"), results
assert results["first"] == results["second"] == "pass", results
assert results["missing"].startswith("fail:") and "no tests to run" in results["missing"], results

if shutil.which("cc") or shutil.which("gcc"):
    build_probes = probes[1:3] + [{"id": "race", "test": "TestProbeFreshProcess", "race": True}]
    for tag in ("race", "!race"):
        broken = root / ("race-build" if tag == "race" else "normal-build")
        shutil.copytree(run, broken)
        (broken / "stock/build_failure_test.go").write_text(
            f"//go:build {tag}\n\npackage stock\n\nvar _ = probeBuildFailure\n"
        )
        results = grader.run_probes(bench, broken, build_probes)
        for probe in build_probes:
            outcome = results[probe["id"]]
            if bool(probe.get("race")) == (tag == "race"):
                assert outcome.startswith("fail:") and "undefined: probeBuildFailure" in outcome, (tag, results)
            else:
                assert outcome == "pass", (tag, results)

assert before == {p.relative_to(run): p.read_bytes() for p in run.rglob("*") if p.is_file()}
PY

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

# Cost is read from the run's metrics ledger, dispatch.json, and question log.
# Wall 09:40-10:19 (39 min); blocking waits 09:45-09:55 and 10:10-10:14 (14 min,
# the advisory one does not count); dispatches 10:00-10:12, 10:13-10:18 and an
# overlapping 10:14-10:17 (20 agent minutes, 17 covered). The 10:10-10:14 wait
# overlaps dispatches and is subtracted once: root-only = 39 - 28 = 11 of 25 active.
W="$T/fixed/.devrites/work/feat"
mkdir -p "$W"
cat >"$W/metrics.jsonl" <<'EOF'
{"ts":"2026-10-20T09:40:00Z","phase":"spec","event":"context","role":"devrites-spec-writer","bytes":1000}
{"ts":"2026-10-20T10:00:00Z","phase":"build","event":"dispatch","role":"devrites-slice-wright"}
{"ts":"2026-10-20T10:12:00Z","phase":"build","event":"return","role":"devrites-slice-wright"}
{"ts":"2026-10-20T10:13:00Z","phase":"build","event":"context","role":"devrites-code-reviewer","bytes":500}
{"ts":"2026-10-20T10:13:00Z","phase":"build","event":"dispatch","role":"devrites-code-reviewer"}
{"ts":"2026-10-20T10:18:00Z","phase":"build","event":"return","role":"devrites-code-reviewer","note":"outcome=findings"}
{"ts":"2026-10-20T10:19:00Z","phase":"build","event":"abandon","note":"host session died"}
EOF
cat >"$W/dispatch.json" <<'EOF'
{"waves": {
  "build-1": {"phase": "build", "state": "complete", "roles": {
    "devrites-slice-wright": {"handle": "Wright1", "started": "2026-10-20T10:00:00Z", "returned": true, "returned_at": "2026-10-20T10:12:00Z"}}},
  "review-1": {"phase": "build", "state": "complete", "roles": {
    "devrites-code-reviewer": {"handle": "Reviewer1", "started": "2026-10-20T10:13:00Z", "returned": true, "returned_at": "2026-10-20T10:18:00Z"},
    "devrites-security-reviewer": {"handle": "Security1", "started": "2026-10-20T10:14:00Z", "returned": true, "returned_at": "2026-10-20T10:17:00Z"}}},
  "build-2": {"phase": "build", "state": "abandoned", "roles": {
    "devrites-slice-wright": {"handle": "Wright2", "started": "2026-10-20T10:18:30Z"}}}
}}
EOF
cat >"$W/questions.md" <<'EOF'
# Questions

## q-2026-10-20-001
status: answered
gate: blocking
raised_at: 2026-10-20T09:45:00Z
answered_at: 2026-10-20T09:55:00Z

## q-2026-10-20-002
status: answered
gate: validating
raised_at: 2026-10-20T10:01:00Z
answered_at: 2026-10-20T10:05:00Z

## q-2026-10-20-003
status: answered
gate: blocking
raised_at: 2026-10-20T10:10:00Z
answered_at: 2026-10-20T10:14:00Z
EOF
python3 -I "$GRADER" --benchmark "$BENCH" --run "$T/fixed" | python3 -I -c '
import json, sys
cost = json.load(sys.stdin)["cost"]
want = {"agent_minutes": 20.0, "active_minutes": 25.0, "root_only_minutes": 11.0, "root_only_fraction": 0.44,
        "context_bytes": 1500, "dispatches": 2, "wright_dispatches": 1, "questions": 3, "wall_hours": 0.65,
        "return_outcomes": {"findings": 1}, "abandoned_waves": 1}
if cost != want:
    sys.exit(f"FAIL: cost = {cost}, want {want}")
'

# --transcripts adds the transcript-usage report as cost.tokens.
mkdir -p "$T/omp/s1"
printf '%s\n' '{"type":"session","timestamp":"2026-10-20T09:59:58Z"}' \
  '{"type":"message","timestamp":"2026-10-20T10:01:00Z","message":{"role":"assistant","model":"m1","usage":{"input":3,"output":7,"cacheRead":0,"cacheWrite":0}}}' \
  >"$T/omp/s1/Wright1.jsonl"
python3 -I "$GRADER" --benchmark "$BENCH" --run "$T/fixed" --transcripts "$T/omp" --host omp | python3 -I -c '
import json, sys
tokens = json.load(sys.stdin)["cost"]["tokens"]
wright = tokens["roles"]["devrites-slice-wright"]
if tokens["usage"] != "available" or (wright["turns"], wright["output"]) != (1, 7):
    sys.exit(f"FAIL: tokens = {tokens}")
'

# refuses <args...> asserts the grader exits 2 before grading anything.
refuses() {
  local code=0
  python3 -I "$GRADER" --benchmark "$BENCH" "$@" >/dev/null 2>"$T/refuse.err" || code=$?
  if [ "$code" -ne 2 ]; then
    echo "FAIL: grader accepted $* (exit $code)"
    exit 1
  fi
}

refuses --run "$T/fixed" --transcripts "$T/omp"
if ! grep -q -- "--transcripts and --host go together" "$T/refuse.err"; then
  echo "FAIL: --transcripts without --host refused for another reason: $(cat "$T/refuse.err")"
  exit 1
fi
refuses --baseline "$T/fixed" "$T/naive" --candidate "$T/fixed" "$T/naive" "$T/drive-by"
refuses --baseline "$T/fixed" "$T/naive" "$T/fixed" --candidate "$T/drive-by" "$T/go-mod" "$T/boundary"
refuses --baseline "$T/fixed" "$T/naive" "$T/drive-by" --candidate "$T/go-mod" "$T/boundary" "$T/validation" \
  --baseline-label omp/model-a --candidate-label omp/model-b
refuses --baseline "$T/fixed" "$T/naive" "$T/drive-by" --candidate "$T/go-mod" "$T/boundary" "$T/validation" \
  --baseline-label omp/model-a

# ledger <variant> <minutes> gives a correct run one wright dispatch of that length.
ledger() {
  variant "$1" "$BENCH/reference/fixed/stock.go"
  mkdir -p "$T/$1/.devrites/work/feat"
  printf '%s\n' '{"ts":"2026-10-20T10:00:00Z","phase":"build","event":"dispatch","role":"devrites-slice-wright"}' \
    "{\"ts\":\"2026-10-20T10:$2:00Z\",\"phase\":\"build\",\"event\":\"return\",\"role\":\"devrites-slice-wright\"}" \
    >"$T/$1/.devrites/work/feat/metrics.jsonl"
  printf '{"waves":{"build-1":{"phase":"build","state":"complete","roles":{"devrites-slice-wright":{"handle":"W","started":"2026-10-20T10:00:00Z","returned":true,"returned_at":"2026-10-20T10:%s:00Z"}}}}}\n' "$2" \
    >"$T/$1/.devrites/work/feat/dispatch.json"
}
for i in 1 2 3; do
  ledger "base$i" 30
  ledger "cand$i" 10
done

# verdict <want> <args...> asserts the compare verdict and its evidence.
verdict() {
  local want="$1"
  shift
  python3 -I "$GRADER" --benchmark "$BENCH" "$@" --baseline-label omp/m --candidate-label omp/m >"$T/compare.json"
  python3 -I - "$T/compare.json" "$want" <<'PY'
import json, sys
report = json.load(open(sys.argv[1]))
want = sys.argv[2]
if report["verdict"] != want or (want == "reject") != bool(report["reasons"]):
    sys.exit(f"FAIL: verdict {report['verdict']} reasons {report['reasons']}, want {want}")
if want == "accept":
    agent = report["cost"]["agent_minutes"]
    if report["cost_drops"] != ["agent_minutes"] or agent["baseline"]["median"] != 30 or agent["candidate"]["median"] != 10:
        sys.exit(f"FAIL: cost {report['cost']} drops {report['cost_drops']}")
elif not any(r.startswith("D1 passed in 2 of 3") for r in report["reasons"]):
    sys.exit(f"FAIL: reasons {report['reasons']} miss the naive candidate's D1")
elif "candidate run 3 has no metrics ledger" not in report["reasons"]:
    sys.exit(f"FAIL: reasons {report['reasons']} miss the naive candidate's missing ledger")
PY
}

verdict accept --baseline "$T/base1" "$T/base2" "$T/base3" --candidate "$T/cand1" "$T/cand2" "$T/cand3"
verdict reject --baseline "$T/base1" "$T/base2" "$T/base3" --candidate "$T/cand1" "$T/cand2" "$T/naive"

echo "lifecycle-benchmark-grader: ok"
