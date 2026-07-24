#!/usr/bin/env bash
# Run the deterministic final-outcome matrix. The canonical fixture passes the
# shell grader and Go seal gate. Each failing case changes one input.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GRADER="$ROOT/scripts/grade-feature.sh"
GOOD="$ROOT/evals/golden/shippable-feature"
MANIFEST="$ROOT/engine/internal/state/workflow_manifest.json"
READINESS="$ROOT/engine/internal/lib/readiness_contract.json"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

ENGINE="$tmp/devrites-engine"
(cd "$ROOT/engine" && CGO_ENABLED=0 go build -trimpath -o "$ENGINE" .)
export DEVRITES_CLI="$ENGINE"

# A fake PATH binary exposes accidental fallback. The grader must use the exact
# binary built above through DEVRITES_CLI.
mkdir -p "$tmp/stale-path"
cat >"$tmp/stale-path/devrites-engine" <<'SH'
#!/usr/bin/env sh
echo "stale PATH shim executed" >&2
exit 99
SH
chmod +x "$tmp/stale-path/devrites-engine"
export PATH="$tmp/stale-path:$PATH"

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

assert_seal() {
  local want_reason="$1"
  printf '%s' "$CAPTURED" | python3 -c '
import json
import sys

data = json.load(sys.stdin)
want = sys.argv[1]
if data["schema"] != "devrites-command/v1" or data["command"] != "seal":
    raise SystemExit(f"unexpected seal envelope: {data!r}")
if data.get("reason_id") != want:
    raise SystemExit("seal reason={!r}, want {!r}".format(data.get("reason_id"), want))
' "$want_reason"
}

stage_workspace() {
  local project="$1"
  local slug="$2"
  local mode="$3"
  local ws="$project/.devrites/work/$slug"
  mkdir -p "$ws"
  cp -R "$GOOD/." "$ws/"
  if [ "$mode" = "readiness" ]; then
    python3 - "$ws/state.md" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text()
for old, new in (
    ("| phase | done |", "| phase | build |"),
    ("| status | done |", "| status | running |"),
):
    if text.count(old) != 1:
        raise SystemExit(f"{path}: expected exactly one {old!r}")
    text = text.replace(old, new)
path.write_text(text)
PY
  fi

  python3 - "$project" "$ws" <<'PY'
from pathlib import Path
import os
import re
import sys

project, workspace = map(Path, sys.argv[1:])
paths = re.findall(r"`([^`]+)`", (workspace / "touched-files.md").read_text())
for relative in paths:
    path = project / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("synthetic outcome-eval source\n")
    os.utime(path, (1_700_000_000, 1_700_000_000))
os.utime(workspace / "evidence.md", (1_700_000_100, 1_700_000_100))
PY
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
    rows = {}
    for line in path.read_text().splitlines():
        key = line.split("|", 1)[0]
        rows[key] = line
    return rows

before, after = load(before_path), load(after_path)
changed = sorted(
    key for key in before.keys() | after.keys()
    if before.get(key) != after.get(key)
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
    unchecked_ac)
      replace_once "$ws/seal.md" "- [x] AC-001:" "- [ ] AC-001:"
      ;;
    wrong_ac_id)
      replace_once "$ws/seal.md" "AC-003:" "AC-999:"
      ;;
    *) fail "unknown final mutation $id" ;;
  esac
}

