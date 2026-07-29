#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SPEC="$ROOT/pack/.claude/skills/rite-spec/SKILL.md"
CLARIFY="$ROOT/pack/.claude/skills/rite-clarify/SKILL.md"
TEMPER="$ROOT/pack/.claude/skills/rite-temper/SKILL.md"
DEFINE="$ROOT/pack/.claude/skills/rite-define/SKILL.md"
VET="$ROOT/pack/.claude/skills/rite-vet/SKILL.md"
VET_ARTIFACTS="$ROOT/pack/.claude/skills/rite-vet/reference/artifacts.md"
PLAN="$ROOT/pack/.claude/skills/rite-plan/SKILL.md"
PLAN_MODES="$ROOT/pack/.claude/skills/rite-plan/reference/replan-and-repair.md"
CONVERGE="$ROOT/pack/.claude/skills/rite-converge/SKILL.md"
BUILD="$ROOT/pack/.claude/skills/rite-build/reference/phase-contract.md"
WRIGHT_DISPATCH="$ROOT/pack/.claude/skills/rite-build/reference/wright-dispatch.md"
CLEANUP="$ROOT/pack/.claude/skills/devrites-debug-recovery/reference/cleanup-and-classify.md"
DRIFT="$ROOT/pack/.claude/skills/rite-build/reference/spec-drift-guard.md"
DOCTOR="$ROOT/pack/.claude/skills/rite-doctor/SKILL.md"
UPGRADE="$ROOT/pack/.claude/skills/rite-upgrade/SKILL.md"
UPGRADE_PLANNER="$ROOT/pack/.claude/agents/devrites-upgrade-planner.md"
RESOLVE="$ROOT/pack/.claude/skills/rite-resolve/SKILL.md"
AUTOCOMPLETE="$ROOT/pack/.claude/skills/rite-autocomplete/SKILL.md"
fail=0

ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

echo "== phase-gate-routing-test =="

if grep -q 'Do not run `devrites-engine analyze`' "$SPEC"; then
  ok "rite-spec defers analyze until tasks exist"
else
  no "rite-spec does not explicitly defer analyze"
fi

write_line="$(grep -n '^6\. \*\*Write\*\*' "$DEFINE" | cut -d: -f1)"
analyze_line="$(grep -n '^[[:space:]]*devrites-engine analyze' "$DEFINE" | head -1 | cut -d: -f1)"
readiness_line="$(grep -n '^7\. \*\*Readiness gate\*\*' "$DEFINE" | cut -d: -f1)"
if [ -n "$write_line" ] && [ -n "$analyze_line" ] && [ -n "$readiness_line" ] \
   && [ "$analyze_line" -gt "$write_line" ] && [ "$analyze_line" -lt "$readiness_line" ]; then
  ok "rite-define runs analyze after writing tasks and before readiness"
else
  no "rite-define does not run analyze after tasks are written"
fi

vet_analyze_count="$(grep -c '^[[:space:]]*devrites-engine analyze' "$VET" || true)"
if [ "$vet_analyze_count" -ge 2 ]; then
  ok "rite-vet re-runs analyze after plan hardening"
else
  no "rite-vet can mutate tasks after its only analyze pass"
fi

if grep -q 'Build-interruption forecast' "$SPEC" \
   && grep -q 'Interruption pre-mortem' "$TEMPER" \
   && grep -q 'Foreseeable-decision sweep' "$DEFINE" \
   && grep -q 'Build-entry preflight' "$VET"; then
  ok "pre-build phases explicitly close foreseeable build interruptions"
else
  no "one or more pre-build phases lack their interruption-closure gate"
fi

if [ -f "$CLARIFY" ] \
   && grep -q 'devrites-interview' "$CLARIFY" \
   && grep -q 'decision-coverage.md' "$CLARIFY" \
   && grep -q 'Next: /rite-clarify' "$SPEC" \
   && grep -q 'decision-coverage.md' "$DEFINE"; then
  ok "rite-clarify reuses the interview engine and leaves auditable decision coverage"
