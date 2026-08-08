#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0

ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
require() {
  if grep -Fq -- "$2" "$1"; then ok "$3"; else no "$3"; fi
}
forbid() {
  if grep -Fq -- "$2" "$1"; then no "$3"; else ok "$3"; fi
}
require_order() {
  local file="$1" label="$2" previous=0 line token
  shift 2
  for token in "$@"; do
    line="$(grep -nF -- "$token" "$file" | head -1 | cut -d: -f1)"
    if [ -z "$line" ] || [ "$line" -le "$previous" ]; then
      no "$label"
      return
    fi
    previous="$line"
  done
  ok "$label"
}

echo "== phase-gate-routing-test =="

SPEC="$ROOT/pack/.claude/skills/rite-spec/SKILL.md"
CLARIFY="$ROOT/pack/.claude/skills/rite-clarify/SKILL.md"
DEFINE="$ROOT/pack/.claude/skills/rite-define/SKILL.md"
PLAN="$ROOT/pack/.claude/skills/rite-plan/SKILL.md"
VET="$ROOT/pack/.claude/skills/rite-vet/SKILL.md"
BUILD="$ROOT/pack/.claude/skills/rite-build/reference/phase-contract.md"
WRIGHT="$ROOT/pack/.claude/agents/devrites-slice-wright.md"
WRIGHT_DISPATCH="$ROOT/pack/.claude/skills/rite-build/reference/wright-dispatch.md"
CORE="$ROOT/pack/.claude/skills/devrites-lib/reference/standards/core.md"
AFK_HITL="$ROOT/pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md"
ONE_SHOT="$ROOT/pack/.claude/skills/devrites-lib/reference/standards/one-shot-actions.md"
WORKFLOW_ARTIFACTS="$ROOT/pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md"
STATE_WORKSPACE="$ROOT/pack/.claude/skills/rite-spec/reference/state-workspace.md"
REPLY="$ROOT/pack/.claude/skills/devrites-lib/reference/reply-contract.md"
PROVE="$ROOT/pack/.claude/skills/rite-prove/SKILL.md"
DRIFT="$ROOT/pack/.claude/skills/rite-build/reference/spec-drift-guard.md"
AUTOCOMPLETE="$ROOT/pack/.claude/skills/rite-autocomplete/SKILL.md"
AUTOCOMPLETE_LOOP="$ROOT/pack/.claude/skills/rite-autocomplete/reference/loop.md"
AUTOCOMPLETE_STOPS="$ROOT/pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md"
BUILD_AFK="$ROOT/pack/.claude/skills/rite-build/reference/afk-discipline.md"
DEBUG_RECOVERY="$ROOT/pack/.claude/skills/devrites-debug-recovery/SKILL.md"
DEBUG_CLASSIFY="$ROOT/pack/.claude/skills/devrites-debug-recovery/reference/cleanup-and-classify.md"
CANDIDATE_INTEGRITY="$ROOT/pack/.claude/skills/devrites-lib/reference/candidate-integrity.md"
PROOF_RUNNER="$ROOT/pack/.claude/agents/devrites-proof-runner.md"
DISCOVERY="$ROOT/pack/.claude/skills/rite-prove/reference/test-command-discovery.md"
FAILURE_TRIAGE="$ROOT/pack/.claude/skills/rite-prove/reference/failure-triage.md"
PLAN_REVIEWER="$ROOT/pack/.claude/agents/devrites-plan-reviewer.md"
PLAN_DRAFTER="$ROOT/pack/.claude/agents/devrites-plan-drafter.md"
VET_ARTIFACTS="$ROOT/pack/.claude/skills/rite-vet/reference/artifacts.md"
CUSTOMIZE="$ROOT/pack/.claude/skills/rite-customize/SKILL.md"
UPGRADE="$ROOT/pack/.claude/skills/rite-upgrade/SKILL.md"
UPGRADE_PLANNER="$ROOT/pack/.claude/agents/devrites-upgrade-planner.md"
RESOLVE="$ROOT/pack/.claude/skills/rite-resolve/SKILL.md"
SEAL="$ROOT/pack/.claude/skills/rite-seal/SKILL.md"
SEAL_CONTRACT="$ROOT/pack/.claude/skills/rite-seal/reference/phase-contract.md"

