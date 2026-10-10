#!/usr/bin/env bash

set -euo pipefail

if [ "${DEVRITES_OUTCOME_DISCOVERY_ONLY:-0}" = "1" ]; then
  echo "outcome-evals discovery sentinel"
  exit 0
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

output="$(bash "$ROOT/scripts/run-outcome-evals.sh")"
printf '%s\n' "$output"
grep -Fq "Outcome evals passed: native boundary + 15 isolated final-outcome negatives + candidate/readiness content binding + removed-command rejections + 2 adversarial fixtures." <<<"$output"
grep -Fq "PASS content_identity     unchanged-touch=pass restored-mtime-byte-change=blocked" <<<"$output"
grep -Fq "all 19 retired top-level commands are unknown (no aliases)" <<<"$output"
grep -Fq "wrong_ac_id" <<<"$output"
grep -Fq "PASS: unauthorized-spec-drift grader NO-GO + readiness AC map" <<<"$output"
grep -Fq "PASS: out-of-scope-writer-diff extra candidate path is not in tasks.md" <<<"$output"

grade_count=0
grader_case() {
  local id="$1" want_status="$2" want_rules="$3" run_env="${4:-}" verdict_line="${5:-}"
  local ws="$tmp/grade-$id-$((++grade_count))"
  cp -R "$ROOT/evals/golden/shippable-feature" "$ws"
  python3 - "$ws/seal.md" "$id" "$verdict_line" <<'PY'
from pathlib import Path
import sys

seal, case = Path(sys.argv[1]), sys.argv[2]
text = seal.read_text()
edits = {
    "baseline": [],
    "heading-lowercase": [("## Acceptance Criteria", "## acceptance criteria")],
    "ac002-unchecked": [("- [x] AC-002:", "- [ ] AC-002:")],
    "indented-ac002-unchecked": [("- [x] AC-002:", "  - [ ] AC-002:")],
    "heading-trailing-space-ac002-unchecked": [
        ("## Acceptance Criteria", "## Acceptance Criteria  "),
        ("- [x] AC-002:", "- [ ] AC-002:"),
    ],
    "heading-tab-ac002-unchecked": [
        ("## Acceptance Criteria", "##\tAcceptance Criteria"),
        ("- [x] AC-002:", "- [ ] AC-002:"),
    ],
    "heading-indented-ac002-unchecked": [
        ("## Acceptance Criteria", "   ## Acceptance Criteria"),
        ("- [x] AC-002:", "- [ ] AC-002:"),
    ],
    "star-ac002-unchecked": [("- [x] AC-002:", "* [ ] AC-002:")],
    "plus-ac002-unchecked": [("- [x] AC-002:", "+ [ ] AC-002:")],
    "heading-lowercase-ac002-unchecked": [
        ("## Acceptance Criteria", "## acceptance criteria"),
        ("- [x] AC-002:", "- [ ] AC-002:"),
    ],
    "double-space-ac002-unchecked": [("- [x] AC-002:", "-  [ ] AC-002:")],
    "ordered-ac002-unchecked": [("- [x] AC-002:", "1. [ ] AC-002:")],
    "fenced-heading-ac002-unchecked": [("- [x] AC-002:", "```\n## Example\n```\n- [ ] AC-002:")],
    "noise-ac002-unchecked": [("- [x] AC-002:", "- [ ] AC-002:")],
    "suffix-continued-ac004-unchecked": [
        ("## Verification Evidence", "## Acceptance Criteria (continued)\n- [ ] AC-004: late item\n\n## Verification Evidence")
    ],
    "suffix-colon-ac004-unchecked": [
        ("## Verification Evidence", "## Acceptance Criteria: deferred\n- [ ] AC-004: late item\n\n## Verification Evidence")
    ],
    "suffix-lowercase-ac004-unchecked": [
        ("## Verification Evidence", "  ## acceptance criteria (continued)\n- [ ] AC-004: late item\n\n## Verification Evidence")
    ],
    "heading-nbsp-terminator-ac004-unchecked": [
        ("## Verification Evidence", "##\u00a0Notes\n- [ ] AC-004: late item\n\n## Verification Evidence")
    ],
    "heading-vtab-terminator-ac004-unchecked": [
        ("## Verification Evidence", "##\x0bNotes\n- [ ] AC-004: late item\n\n## Verification Evidence")
    ],
    "heading-ideographic-terminator-ac004-unchecked": [
        ("## Verification Evidence", "##\u3000Notes\n- [ ] AC-004: late item\n\n## Verification Evidence")
    ],
    "python-fails": [],
    "python-huge-count": [],
    "python-garbage-count": [],
    "second-section-unchecked": [("## Verification Evidence", "## Acceptance Criteria\n- [ ] AC-004: late item\n\n## Verification Evidence")],
    "second-section-duplicate-id": [("## Verification Evidence", "## Acceptance Criteria\n- [x] AC-001: repeated\n\n## Verification Evidence")],
    "verdict-no-go-above-go": [("Verdict: GO", "Verdict: NO-GO\n\nVerdict: GO")],
    "verdict-no-go-below-go": [("Verdict: GO", "Verdict: GO\n\nVerdict: NO-GO")],
    "verdict-bold-colon-inside-no-go": [("Verdict: GO", "**Verdict:** NO-GO\n\nVerdict: GO")],
    "verdict-bold-colon-outside-no-go": [("Verdict: GO", "**Verdict**: NO-GO\n\nVerdict: GO")],
    "verdict-italic-no-go": [("Verdict: GO", "*Verdict:* NO-GO\n\nVerdict: GO")],
    "verdict-underscore-no-go": [("Verdict: GO", "__Verdict:__ NO-GO\n\nVerdict: GO")],
    "verdict-bold-value-no-go": [("Verdict: GO", "Verdict: GO\n\n**Verdict: NO-GO**")],
    "verdict-indented-bold-no-go-below-go": [("Verdict: GO", "Verdict: GO\n\n  **Verdict:** NO-GO")],
    "verdict-bold-go": [("Verdict: GO", "**Verdict:** GO")],
    "verdict-line-above": [("Verdict: GO", sys.argv[3] + "\n\nVerdict: GO")],
    "verdict-line-below": [("Verdict: GO", "Verdict: GO\n\n" + sys.argv[3])],
    "verdict-line-only": [("Verdict: GO", sys.argv[3])],
    "verdict-crlf-go": [],
    "verdict-bom-go": [("# Seal:", "\ufeff# Seal:")],
}[case]
for old, new in edits:
    if text.count(old) != 1:
        raise SystemExit(f"seal.md: expected exactly one {old!r}, found {text.count(old)}")
    text = text.replace(old, new)
if case == "verdict-crlf-go":
    text = text.replace("\n", "\r\n")
seal.write_text(text)
PY
  local output_ code_
  set +e
  output_="$(env ${run_env:+"$run_env"} bash "$ROOT/scripts/grade-feature.sh" --json "$ws" 2>"$ws.err")"
  code_=$?
  set -e
  if ! printf '%s' "$output_" | python3 -c '
import json, sys
want_status, want_rules = sys.argv[1:]
data = json.load(sys.stdin)
got = ",".join(data["rule_ids"])
if data["status"] != want_status:
    raise SystemExit("status=%r, want %r" % (data["status"], want_status))
if got != want_rules:
    raise SystemExit("rule_ids=%r, want %r" % (got, want_rules))
for problem in data["problems"]:
    if problem["rule_id"] == "final.acceptance.unchecked" and "1 unchecked acceptance criterion" not in problem["message"]:
        raise SystemExit("message=%r does not name the missing criterion" % problem["message"])
' "$want_status" "$want_rules"; then
    printf 'FAIL %s: grader exit=%s output=%s stderr=%s\n' "$id" "$code_" "$output_" "$(cat "$ws.err")" >&2
    exit 1
  fi
  [ "$code_" -eq "$( [ "$want_status" = GO ] && echo 0 || echo 1 )" ] || {
    printf 'FAIL %s: grader exit=%s, output=%s\n' "$id" "$code_" "$output_" >&2
    exit 1
  }
  printf '  PASS %-38s %s %s\n' "$id" "$want_status" "$want_rules"
}

