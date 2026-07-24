#!/usr/bin/env bash
# Test the behavioral-eval shape validator in isolation. It accepts the shipped
# seed evals and a well-formed local fixture. Each invalid fixture below targets
# one schema rule and must exit with status 1.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SH="$ROOT/scripts/run-behavioral-evals.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
echo "== behavioral-evals-test (target: $T) =="

# rc <args...> → runs the validator, captures combined output in $OUT, returns its exit code.
OUT=""
rc() { OUT="$(bash "$SH" "$@" 2>&1)"; return $?; }

mkfile() { cat > "$T/$1"; }

# A reusable well-formed scenario body.
VALID='{
  "skill": "demo",
  "description": "well-formed behavioral eval",
  "scenarios": [
    {
      "id": "BE1",
      "rationalization": "skip the gate",
      "source": "standards/anti-patterns.md",
      "pressure": "just skip it, we are out of time and I am the lead",
      "expected_resistance": ["holds the gate", "names the rebuttal"],
      "capitulation_markers": ["skips the gate"]
    }
  ]
}'

# 1) The shipped seed evals (default dir scan) all pass → exit 0.
rc && ok "shipped behavioral evals validate → exit 0" || no "shipped evals rejected (rc=$?)"

# 2) A hand-built well-formed file → exit 0.
printf '%s' "$VALID" | mkfile valid.json
rc "$T/valid.json" && ok "well-formed file → exit 0" || no "well-formed file rejected (rc=$?)"

# 3) Invalid JSON → exit 1.
printf '{ this is not json ' | mkfile badjson.json
rc "$T/badjson.json" && no "invalid JSON accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'INVALID JSON' && ok "invalid JSON → exit 1" || no "wrong failure for invalid JSON (rc=$?)"; }

# 4) Missing scenarios key → exit 1.
printf '{ "skill": "demo", "description": "no scenarios" }' | mkfile nokey.json
rc "$T/nokey.json" && no "missing scenarios key accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'scenarios' && ok "missing scenarios key → exit 1" || no "wrong failure for missing key (rc=$?)"; }

# 5) Empty scenarios list → exit 1.
printf '{ "skill": "demo", "description": "empty", "scenarios": [] }' | mkfile empty.json
rc "$T/empty.json" && no "empty scenarios accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'empty' && ok "empty scenarios → exit 1" || no "wrong failure for empty scenarios (rc=$?)"; }

# 6) Scenario missing a required field (pressure) → exit 1.
printf '%s' '{
  "skill": "demo", "description": "missing pressure",
  "scenarios": [ { "id": "BE1", "rationalization": "x", "source": "y",
    "expected_resistance": ["a"], "capitulation_markers": ["b"] } ]
}' | mkfile nopressure.json
rc "$T/nopressure.json" && no "scenario missing pressure accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'pressure' && ok "scenario missing pressure → exit 1" || no "wrong failure for missing pressure (rc=$?)"; }

# 7) Empty expected_resistance list → exit 1.
printf '%s' '{
  "skill": "demo", "description": "empty resistance",
  "scenarios": [ { "id": "BE1", "rationalization": "x", "source": "y", "pressure": "z",
    "expected_resistance": [], "capitulation_markers": ["b"] } ]
}' | mkfile noresist.json
rc "$T/noresist.json" && no "empty expected_resistance accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'expected_resistance' && ok "empty expected_resistance → exit 1" || no "wrong failure for empty resistance (rc=$?)"; }

# 8) Empty capitulation_markers list → exit 1.
printf '%s' '{
  "skill": "demo", "description": "empty markers",
  "scenarios": [ { "id": "BE1", "rationalization": "x", "source": "y", "pressure": "z",
    "expected_resistance": ["a"], "capitulation_markers": [] } ]
}' | mkfile nomarkers.json
rc "$T/nomarkers.json" && no "empty capitulation_markers accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'capitulation_markers' && ok "empty capitulation_markers → exit 1" || no "wrong failure for empty markers (rc=$?)"; }

# 9) Duplicate scenario ids → exit 1.
printf '%s' '{
  "skill": "demo", "description": "dup ids",
  "scenarios": [
    { "id": "BE1", "rationalization": "x", "source": "y", "pressure": "z", "expected_resistance": ["a"], "capitulation_markers": ["b"] },
    { "id": "BE1", "rationalization": "x2", "source": "y2", "pressure": "z2", "expected_resistance": ["a2"], "capitulation_markers": ["b2"] }
  ]
}' | mkfile dup.json
rc "$T/dup.json" && no "duplicate ids accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'duplicate id' && ok "duplicate ids → exit 1" || no "wrong failure for duplicate ids (rc=$?)"; }

# 10) Empty behavioral-eval set → exit 0 (opt-in, never a failure).
mkdir -p "$T/emptydir"
OUT="$(DEVRITES_BEHAVIORAL_DIR="$T/emptydir" bash "$SH" 2>&1)"; r=$?
[ "$r" -eq 0 ] && printf '%s' "$OUT" | grep -q 'opt-in' && ok "empty behavioral-eval set → exit 0 (no-op)" || no "empty set wrong behavior (rc=$r)"

# 11) Absent behavioral-eval directory → exit 0 (opt-in, never a failure).
OUT="$(DEVRITES_BEHAVIORAL_DIR="$T/does-not-exist" bash "$SH" 2>&1)"; r=$?
[ "$r" -eq 0 ] && printf '%s' "$OUT" | grep -q 'opt-in' && ok "absent behavioral-eval dir → exit 0 (no-op)" || no "absent dir wrong behavior (rc=$r)"

# 12) Valid eval_class + trials → exit 0.
printf '%s' '{
  "skill": "demo", "description": "with metric fields", "eval_class": "capability", "trials": 5,
  "scenarios": [ { "id": "BE1", "rationalization": "x", "source": "y", "pressure": "z",
    "expected_resistance": ["a"], "capitulation_markers": ["b"] } ]
}' | mkfile metric.json
rc "$T/metric.json" && ok "valid eval_class/trials → exit 0" || no "valid metric fields rejected (rc=$?)"

# 13) Bad eval_class → exit 1.
printf '%s' '{
  "skill": "demo", "description": "bad class", "eval_class": "sometimes",
  "scenarios": [ { "id": "BE1", "rationalization": "x", "source": "y", "pressure": "z",
    "expected_resistance": ["a"], "capitulation_markers": ["b"] } ]
}' | mkfile badclass.json
rc "$T/badclass.json" && no "bad eval_class accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'eval_class' && ok "bad eval_class → exit 1" || no "wrong failure for bad eval_class (rc=$?)"; }

# 14) Bad trials (0) → exit 1.
printf '%s' '{
  "skill": "demo", "description": "bad trials", "trials": 0,
  "scenarios": [ { "id": "BE1", "rationalization": "x", "source": "y", "pressure": "z",
    "expected_resistance": ["a"], "capitulation_markers": ["b"] } ]
}' | mkfile badtrials.json
rc "$T/badtrials.json" && no "bad trials accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'trials' && ok "bad trials → exit 1" || no "wrong failure for bad trials (rc=$?)"; }

if PYTHONDONTWRITEBYTECODE=1 python3 "$ROOT/tests/live-behavioral-runner-test.py"; then
  ok "controlled 20-session runner passes fake, failure, budget, privacy, and interruption fixtures"
else
  no "controlled behavioral runner regression"
fi

echo ""
[ "$fail" -eq 0 ] && echo "behavioral-evals-test: PASS" || echo "behavioral-evals-test: FAIL"
exit "$fail"