require "$CORE" 'Immediately before its final response' 'core loads the shared reply contract at the response boundary'
require "$CORE" 'reply-contract.md' 'core names the universal reply contract'
require "$CORE" 'devrites-engine check readiness <slug>' 'core preserves the structural lifecycle rest point'
require "$CORE" 'devrites-engine check seal <slug>' 'core preserves the structural seal rest point'
require "$REPLY" 'Use exactly one recommended next action' 'reply contract keeps one next action'

require "$SPEC" '/rite-clarify' 'spec routes topology coverage to clarify'
require "$SPEC" 'update the existing workspace rather than overwrite it' 'spec updates an existing workspace safely'
require "$SPEC" 'Acceptance delta' 'spec shows acceptance changes'
require "$SPEC" 'Open-question delta' 'spec shows question changes'
require "$SPEC" 'same intent, >50% scope' 'spec preserves the ACTIVE overlap route'
require "$SPEC" 'Native grammar re-read checklist' 'spec checks normative grammar without an engine parser'
forbid "$SPEC" 'devrites-engine check spec' 'spec has no removed grammar command'
require "$CLARIFY" 'devrites-interview' 'clarify owns the coverage scan'
require "$CLARIFY" 'Decision coverage: CLEAR' 'clarify emits the readiness verdict'
require "$CLARIFY" 'Next step: /rite-temper' 'clarify names the next phase'
require "$CLARIFY" 'return_phase' 'clarify preserves the later-phase return cursor natively'
require "$CLARIFY" 'preserve unrelated Markdown' 'clarify edits only its cursor fields'
forbid "$CLARIFY" 'devrites-engine state clarify' 'clarify has no removed state helper'
require "$DEFINE" 'Decision coverage: CLEAR' 'define requires clarified intent'
require "$DEFINE" '/rite-clarify' 'define returns missing coverage to clarify'

require "$BUILD" 'devrites-engine check readiness <slug>' 'build uses the structural readiness gate'
require "$BUILD" 'dispatch the exact `devrites-slice-wright`' 'build does not bypass the writer agent'
require "$BUILD" 'exact project-relative source/test path list directly in the task' 'build states exact writer paths inline'
require "$BUILD" 'git diff --name-only' 'build inspects the returned source delta'
require "$BUILD" 'test hunks for deletion, skipping/focus, tautology, or weaker expectations' 'build reviews test integrity without snapshot machinery'
require "$BUILD" 'devrites-test-analyst' 'build delegates semantic test analysis natively'
forbid "$BUILD" 'devrites-engine test-integrity' 'build has no heuristic engine test parser'
forbid "$BUILD" 'devrites-engine build-readiness' 'build has no semantic engine readiness parser'
forbid "$BUILD" 'devrites-engine reconcile' 'build has no engine source-window gate'
require "$BUILD" 'devrites-debug-recovery' 'build routes technical failures to bounded recovery'
require "$BUILD" 'exactly once after each green built slice' 'build accounts for AFK slices natively'
forbid "$BUILD" 'devrites-engine state tick-afk' 'build has no removed AFK counter command'
forbid "$BUILD" 'devrites-engine preamble' 'build has no orientation renderer'
forbid "$BUILD" 'devrites-engine progress' 'build has no decorative progress renderer'
forbid "$BUILD" 'devrites-engine footprint' 'build has no dispatch telemetry'

for key in reuse conventions principles sources assumptions follow_ups; do
  require "$WRIGHT" "$key: []" "wright result requires $key bookkeeping"
done
require "$WRIGHT_DISPATCH" 'Reject a result that omits any required key' 'root rejects incomplete wright results'
require "$WRIGHT_DISPATCH" 'Persist the' 'root persists returned wright facts'
require "$WRIGHT" 'no-progress attempts' 'wright shares the per-fingerprint no-progress budget'
forbid "$WRIGHT" 'devrites-engine state recovery' 'wright has no removed recovery counter command'

