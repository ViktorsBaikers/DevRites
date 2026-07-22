#!/usr/bin/env bash
# scripts/run-behavioral-evals.sh: validate the SHAPE of DevRites behavioral evals.
#
# Trigger evals (run-evals.sh) test whether the right skill *fires*. Behavioral
# evals test whether a gating skill's discipline *holds under pressure*: does it
# resist the rationalizations it documents in `standards/anti-patterns.md` (and its own
# `reference/anti-patterns.md`), or does the agent talk itself past the gate. Each
# scenario pairs a pressure prompt with the resistance a holding response shows and
# the capitulation markers a failed one shows.
#
# This script is the DETERMINISTIC, zero-token CI gate: the analog of run-evals.sh's
# schema path and the engine spec-validate gate: it checks that every behavioral eval is well-formed
# so a malformed one can't reach the live grader. It does NOT invoke a model. Executing
# the scenarios against a live Claude (does the skill resist?) is the labeled /
# nightly rung documented in evals/behavioral/README.md.
#
# In addition to the original DevRites pressure schema, scenarios may carry the
# portable agent-skills / Anthropic skill-creator shape:
#   prompt, expected_output, expectations[], trust_level, fixtures[]
# Those fields are validated when present and normalized by the live-grader layer.
#
# Usage:
#   scripts/run-behavioral-evals.sh                                  # validate every evals/behavioral/*.json
#   scripts/run-behavioral-evals.sh evals/behavioral/rite-prove.json # validate one file
#
# Exit: 0 all valid (or no behavioral evals present: opt-in, never a failure) ·
#       1 shape violation(s) · 2 missing parser

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Overridable so the test harness can exercise the opt-in no-op path against a
# scratch directory; defaults to the real behavioral-eval set.
BEHAVIORAL_DIR="${DEVRITES_BEHAVIORAL_DIR:-$ROOT/evals/behavioral}"