mutation_path() {
  local id="$1"
  local slug="$2"
  case "$id" in
    missing_review) printf '.devrites/work/%s/review.md\n' "$slug" ;;
    missing_evidence|empty_evidence) printf '.devrites/work/%s/evidence.md\n' "$slug" ;;
    missing_seal|non_go|unresolved_blocker|unchecked_ac|wrong_ac_id)
      printf '.devrites/work/%s/seal.md\n' "$slug"
      ;;
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

  stage_workspace "$project" "$slug" final
  snapshot_tree "$project" "$before"
  mutate_final "$id" "$project" "$slug"
  snapshot_tree "$project" "$after"
  assert_one_mutation "$before" "$after" "$(mutation_path "$id" "$slug")"

  run_capture bash "$GRADER" --json "$ws"
  [ "$CAPTURE_CODE" -eq 1 ] || fail "$id grader exit=$CAPTURE_CODE; $CAPTURED"
  assert_grade "NO-GO" "$want_rule"

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" seal --json "$slug"
  [ "$CAPTURE_CODE" -eq "$want_seal_code" ] \
    || fail "$id seal exit=$CAPTURE_CODE, want $want_seal_code; $CAPTURED"
  assert_seal "$want_reason"
  printf '  PASS %-20s rule=%s reason=%s\n' "$id" "$want_rule" "$want_reason"
}

readiness_code() {
  local id="$1"
  python3 - "$READINESS" "$id" <<'PY'
import json
import sys

contract = json.load(open(sys.argv[1]))
want = sys.argv[2]
matches = [row for row in contract["reasons"] if row["id"] == want]
if len(matches) != 1:
    raise SystemExit(f"readiness reason {want!r} is not unique")
print(matches[0]["code"])
PY
}

run_readiness_case() {
  local id="$1"
  local file="$2"
  local readiness_id="$3"
  local project="$tmp/readiness-$id"
  local slug="case-$id"
  local ws="$project/.devrites/work/$slug"
  local before="$tmp/$id.before"
  local after="$tmp/$id.after"
  local want_code
  want_code="$(readiness_code "$readiness_id")"

  stage_workspace "$project" "$slug" readiness
  run_capture env DEVRITES_ROOT="$project" "$ENGINE" build-readiness "$slug"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id readiness baseline failed: $CAPTURED"

  snapshot_tree "$project" "$before"
  printf '\nOutcome-eval drift.\n' >>"$ws/$file"
  snapshot_tree "$project" "$after"
  assert_one_mutation "$before" "$after" ".devrites/work/$slug/$file"

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" build-readiness "$slug"
  [ "$CAPTURE_CODE" -eq "$want_code" ] \
    || fail "$id readiness exit=$CAPTURE_CODE, want $want_code; $CAPTURED"

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" seal --json "$slug"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id structural seal failed: $CAPTURED"
  assert_seal "DRV-GATE-SEAL-PASSED"
  printf '  PASS %-20s readiness=%s(code=%s) reason=DRV-GATE-SEAL-PASSED\n' \
    "$id" "$readiness_id" "$want_code"
}

run_stale_case() {
  local id="stale_proof"
  local project="$tmp/$id"
  local slug="case-$id"
  local ws="$project/.devrites/work/$slug"
  local source="src/routes/transactions/export.ts"
  local before="$tmp/$id.before"
  local after="$tmp/$id.after"

  stage_workspace "$project" "$slug" final
  snapshot_tree "$project" "$before"
  python3 - "$project/$source" <<'PY'
import os
import sys
os.utime(sys.argv[1], (1_700_000_200, 1_700_000_200))
PY
  snapshot_tree "$project" "$after"
  assert_one_mutation "$before" "$after" "$source"

  run_capture bash "$GRADER" --json "$ws"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id final-only grader failed: $CAPTURED"
  assert_grade "GO" ""

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" evidence-fresh "$slug"
  [ "$CAPTURE_CODE" -eq 3 ] || fail "$id evidence exit=$CAPTURE_CODE, want 3; $CAPTURED"

  run_capture env DEVRITES_ROOT="$project" "$ENGINE" seal --json "$slug"
  [ "$CAPTURE_CODE" -eq 0 ] || fail "$id structural seal failed: $CAPTURED"
  assert_seal "DRV-GATE-SEAL-PASSED"
  printf '  PASS %-20s evidence=stale(code=3) reason=DRV-GATE-SEAL-PASSED\n' "$id"
}

printf '== canonical owner inventory ==\n'
python3 - "$MANIFEST" "$READINESS" "$GOOD" <<'PY'
from pathlib import Path
import json
import sys

