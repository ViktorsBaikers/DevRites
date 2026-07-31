#!/usr/bin/env bash
# Exercise the deterministic artifact boundary with one freshly built engine.
# Native agents own semantic readiness/review; the engine owns structure,
# freshness, and state mechanics.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GRADER="$ROOT/scripts/grade-feature.sh"
GOOD="$ROOT/evals/golden/shippable-feature"
MANIFEST="$ROOT/engine/internal/state/workflow_manifest.json"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

ENGINE="$tmp/devrites-engine"
(cd "$ROOT/engine" && GOCACHE="$tmp/go-cache" CGO_ENABLED=0 go build -trimpath -o "$ENGINE" .)

CAPTURED=""
CAPTURE_CODE=0
run_capture() {
  set +e
  CAPTURED="$("$@" 2>&1)"
  CAPTURE_CODE=$?
  set -e
}

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_grade() {
  local want_status="$1"
  local want_rules="$2"
  printf '%s' "$CAPTURED" | python3 -c '
import json
import sys

want_status, want_rules = sys.argv[1:]
data = json.load(sys.stdin)
got_rules = ",".join(data["rule_ids"])
if data["schema"] != "devrites-outcome-grade/v1":
    raise SystemExit("grade schema={!r}".format(data.get("schema")))
if data["status"] != want_status:
    raise SystemExit("grade status={!r}, want {!r}".format(data["status"], want_status))
if got_rules != want_rules:
    raise SystemExit("grade rules={!r}, want {!r}".format(got_rules, want_rules))
' "$want_status" "$want_rules"
}

assert_check_seal() {
  local want_reason="$1"
  printf '%s\n' "$CAPTURED" | grep -Fxq "reason: $want_reason" \
    || fail "check-seal reason missing: want $want_reason; $CAPTURED"
}

stage_workspace() {
  local project="$1"
  local slug="$2"
  local ws="$project/.devrites/work/$slug"
  mkdir -p "$ws"
  cp -R "$GOOD/." "$ws/"

  python3 - "$project" "$ws" <<'PY'
from pathlib import Path
import re
import sys

project, workspace = map(Path, sys.argv[1:])
paths = []
for line in (workspace / "touched-files.md").read_text().splitlines():
    match = re.fullmatch(r"\| present \| `([^`]+)` \| [^|]+ \| [^|]+ \|", line)
    if match:
        paths.append(match.group(1))
for relative in paths:
    path = project / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("synthetic outcome-eval source\n")
PY

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" check readiness --emit-binding "$slug"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "stage readiness binding failed: $CAPTURED"
  local readiness_digest
  readiness_digest="$(printf '%s\n' "$CAPTURED" | sed -n 's/^Readiness inputs SHA-256: \([0-9a-f]\{64\}\)$/\1/p')"
  [ "${#readiness_digest}" -eq 64 ] || fail "stage readiness binding malformed: $CAPTURED"
  replace_once "$ws/eng-review.md" "__READINESS_SHA256__" "$readiness_digest"

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" check candidate "$slug"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "stage candidate failed: $CAPTURED"
  local digest
  digest="$(printf '%s\n' "$CAPTURED" | sed -n 's/^candidate-sha256: \([0-9a-f]\{64\}\)$/\1/p')"
  [ "${#digest}" -eq 64 ] || fail "stage candidate digest malformed: $CAPTURED"
  for artifact in evidence.md review.md seal.md; do
    replace_once "$ws/$artifact" "__CANDIDATE_SHA256__" "$digest"
  done
}

snapshot_tree() {
  local project="$1"
  local output="$2"
  python3 - "$project" "$output" <<'PY'
from pathlib import Path
import hashlib
import sys

root, output = map(Path, sys.argv[1:])
rows = []
for path in sorted(root.rglob("*")):
    if not path.is_file():
        continue
    stat = path.stat()
    rows.append(
        f"{path.relative_to(root)}|{stat.st_mode & 0o777:o}|{stat.st_mtime_ns}|"
        f"{hashlib.sha256(path.read_bytes()).hexdigest()}"
    )
output.write_text("\n".join(rows) + "\n")
PY
}

