#!/usr/bin/env bash
# agent-skills-refresh-validation-test.sh — regression coverage for the
# agent-skills refresh hardening layer: deterministic routing reports, skill
# anatomy, host command parity, and agent composition.
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
run_ok "live behavioral eval dry-run plans shipped portable scenarios" python3 "$ROOT/scripts/run-live-behavioral-evals.py" --dry-run "$ROOT/evals/behavioral/rite-ship.json"
run_ok "skill pruning audit runs" node "$ROOT/scripts/skill-pruning-audit.mjs" --quiet
run_ok "skill anatomy validator passes shipped pack" python3 "$ROOT/scripts/validate-skill-anatomy.py" --quiet
run_ok "host command parity validator passes" python3 "$ROOT/scripts/validate-command-parity.py" --quiet
run_ok "agent composition validator passes" python3 "$ROOT/scripts/validate-agent-composition.py" --quiet

# Fixture: anatomy validator rejects a public skill without a stop/output contract.
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

# A legacy skill exemption may waive named sections, but must not bypass every
# other anatomy contract.
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

# Fixture: routing eval owner negatives are pairwise, not vacuous.
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

cat > "$T/routing-baseline.json" <<'JSON'
{"recorded_at":"2026-07-10","skills":99,"queries":1,"positive_queries":0,"negative_queries":1}
JSON
run_fail_contains "routing eval rejects stale baseline metadata" "baseline metadata: skills" python3 "$ROOT/scripts/run-routing-evals.py" --skills-dir "$T/routing-skills" --evals-dir "$T/routing-evals" --baseline "$T/routing-baseline.json"

# Fixture: host parity rejects docs without a Codex equivalent.
cp -R "$ROOT/pack/.claude/skills" "$T/parity-skills"
cp "$ROOT/docs/skills.md" "$T/skills.md"
cp "$ROOT/docs/command-map.md" "$T/command-map.md"
python3 - "$T/skills.md" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
s = p.read_text()
s = s.replace('$rite-build', 'RITE_BUILD_CODEX_REMOVED')
s = s.replace('$rite build', 'RITE_BUILD_MENU_REMOVED')
p.write_text(s)
PY
run_fail_contains "host parity rejects missing Codex wording" "Codex" python3 "$ROOT/scripts/validate-command-parity.py" --skills-dir "$T/parity-skills" --docs-skills "$T/skills.md" --docs-command-map "$T/command-map.md" --readme "$ROOT/README.md" --quiet

# Fixture: agent validator rejects a write-capable reviewer.
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
