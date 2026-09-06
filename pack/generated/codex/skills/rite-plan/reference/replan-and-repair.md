# Re-plan & repair modes

`$rite-plan` runs in one of these modes. Pick from `$ARGUMENTS` or infer from state.

Authority: `.agents/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

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
**by the sizing rule, not to a target count**. Update `tasks.md`; renumber; fix
dependency edges.

## repair (after Spec Drift Guard)

A drift event stopped the build. Steps:

1. Read the `drift.md` entry and its classification.
2. If the user answered a drift question, encode that decision in `decisions.md`.
3. Update `spec.md` (only the affected sections) and `plan.md`/`tasks.md` to match
   reality. Adjust acceptance criteria if they were wrong.
4. Mark the `drift.md` entry **resolved** with the resolution + date.
5. Resume at the corrected slice.
Never quietly delete a requirement; use the marked action before contract edits.

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

## revise

Artifact-only reconciliation of `spec.md`, `architecture.md`, `plan.md`, `tasks.md`,
or `traceability.md` — never source. Propose/confirm the file set first. Only explicit
`$rite-upgrade` with a `repairable` assessment may authorize neutral workspace edits.

Unchanged acceptance/behavior with non-equivalent `proposed_coverage` is a
contradictory input: the canonical classifier blocks it — there is no fourth
route (see `acceptance-preserving-reslice.md`).

## Always

Update `state.md` (phase, next step) and append a dated line to `decisions.md`
explaining *why* the plan changed. Preserve mapped-action decisions.