require "$PROVE" 'sole approved runtime' 'prove treats test-plan as sole command authority'
require "$PROVE" 'return to the current Vet contract' 'prove routes newly discovered commands through Vet'
require "$CORE" 'nested phase boundary, not a user-facing handoff' 'nested recovery returns to its controlling rite'
require "$CORE" 'not from a stale `state.md` label' 'shared caller contract verifies recovery exhaustion from durable attempts'
require "$STATE_WORKSPACE" 'agent-owned technical backtracking' 'state cursor preserves the recovery origin'
require "$STATE_WORKSPACE" 'terminal `next_action` is a claim to verify' 'cold resume reconciles stale terminal cursors'
require "$STATE_WORKSPACE" 'spent action authorization blocks another execution' 'state recovery separates action authority from offline repair'
require "$DRIFT" 'invoke `/rite-plan repair` and `/rite-vet` inline' 'spec drift recovery stays inside the active caller'
require "$PROVE" 'Prove remains the controlling caller' 'prove owns repair and re-vet continuation'
require "$PLAN" 'preserve any valid return cursor' 'plan repair preserves its caller recovery target'
require "$VET" 'restore and consume the return cursor' 'vet returns a repaired plan to the originating phase'
require "$AUTOCOMPLETE" 'follow agent-owned backward edges' 'autocomplete follows technical backtracking internally'
require "$AUTOCOMPLETE_LOOP" 'do not hand the intermediate command to the user' 'autocomplete loop retains phase orchestration ownership'
require "$AUTOCOMPLETE_STOPS" 'Agent-owned backtracking is not a stop condition' 'autocomplete pauses only on a real stop condition'
require "$FAILURE_TRIAGE" 'Ask only when the remaining decision is human-owned' 'prove exhaustion does not ask for mechanical recovery'
require "$AFK_HITL" 'Next step: none — technical recovery exhausted' 'technical exhaustion is terminal without a phase command'
forbid "$AFK_HITL" 'Next step: /rite-plan unblock' 'technical exhaustion cannot hand plan unblock to the user'
require "$BUILD_AFK" 'Next step: none — technical recovery exhausted' 'build exhaustion is terminal without a phase command'
forbid "$BUILD_AFK" 'Next step: /rite-plan unblock' 'build exhaustion cannot hand plan unblock to the user'
require "$BUILD" 'no-progress attempts' 'build counts only unchanged fingerprint failures'
require "$BUILD_AFK" 'no-progress attempts' 'afk build recovery uses progress-aware accounting'
require "$DEBUG_RECOVERY" 'no-progress attempts' 'debug recovery shares progress-aware accounting'
require "$DEBUG_RECOVERY" 'Next: none' 'debug exhaustion has no runnable recovery command'
forbid "$DEBUG_RECOVERY" 'Next: /rite-plan unblock' 'debug exhaustion cannot restart manual plan ping-pong'
require "$DEBUG_CLASSIFY" 'no-progress attempts' 'debug classification derives the no-progress count'
require "$FAILURE_TRIAGE" 'no-progress attempts' 'prove triage consumes only no-progress budget'
require "$AUTOCOMPLETE_STOPS" 'no runnable recovery command' 'autocomplete terminal blockers do not advertise a retry command'
require "$STATE_WORKSPACE" 'terminal: none' 'state cursor represents terminal technical exhaustion explicitly'
require "$REPLY" 'Technical recovery exhausted' 'reply contract has a terminal technical blocker shape'
require "$REPLY" 'No runnable recovery command' 'reply contract does not turn exhaustion into another user command'
require "$AFK_HITL" 'three no-progress attempts per exact causal fingerprint' 'recovery budget attaches to the exact unresolved cause'
require "$AFK_HITL" 'Closing a prior finding with discriminating evidence is progress' 'closed findings do not consume no-progress budget'
require "$AFK_HITL" 'new Critical or Important finding' 'new high-severity blockers receive an independent fingerprint'
require "$AFK_HITL" '`drift.md` and `evidence.md`' 'recovery progress uses existing durable artifacts'
require "$AUTOCOMPLETE_LOOP" 'narrow Vet recheck' 'autocomplete rechecks repaired findings without restarting Full Vet'
require "$VET" 'Recovery recheck' 'vet has an explicit bounded recovery mode'
require "$VET" 'does not start another Full Vet' 'recovery recheck cannot restart the full review cycle'
require "$VET" 'Suggestion, Nit, or FYI' 'lower-severity novelty cannot perpetuate recovery'
require "$PROVE" 'three no-progress attempts on the exact same fingerprint' 'standalone prove uses progress-aware recovery accounting'
require "$ONE_SHOT" 'unknown but lexically well-formed non-secret values survive' 'one-shot evidence preserves safe unknown diagnostics'
require "$ONE_SHOT" 'cleanup cannot delete or overwrite' 'one-shot gate proves failure evidence survives cleanup'
require "$ONE_SHOT" 'Do not rerun the action during triage' 'one-shot failure handling forbids blind reproduction'
require "$ONE_SHOT" 'consumes only the authorization' 'one-shot execution budget does not erase offline recovery'
require "$ONE_SHOT" 'immediately runs' 'retained new evidence starts offline recovery in the caller'
require "$ONE_SHOT" 'stable non-secret `boundary_id`' 'one-shot evidence identifies one actionable boundary'
require "$ONE_SHOT" 'injective' 'one-shot gate rejects diagnostic collisions'
require "$ONE_SHOT" 'diagnostic-amplification attempt' 'one-shot recovery can safely acquire a missing discriminator'
require "$ONE_SHOT" 'past attempt is irretrievable' 'past evidence loss alone is not terminal when amplification is safe'
require "$VET" 'one-shot evidence completeness' 'vet blocks READY without consumptive-action evidence retention'
require "$VET" 'collision mutant' 'vet rejects two failure seams sharing one retained fingerprint'
require "$PROVE" 'Record the admitted artifact identity before execution' 'prove checks one-shot evidence before execution'
require "$PROVE" 'action budget is zero' 'prove does not misclassify spent execution authority as recovery exhaustion'
require "$FAILURE_TRIAGE" 'never rerun it' 'prove triage uses retained evidence for consumptive actions'
require "$FAILURE_TRIAGE" 'does not stop offline recovery' 'prove triage continues from retained new evidence'
require "$DEBUG_RECOVERY" 'MUST NOT be rerun during diagnosis' 'debug recovery does not reproduce consumptive actions'
require "$DEBUG_RECOVERY" 'not a spent recovery budget' 'debug recovery separates execution authority from repair budget'
require "$AUTOCOMPLETE" 'Do not confuse an action budget with recovery exhaustion' 'autocomplete continues offline after a one-shot failure'
require "$AUTOCOMPLETE" 'one-shot-actions.md` before any' 'autocomplete loads the consumptive-action contract before routing'
require "$AUTOCOMPLETE_LOOP" 'spent authorization blocks only another' 'autocomplete loop uses retained evidence before requesting a new GO'
require "$AUTOCOMPLETE_STOPS" 'is not technical-recovery' 'autocomplete stop contract separates one-shot authority from recovery'
require "$ONE_SHOT" 'Cold resume does not make that fingerprint old' 'one-shot recovery survives a session boundary'
require "$AUTOCOMPLETE" 'Before honoring `blocked`' 'autocomplete reconciles retained evidence before accepting a stale terminal cursor'
require "$AUTOCOMPLETE_LOOP" 'On cold resume, reconcile the terminal cursor' 'autocomplete reopens unfinished retained recovery after restart'
require "$AUTOCOMPLETE_STOPS" 'three recorded' 'unchanged terminal state requires actual fingerprint exhaustion'
require "$PROVE" 'blocked Prove cold' 'prove resumes retained offline recovery after restart'
require "$DEBUG_RECOVERY" 'previous action wrote a terminal cursor' 'debug recovery ignores stale terminal state while budget remains'
require "$REPLY" 'consumptive-action authorization plus' 'terminal reply cannot suppress retained offline recovery'
require "$DEBUG_CLASSIFY" 'offline diagnosis or correction from retained evidence' 'fresh action authorization is not required for offline diagnosis'
require "$FAILURE_TRIAGE" 'fix it when agent-owned' 'environment failures are repaired before they become blockers'
require "$AUTOCOMPLETE" 'blocked` label alone is not a stop condition' 'autocomplete routes technical blockers before stopping'
require "$AUTOCOMPLETE" 'Red gates block forward advancement and enter' 'autocomplete backtracks on red without advancing or stopping early'
require "$AUTOCOMPLETE_STOPS" 'Past evidence being irretrievable is not by itself terminal' 'autocomplete prepares diagnostic amplification before terminal exhaustion'
require "$PROVE" 'does not prove that no safe future acquisition design exists' 'prove distinguishes missing past evidence from impossible future evidence'
require "$FAILURE_TRIAGE" 'diagnostic-amplification plan gap' 'prove triage routes ambiguous one-shot evidence to plan repair'
require "$DEBUG_RECOVERY" 'diagnostic amplification' 'debug recovery can improve evidence without guessing a runtime fix'
require "$PLAN_REVIEWER" 'Missing evidence completeness is `broken`' 'plan reviewer rejects unsafe one-shot plans'
require "$PLAN_REVIEWER" 'one actionable failure seam' 'plan reviewer requires causal diagnostic uniqueness'
require "$PLAN_DRAFTER" 'diagnostic-amplification' 'plan drafter designs bounded evidence acquisition when past evidence is ambiguous'
require "$VET_ARTIFACTS" '## Consumptive action gates' 'test plan records one-shot evidence authority durably'
require "$VET_ARTIFACTS" 'Boundary map + collision proof' 'test plan binds diagnostic actionability proof'
require "$REPLY" 'no safe in-scope diagnostic-amplification seam' 'terminal reply requires proof that amplification is unavailable'
require "$WORKFLOW_ARTIFACTS" 'controlling root is the sole materializer' 'root owns executable workflow proof artifacts'
require "$WORKFLOW_ARTIFACTS" 'never dispatch `devrites-slice-wright`' 'workflow proof artifacts never widen the product writer'
require "$WORKFLOW_ARTIFACTS" 'exact file list' 'workflow artifact materialization is path bounded'
require "$WORKFLOW_ARTIFACTS" 'candidate digest remains identical' 'workflow artifacts cannot mutate the product candidate'
require "$PLAN_DRAFTER" 'never return implementation bodies' 'plan drafter cannot be used as a proof-artifact writer'
require "$PLAN" 'materializes the exact vetted workflow-artifact paths' 'plan repair gives executable workflow artifacts to the root'
require "$BUILD" 'Executable workflow-artifact branch' 'build separates workflow artifacts from product slices'
require "$BUILD" 'does not dispatch the wright' 'build never sends .devrites workflow artifacts to slice-wright'
require "$PROVE" 'workflow-artifacts.md' 'prove routes proof-artifact implementation to root ownership'
require "$DEBUG_RECOVERY" 'workflow-artifacts.md' 'debug recovery preserves proof-artifact root ownership'
require "$WRIGHT_DISPATCH" 'reject' 'wright dispatch remains fail closed'
require "$WRIGHT_DISPATCH" '`.devrites/**`' 'wright still rejects workflow artifact paths'
require "$DISCOVERY" 'discovery evidence, not authorization' 'command discovery cannot authorize execution'
require "$PROOF_RUNNER" 'reject missing, synthesized, or unapproved commands' 'proof runner rejects commands outside the approved plan'
require "$SEAL_CONTRACT" 'devrites-proof-runner' 'seal delegates acceptance proof judgment natively'
require "$SEAL_CONTRACT" 'devrites-spec-reviewer' 'seal delegates spec coverage judgment natively'
require "$SEAL" 'devrites-engine check seal' 'seal retains only deterministic structure and freshness'

