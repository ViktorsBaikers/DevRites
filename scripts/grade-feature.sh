#!/usr/bin/env bash
# Deterministic outcome grader for a completed DevRites feature workspace.
#
# Trigger evals in evals/*.json check skill selection. This script checks
# whether a completed run is shippable and has proof. It reads only committed
# Markdown artifacts, so CI can grade a golden fixture without an API key or
# model harness.
#
# Git fixtures do not preserve proof mtimes. Live freshness belongs to
# `devrites-engine check seal`; this grader checks committed artifact content.
#
# Usage: grade-feature.sh [--json] <workspace-dir>
#   e.g. evals/golden/shippable-feature | .devrites/work/<slug> | .devrites/archive/<slug>
#
# Checks from rite-seal/reference/{seal-template,go-no-go,final-evidence}.md:
#   1. seal.md present with "Verdict: GO" (not NO-GO)
#   2. seal.md "## Acceptance Criteria" has no unchecked "- [ ]" item
#   3. seal.md "## Blockers" is empty / "none"
#   4. evidence.md present and non-empty
#   5. review.md present
#   6. questions.md has no open question (later lifecycle phases block them)
#   7. state.md Phase in {seal, ship, done}; Status not awaiting_human / blocked
#   8. spec.md and seal.md contain the same nonempty, unique canonical AC-### IDs
#
# Exit codes: 0 shippable; 1 one or more invariants failed; 2 bad usage.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
json=0
if [ "${1:-}" = "--json" ]; then
  json=1
  shift
fi
ws="${1:-}"
if [ "$#" -ne 1 ] || [ ! -d "$ws" ]; then
  printf 'usage: grade-feature.sh [--json] <workspace-dir>\n' >&2
  exit 2
fi

seal="$ws/seal.md"; ev="$ws/evidence.md"; rev="$ws/review.md"
q="$ws/questions.md"; st="$ws/state.md"
problems=()
rules=()
unchecked=0
acceptance_ready=0

add_problem() {
  rules+=("$1")
  problems+=("$2")
}

# One recogniser for "## Acceptance criteria" sections, shared by the
# unchecked-item rule (check 2) and the ID rule (check 8); repeated sections are
# concatenated so an item cannot hide in a later one. The unchecked rule also
# counts sections whose heading merely starts with the title (a suffix such as
# "(continued)" must not hide an item); the ID rule needs the exact heading.
# Headings are matched on text with fenced code masked, so a heading inside a
# fence does not end the section; unchecked items are counted on the raw lines so fenced text is not
# silently ignored. Up to three leading spaces are allowed on the heading
# (CommonMark); four or more is code, not a heading. Heading whitespace is only
# space or tab, so other Unicode whitespace after "##" does not make a heading.
read -r -d '' ACCEPT_PY <<'PY' || true
from pathlib import Path
import re
import sys

from workflow_schema import structural_markdown

canonical = re.compile(r"\bAC-\d{3}\b")
any_ac = re.compile(r"\bAC(?:-\d+|\d+)\b", re.IGNORECASE)
heading = re.compile(r"^ {0,3}##[ \t]+Acceptance criteria[ \t]*#*[ \t]*$", re.IGNORECASE)
heading_prefix = re.compile(r"^ {0,3}##[ \t]+Acceptance criteria", re.IGNORECASE)
h2 = re.compile(r"^ {0,3}##[ \t]+")
unchecked = re.compile(r"^\s*(?:[-*+]|\d{1,9}[.)])\s+\[ \]")


def acceptance(path: Path, opening=heading) -> tuple[list[str], list[str]]:
    if not path.is_file():
        raise ValueError(f"{path.name} missing")
    raw = path.read_text(encoding="utf-8")
    masked = structural_markdown(raw, path).split("\n")
    rows, found, inside = [], False, False
    for i, line in enumerate(masked):
        if opening.match(line.rstrip("\r")):
            found = inside = True
        elif h2.match(line):
            inside = False
        elif inside:
            rows.append(i)
    if not found:
        raise ValueError(f'{path.name} has no "## Acceptance criteria" section')
    raw_lines = raw.split("\n")
    return [raw_lines[i] for i in rows], [masked[i] for i in rows]


def sections(*paths: str, opening=heading) -> list[tuple[list[str], list[str]]]:
    try:
        return [acceptance(Path(p), opening) for p in paths]
    except (OSError, UnicodeError, ValueError) as exc:
        raise SystemExit(str(exc))


