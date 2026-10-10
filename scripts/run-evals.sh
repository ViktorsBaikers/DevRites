#!/usr/bin/env bash
# scripts/run-evals.sh: validate the structure of DevRites trigger evals.
#
# Schema check + summary. CI runs this script to catch broken JSON, missing
# skills, and empty/one-sided corpora. Native hosts own actual skill routing.
# Two offline routing rules are also enforced: an explicit /name or $name
# query must route to that skill, and a natural-language negative must not name
# an explicit-only owner. The jq fallback checks shape only.
#
# Usage:
#   scripts/run-evals.sh                         # validate every evals/*.json
#   scripts/run-evals.sh evals/rite-spec.json    # validate one file

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EVALS_DIR="$ROOT/evals"

if [[ $# -gt 0 ]]; then
  FILES=("$@")
else
  if [[ ! -d "$EVALS_DIR" ]]; then
    echo "No evals/ directory at $EVALS_DIR" >&2
    exit 1
  fi
  FILES=()
  while IFS= read -r f; do
    FILES+=("$f")
  done < <(find "$EVALS_DIR" -maxdepth 1 -type f -name '*.json' ! -name 'coverage.json' | sort)
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
    if OUT=$(python3 - "$file" "$ROOT/pack/.claude/skills" <<'PY'
import json, re, sys, pathlib
skills_dir = pathlib.Path(sys.argv[2])
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
elif not queries:
    errors.append("queries is empty")

trig = noTrig = 0
for i, q in enumerate(queries if isinstance(queries, list) else []):
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
        owner = q.get("owner")
        if "owner" not in q:
            errors.append(f"query[{i}] missing key: owner")
        elif owner is None:
            if not str(q.get("owner_rationale") or "").strip():
                errors.append(f"query[{i}] owner is null without owner_rationale")
        elif not (
            isinstance(owner, str)
            and re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", owner)
            and (skills_dir / owner / "SKILL.md").is_file()
        ):
            errors.append(f"query[{i}] owner {owner!r} is not a skill in pack/.claude/skills")
    else:
        errors.append(f"query[{i}] invalid expected: {q.get('expected')!r}")
        continue
    text = str(q.get("text", ""))
    owner = q.get("owner")
    m = re.match(r"\s*[/$]([a-z0-9][a-z0-9-]*)(?:\s+([a-z0-9][a-z0-9-]*))?", text)
    target = None
    if m:
        for cand in ([f"rite-{m.group(2)}"] if m.group(1) == "rite" and m.group(2) else []) + [m.group(1)]:
            if (skills_dir / cand / "SKILL.md").is_file():
                target = cand
                break
    if target:
        if q["expected"] == "should_trigger" and target != data.get("skill"):
            errors.append(f"query[{i}] explicitly invokes {target}, not {data.get('skill')}")
        elif q["expected"] == "should_not_trigger" and owner != target:
            errors.append(f"query[{i}] explicitly invokes {target}; owner must be {target}")
    elif q["expected"] == "should_not_trigger" and isinstance(owner, str):
        owner_md = skills_dir / owner / "SKILL.md"
        if owner_md.is_file() and re.search(
            r"^disable-model-invocation:\s*true", owner_md.read_text(), re.M
        ):
            errors.append(
                f"query[{i}] owner {owner} is explicit-only; natural language has no implicit owner"
            )

if isinstance(queries, list) and queries:
    if trig == 0:
        errors.append("corpus has no should_trigger query")
    if noTrig == 0:
        errors.append("corpus has no should_not_trigger query")

if errors:
    for e in errors:
        print(f"  FAIL: {e}")
    sys.exit(1)

print(f"  skill: {data['skill']}")
print(f"  queries: {len(queries)} (should_trigger={trig}, should_not_trigger={noTrig})")
PY
    ); then rc=0; else rc=$?; fi
  else
    if OUT=$(jq -r '
      if (.skill and .description and (.queries|type=="array")) then
        if ((.queries|length) > 0 and (.queries|map(select(.expected=="should_trigger"))|length) > 0 and (.queries|map(select(.expected=="should_not_trigger"))|length) > 0) then
          "  skill: \(.skill)\n  queries: \(.queries|length) (should_trigger=\(.queries|map(select(.expected=="should_trigger"))|length), should_not_trigger=\(.queries|map(select(.expected=="should_not_trigger"))|length))"
        else
          "  FAIL: queries must be non-empty and include should_trigger + should_not_trigger"
        end
      else
        "  FAIL: missing required keys"
      end
    ' "$file"); then rc=0; else rc=$?; fi
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
