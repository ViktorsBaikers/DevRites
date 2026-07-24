#!/usr/bin/env bash
# Regression checks for the agent-skills refresh validators: routing reports,
# skill anatomy, host command parity, and agent composition.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== agent-skills-refresh-validation-test =="

run_ok() {
  label="$1"; shift
  out="$T/${label//[^A-Za-z0-9_]/_}.out"
  if "$@" >"$out" 2>&1; then
    ok "$label"
  else
    no "$label"
    sed -n '1,80p' "$out"
  fi
}

run_fail_contains() {
  label="$1"; needle="$2"; shift 2
  out="$T/${label//[^A-Za-z0-9_]/_}.out"
  if "$@" >"$out" 2>&1; then
    no "$label accepted invalid fixture"
    sed -n '1,80p' "$out"
  else
    if grep -q "$needle" "$out"; then ok "$label"; else no "$label wrong failure"; sed -n '1,80p' "$out"; fi
  fi
}

run_ok "routing eval report passes" python3 "$ROOT/scripts/run-routing-evals.py" --quiet
python3 "$ROOT/scripts/run-routing-evals.py" --json-out "$T/routing.json" --quiet >/dev/null 2>&1 \
  && grep -q '"rank1_rate"' "$T/routing.json" \
  && grep -q '"host_wording_confusion"' "$T/routing.json" \
  && ok "routing eval writes machine-readable rank report" \
  || no "routing eval missing JSON report fields"
run_ok "controlled behavioral dry-run validates the frozen 20-session trial" python3 "$ROOT/scripts/run-live-behavioral-evals.py" --dry-run
run_ok "skill pruning audit runs" node "$ROOT/scripts/skill-pruning-audit.mjs" --quiet
run_ok "skill anatomy validator passes shipped pack" python3 "$ROOT/scripts/validate-skill-anatomy.py" --quiet
run_ok "host command parity validator passes" python3 "$ROOT/scripts/validate-command-parity.py" --quiet
run_ok "agent composition validator passes" python3 "$ROOT/scripts/validate-agent-composition.py" --quiet

# The schema accepts policy-sized corpora and rejects empty ones.
cat > "$T/small-eval.json" <<'JSON'
{"skill":"rite-demo","description":"Small direct-command corpus.","queries":[{"text":"/rite-demo","expected":"should_trigger","rationale":"Direct invocation."},{"text":"run something else","expected":"should_not_trigger","rationale":"Negative boundary."}]}
JSON
run_ok "trigger eval schema accepts variable corpus size" bash "$ROOT/scripts/run-evals.sh" "$T/small-eval.json"
cat > "$T/empty-eval.json" <<'JSON'
{"skill":"rite-demo","description":"Invalid empty corpus.","queries":[]}
JSON
run_fail_contains "trigger eval schema rejects empty corpus" "queries is empty" bash "$ROOT/scripts/run-evals.sh" "$T/empty-eval.json"