else
  no "clarification is not an enforced, auditable pre-plan phase"
fi

autocomplete_arc="$(tr '\n' ' ' < "$AUTOCOMPLETE" | sed -E 's/[[:space:]]+/ /g')"
if [[ "$autocomplete_arc" == *'/rite-spec` → **`/rite-clarify`** → **`/rite-temper`** → `/rite-define`'* ]]; then
  ok "rite-autocomplete runs clarify before temper and define"
else
  no "rite-autocomplete does not enforce clarify before technical planning"
fi

if grep -q 'Implementation readiness: READY' "$VET" \
   && grep -q 'NEEDS CLARIFICATION' "$VET" \
   && grep -q 'NEEDS REPLAN' "$VET"; then
  ok "rite-vet records a typed implementation-readiness verdict"
else
  no "rite-vet lacks a typed final implementation-readiness verdict"
fi

vet_recheck_line="$(grep -n '^6\. \*\*One narrow recheck' "$VET" | cut -d: -f1)"
vet_engineering_digest_line="$(grep -n 'devrites-engine readiness-digest engineering <slug>' "$VET" | tail -1 | cut -d: -f1)"
vet_ready_line="$(grep -n 'Only READY sets `Phase: vet`' "$VET" | tail -1 | cut -d: -f1)"
if [ -n "$vet_recheck_line" ] \
   && [ -n "$vet_engineering_digest_line" ] \
   && [ -n "$vet_ready_line" ] \
   && [ "$vet_engineering_digest_line" -gt "$vet_recheck_line" ] \
   && [ "$vet_ready_line" -gt "$vet_recheck_line" ] \
   && grep -q 'Keep `state.md` non-READY' "$VET"; then
  ok "rite-vet finalizes digest and READY only after the mandatory narrow recheck"
else
  no "rite-vet can expose READY before its final reviewer recheck"
fi

if grep -q '`6` → `/rite-clarify`' "$BUILD" \
   && grep -q '`7` → `/rite-vet`' "$BUILD" \
   && grep -q '`8` → `/rite-upgrade`' "$BUILD"; then
  ok "rite-build routes missing upstream evidence to its owning phase"
else
  no "rite-build does not route clarification/vet readiness gaps upstream"
fi

if grep -q 'devrites.readiness-artifacts.v2' "$CLARIFY" \
   && grep -q 'devrites.readiness-artifacts.v2' "$VET" \
   && grep -q 'devrites-engine migrate' "$UPGRADE" \
   && grep -q 'devrites-upgrade-planner' "$UPGRADE" \
   && grep -q 'devrites-engine build-readiness' "$UPGRADE" \
   && grep -q 'second `/rite-upgrade`' "$UPGRADE" \
   && grep -q 'no-op path' "$UPGRADE" \
   && grep -q 'Read-only' "$UPGRADE_PLANNER"; then
  ok "rite-upgrade reconciles legacy workspaces to the stamped current contract"
else
  no "rite-upgrade lacks a current-contract, fresh-agent, idempotent readiness path"
fi

if grep -q 'devrites-engine doctor; echo' "$DOCTOR" \
   && grep -q 'devrites-engine doctor; echo' "$UPGRADE" \
   && ! grep -q 'doctor --verbose' "$DOCTOR" "$UPGRADE"; then
  ok "doctor and upgrade use the engine's stable doctor contract"
else
  no "doctor or upgrade depends on an unsupported doctor flag"
fi

if grep -q 'Implementation readiness: NEEDS REPLAN' "$PLAN" \
   && grep -q 'Next step: /rite-vet' "$PLAN" \
   && grep -q 'Implementation readiness: NEEDS REPLAN' "$CONVERGE" \
   && grep -q 'Next step: /rite-vet' "$CONVERGE"; then
  ok "planning mutations invalidate stale vet readiness"
