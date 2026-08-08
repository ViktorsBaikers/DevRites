---
name: rite-prove
description: Prove a completed feature with tests and the full test suite, build/typecheck/lint, end-to-end/browser evidence, screenshots, commands, and outputs for seal. Not for single-slice proof.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-prove: prove the completed feature

Prove the **whole feature**. Read the active workspace; if absent, run
`/rite-spec <feature>`.

> **Scope:** `/verify` proves one change; `/run` launches the app. `/rite-prove`
> walks feature acceptance, relevant suite/build/typecheck/lint, and browser
> proof, writing evidence under the active workspace. Without one, use
> `/verify` or `/run`.

## Gate: all slices must be built first
Read `tasks.md` + `state.md`. If any slice is pending, stop for `/rite-build`.
This phase proves the assembled feature once, after per-slice tests.

**Never report an unobserved pass.** If a command could not run, report that and give
exact manual steps.

After polish/review edits, rerun affected criteria/routes so evidence binds the
changed candidate digest before Seal.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when relevant:
- `testing.md`: pyramid, determinism, no-flake discipline.
- `test-proof-checklist.md`: compact proof-quality gate for tests and recorded evidence.
- `browser-proof-checklist.md`: required UI states and Visual Verdict evidence.
- `spec-grammar.md`: when the spec uses structured `### Requirement:` / `#### Scenario:`
  blocks, each scenario (WHEN/THEN) is one observable behavior to walk and prove.
- `performance.md`: measure first when perf is in scope.
- `observability.md`: runtime telemetry must be observed to emit.
- `developer-experience.md`: when the change ships a developer-facing surface (API / CLI / SDK /
  webhook / config / error messages / getting-started): **measure** the DX scorecard (run the flow,
  measure time-to-hello-world, and capture verbatim error text), rather than asserting it.
- `definition-of-done.md`: acceptance, proof, gates, scope, rollback/docs.
- `one-shot-actions.md`: pre-attempt evidence completeness and no-rerun handling for
  consumptive proof actions.


## Operating rules
- Evidence over confidence. Feature scope only: fix within the feature or record a
  blocker; don't refactor unrelated code.
- Apply `testing.md`'s positive, discriminating proof rule to every behavioral claim.
  A green command without an executing assertion and decisive observed signal is unproven.
- Follow the shared
  [`candidate-integrity.md`](../devrites-lib/reference/candidate-integrity.md).
  Prove owns proof binding, not manifest grammar or candidate hashing.
- Spec Drift Guard applies: if tests/evidence reveal the spec is wrong, stop and handle
  drift (`rite-build/reference/spec-drift-guard.md`).
- **Root executes; runner validates; root records; wright fixes.** Use the native
  fresh-context contract in
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The root owns exact
  vetted gate execution, browser capability, the evidence verdict, and canonical
  writes. The proof runner is read-only and validates immutable logs/artifacts.
  Every accepted source/test correction is one bounded
  `devrites-slice-wright` task, never an inline edit.
- **Prove remains the controlling caller during technical backtracking.** Save
  its return cursor, invoke Plan/Vet or bounded remediation inline, consume each
  nested phase boundary, then resume the failed Prove step. Never make the human
  submit an agent-owned repair, re-vet, or proof-rerun command. Apply `afk-hitl.md`
  progress accounting: only three no-progress attempts on the exact same fingerprint
  exhaust recovery; a closed prior finding or a genuinely new
  Critical/Important fingerprint continues inside this invocation.

## Released-workspace refresh entry

Only an admitted `/rite-upgrade` assessment that proves a released-format candidate
defect may enter this branch. Normal Prove never reconstructs candidate scope.

Before proof, require that the legacy touched-file scope, live diff, tasks, and traceability agree unambiguously on every candidate path and state. Missing, unknown, or
ambiguous scope is a recorded gap and HITL stop; never guess from Git alone. When they
agree, preserve unrelated `touched-files.md` content and refresh only its strict manifest
from the currently observed project bytes under the workspace schema.

Discard every old pass as proof. Run all current approved real proof from scratch under
the positive, discriminating rules below, and establish the pre-proof and post-proof engine digest
with `devrites-engine check candidate <slug>`. Require an identical digest,
then write one fresh exact binding in `evidence.md` and in `browser-evidence.md` when it
exists. Historical evidence cannot substitute for any current gate. A current proof
failure is a blocker; this Upgrade admission never authorizes source or test changes.

## Workflow
0. Read core, load relevant rules above, then resolve the active slug,
   require its `state.md`, and read the cursor directly. On a blocked Prove cold
   resume, reconcile any retained consumptive-action artifact against recorded
   no-progress corrections before accepting `next_action: none`; a distinct
   fingerprint below its cap resumes offline triage without another real action.