# "Done" is valid only for green states, not blocked, unproven, or awaiting ones.
reply_case() {
  label="$1" line="$2" expected="$3"
  dir="$T/reply-$label/rite-demo"
  mkdir -p "$dir"
  cat > "$dir/SKILL.md" <<SKILL
---
name: rite-demo
description: Demo.
user-invocable: true
---
# rite-demo
## Output
Uses devrites-lib/reference/reply-contract.md.
\`\`\`
Done: demo complete.
Evidence: checks pass
$line
Next: /rite-prove
\`\`\`
SKILL
  if [ "$expected" = pass ]; then
    run_ok "reply contract accepts $label" env DEVRITES_SKILLS_DIR="${dir%/rite-demo}" bash "$ROOT/scripts/check-reply-contract.sh"
  else
    run_fail_contains "reply contract rejects $label" "unresolved state" env DEVRITES_SKILLS_DIR="${dir%/rite-demo}" bash "$ROOT/scripts/check-reply-contract.sh"
  fi
}
reply_case green "Open: none" pass
reply_case blocked "Open: blocker remains" fail
reply_case unproven "Evidence: acceptance unproven" fail
reply_case awaiting "Open: awaiting human" fail

# An ordered step needs an explicit completion signal or an observable action
# and target. A generic exhortation is not a contract.
mkdir -p "$T/pruning-skills/rite-demo"
cat > "$T/pruning-skills/rite-demo/SKILL.md" <<'SKILL'
---
name: rite-demo
description: Use when demonstrating an invalid step.
user-invocable: true
---
# rite-demo
## Workflow
1. Think carefully.
SKILL
run_fail_contains "step audit rejects vague ordered step" "needs a checkable completion criterion" node "$ROOT/scripts/skill-pruning-audit.mjs" --skills-dir "$T/pruning-skills" --quiet

# The anatomy validator rejects a public skill without a stop or output contract.
mkdir -p "$T/skills/rite-demo"
cat > "$T/skills/rite-demo/SKILL.md" <<'SKILL'
---
name: rite-demo
description: Use when demo. Not for real work.
user-invocable: true
---
# /rite-demo

## Rules consulted
Read core.md.

## Operating rules
- Do the thing.

## Workflow
- Run a step.
SKILL
run_fail_contains "skill anatomy rejects missing output contract" "output/reply contract" python3 "$ROOT/scripts/validate-skill-anatomy.py" --skills-dir "$T/skills" --quiet

# A legacy exemption may waive named sections, but the other anatomy rules
# still apply.
mkdir -p "$T/exempt-skills/rite-doctor"
cat > "$T/exempt-skills/rite-doctor/SKILL.md" <<'SKILL'
---
name: rite-doctor
description: Use when diagnosing DevRites. Not for feature work.
user-invocable: true
---
# /rite-doctor
SKILL
run_fail_contains "skill anatomy exemptions are section-scoped" "output/reply contract" python3 "$ROOT/scripts/validate-skill-anatomy.py" --skills-dir "$T/exempt-skills" --quiet

# Routing eval owner negatives must compare a real pair.
mkdir -p "$T/routing-skills/alpha" "$T/routing-skills/beta" "$T/routing-evals"
cat > "$T/routing-skills/alpha/SKILL.md" <<'SKILL'
---
name: alpha
description: Alpha handles beta task wording.
user-invocable: true
---
# alpha
SKILL
cat > "$T/routing-skills/beta/SKILL.md" <<'SKILL'
---
name: beta
description: Beta handles other wording.
user-invocable: true
---
# beta
SKILL
cat > "$T/routing-evals/alpha.json" <<'JSON'
{"skill":"alpha","queries":[{"text":"alpha task","expected":"should_not_trigger","owner":"beta","rationale":"beta owns it"}]}
JSON
run_fail_contains "routing eval rejects owner that does not outrank target" "declared owner" python3 "$ROOT/scripts/run-routing-evals.py" --skills-dir "$T/routing-skills" --evals-dir "$T/routing-evals" --baseline "$T/no-baseline.json"

# Invocation policy determines the required routing corpus shape.
mkdir -p "$T/explicit-skills/rite-explicit" "$T/explicit-evals"
cat > "$T/explicit-skills/rite-explicit/SKILL.md" <<'SKILL'
---
name: rite-explicit
description: Explicit demo.
user-invocable: true
disable-model-invocation: true
---
# rite-explicit
SKILL
cat > "$T/explicit-evals/rite-explicit.json" <<'JSON'
{"skill":"rite-explicit","queries":[{"text":"please run the explicit demo","expected":"should_trigger"},{"text":"ignore the demo","expected":"should_not_trigger","owner":null,"owner_rationale":"No owner."}]}
JSON
run_fail_contains "routing eval rejects implicit positive for explicit-only skill" "positives must all directly invoke" python3 "$ROOT/scripts/run-routing-evals.py" --skills-dir "$T/explicit-skills" --evals-dir "$T/explicit-evals" --baseline "$T/no-baseline.json"

mkdir -p "$T/model-skills/rite-model" "$T/model-evals"
cat > "$T/model-skills/rite-model/SKILL.md" <<'SKILL'
---
name: rite-model
description: Model demo. Use when asked for model routing.
user-invocable: true
---
# rite-model
SKILL
cat > "$T/model-evals/rite-model.json" <<'JSON'
{"skill":"rite-model","queries":[{"text":"/rite-model","expected":"should_trigger"},{"text":"ignore the demo","expected":"should_not_trigger","owner":null,"owner_rationale":"No owner."}]}
JSON
run_fail_contains "routing eval rejects direct-only model corpus" "needs implicit positive and negative queries" python3 "$ROOT/scripts/run-routing-evals.py" --skills-dir "$T/model-skills" --evals-dir "$T/model-evals" --baseline "$T/no-baseline.json"

cat > "$T/routing-baseline.json" <<'JSON'
{"recorded_at":"2026-07-10","skills":99,"queries":1,"positive_queries":0,"negative_queries":1}
JSON
run_fail_contains "routing eval rejects stale baseline metadata" "baseline metadata: skills" python3 "$ROOT/scripts/run-routing-evals.py" --skills-dir "$T/routing-skills" --evals-dir "$T/routing-evals" --baseline "$T/routing-baseline.json"

# Host parity rejects a missing canonical command-map entry.
cp -R "$ROOT/pack/.claude/skills" "$T/parity-skills"
cp "$ROOT/docs/skills.md" "$T/skills.md"
cp "$ROOT/docs/command-map.md" "$T/command-map.md"
python3 - "$T/command-map.md" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
s = p.read_text()
s = s.replace('/rite-build', 'RITE_BUILD_CLAUDE_REMOVED')
p.write_text(s)
PY
run_fail_contains "host parity rejects missing command-map entry" "docs/command-map Claude direct" python3 "$ROOT/scripts/validate-command-parity.py" --skills-dir "$T/parity-skills" --docs-skills "$T/skills.md" --docs-command-map "$T/command-map.md" --readme "$ROOT/README.md" --quiet

# The agent validator rejects a write-capable reviewer.
mkdir -p "$T/agents"
cat > "$T/agents/devrites-code-reviewer.md" <<'AGENT'
---
name: devrites-code-reviewer
description: Bad reviewer.
tools: Read, Write, Bash
---
## Role / scope
Reviewer.
## Tools / read-write mode
Write-capable.
## Output format
Findings.
## Composition
Invoke directly when reviewing.
Do not invoke another agent.
AGENT
run_fail_contains "agent validator rejects extra writer" "only devrites-slice-wright" python3 "$ROOT/scripts/validate-agent-composition.py" --agents-dir "$T/agents" --quiet

echo ""
[ "$fail" -eq 0 ] && echo "agent-skills-refresh-validation-test: PASS" || echo "agent-skills-refresh-validation-test: FAIL"
exit "$fail"
