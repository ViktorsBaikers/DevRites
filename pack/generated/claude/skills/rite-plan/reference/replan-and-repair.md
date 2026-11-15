# Re-plan & repair modes

`/rite-plan` runs in one of these modes. Pick from `$ARGUMENTS` or infer from state.

Authority: `.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → reconcile `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md` atomically; invalidate Vet/readiness; Vet.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## decompose

First (or fresh) breakdown into vertical slices. Use `task-breakdown.md` +
`slicing.md`. Output: a populated `tasks.md` and an ordered `plan.md` graph.

## reslice

A slice proved too large (couldn't build+prove in one cycle, or its goal has multiple
"and"s). Split into thinner end-to-end slices, preserving acceptance coverage: split
**by the sizing rule, not to a target count**. Update `tasks.md` under the canonical
[ID contract](../../devrites-lib/reference/workspace-artifact-schema.md#id-contract):
preserve unaffected IDs; never renumber or fill retired gaps. New meanings get the
next unused IDs; record relationships in `decisions.md` or `drift.md` before updating
downstream dependency edges and traceability.

## repair (after Spec Drift Guard)

Root preflight follows the canonical blocked-input action above for missing/empty
`drift.md`. Reconcile cited briefs/findings against current artifacts before dispatch.
Record only evidenced open drift/provenance; explicitly supersede stale instructions.
No remaining repair means select the supported next action, not manufacture drift.
Missing authority/producer input requires a source-specific `frontier-blocked:`
record with owner/resume trigger, then refresh. Narrow no-findings is not READY.

1. Fold every open classified drift entry and accepted finding for affected
   artifacts in one pass.
2. Record answered drift decisions in `decisions.md`; carry their constraints into
   affected remaining slices and their `Prior-slice learnings`.
3. Update affected `spec.md` sections and `plan.md`/`tasks.md`. Acceptance meaning
   changes require a user directive or recorded user decision
   ([provenance](../../devrites-lib/reference/standards/acceptance-preserving-reslice.md));
   otherwise ask with a recommended option. Root cannot authorize its own AC change.
   **Write the pointer, not the history:** closed finding ID + one-line meaning.
   Keep prior folds/DEC lists/round narratives out of architecture/plan/tasks/traceability;
   use `decisions.md` or [Budget relocation](#revise) into history when closed.
4. Mark resolved drift with resolution and date.
5. Before handoff, self-check and self-attack the delta: acyclic dependencies,
   bidirectional traceability, Shared contract proof/boundary coverage, sizing and
   correction-created regression; probe races/observation windows, aliases, ordering,
   capabilities, unbounded blocking and happy-path-only coverage. Fold fixes or named
   bounded residuals into this pass. Record both checks in the Plan checkpoint for
   Vet recheck, not rediscovery.
6. Resume the corrected slice. Preserve requirements; use the marked action before
   contract edits.

## reorder

Dependency order is wrong or suboptimal. Recompute the graph (`dependency-graph.md`),
re-sort risk-first within tiers, update `plan.md`'s implementation order.

## split (backend/frontend contract)

A slice couples two sides too tightly. Define the contract first (shape, status codes,
errors: `devrites-api-interface`), then split into a backend slice (can land with a
stub consumer) and a frontend slice (can land against a mock/real contract).

## unblock

An exhausted fingerprint blocks diagnosis, not symptom. Route around/reslice; escalate
only for scope change. If proof removed that cause but symptom remains, keep the dead
end and plan a new diagnosis/proof fingerprint. Never clear/reuse old one.

## course-correct

Mid-build user pivot. Apply the marked action, choose rollback or forward-fix, and
update permitted artifacts atomically (`MVP cut` is the named retreat). Invalidate
Vet/readiness when contract or topology changes.

**Feature-ceiling cut.** When unbuilt slices exceed the
[feature ceiling](slicing.md#feature-ceiling-split-an-epic-never-override-the-budget),
`MVP cut` keeps the built slices plus the thinnest shippable unbuilt subset here; the
remaining slices, their REQ/AC IDs and proof rows move verbatim into a named
continuation list (spec `Non-goals` → future `/rite-spec <slug>-2`, `-3`, …) through
the Spec Drift Guard. IDs never renumber; each ID is owned by exactly one workspace.
Record the sequence once in `decisions.md`.

## revise

Artifact-only reconciliation of `spec.md`, `architecture.md`, `plan.md`, `tasks.md`,
or `traceability.md` — never source. Propose/confirm the file set first. A neutral workspace
edit is authorized by explicit `/rite-upgrade` with a `repairable` assessment, or, for budget
relocation only, by the phase owner when `orient` reports `over: true`.

Unchanged acceptance/behavior with non-equivalent `proposed_coverage` is a
contradictory input: the canonical classifier blocks it — there is no fourth
route (see [`acceptance-preserving-reslice.md`](../../devrites-lib/reference/standards/acceptance-preserving-reslice.md)).

**Budget relocation** (authorized as above): move checkpoint narrative, historical fold text and
closed-round accounts verbatim from `state.md`/`tasks.md`/`plan.md`/`architecture.md`/
`eng-review.md` into `history/<file>-<YYYYMMDD>.md`, packets/accounts from the root
into `packets/`, leaving a one-line pointer. Cursor, live slices, open findings, IDs,
answers, decisions and bindings stay unchanged; nothing is deleted or reworded.
Re-emit the readiness binding when a readiness input moved.

## Always

Update `state.md` (phase, next step) and append a dated line to `decisions.md`
explaining *why* the plan changed. Preserve mapped-action decisions.