mode, *paths = sys.argv[1:]
if mode == "unchecked":
    print(sum(1 for line in sections(*paths, opening=heading_prefix)[0][0] if unchecked.match(line)))
    raise SystemExit(0)

spec, sealed = ("\n".join(masked) for _, masked in sections(*paths))
spec_ids = canonical.findall(spec)
seal_ids = canonical.findall(sealed)
for name, body, ids in (("spec.md", spec, spec_ids), ("seal.md", sealed, seal_ids)):
    invalid = sorted({token for token in any_ac.findall(body) if not canonical.fullmatch(token)})
    if invalid:
        raise SystemExit(f"{name} has noncanonical acceptance IDs: {' '.join(invalid)}")
    if not ids:
        raise SystemExit(f"{name} acceptance IDs are empty")
    duplicates = sorted({item for item in ids if ids.count(item) > 1})
    if duplicates:
        raise SystemExit(f"{name} has duplicate acceptance IDs: {' '.join(duplicates)}")

missing = sorted(set(spec_ids) - set(seal_ids))
extra = sorted(set(seal_ids) - set(spec_ids))
if missing or extra:
    raise SystemExit(
        "acceptance ID mismatch: missing={} extra={}".format(
            " ".join(missing) or "none", " ".join(extra) or "none"
        )
    )
PY
acceptance_py() {
  PYTHONPATH="$ROOT/scripts${PYTHONPATH:+:$PYTHONPATH}" python3 -c "$ACCEPT_PY" "$@"
}

# Checks 1 through 3: seal verdict, acceptance, and blockers.
if [ ! -f "$seal" ]; then
  add_problem "final.seal.missing" "seal.md missing: feature never sealed"
else
  # GO needs at least one Verdict field and every one of them GO; any other
  # value, or a failure to read the file, is NO-GO. The field itself is read by
  # cursor_values from validate-workspace-schema.py, so this grader and the
  # schema validator share one rule. Quote, heading, list, checkbox and
  # emphasis markers are removed first because a reader still sees those lines
  # as verdicts. Fenced text is not masked, so a NO-GO inside a fence counts.
  if ! PYTHONPATH="$ROOT/scripts${PYTHONPATH:+:$PYTHONPATH}" python3 - "$seal" "$ROOT/scripts/validate-workspace-schema.py" <<'PY'
import importlib.util
import re
import sys
from pathlib import Path

spec = importlib.util.spec_from_file_location("validate_workspace_schema", sys.argv[2])
schema = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = schema
spec.loader.exec_module(schema)

marker = re.compile(r"^\s*(?:>|#{1,6}(?=\s)|\d{1,9}[.)](?=\s)|[-*+](?=\s)|\[[ xX]\](?=\s))")
lines = []
for line in Path(sys.argv[1]).read_text(encoding="utf-8-sig").splitlines():
    while (found := marker.match(line)):
        line = line[found.end():]
    lines.append(re.sub(r"[*_]", "", line))
values = schema.cursor_values("\n".join(lines), "Verdict")
raise SystemExit(0 if values and all(value.lower() == "go" for value in values) else 1)
PY
  then
    add_problem "final.verdict.not-go" "seal.md Verdict is not GO"
  fi
  # stdout is the count and nothing else; stderr is kept apart so interpreter
  # noise cannot corrupt it, and any failure or non-numeric count is NO-GO. The
  # digit cap keeps the integer comparisons below from overflowing into a pass.
  accept_err="$(mktemp)"
  if unchecked=$(acceptance_py unchecked "$seal" 2>"$accept_err") && [[ "$unchecked" =~ ^[0-9]{1,9}$ ]]; then
    :
  else
    accept_msg="$(tail -n 1 "$accept_err")"
    add_problem "final.acceptance.ids" "acceptance: ${accept_msg:-unreadable count '${unchecked:-}'}"
    unchecked=-1
  fi
  rm -f "$accept_err"
  if [ "$unchecked" -gt 0 ]; then
    add_problem "final.acceptance.unchecked" "seal.md has ${unchecked} unchecked acceptance criterion(s)"
  elif [ "$unchecked" -eq 0 ]; then
    acceptance_ready=1
  fi
  if awk '
      /^## /{ insec=($0 ~ /^## Blockers/); next }
      insec { l=$0; gsub(/^[[:space:]]+|[[:space:]]+$/,"",l)
              if (l=="") next
              ll=tolower(l); if (ll ~ /^-?[[:space:]]*(none|n\/a)$/) next
              nz=1 }
      END { exit(nz?0:1) }' "$seal"; then
    add_problem "final.blockers.unresolved" "seal.md lists unresolved blockers"
  fi