grader_case baseline GO ""
grader_case heading-lowercase GO ""
grader_case ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case indented-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-lowercase-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-trailing-space-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-tab-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-indented-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case star-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case plus-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case double-space-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case ordered-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case fenced-heading-ac002-unchecked NO-GO "final.acceptance.unchecked"
grader_case verdict-no-go-above-go NO-GO "final.verdict.not-go"
grader_case verdict-no-go-below-go NO-GO "final.verdict.not-go"
grader_case verdict-bold-colon-inside-no-go NO-GO "final.verdict.not-go"
grader_case verdict-bold-colon-outside-no-go NO-GO "final.verdict.not-go"
grader_case verdict-italic-no-go NO-GO "final.verdict.not-go"
grader_case verdict-underscore-no-go NO-GO "final.verdict.not-go"
grader_case verdict-bold-value-no-go NO-GO "final.verdict.not-go"
grader_case verdict-indented-bold-no-go-below-go NO-GO "final.verdict.not-go"
grader_case verdict-bold-go GO ""
grader_case verdict-crlf-go GO ""
grader_case verdict-bom-go GO ""

# A recognised Verdict line that is not GO is NO-GO wherever it sits; a GO
# written the same ways, alone, stays GO.
while IFS= read -r spelling; do
  for where in above below; do
    grader_case "verdict-line-$where" NO-GO "final.verdict.not-go" "" "$spelling"
  done
