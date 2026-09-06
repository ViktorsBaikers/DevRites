---
name: rite-plan
description: Re-plan existing work when reality invalidates the plan: reslice a slice that is too big, repair drift, reorder dependencies, split boundaries, or unblock work. Not first-pass decomposition.
argument-hint: "[mode: decompose|reslice|repair|reorder|split|unblock|course-correct|revise]"
user-invocable: true
---

# /rite-plan: (re)plan an active feature

Update a plan invalidated by evidence, drift, or user decision. **Read the active workspace
first.** Missing `.devrites/ACTIVE`/workspace stops to `/rite-spec <feature>`. **Revise is
artifact-only**: reconcile
`spec.md` / `architecture.md` / `plan.md` / `tasks.md` / `traceability.md` without
editing source code.

## Rules consulted (read on demand from `.omp/skills/devrites-lib/reference/standards/`)
Pull `development-workflow.md` via `Read` when reshaping slice cadence or DoD
criteria.
Load `repository-topology.md`, `data-integrity.md`, or `integration-reliability.md`
when the spec applicability map or observed drift triggers them.
Before classifying any Reslice, read `.omp/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → reconcile `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md` atomically; invalidate Vet/readiness; Vet.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Operating rules
- Update the spec when needed, but never plan around a known-wrong assumption.
- Keep each slice small enough for one focused build → prove cycle.
- **Derive slice count:** reslice when work cannot build+prove in one cycle, not to hit a
  requested tally. Treat counts as hints and explain honest differences; see
  [`slicing.md`](reference/slicing.md).
- **Size by complexity, order by dependency.** A slice carries a `Complexity: N/5` score (from
  `/rite-define`); a slice scoring **>3** is a reslice trigger unless its inline reason justifies
  the irreducible complexity. Honor each slice's `depends_on:`: the next *buildable* slice is the
  lowest pending one whose dependencies are all built. This preserves one-slice-at-a-time execution.
- **Root writes; drafter proposes.** Follow
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The controlling chat owns
  human questions, decisions, reconciliation, and planning writes.
- **Nested repair preserves its caller.** When `state.md` contains a valid
  technical-backtracking return cursor, preserve any valid return cursor
  byte-for-byte. `/rite-vet` is the next internal prerequisite, not a command to
  hand back to the human.

## Workflow
0. Read `.omp/skills/devrites-lib/reference/standards/core.md` (operating rules) before reshaping anything.
   Then resolve the explicit or active slug, require its `state.md`, and read
   the cursor directly.
1. Read `spec.md`, `decision-coverage.md`, `plan.md`, `tasks.md`, `state.md`, `drift.md`,
   `eng-review.md`, and the current `git diff` (if a repo). Read `decisions.md` and
   `assumptions.md`. Require `Decision coverage: CLEAR`; otherwise STOP → `/rite-clarify`.
   Apply `.omp/skills/devrites-lib/reference/standards/tooling.md`: use the
   primary available structural index, and cross-check only for a named unresolved
   predicate rather than reassurance. For an external
   dependency's current API surface, consult context7 if available.
2. **Pick the mode** (`$ARGUMENTS` or infer): Apply the marked action before writes and
   retain its decision/coverage evidence.
   - **decompose:** first/again break the feature into vertical slices.
   - **reslice:** a slice is too large; split into thinner end-to-end slices.
   - **repair:** a Spec Drift Guard event; fold the resolution into plan + tasks.
   - **reorder:** fix the dependency order.
   - **split:** separate backend/frontend contracts (see `devrites-api-interface`).
   - **unblock:** a verification failed; re-route around the blocker.
   - **course-correct:** mid-build user pivot; apply the marked action, choose
     rollback/forward-fix, and update permitted artifacts atomically (`MVP cut` is the named retreat).
   - **revise:** reconcile a requested artifact revision; propose/confirm its file set and
     **never edit source**. Only explicit `/rite-upgrade` with a `repairable` assessment naming
     rule, evidence, gate, paths, and delta can authorize its neutral workspace edit—not source/history.
     **Revise or new?** Same intent? >50% scope survives? Original not completable without it?
     Two “no” answers mean new work: seal/ship current scope (`MVP cut` if named), then
     `/rite-spec`, and stop.
   See [replan-and-repair](reference/replan-and-repair.md) for each mode's steps.
2a. **Draft fresh.** Dispatch `devrites-plan-drafter` in `repair` mode with frozen mode,
   affected artifacts, settled contract, and failure/drift. Await one atomic, read-only
   the drafter's `candidate_files` bundle; human choices return separately.
   When the observed failure is missing or ambiguous consumptive-action evidence,
   require the drafter to apply `one-shot-actions.md` and return the bounded
   diagnostic-amplification design, injective boundary map, per-seam fixtures, and
   collision mutant. Past evidence loss is not terminal. The drafter supplies no bodies.
   Bind exact active `.devrites/work/<slug>/` targets and executable contract; after Vet READY, root
   materializes the exact vetted workflow-artifact paths under
   [`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md)
   without dispatching the product wright.
