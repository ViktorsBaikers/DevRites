#!/usr/bin/env bash
# The reviewer replay is only a quality guard if its grader is right: a known hit
# must score 1, a miss 0, and a line just outside the window must not count.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
CASES="$ROOT/evals/replay/cases"
GRADER="$ROOT/scripts/grade-reviewer-replay.py"
RUNNER="$ROOT/scripts/run-reviewer-replay.py"
EXAMPLE="example-quota-boundary"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

# The shipped corpus validates and plans without starting any host session.
python3 -I "$RUNNER" --validate-only >/dev/null
plan="$(python3 -I "$RUNNER" --arm A --dry-run)"
case "$plan" in
  *"3 calls planned for arm A, at most 6 at once"*) ;;
  *) echo "FAIL: dry-run plan: $plan"; exit 1 ;;
esac

# ret <run> <case> <trial> <finding row or empty> writes one sanitized return.
ret() {
  mkdir -p "$T/$1/$2"
  {
    echo "Outcome: findings"
    echo "Counts: Important=1"
    [ -z "$4" ] || echo "Finding: $4"
  } >"$T/$1/$2/trial-$3.md"
}

# recall <run> <want> checks the example case's single-trial recall and unlabeled count.
recall() {
  python3 -I "$GRADER" --cases "$CASES" --run "$T/$1" | python3 -I -c '
import json, sys
name, want_recall, want_unlabeled = sys.argv[1], float(sys.argv[2]), int(sys.argv[3])
trial = json.load(sys.stdin)["cases"]["'"$EXAMPLE"'"]["trials"][0]
if trial["recall"] != want_recall or trial["unlabeled"] != want_unlabeled:
    sys.exit(f"FAIL: {name}: got {trial}, want recall {want_recall} unlabeled {want_unlabeled}")
' "$1" "$2" "$3"
}

ret hit "$EXAMPLE" 1 'Important conf 4 | internal/quota/quota.go:8 | allows request N+1 | when used == limit -> allowed | breaks AC-001 "return used <= limit" | use used < limit'
recall hit 1 0
ret miss "$EXAMPLE" 1 ''
recall miss 0 0
ret edge "$EXAMPLE" 1 'Important conf 3 | internal/quota/quota.go:18 | boundary | breaks standard'
recall edge 1 0
ret off-window "$EXAMPLE" 1 'Important conf 3 | internal/quota/quota.go:19 | boundary | breaks standard'
recall off-window 0 1
ret basename "$EXAMPLE" 1 'Important conf 3 | quota.go:8 | boundary | breaks standard'
recall basename 1 0
ret other-file "$EXAMPLE" 1 'Important conf 3 | internal/other/quota.go:8 | boundary | breaks standard'
recall other-file 0 1
ret cited-id "$EXAMPLE" 1 'Important conf 3 | internal/limits.go:40 | boundary | breaks AC-001 | fix'
recall cited-id 1 0
ret wrong-id "$EXAMPLE" 1 'Important conf 3 | internal/limits.go:40 | boundary | breaks AC-0011 | fix'
recall wrong-id 0 1

# Route-style, scoped-package, and parenthesized locations keep their whole path.
python3 -I - "$GRADER" <<'PY'
import importlib.util, sys
spec = importlib.util.spec_from_file_location("grader", sys.argv[1])
grader = importlib.util.module_from_spec(spec)
spec.loader.exec_module(grader)
for file, row in (
    ("app/[id]/page.tsx", " Important | app/[id]/page.tsx:8 | boundary"),
    ("app/[id]/page.tsx", " Important | see (page.tsx:9) | boundary"),
    ("packages/@scope/ui+x/index.ts", " Important | packages/@scope/ui+x/index.ts:8 | boundary"),
):
    assert grader.matches(row, {"file": file, "line": 8, "severity": "Important"}), row
assert not grader.matches(" Important | app/[slug]/page.tsx:8 | boundary", {"file": "app/[id]/page.tsx", "line": 8, "severity": "Important"})
PY

