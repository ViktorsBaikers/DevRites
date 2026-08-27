---
name: rite-prove
description: Prove a completed feature with tests and the full test suite, build/typecheck/lint, end-to-end/browser evidence, screenshots, commands, and outputs for seal. Not for single-slice proof.
argument-hint: "[feature-slug]"
user-invocable: true
---

# $rite-prove: prove the completed feature

Read the active workspace and prove the whole assembled feature once. Without
one, use `/verify` or `/run`; if any task is pending, stop for `$rite-build`.
Never report an unobserved pass. After Polish/Review code edits, repeat affected
criteria and bind evidence to the changed candidate digest.

## Rules

Read the applicable standards: `testing.md`, `test-proof-checklist.md`,
`browser-proof-checklist.md` for UI, `spec-grammar.md`, `performance.md`,
`observability.md`, topology/data/integration rules named by the plan,
`developer-experience.md`, `definition-of-done.md`, `one-shot-actions.md`, and
`workflow-artifacts.md`. Developer surfaces require an observed flow, measured
TTHW, and exact signal-bearing errors; never assert DX.

- Evidence over confidence. Apply positive, discriminating proof to every
  behavioral claim; a green command without executing assertion and decisive
  signal is unproven.
- Follow `candidate-integrity.md`. Prove owns proof binding, not candidate
  grammar/hashing. Spec Drift Guard owns revealed contract drift.
- Root executes vetted gates/browser and records evidence. A read-only proof
  runner validates immutable evidence. The sole bounded wright fixes product
  source/tests; root never does.
- **Prove remains the controlling caller** during technical backtracking. Save
  its cursor; run Plan repair, Recovery Vet, remediation, and re-proof inline;
  consume nested boundaries; resume the failed rung. Recovery exhausts after
  three no-progress attempts on the exact same fingerprint. Closing one or exposing a distinct
  Critical/Important invariant is progress, not a reason to hand work off.
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"Prove consumes installed Workflow Artifact or CLEANED rerun","action":"VERIFY_EXISTING or admitted proof path ending PROVE_AND_RETURN","return":"saved Prove cursor; stop before real action"} -->
## Released-workspace refresh entry

Only an admitted `$rite-upgrade` assessment proving a released-format candidate
defect enters. Require legacy touched-file scope, live diff, tasks, and traceability agree unambiguously
on every candidate path/state. Missing or
ambiguous scope stops; never guess from Git. Preserve unrelated manifest content
and refresh only the strict manifest from observed bytes.

Discard old passes. Run all current approved real proof from scratch. Establish
pre-proof and post-proof engine digest with `devrites-engine check candidate
<slug>` and require equality, then write one fresh exact binding in evidence and
browser evidence. Current failure blocks; Upgrade does not authorize source or
test changes.

## Workflow

1. **Orient.** Read core, resolve active slug, require `state.md`, and read its
   cursor. On blocked Prove cold resume, reconcile retained consumptive evidence
   and no-progress corrections before accepting terminal none; a distinct
   fingerprint below cap resumes offline triage without another real action.
2. **Freeze scope.** Read spec, tasks, state, test plan, and full diff. Require
   every slice built. Missing `test-plan.md` invokes caller-owned Vet and returns
   here; it never authorizes ad hoc proof.
3. **Approve commands.** If absent, use
   [test-command-discovery](reference/test-command-discovery.md) over repository
   manifests/CI. Discovery is evidence only: `test-plan.md` is the sole approved runtime
   command list. A newly found command must return to the current Vet contract,
   refresh readiness, then resume without user handoff.
4. **Execute a frozen candidate.** Run candidate check and retain its digest.
   The root runs only commands declared by `test-plan.md`, with exact approved
   command, cwd, prerequisites, exit, and sanitized
   decisive output. Run relevant suite plus build/typecheck/lint. Recheck the
   candidate and require identical digest/no source mutation. Reject substituted
   commands, malformed manifests, zero-test/skipped/filtered behavioral claims,
   exit-status-only claims, and source drift. Static gates prove only their named
   static criterion.
5. **Gate consumptive actions.** Immediately before execution, apply
   `one-shot-actions.md`: current retained identity, bounds/sanitization,
   injective boundary map, per-seam fault fixtures, collision mutant, terminal
   fixtures, and cleanup-survival proof must be green. Missing/stale/disposable-
   only evidence returns to Vet without spending an attempt. Record the admitted artifact identity before execution.
   After failure, retained evidence is the
   reproduction; never rerun for diagnosis. Spent authorization blocks another
   action, but when action budget is zero it does not exhaust a newly identified
   offline fingerprint.
6. **Prove UI when applicable.** Use design brief/references, browser harness,
   allowed scratch path, [proof ladder](reference/proof-ladder.md), and
   [browser proof](reference/browser-proof.md): routes, viewports, opened and
   described screenshots, console/network, interaction, target comparisons, and
   every required state. Material mismatch fails.
7. **Independent validation.** Dispatch exact fresh read-only
   `devrites-proof-runner` and `devrites-spec-reviewer` on the same candidate,
   commands, immutable evidence, and acceptance map. Reject missing/stale,
   invented-ID, label-only, or self-attested reports; neither executes commands.
8. **Map proof.** Reconcile both verdicts and
   [acceptance-proof.md](reference/acceptance-proof.md). Every criterion,
   scenario, interaction, and planned key link gets positive discriminating
   evidence or a blocker, including applicable critical-path, observability,
   developer, and wiring branches.
9. **Recover red.** Use [failure triage](reference/failure-triage.md) and Debug
   Recovery. Reconcile reproduction; send accepted in-scope source/test correction
   to the sole wright; update actual manifest; rerun affected proof, both candidate
   checks, and fresh proof runner. Scope growth blocks.

   Agent-owned durable-plan errors run Spec Drift Guard, Plan, and Vet inline,
   preserving Prove as origin. Consumptive failures use retained fixtures, repair,
   and Vet, then stop for fresh GO after changed conditions. Ambiguous retained
   evidence does not prove that no safe future acquisition design exists: treat
   an in-scope discriminator as diagnostic-amplification Plan gap, repair its
   finite map/collision proof, narrow-Vet, then seek fresh GO before one evidence
   attempt. Never guess the runtime fix or reuse old GO.
10. **Record.** Root updates `evidence.md`, optional `browser-evidence.md`,
    traceability, and state, with exactly one observed candidate binding.

> Do not claim an unobserved pass, skip browser proof, or proceed with pending
> slices. Load `reference/anti-patterns.md` when tempted to do so.

## Phase exit (observable)

**Complete when:** every criterion in `acceptance-proof.md` has discriminating
evidence bound to the current candidate digest, both independent validators admit
accounts, and `state.md` records Prove complete with no open `cannot_verify` rows.

**Failing case:** narrative "all tests passed" without `evidence.md` binding and
proof-runner admission → phase not complete; Seal blocks.