3. Reason about dependencies: [dependency-graph](reference/dependency-graph.md).
   Reconcile the horizon register against `../rite-define/reference/plan-template.md`.
   Preserve every `HZN-###` and unresolved item. Cite evidence when reclassifying the earliest
   honest decision point; resolution or supersession also needs evidence. Never silently delete
   an item. Planning choices resolve from source evidence or become bounded risk spikes with
   discriminating criteria and fallback branches; local/checkpoint entries retain their owner,
   trigger, bounds/fallback, and proof. A new human-owned blocker stops to `/rite-clarify`.
   Reconcile `plan.md`'s canonical `Shared contract proof`: changed provider/consumer
   boundaries keep one reused contract artifact ahead of both asserting tests, and unaffected
   plans retain the specific no-impact statement. Missing, one-sided, duplicated-contract, vague, or
   non-consuming proof routes to `/rite-vet` only after repair.
   Reconcile the spec applicability map and retain every applicable standard's required
   topology/data/integration owner, failure/recovery, deployment order, and proof output.
   **Completion:** the slice graph is cycle-free and every dependency names an existing slice.
4. Re-slice using vertical-slice rules: [slicing](reference/slicing.md) and
   [task-breakdown](reference/task-breakdown.md). Prefer thin, shippable, verifiable.
   **Completion:** every slice is independently shippable/provable or carries an irreducibility reason.
5. Reconcile the candidate against steps 3 and 4, then the root updates `plan.md`, `tasks.md`,
   `state.md`, and appends rationale to `decisions.md`.
   Any change to `architecture.md`, `plan.md`, `tasks.md`, or `traceability.md` invalidates
   the previous vet verdict: set `Phase: plan`, `Next step: /rite-vet`, and, when
   `eng-review.md` exists, set `Implementation readiness: NEEDS REPLAN`. Never retain READY
   across changed planning inputs. Preserve `Plan approved` only for behavior/acceptance-neutral
   technical repair; clear it for a contract-changing or newly blocking horizon item and
   reconfirm only after Clarify and Vet close it. If you stopped for drift,
   mark the `drift.md` entry resolved. Never remove or overwrite a valid caller
   return cursor while writing the Plan checkpoint.
6. After editing `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, or `questions.md`,
   re-scan affected coverage, assumptions, uncertainty, and gates. Partial/Missing, unowned
   material assumption, or open blocking/escalating question routes `/rite-clarify`/HITL.
   Restore `CLEAR` only from current evidence.
7. **Done when:** every slice is sized (builds + proves in one cycle; no slice scoring >3
   left unjustified), the dependency order is acyclic, every `drift.md` entry you stopped for
   is marked resolved, revised artifacts agree with each other, no source files changed in
   `revise` mode, the marked action is complete, and every changed
   plan ends at `/rite-vet` rather than returning directly to build; horizon IDs stay stable,
   reclassifications cite evidence, and no unresolved item vanished. The `Shared contract proof`
   table or justified no-impact statement must still match the revised boundary set.
   If any check fails, loop back: don't hand off a half-reshaped plan.

When invoked inline by a controlling rite, return the completed Plan checkpoint
to that caller so it can invoke Vet immediately. The phase boundary is not a
user-facing stop unless a genuine HITL, safety/access, or exhausted-recovery
condition was recorded.

> **Mid-flight discipline.** Do not change product behavior without confirmation or
> absorb drift silently. See [`anti-patterns`](reference/anti-patterns.md).