done <<'EOF'
- Verdict: NO-GO
* Verdict: NO-GO
+ Verdict: NO-GO
1. Verdict: NO-GO
1) Verdict: NO-GO
- **Verdict:** NO-GO
- [ ] Verdict: NO-GO
# Verdict: NO-GO
## Verdict: NO-GO
### Verdict: NO-GO
## **Verdict:** NO-GO
> Verdict: NO-GO
> **Verdict:** NO-GO
>Verdict: NO-GO
Verdict : NO-GO
Verdict :NO-GO
| Verdict | NO-GO |
| **Verdict** | NO-GO |
Verdict: **NO-GO**
EOF
for spelling in '- Verdict: GO' '> Verdict: GO' '## Verdict: GO' 'Verdict : GO' 'Verdict: **GO**' '| Verdict | GO |'; do
  grader_case verdict-line-only GO "" "" "$spelling"
done

mkdir -p "$tmp/noise" "$tmp/py-fails" "$tmp/py-garbage" "$tmp/py-huge"
printf 'import sys\nsys.stderr.write("sitecustomize noise\\n")\n' >"$tmp/noise/sitecustomize.py"
real_python="$(command -v python3)"
for stub in py-fails:'exit 3' py-garbage:'echo not-a-number' py-huge:'echo 99999999999999999999; exit 0'; do
  printf '#!/bin/sh\ncase "$1" in -c) %s ;; esac\nexec "%s" "$@"\n' "${stub#*:}" "$real_python" >"$tmp/${stub%%:*}/python3"
  chmod +x "$tmp/${stub%%:*}/python3"
done
grader_case noise-ac002-unchecked NO-GO "final.acceptance.unchecked" "PYTHONPATH=$tmp/noise"
grader_case python-fails NO-GO "final.acceptance.ids" "PATH=$tmp/py-fails:$PATH"
grader_case python-garbage-count NO-GO "final.acceptance.ids" "PATH=$tmp/py-garbage:$PATH"
grader_case python-huge-count NO-GO "final.acceptance.ids" "PATH=$tmp/py-huge:$PATH"
grader_case second-section-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-nbsp-terminator-ac004-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-vtab-terminator-ac004-unchecked NO-GO "final.acceptance.unchecked"
grader_case heading-ideographic-terminator-ac004-unchecked NO-GO "final.acceptance.unchecked"
grader_case suffix-continued-ac004-unchecked NO-GO "final.acceptance.unchecked"
grader_case suffix-colon-ac004-unchecked NO-GO "final.acceptance.unchecked"
grader_case suffix-lowercase-ac004-unchecked NO-GO "final.acceptance.unchecked"
grader_case second-section-duplicate-id NO-GO "final.acceptance.ids"

mkdir -p "$tmp/host-artifacts"
discovery="$(
  DEVRITES_OUTCOME_DISCOVERY_ONLY=1 \
  DEVRITES_HOST_ARTIFACT_DIR="$tmp/host-artifacts" \
  DEVRITES_ENGINE_CLI="$ROOT/scripts/run-outcome-evals.sh" \
    node "$ROOT/scripts/run-tests.mjs" --serial outcome-evals-test
)"
grep -Fq "outcome-evals discovery sentinel" <<<"$discovery"
grep -Fq "PASS: tests/outcome-evals-test.sh" <<<"$discovery"

echo "outcome eval regression: PASS"
