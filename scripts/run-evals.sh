#!/usr/bin/env bash
# scripts/run-evals.sh — validate the structure of DevRites trigger evals.
#
# Schema check + summary. Does NOT actually invoke Claude — full eval
# execution requires CLAUDE_API_KEY and a `claude` CLI invocation that is
# gated to the user, not CI. CI runs this script to enforce the shape and
# catch broken JSON / missing skills / wrong query counts.
#
# Usage:
#   scripts/run-evals.sh                      # validate every evals/*.json
#   scripts/run-evals.sh evals/rite-spec.json # validate one file

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EVALS_DIR="$ROOT/evals"
EXPECTED_QUERIES=20

if [[ $# -gt 0 ]]; then
  FILES=("$@")
else
  if [[ ! -d "$EVALS_DIR" ]]; then
    echo "No evals/ directory at $EVALS_DIR" >&2
    exit 1
  fi
  FILES=()
  while IFS= read -r f; do
    [[ "$f" == */README.md ]] && continue
    FILES+=("$f")
  done < <(find "$EVALS_DIR" -type f -name '*.json' | sort)
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "No eval files found." >&2
  exit 1
fi

# Need either python3 or jq for JSON parsing. Prefer python3 (already a
# DevRites build dep).
if command -v python3 >/dev/null 2>&1; then
  PARSER="python3"
elif command -v jq >/dev/null 2>&1; then
  PARSER="jq"
else
  echo "Need python3 or jq to validate JSON." >&2
  exit 1
fi

FAILED=0
TOTAL=0

for file in "${FILES[@]}"; do
  TOTAL=$((TOTAL + 1))
  printf '== %s ==\n' "$file"

  if [[ "$PARSER" == "python3" ]]; then
    OUT=$(python3 - "$file" <<'PY'
import json, sys, pathlib
path = pathlib.Path(sys.argv[1])
try:
    data = json.loads(path.read_text())
except Exception as e:
    print(f"INVALID JSON: {e}")
    sys.exit(1)

errors = []

for key in ("skill", "description", "queries"):
    if key not in data:
        errors.append(f"missing top-level key: {key}")

queries = data.get("queries", [])
if not isinstance(queries, list):
    errors.append("queries is not a list")
elif len(queries) != 20:
    errors.append(f"expected 20 queries, got {len(queries)}")

trig = noTrig = 0
for i, q in enumerate(queries):
    if not isinstance(q, dict):
        errors.append(f"query[{i}] not an object")
        continue
    for k in ("text", "expected", "rationale"):
        if k not in q:
            errors.append(f"query[{i}] missing key: {k}")
    if q.get("expected") == "should_trigger":
        trig += 1
    elif q.get("expected") == "should_not_trigger":
        noTrig += 1
    else:
        errors.append(f"query[{i}] invalid expected: {q.get('expected')!r}")

if errors:
    for e in errors:
        print(f"  FAIL: {e}")
    sys.exit(1)

print(f"  skill: {data['skill']}")
print(f"  queries: {len(queries)} (should_trigger={trig}, should_not_trigger={noTrig})")
PY
    )
    rc=$?
  else
    OUT=$(jq -r '
      if (.skill and .description and (.queries|type=="array")) then
        if (.queries|length) == 20 then
          "  skill: \(.skill)\n  queries: \(.queries|length) (should_trigger=\(.queries|map(select(.expected=="should_trigger"))|length), should_not_trigger=\(.queries|map(select(.expected=="should_not_trigger"))|length))"
        else
          "  FAIL: expected 20 queries, got \(.queries|length)"
        end
      else
        "  FAIL: missing required keys"
      end
    ' "$file") || rc=1
    rc=${rc:-0}
  fi

  printf '%s\n' "$OUT"
  if [[ ${rc:-0} -ne 0 ]] || [[ "$OUT" == *"FAIL"* ]]; then
    FAILED=$((FAILED + 1))
  fi
done

echo
printf 'Validated %d eval files; %d failed.\n' "$TOTAL" "$FAILED"

if [[ $FAILED -gt 0 ]]; then
  exit 1
fi

if [[ -z "${CLAUDE_API_KEY:-}" ]]; then
  echo
  echo "Note: CLAUDE_API_KEY is not set; ran schema validation only."
  echo "To execute the evals against a live Claude model:"
  echo "  pip install anthropic"
  echo "  CLAUDE_API_KEY=sk-... python3 scripts/eval-runner.py evals/*.json"
  echo "Override the model with DEVRITES_EVAL_MODEL=claude-... ."
  exit 0
fi

# Live execution path — runs each eval against a real Claude model.
if ! command -v python3 >/dev/null 2>&1; then
  echo "error: CLAUDE_API_KEY is set but python3 is required for the live runner." >&2
  exit 1
fi

echo
echo "CLAUDE_API_KEY set — executing live trigger evals via scripts/eval-runner.py …"
exec python3 "$ROOT/scripts/eval-runner.py" "${FILES[@]}"