assert_one_mutation() {
  local before="$1"
  local after="$2"
  local want_path="$3"
  python3 - "$before" "$after" "$want_path" <<'PY'
from pathlib import Path
import sys

before_path, after_path = map(Path, sys.argv[1:3])
want = sys.argv[3]


def load(path):
    return {line.split("|", 1)[0]: line for line in path.read_text().splitlines()}


before, after = load(before_path), load(after_path)
changed = sorted(
    key for key in before.keys() | after.keys() if before.get(key) != after.get(key)
)
if changed != [want]:
    raise SystemExit(f"causal paths={changed!r}, want {[want]!r}")
PY
}

replace_once() {
  local path="$1"
  local old="$2"
  local new="$3"
  python3 - "$path" "$old" "$new" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
old, new = sys.argv[2:]
text = path.read_text()
if text.count(old) != 1:
    raise SystemExit(f"{path}: expected exactly one {old!r}, found {text.count(old)}")
path.write_text(text.replace(old, new))
PY
}

remove_acceptance_ids() {
  python3 - "$1" <<'PY'
from pathlib import Path
import re
import sys

path = Path(sys.argv[1])
text = path.read_text()
match = re.search(r"(?ms)(^## Acceptance criteria\s*$\n)(.*?)(?=^##\s)", text)
if not match:
    raise SystemExit(f"{path}: acceptance section missing")
body = match.group(2).replace("AC-", "XX-")
path.write_text(text[:match.start(2)] + body + text[match.end(2):])
PY
}

mutate_final() {
  local id="$1"
  local project="$2"
  local slug="$3"
  local ws="$project/.devrites/work/$slug"
  case "$id" in
    missing_review) rm "$ws/review.md" ;;
    missing_evidence) rm "$ws/evidence.md" ;;
    empty_evidence) : >"$ws/evidence.md" ;;
    missing_seal) rm "$ws/seal.md" ;;
    non_go) replace_once "$ws/seal.md" "Verdict: GO" "Verdict: NO-GO" ;;
    unresolved_blocker)
      replace_once "$ws/seal.md" $'## Blockers\nnone' $'## Blockers\n- release blocker'
      ;;
    open_question)
      replace_once "$ws/questions.md" "| Q-001 | answered |" "| Q-001 | open |"
      ;;
    wrong_phase) replace_once "$ws/state.md" "| phase | done |" "| phase | review |" ;;
    awaiting_human)
      replace_once "$ws/state.md" "| status | done |" "| status | awaiting_human |"
      ;;
    blocked_status) replace_once "$ws/state.md" "| status | done |" "| status | blocked |" ;;
    unchecked_ac) replace_once "$ws/seal.md" "- [x] AC-001:" "- [ ] AC-001:" ;;
    wrong_ac_id) replace_once "$ws/seal.md" "AC-003:" "AC-999:" ;;
    noncanonical_ac) replace_once "$ws/seal.md" "AC-001:" "AC1:" ;;
    duplicate_ac) replace_once "$ws/seal.md" "- [x] AC-001:" "- [x] AC-001: AC-001" ;;
    empty_ac) remove_acceptance_ids "$ws/spec.md" ;;
    *) fail "unknown final mutation $id" ;;
  esac
}

mutation_path() {
  local id="$1"
  local slug="$2"
  case "$id" in
    missing_review) printf '.devrites/work/%s/review.md\n' "$slug" ;;
    missing_evidence|empty_evidence) printf '.devrites/work/%s/evidence.md\n' "$slug" ;;
    missing_seal|non_go|unresolved_blocker|unchecked_ac|wrong_ac_id|noncanonical_ac|duplicate_ac)
      printf '.devrites/work/%s/seal.md\n' "$slug"
      ;;
    empty_ac) printf '.devrites/work/%s/spec.md\n' "$slug" ;;
    open_question) printf '.devrites/work/%s/questions.md\n' "$slug" ;;
    wrong_phase|awaiting_human|blocked_status)
      printf '.devrites/work/%s/state.md\n' "$slug"
      ;;
    *) fail "unknown mutation path $id" ;;
  esac
}