else
  no "a replan or convergence append can retain a stale READY verdict"
fi

coverage_refresh_ok=1
for owner in "$TEMPER" "$PLAN" "$VET"; do
  grep -q 'Partial/Missing' "$owner" || coverage_refresh_ok=0
  grep -q 'devrites-engine readiness-digest coverage <slug>' "$owner" || coverage_refresh_ok=0
done
coverage_digest_line="$(grep -n 'devrites-engine readiness-digest coverage <slug>' "$VET_ARTIFACTS" | tail -1 | cut -d: -f1)"
engineering_digest_line="$(grep -n 'devrites-engine readiness-digest engineering <slug>' "$VET_ARTIFACTS" | tail -1 | cut -d: -f1)"
if [ "$coverage_refresh_ok" -eq 1 ] \
   && [ -n "$coverage_digest_line" ] \
   && [ -n "$engineering_digest_line" ] \
   && [ "$coverage_digest_line" -lt "$engineering_digest_line" ]; then
  ok "coverage-bound mutation owners revalidate and refresh before engineering digest"
else
  no "a downstream ledger mutation can stale decision coverage before build readiness"
fi

if grep -q 'devrites-debug-recovery' "$BUILD" \
   && grep -q 'three total attempts' "$BUILD" \
   && ! grep -q 'Still red after the one retry' "$BUILD" \
   && grep -q 'Do not ask for retry authorization' "$DRIFT" \
   && grep -q 'Re-plan only when the durable plan changed' "$DRIFT"; then
  ok "rite-build routes objective failures through bounded debug recovery"
else
  no "rite-build still converts the first retry failure directly into a human gate"
fi

if grep -q 'Fingerprint the causal diagnosis, never the failing test or symptom' "$CLEANUP" \
   && grep -q 'discriminating fix proves the cause absent' "$CLEANUP" \
   && grep -q 'symptom remains' "$CLEANUP" \
   && grep -q 'Failure alone' "$CLEANUP" \
   && grep -q 'never resets a budget' "$CLEANUP" \
   && grep -q 'cleanup-and-classify.md' "$BUILD"; then
  ok "rite-build separates a falsified diagnosis from a repeated symptom"
else
  no "rite-build can exhaust a new diagnosis under an old symptom fingerprint"
fi

if grep -q 'exhausted fingerprint blocks diagnosis, not symptom' "$PLAN_MODES" \
   && grep -q 'proof removed that cause but symptom remains' "$PLAN_MODES" \
   && grep -q 'new diagnosis/proof fingerprint' "$PLAN_MODES" \
   && grep -q 'Never clear/reuse old one' "$PLAN_MODES"; then
  ok "rite-plan unblock reroutes a falsified diagnosis without resetting its budget"
else
  no "rite-plan unblock can reuse or clear an exhausted diagnosis"
fi

stuck_line="$(grep -n 'devrites-engine stuck log' "$WRIGHT_DISPATCH" | head -1 | cut -d: -f1)"
snapshot_line="$(grep -n 'devrites-engine reconcile snapshot' "$WRIGHT_DISPATCH" | head -1 | cut -d: -f1)"
if [ -n "$stuck_line" ] \
   && [ -n "$snapshot_line" ] \
   && [ "$stuck_line" -lt "$snapshot_line" ]; then
  ok "rite-build records stuck telemetry before arming reconciliation"
else
  no "rite-build mutates action.log after the reconcile snapshot"
fi

if grep -q 'explicit consent' "$RESOLVE" && ! grep -q 'confirm? (y/N)' "$RESOLVE"; then
  ok "rite-resolve does not ask for redundant confirmation"
else
  no "rite-resolve still double-confirms an explicit answer"
fi

echo ""
[ "$fail" -eq 0 ] && echo "phase-gate-routing-test: PASS" || echo "phase-gate-routing-test: FAIL"
exit "$fail"