manifest = json.load(open(sys.argv[1]))
readiness = json.load(open(sys.argv[2]))
workspace = Path(sys.argv[3])
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
contract_artifacts = [
    readiness["coverage"]["artifact"],
    readiness["engineering"]["artifact"],
    readiness["testPlan"]["artifact"],
]
missing_contract = [name for name in contract_artifacts if not (workspace / name).is_file()]
if missing_contract:
    raise SystemExit(f"canonical fixture missing readiness artifacts: {missing_contract}")
for name in ("seal.md", "review.md"):
    if not (workspace / name).is_file():
        raise SystemExit(f"canonical fixture missing final-only artifact: {name}")
print(f"  PASS: {len(required)} seal artifacts + {len(contract_artifacts)} readiness bindings + final-only files")
PY
run_capture python3 "$ROOT/scripts/validate-workspace-schema.py" "$GOOD"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical workspace schema failed: $CAPTURED"
printf '  PASS: workspace schema validator\n'

printf '\n== canonical base: shell grader + exact Go gates ==\n'
run_capture bash "$GRADER" --json "$GOOD"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical grader failed: $CAPTURED"
assert_grade "GO" ""

base_project="$tmp/canonical-base"
base_slug="shippable-feature"
stage_workspace "$base_project" "$base_slug" final
run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" evidence-fresh "$base_slug"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical evidence freshness failed: $CAPTURED"
run_capture env DEVRITES_ROOT="$base_project" "$ENGINE" seal --json "$base_slug"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical seal failed: $CAPTURED"
assert_seal "DRV-GATE-SEAL-PASSED"

readiness_project="$tmp/canonical-readiness"
readiness_slug="readiness-feature"
stage_workspace "$readiness_project" "$readiness_slug" readiness
run_capture env DEVRITES_ROOT="$readiness_project" "$ENGINE" build-readiness "$readiness_slug"
[ "$CAPTURE_CODE" -eq 0 ] || fail "canonical readiness projection failed: $CAPTURED"
printf '  PASS: grader GO, seal DRV-GATE-SEAL-PASSED, evidence fresh, readiness ready\n'

printf '\n== one-delta final-outcome matrix ==\n'
run_final_case missing_review final.review.missing 0 DRV-GATE-SEAL-PASSED
run_final_case missing_evidence final.evidence.missing-or-empty 3 DRV-GATE-SEAL-MISSING
run_final_case empty_evidence final.evidence.missing-or-empty 3 DRV-GATE-SEAL-MISSING
run_final_case missing_seal final.seal.missing 0 DRV-GATE-SEAL-PASSED
run_final_case non_go final.verdict.not-go 0 DRV-GATE-SEAL-PASSED
run_final_case unresolved_blocker final.blockers.unresolved 0 DRV-GATE-SEAL-PASSED
run_final_case open_question final.questions.open 0 DRV-GATE-SEAL-PASSED
run_final_case wrong_phase final.state.phase 0 DRV-GATE-SEAL-PASSED
run_final_case awaiting_human final.state.status 0 DRV-GATE-SEAL-PASSED
run_final_case blocked_status final.state.status 0 DRV-GATE-SEAL-PASSED
run_final_case unchecked_ac final.acceptance.unchecked 0 DRV-GATE-SEAL-PASSED
run_final_case wrong_ac_id final.acceptance.ids 0 DRV-GATE-SEAL-PASSED
run_stale_case
run_readiness_case coverage_digest_drift brief.md coverage-not-clear
run_readiness_case engineering_digest_drift architecture.md engineering-not-ready

printf '\n== legacy compatibility fixtures (not matrix proof) ==\n'
for legacy in blocked-feature near-miss-unproven-ac; do
  run_capture bash "$GRADER" --json "$ROOT/evals/golden/$legacy"
  [ "$CAPTURE_CODE" -eq 1 ] || fail "$legacy should remain NO-GO"
  printf '  PASS: %s remains NO-GO\n' "$legacy"
done

printf '\nOutcome evals passed: canonical parity + 15 isolated negative rows.\n'