if [[ $# -gt 0 ]]; then
  FILES=("$@")
else
  if [[ ! -d "$BEHAVIORAL_DIR" ]]; then
    echo "No evals/behavioral/ directory: behavioral evals are opt-in; nothing to validate."
    exit 0
  fi
  FILES=()
  while IFS= read -r f; do
    [[ "$f" == */README.md ]] && continue
    FILES+=("$f")
  done < <(find "$BEHAVIORAL_DIR" -type f -name '*.json' | sort)
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "No behavioral eval files: behavioral evals are opt-in; nothing to validate."
  exit 0
fi

if command -v python3 >/dev/null 2>&1; then
  PARSER="python3"
elif command -v jq >/dev/null 2>&1; then
  PARSER="jq"
else
  echo "Need python3 or jq to validate JSON." >&2
  exit 2
fi

FAILED=0
TOTAL=0
SCENARIOS=0

for file in "${FILES[@]}"; do
  TOTAL=$((TOTAL + 1))
  printf '== %s ==\n' "$file"

  if [[ "$PARSER" == "python3" ]]; then
    if OUT=$(python3 - "$file" <<'PY'
import json, sys, pathlib
path = pathlib.Path(sys.argv[1])
try:
    data = json.loads(path.read_text())
except Exception as e:
    print(f"  FAIL: INVALID JSON: {e}")
    sys.exit(1)

errors = []
for key in ("skill", "description", "scenarios"):
    if key not in data:
        errors.append(f"missing top-level key: {key}")

scenarios = data.get("scenarios", [])
if not isinstance(scenarios, list):
    errors.append("scenarios is not a list")
    scenarios = []
elif len(scenarios) == 0:
    errors.append("scenarios is empty: a behavioral eval needs at least one")

# Metric fields (optional: default to the regression discipline). A behavioral
# eval is a regression gate by nature (the discipline must hold every trial ->
# pass^k = 100%); a capability eval is the exploratory pass@k variant.
ec = data.get("eval_class", "regression")
if ec not in ("regression", "capability"):
    errors.append(f"eval_class must be 'regression' or 'capability', got {ec!r}")
tr = data.get("trials", 3)
if isinstance(tr, bool) or not isinstance(tr, int) or tr < 1:
    errors.append(f"trials must be an integer >= 1, got {tr!r}")

seen_ids = {}
for i, s in enumerate(scenarios):
    if not isinstance(s, dict):
        errors.append(f"scenario[{i}] not an object")
        continue
    sid = s.get("id")
    if not sid or not isinstance(sid, str):
        errors.append(f"scenario[{i}] missing non-empty string id")
    elif sid in seen_ids:
        errors.append(f"scenario[{i}] duplicate id {sid!r} (also scenario[{seen_ids[sid]}])")
    else:
        seen_ids[sid] = i
    where = sid or f"[{i}]"
    # Non-empty string fields.
    for k in ("rationalization", "pressure", "source"):
        v = s.get(k)
        if not isinstance(v, str) or not v.strip():
            errors.append(f"scenario {where} missing non-empty {k}")
    # Optional portable behavioral-eval fields borrowed from agent-skills /
    # Anthropic skill-creator. When one is present, validate the full portable
    # row so fixtures and transcript/tool-call graders can consume it later.
    portable_present = any(k in s for k in ("prompt", "expected_output", "expectations", "trust_level", "fixtures"))
    if portable_present:
        for k in ("prompt", "expected_output", "trust_level"):
            v = s.get(k)
            if not isinstance(v, str) or not v.strip():
                errors.append(f"scenario {where} missing non-empty portable field {k}")
        expectations = s.get("expectations")
        if not isinstance(expectations, list) or not expectations or not all(isinstance(x, str) and x.strip() for x in expectations):
            errors.append(f"scenario {where} portable expectations must be a non-empty string list")
        fixtures = s.get("fixtures", [])
        if not isinstance(fixtures, list) or not all(isinstance(x, str) and x.strip() for x in fixtures):
            errors.append(f"scenario {where} portable fixtures must be a string list when present")
    # Non-empty list-of-string fields.
    for k in ("expected_resistance", "capitulation_markers"):
        v = s.get(k)
        if not isinstance(v, list) or len(v) == 0:
            errors.append(f"scenario {where} missing non-empty {k} list")
        elif not all(isinstance(x, str) and x.strip() for x in v):
            errors.append(f"scenario {where} {k} has an empty / non-string entry")

if errors:
    for e in errors:
        print(f"  FAIL: {e}")
    sys.exit(1)

gate = "pass^k (all trials must hold)" if ec == "regression" else "pass@k (any trial holds)"
print(f"  skill: {data['skill']}")
print(f"  class: {ec} · trials: {tr} · gate: {gate}")
print(f"  scenarios: {len(scenarios)}")
print(f"__COUNT__ {len(scenarios)}")
PY
    ); then rc=0; else rc=$?; fi
  else
    if OUT=$(jq -r '
      if (.skill and .description and (.scenarios|type=="array") and (.scenarios|length > 0)) then
        ( [ .scenarios[]
            | select(
                ((.id|type=="string") and (.id|length>0))
                and ((.rationalization|type=="string") and (.rationalization|length>0))
                and ((.pressure|type=="string") and (.pressure|length>0))
                and ((.source|type=="string") and (.source|length>0))
                and ((.expected_resistance|type=="array") and (.expected_resistance|length>0))
                and ((.capitulation_markers|type=="array") and (.capitulation_markers|length>0))
              ) ] | length
        ) as $ok
        | if $ok == (.scenarios|length)
          then "  skill: \(.skill)\n  scenarios: \(.scenarios|length)\n__COUNT__ \(.scenarios|length)"
          else "  FAIL: \((.scenarios|length) - $ok) scenario(s) missing required fields"
          end
      else
        "  FAIL: missing required keys or empty scenarios"
      end
    ' "$file"); then rc=0; else rc=$?; fi
  fi

  COUNT=$(printf '%s\n' "$OUT" | sed -n 's/^__COUNT__ //p')
  OUT=$(printf '%s\n' "$OUT" | grep -v '^__COUNT__' || true)
  printf '%s\n' "$OUT"

  if [[ ${rc:-0} -ne 0 ]] || [[ "$OUT" == *"FAIL"* ]]; then
    FAILED=$((FAILED + 1))
  else
    SCENARIOS=$((SCENARIOS + ${COUNT:-0}))
  fi
done

echo
printf 'Validated %d behavioral eval file(s); %d scenario(s); %d failed.\n' "$TOTAL" "$SCENARIOS" "$FAILED"

if [[ $FAILED -gt 0 ]]; then
  exit 1
fi

echo
echo "Note: this is shape validation only: it does not execute the scenarios."
echo "To grade whether the skills resist under pressure (live model):"
echo "  see evals/behavioral/README.md: the labeled / nightly rung."
exit 0