1. **Confirm all slices built.** Read `spec.md`, `tasks.md`, `state.md`,
   `test-plan.md`, and the full diff.
   A missing `test-plan.md` enters caller-owned Vet backtracking; invoke Vet
   inline and resume this step. It never authorizes ad hoc proof.
2. **Discover commands** if not recorded:
   [test-command-discovery](reference/test-command-discovery.md): README, package
   scripts, Makefile, CI configs, Gemfile/Rakefile, pyproject, go.mod, Cargo.toml.
   Discovery only supplies evidence. `test-plan.md` is the sole approved runtime
   command list. If a discovered command is absent from it, do not run or
   silently approve the command: return to the current Vet contract to add and
   vet it inline, refresh readiness, then return to Prove without a user handoff.
   **Completion:** exact commands are approved in `test-plan.md` or unavailable.
3. **Execute proof against a frozen candidate.** Run
   `devrites-engine check candidate <slug>` before any approved proof and retain
   its exact digest. The root runs only commands declared by `test-plan.md`,
   byte-for-byte, with exact cwd and prerequisites, capturing exit code and
   decisive output in secure external scratch. Run the full relevant test suite
   plus **build / typecheck / lint**. Then rerun the candidate check: require the
   identical digest and no candidate-source mutation. Reject synthesized or
   substituted commands, malformed manifests, zero-test/skipped/filtered results used as
   behavioral proof, success inferred only from exit status, and any source drift. Static
   gates prove only their named static criterion.
   Immediately before any consumptive action, apply `one-shot-actions.md` to the
   live candidate: require the vetted retained-artifact identity, bounds,
   sanitization, terminal-path fixtures, and cleanup-survival proof. A missing,
   stale, or disposable-only evidence surface returns to Vet inline and consumes
   no attempt. Record the admitted artifact identity before execution. After a
   failed consumptive action, triage from that artifact; never reproduce it by
   rerunning the action. Consuming its authorization blocks only another real
   execution. If the artifact identifies a new Critical/Important fingerprint,
   continue offline recovery inside this Prove invocation; do not label the new
   fingerprint exhausted because the action budget is zero.
4. **UI feature?** The root applies the browser proof ladder with
   `design-brief.md`, `references.md`, the requested routes, browser harness, and
   allowed scratch path:
   [proof-ladder](reference/proof-ladder.md) + [browser-proof](reference/browser-proof.md)
   (`devrites-browser-proof`): routes, viewports, screenshots (opened + described),
   console, network, interaction paths, and the brief's proof targets. Compare screenshots
   with target references and record deltas. An unresolved material mismatch is a failed
   result; the root handles any accepted correction at step 7 before re-rendering.
5. **Validate proof in fresh context.** In parallel, dispatch the exact read-only
   `devrites-proof-runner` on candidate identity, approved commands, immutable
   root evidence, and acceptance map, and `devrites-spec-reviewer` on the same
   candidate. Wait for both; reject missing/stale, invented-ID, label-only, or
   self-attested evidence. Neither executes commands.
6. **Map proof completely.** Reconcile both exact verdicts, then follow
   [`reference/acceptance-proof.md`](reference/acceptance-proof.md) for acceptance/scenario
   coverage and the conditional critical-path, observability, developer-surface, and wiring
   branches. Completion: every criterion, planned interaction, and declared key link has a
   proof class plus positive, discriminating passing evidence, or is recorded as a blocker.
7. **On failure** → [failure-triage](reference/failure-triage.md) +
   `devrites-debug-recovery`. The root reconciles the reproduction. Send an accepted,
   in-scope correction to the sole writer, `devrites-slice-wright`; update the
   manifest from its actual scoped diff, rerun affected real proof and both
   candidate checks, then dispatch a fresh proof runner. If a fix would exceed
   scope, record a blocker.
   If triage shows an agent-owned durable-plan error, apply the Spec Drift
   Guard's inline return contract: preserve Prove as the origin, run repair and
   Vet inside this invocation, and resume the exact failed proof rung. Ask only
   for a human-owned decision; causal-fingerprint exhaustion stops once with a
   technical blocker rather than another phase command.
   For a consumptive-action failure, use fixtures or another non-consumptive
   reproduction to fix and re-vet the retained fingerprint. Once the failure
   condition is demonstrably changed, stop at the fresh-authorization boundary;
   only a new human GO may admit the next real attempt.
8. The root updates `evidence.md`, `browser-evidence.md` (when present),
   `traceability.md`, and `state.md`. Record exactly one binding for the observed
   digest in evidence and browser evidence. New proof goes to canonical
   `evidence.md`.

> **Mid-flight discipline.** When tempted to claim an un-observed pass, skip a rung of the browser-proof ladder, or proceed with slices pending: see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.