# Paired mode: three copies of the example in two clusters; c2 is a Critical must-catch.
mkdir -p "$T/cases"
for c in c1 c2 c3; do cp -R "$CASES/$EXAMPLE" "$T/cases/$c"; done
python3 -I - "$T/cases" <<'PY'
import json, pathlib, sys
root = pathlib.Path(sys.argv[1])
for case, cluster, severity in (("c1", "x", "Important"), ("c2", "x", "Critical"), ("c3", "y", "Important")):
    path = root / case / "expected.json"
    data = json.loads(path.read_text())
    data["cluster"] = cluster
    data["findings"][0]["severity"] = severity
    path.write_text(json.dumps(data))
PY
HIT='Critical conf 4 | internal/quota/quota.go:8 | allows request N+1 | breaks AC-001'
for n in 1 2 3; do
  for c in c1 c2 c3; do ret A "$c" "$n" "$HIT"; done
  ret B c1 "$n" "$HIT"
  ret B c2 "$n" ''
  ret B c3 "$n" "$HIT"
done

# paired <a> <b> <aa-floor or empty> <python assertions on role r and report rep>
paired() {
  local floor=()
  [ -z "$3" ] || floor=(--aa-noise-floor "$3")
  python3 -I "$GRADER" --cases "$T/cases" --a "$T/$1" --b "$T/$2" "${floor[@]+"${floor[@]}"}" >"$T/paired.json"
  python3 -I -c '
import json, sys
rep = json.load(open(sys.argv[1]))
r = rep["roles"]["devrites-code-reviewer"]
'"$4" "$T/paired.json"
}

# d = [0, -1, 0]; residuals sum to -1/3 in cluster x and 1/3 in y; SE = sqrt(2/9)/3.
paired A B 0.10 '
assert (r["cases"], r["clusters"]) == (3, 2), r
assert (r["recall_a"], r["recall_b"], r["diff"]) == (1.0, 0.6667, -0.3333), r
assert (r["se"], r["ci_half_width"]) == (0.1571, 0.308), r
assert r["correlation"] is None, r
assert [(v["case"], v["a_hits"], v["b_trials"]) for v in r["must_catch_violations"]] == [("c2", 3, 3)], r
assert r["gate"] == "fail" and rep["gate"] == "fail" and rep["margin"] == 0.1, rep
'
# Same returns on both sides (A/A): zero difference passes once calibrated, never before.
paired A A '' '
assert (r["diff"], r["ci_half_width"], r["must_catch_violations"]) == (0.0, 0.0, []), r
assert r["gate"] == "needs_aa" and rep["gate"] == "needs_aa", rep
'
paired A A 0.02 '
assert r["gate"] == "pass" and rep["margin"] == 0.05, rep
'
# A case missing from one arm (c2 timed out in every B trial) is reported and blocks a pass.
rm -rf "$T/B/c2"
paired A B 0.10 '
assert r["cases"] == 2 and r["must_catch_violations"] == [] and rep["unpaired"] == ["c2"], rep
assert r["gate"] == "incomplete" and rep["gate"] == "incomplete", rep
'
# A role whose cases all share one cluster has SE 0, so it cannot pass either.
rm -rf "$T/cases/c3"
paired A A 0.02 '
assert (r["cases"], r["clusters"], r["ci_half_width"]) == (2, 1, 0.0), r
assert r["gate"] == "insufficient_clusters" and rep["gate"] == "insufficient_clusters", rep
'

# Malformed cases are rejected with exit 2.
cp -R "$CASES/$EXAMPLE" "$T/bad"
mkdir -p "$T/badcases" && mv "$T/bad" "$T/badcases/bad"
printf '{"role":"devrites-code-reviewer","cluster":"x","findings":[{"file":"a.go","line":0,"severity":"Important"}]}' >"$T/badcases/bad/expected.json"
set +e
python3 -I "$GRADER" --cases "$T/badcases" --run "$T/hit" >/dev/null 2>&1
status=$?
set -e
if [ "$status" -ne 2 ]; then
  echo "FAIL: malformed case exit $status, want 2"
  exit 1
fi

echo "reviewer-replay-grader: ok"