require "$CUSTOMIZE" '--import-legacy' 'customize exposes legacy import mode'
require "$CUSTOMIZE" '.devrites/extensions/' 'legacy import inventories extensions'
require "$CUSTOMIZE" '.devrites/overrides/' 'legacy import inventories overrides'
require "$CUSTOMIZE" '.devrites/runbooks/' 'legacy import inventories runbooks'
require "$CUSTOMIZE" 'native skill with explicit gate,' 'runbooks map to native skills with control semantics'
require "$CUSTOMIZE" 'Leave the legacy files intact until native' 'legacy data remains until native validation passes'

require "$UPGRADE" '/rite-doctor' 'upgrade routes installed-contract diagnosis to the native doctor'
require "$UPGRADE" 'Recognize only released workspace forms' 'upgrade limits compatibility to released cursors'
require "$UPGRADE" 'Older provenance, cursor form, or pack version alone is never a defect' 'upgrade requires an observed current-contract failure'
require "$UPGRADE" 'current rule, exact workspace evidence, affected gate, owning rite' 'upgrade admits only evidence-backed repair deltas'
require "$UPGRADE" 'devrites-upgrade-planner' 'upgrade uses the native read-only planner'
require "$UPGRADE" '/rite-clarify' 'upgrade delegates decision repair to clarify'
require "$UPGRADE" '/rite-plan repair' 'upgrade delegates planning repair to plan'
require "$UPGRADE" '/rite-converge' 'upgrade delegates code-intent repair to converge'
require "$UPGRADE" '/rite-vet' 'upgrade delegates readiness to vet'
require_order "$UPGRADE" 'upgrade sequences candidate repair owners in lifecycle order' \
  '→ `/rite-prove`' '→ `/rite-polish`' '→ `/rite-review`' '→ `/rite-seal`'