run_final_case() {
  local id="$1"
  local want_rule="$2"
  local want_seal_code="$3"
  local want_reason="$4"
  local project="$tmp/final-$id"
  local slug="case-$id"
  local before="$tmp/$id.before"
  local after="$tmp/$id.after"
  local ws="$project/.devrites/work/$slug"

  stage_workspace "$project" "$slug"
  snapshot_tree "$project" "$before"
  mutate_final "$id" "$project" "$slug"
  snapshot_tree "$project" "$after"
  assert_one_mutation "$before" "$after" "$(mutation_path "$id" "$slug")"

  run_capture bash "$GRADER" --json "$ws"
  [ "$CAPTURE_CODE" -eq 1 ] || fail "$id grader exit=$CAPTURE_CODE; $CAPTURED"
  assert_grade "NO-GO" "$want_rule"
  case "$id" in
    wrong_ac_id) printf '%s' "$CAPTURED" | grep -Fq 'acceptance ID mismatch' || fail "$id did not prove exact ID equality" ;;
    noncanonical_ac) printf '%s' "$CAPTURED" | grep -Fq 'noncanonical acceptance IDs' || fail "$id did not prove canonical IDs" ;;
    duplicate_ac) printf '%s' "$CAPTURED" | grep -Fq 'duplicate acceptance IDs' || fail "$id did not prove ID uniqueness" ;;
    empty_ac) printf '%s' "$CAPTURED" | grep -Fq 'acceptance IDs are empty' || fail "$id did not prove nonempty IDs" ;;
  esac

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" check seal "$slug"
  [ "$CAPTURE_CODE" -eq "$want_seal_code" ] \
    || fail "$id check seal exit=$CAPTURE_CODE, want $want_seal_code; $CAPTURED"
  assert_check_seal "$want_reason"
  printf '  PASS %-20s rule=%s structural=%s\n' "$id" "$want_rule" "$want_reason"
}

run_content_identity_case() {
  local id="content_identity"
  local project="$tmp/$id"
  local slug="case-$id"
  local ws="$project/.devrites/work/$slug"
  local source="src/routes/transactions/export.ts"
  local before="$tmp/$id.before"
  local after="$tmp/$id.after"
  local before_content="$tmp/$id.before-content"
  local after_content="$tmp/$id.after-content"

  stage_workspace "$project" "$slug"
  snapshot_tree "$project" "$before"
  python3 - "$project/$source" <<'PY'
import os
import sys

os.utime(sys.argv[1], (1_700_000_200, 1_700_000_200))
PY
  snapshot_tree "$project" "$after"
  assert_one_mutation "$before" "$after" "$source"

  run_capture bash "$GRADER" --json "$ws"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id content grader failed: $CAPTURED"
  assert_grade "GO" ""

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" check seal "$slug"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id unchanged touch check seal exit=$CAPTURE_CODE, want 0: $CAPTURED"
  assert_check_seal "DRV-GATE-SEAL-PASSED"

  snapshot_tree "$project" "$before_content"
  python3 - "$project/$source" <<'PY'
import os
from pathlib import Path
import sys

path = Path(sys.argv[1])
stat = path.stat()
path.write_text("synthetic outcome-eval source changed\n")
os.utime(path, ns=(stat.st_atime_ns, stat.st_mtime_ns))
PY
  snapshot_tree "$project" "$after_content"
  assert_one_mutation "$before_content" "$after_content" "$source"

  run_capture bash "$GRADER" --json "$ws"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id changed-content grader failed: $CAPTURED"
  assert_grade "GO" ""
  run_capture env DEVRITES_ROOT="$project" "$ENGINE" check seal "$slug"
  [ "$CAPTURE_CODE" -eq 3 ] || fail "$id changed-content check seal exit=$CAPTURE_CODE, want 3: $CAPTURED"
  assert_check_seal "DRV-GATE-SEAL-MISSING"
  printf '  PASS %-20s unchanged-touch=pass restored-mtime-byte-change=blocked\n' "$id"
}

printf '== canonical structural owner ==\n'
python3 - "$MANIFEST" "$GOOD" <<'PY'
from pathlib import Path
import json
import sys

manifest = json.load(open(sys.argv[1]))
workspace = Path(sys.argv[2])
seal = [phase for phase in manifest["phases"] if phase["id"] == "seal"]
if len(seal) != 1 or not seal[0].get("shippable"):
    raise SystemExit("workflow manifest must own one shippable seal phase")
required = seal[0]["workspaceRequired"]
missing = [
    name for name in required
    if not (workspace / name).is_file() or not (workspace / name).read_text().strip()
]
if missing:
    raise SystemExit(f"canonical fixture missing seal artifacts: {missing}")
for name in ("seal.md", "review.md"):
    if not (workspace / name).is_file():
        raise SystemExit(f"canonical fixture missing final-only artifact: {name}")
print(f"  PASS: {len(required)} seal artifacts + final-only files")
PY
run_capture python3 "$ROOT/scripts/validate-workspace-schema.py" "$GOOD"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical workspace schema failed: $CAPTURED"
printf '  PASS: workspace schema\n'