fi

# Check 4: evidence.
{ [ -f "$ev" ] && [ -s "$ev" ]; } \
  || add_problem "final.evidence.missing-or-empty" "evidence.md missing or empty: acceptance unproven"

# Check 5: review.
[ -f "$rev" ] || add_problem "final.review.missing" "review.md missing: feature not reviewed"

# Check 6: open questions block sealing.
if [ -f "$q" ]; then
  if python3 - "$q" <<'PY'
from pathlib import Path
import sys

lines = Path(sys.argv[1]).read_text().splitlines()
in_question = False
status = ""
for line in lines:
    stripped = line.strip()
    if stripped.startswith("## "):
        if in_question and status.lower() == "open":
            raise SystemExit(0)
        in_question = stripped[3:].lower().startswith("q-")
        status = ""
    elif in_question and stripped.lower().startswith("status:"):
        status = stripped.split(":", 1)[1].strip()
if in_question and status.lower() == "open":
    raise SystemExit(0)

status_index = None
for line in lines:
    stripped = line.strip()
    if not (stripped.startswith("|") and stripped.endswith("|")):
        status_index = None
        continue
    cells = [cell.strip() for cell in stripped.strip("|").split("|")]
    lowered = [cell.lower() for cell in cells]
    if status_index is None:
        status_index = lowered.index("status") if "status" in lowered else None
        continue
    if status_index < len(cells) and cells[status_index].lower() == "open":
        raise SystemExit(0)
raise SystemExit(1)
PY
  then
    add_problem "final.questions.open" "questions.md contains an open question"
  fi
fi

# Check 7: state phase and status.
if [ -f "$st" ]; then
  ph="$(python3 "$ROOT/scripts/workflow_schema.py" field "$st" phase 2>/dev/null || true)"
  stt="$(python3 "$ROOT/scripts/workflow_schema.py" field "$st" status 2>/dev/null || true)"
  if ! python3 "$ROOT/scripts/workflow_schema.py" phase-property "$ph" shippable >/dev/null; then
    add_problem "final.state.phase" "state.md Phase='${ph}' (expected a shippable phase)"
  fi
  case "$stt" in
    awaiting_human|blocked)
      add_problem "final.state.status" "state.md Status='${stt}' (not shippable)"
      ;;
  esac
else
  add_problem "final.state.missing" "state.md missing"
fi

# Check 8: canonical acceptance IDs are nonempty, unique, and exactly equal in
# spec.md and seal.md. The unchecked rule above separately proves every seal row
# is checked.
run_acceptance_check() {
  acceptance_py ids "$ws/spec.md" "$seal"
}

if [ "$acceptance_ready" -eq 1 ] && ! acout=$(run_acceptance_check 2>&1); then
  add_problem "final.acceptance.ids" "acceptance: $(printf '%s' "$acout" | tail -1)"
fi

slug="$(basename "$ws")"
if [ "$json" -eq 1 ]; then
  status="GO"
  [ "${#problems[@]}" -gt 0 ] && status="NO-GO"
  pairs=()
  for ((i = 0; i < ${#problems[@]}; i++)); do
    pairs+=("${rules[$i]}" "${problems[$i]}")
  done
  python3 - "$slug" "$status" ${pairs[@]+"${pairs[@]}"} <<'PY'
import json
import sys

slug, status, *pairs = sys.argv[1:]
items = [
    {"rule_id": pairs[i], "message": pairs[i + 1]}
    for i in range(0, len(pairs), 2)
]
print(json.dumps({
    "schema": "devrites-outcome-grade/v1",
    "workspace": slug,
    "status": status,
    "rule_ids": [item["rule_id"] for item in items],
    "problems": items,
}, separators=(",", ":")))
PY
  if [ "${#problems[@]}" -eq 0 ]; then
    exit 0
  fi
  exit 1
fi

if [ "${#problems[@]}" -eq 0 ]; then
  printf 'GO    %s: shippable: sealed GO, acceptance proven, no blockers, no open question.\n' "$slug"
  exit 0
fi
printf 'NO-GO %s: %d blocker(s):\n' "$slug" "${#problems[@]}"
for p in "${problems[@]}"; do printf '  - %s\n' "$p"; done
exit 1