require "$UPGRADE" 'Upgrade writes no workspace artifact' 'upgrade remains a read-only assessor and orchestrator'
require "$UPGRADE" 'Upgrade never restores it itself' 'upgrade does not mutate files during preservation failure'
require "$UPGRADE" 'never synthesize or guess' 'upgrade never invents candidate scope or historical proof'
require "$UPGRADE" 'v1/v2' 'upgrade preserves released v1 and v2 cursor support'
require "$UPGRADE" 'v3' 'upgrade preserves released v3 cursor support'
require "$UPGRADE" 'devrites-engine check candidate <slug>' 'upgrade rechecks a post-build candidate'
require "$UPGRADE" 'devrites-engine check seal <slug>' 'upgrade rechecks a sealed candidate'
require "$UPGRADE" 'devrites-engine check readiness <slug>' 'upgrade proves structural readiness'
forbid "$UPGRADE" 'devrites-engine migrate' 'upgrade has no engine migrator'
forbid "$UPGRADE" 'devrites-engine build-readiness' 'upgrade has no semantic engine readiness parser'
forbid "$UPGRADE" 'doctor --verbose' 'upgrade uses no unsupported doctor flag'
forbid "$UPGRADE" 'devrites-engine doctor' 'upgrade has no removed engine doctor command'
require "$UPGRADE" 'Resolve the explicit or active slug' 'upgrade reads native workspace orientation directly'
require "$UPGRADE" 'state.md' 'upgrade requires the authoritative workspace ledger'
require "$UPGRADE_PLANNER" 'Outcome: <current | repairable | unsupported | gap>' 'upgrade planner returns a fail-closed typed outcome'
require "$UPGRADE_PLANNER" 'current_rule:' 'upgrade planner cites the current contract'
require "$UPGRADE_PLANNER" 'workspace_evidence:' 'upgrade planner cites the observed workspace defect'
require "$UPGRADE_PLANNER" 'Older provenance is not evidence' 'upgrade planner cannot infer staleness from age'
require "$UPGRADE_PLANNER" 'Missing input or unverifiable current rules produce `gap`' 'upgrade planner fails closed on incomplete evidence'
require "$UPGRADE_PLANNER" 'candidate integrity' 'upgrade planner assesses the current candidate-integrity axis'
require "$UPGRADE_PLANNER" 'Prove, Polish, Review, or Seal' 'upgrade planner may route candidate defects only to current owners'
require "$UPGRADE_PLANNER" 'ambiguous candidate scope produces' 'upgrade planner fails closed before candidate reconstruction'
require "$UPGRADE_PLANNER" '`unsupported`/`gap` return empty findings/route and no writable path or delta' 'unsupported and gap assessments remain pathless'
require "$PROVE" 'admitted `/rite-upgrade` assessment' 'prove limits legacy refresh to an admitted upgrade assessment'
require "$PROVE" 'legacy touched-file scope, live diff, tasks, and traceability agree unambiguously' 'prove refuses ambiguous legacy candidate scope'
require "$PROVE" 'all current approved real proof from scratch' 'prove never reuses a legacy pass'
require "$PROVE" 'pre-proof and post-proof engine digest' 'prove establishes both candidate digest observations'
require "$PROVE" 'fresh exact binding' 'prove writes fresh exact candidate bindings after legacy proof'
require "$CANDIDATE_INTEGRITY" 'Upgrade routes a released-workspace candidate defect to Prove' 'candidate integrity names the fail-closed legacy owner'
require "$RESOLVE" 'explicit consent' 'resolve consumes the explicit answer'
require "$RESOLVE" 'devrites-engine state resolve' 'resolve uses the nested state writer'
forbid "$RESOLVE" 'confirm? (y/N)' 'resolve does not ask twice'

echo ""
[ "$fail" -eq 0 ] && echo "phase-gate-routing-test: PASS" || echo "phase-gate-routing-test: FAIL"
exit "$fail"
