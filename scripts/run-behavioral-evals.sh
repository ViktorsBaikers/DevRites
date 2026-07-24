#!/usr/bin/env bash
# Validate DevRites behavioral eval files without running a model.
#
# Trigger evals in run-evals.sh check skill selection. Behavioral evals check
# whether a gating skill holds its boundary under pressure. Each scenario pairs
# a pressure prompt with markers for an acceptable response and for capitulation.
# The pressure cases come from standards/anti-patterns.md and each skill's
# reference/anti-patterns.md.
#
# This schema check invokes no model and stops malformed evals before they reach
# the live grader. The manual live-model trial is documented in
# evals/behavioral/README.md.
#
# Scenarios may also use the portable agent-skills and Anthropic skill-creator
# fields:
#   prompt, expected_output, expectations[], trust_level, fixtures[]
# The live grader normalizes these fields after validation.
#
# Usage:
#   scripts/run-behavioral-evals.sh                                  # validate every evals/behavioral/*.json
#   scripts/run-behavioral-evals.sh evals/behavioral/rite-prove.json # validate one file
#
# Exit: 0 all valid (or no behavioral evals present: opt-in, never a failure) ·
#       1 shape violation(s) · 2 missing parser

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Tests can point this at a scratch directory. Production uses the repository's
# behavioral evals by default.
BEHAVIORAL_DIR="${DEVRITES_BEHAVIORAL_DIR:-$ROOT/evals/behavioral}"

if [[ $# -gt 0 ]]; then
  FILES=("$@")
else
  if [[ ! -d "$BEHAVIORAL_DIR" ]]; then
    echo "Behavioral evals are opt-in, and no eval directory was found. Nothing to validate."
    exit 0
  fi
  FILES=()
  while IFS= read -r f; do
    [[ "$f" == */README.md ]] && continue
    FILES+=("$f")
  done < <(find "$BEHAVIORAL_DIR" -type f -name '*.json' | sort)
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "Behavioral evals are opt-in, and no eval files were found. Nothing to validate."
  exit 0
fi

if command -v python3 >/dev/null 2>&1; then
  PARSER="python3"
elif command -v jq >/dev/null 2>&1; then
  PARSER="jq"
else
  echo "python3 or jq is required to validate JSON." >&2
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

# Regression evals require every trial to pass (pass^k = 100%). Capability
# evals use the exploratory pass@k rule.
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
    # If one portable field appears, validate the full row so fixture and
    # transcript or tool-call graders can consume it later.
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
    # Fields that must be non-empty lists of strings.
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
echo "This checks the schema only; it does not run the scenarios."
echo "For a live-model pressure test, follow the manual trial in evals/behavioral/README.md."
exit 0