printf '\n== canonical content grader + structural gates ==\n'
run_capture bash "$GRADER" --json "$GOOD"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical grader failed: $CAPTURED"
assert_grade "GO" ""

base_project="$tmp/canonical-base"
base_slug="shippable-feature"
stage_workspace "$base_project" "$base_slug"
run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" check readiness "$base_slug"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical check readiness failed: $CAPTURED"
run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" check seal "$base_slug"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical check seal failed: $CAPTURED"
assert_check_seal "DRV-GATE-SEAL-PASSED"
printf '  PASS: grader GO + structural readiness + fresh seal\n'

printf '\n== one-delta final-outcome matrix ==\n'
run_final_case missing_review final.review.missing 3 DRV-GATE-SEAL-MISSING
run_final_case missing_evidence final.evidence.missing-or-empty 3 DRV-GATE-SEAL-MISSING
run_final_case empty_evidence final.evidence.missing-or-empty 3 DRV-GATE-SEAL-MISSING
run_final_case missing_seal final.seal.missing 3 DRV-GATE-SEAL-MISSING
run_final_case non_go final.verdict.not-go 0 DRV-GATE-SEAL-PASSED
run_final_case unresolved_blocker final.blockers.unresolved 0 DRV-GATE-SEAL-PASSED
run_final_case open_question final.questions.open 3 DRV-GATE-SEAL-MISSING
run_final_case wrong_phase final.state.phase 0 DRV-GATE-SEAL-PASSED
run_final_case awaiting_human final.state.status 0 DRV-GATE-SEAL-PASSED
run_final_case blocked_status final.state.status 0 DRV-GATE-SEAL-PASSED
run_final_case unchecked_ac final.acceptance.unchecked 0 DRV-GATE-SEAL-PASSED
run_final_case wrong_ac_id final.acceptance.ids 0 DRV-GATE-SEAL-PASSED
run_final_case noncanonical_ac final.acceptance.ids 0 DRV-GATE-SEAL-PASSED
run_final_case duplicate_ac final.acceptance.ids 0 DRV-GATE-SEAL-PASSED
run_final_case empty_ac final.acceptance.ids 3 DRV-GATE-READINESS-STALE
run_content_identity_case

printf '\n== removed top-level commands ==\n'
removed=(
  readiness seal spec-validate check-acceptance evidence-fresh coverage
  doubt-coverage test-integrity review-integrity build-readiness
  readiness-digest analyze ledger resolve clarify-return tick-afk recovery
  close-out migrate doctor
)
[ "${#removed[@]}" -eq 20 ] || fail "removed command inventory is not 20"
for command in "${removed[@]}"; do
  run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" "$command"
  [ "$CAPTURE_CODE" -eq 2 ] || fail "$command exit=$CAPTURE_CODE, want 2; $CAPTURED"
  printf '%s' "$CAPTURED" | grep -Fq "unknown command \"$command\"" \
    || fail "$command did not use unknown-command path: $CAPTURED"
done
printf '  PASS: all 20 retired top-level commands are unknown (no aliases)\n'

printf '\n== removed nested policy commands ==\n'
assert_removed_nested() {
  run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" "$@"
  [ "$CAPTURE_CODE" -ne 0 ] || fail "$* unexpectedly succeeded: $CAPTURED"
}
assert_removed_nested check spec "$GOOD"
assert_removed_nested state clarify enter "$base_slug"
assert_removed_nested state tick-afk "$base_project/.devrites/work/$base_slug/state.md"
assert_removed_nested state recovery check synthetic-cause "$base_slug"
printf '%s\n' "$base_slug" >"$base_project/.devrites/ACTIVE"
run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" state resolve next-qid "$GOOD/questions.md"
[ "$CAPTURE_CODE" -ne 0 ] || fail "state resolve next-qid unexpectedly generated an id: $CAPTURED"
printf '  PASS: removed nested policy forms cannot execute\n'

printf '\n== compatibility examples ==\n'
for legacy in blocked-feature near-miss-unproven-ac; do
  run_capture bash "$GRADER" --json "$ROOT/evals/golden/$legacy"
  [ "$CAPTURE_CODE" -eq 1 ] || fail "$legacy should remain NO-GO"
  printf '  PASS: %s remains NO-GO\n' "$legacy"
done

printf '\nOutcome evals passed: native boundary + 15 isolated final-outcome negatives + candidate/readiness content binding + removed-command rejections.\n'
