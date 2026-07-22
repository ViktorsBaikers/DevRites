# Re-plan & repair modes

`/rite-plan` runs in one of these modes. Pick from `$ARGUMENTS` or infer from state.

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
Never repair by quietly deleting the inconvenient requirement, if behavior changes,
that was a user question.

## reorder
Dependency order is wrong or suboptimal. Recompute the graph (`dependency-graph.md`),
re-sort risk-first within tiers, update `plan.md`'s implementation order.

## split (backend/frontend contract)
A slice couples two sides too tightly. Define the contract first (shape, status codes,
errors: `devrites-api-interface`), then split into a backend slice (can land with a
stub consumer) and a frontend slice (can land against a mock/real contract).

## unblock
A `/rite-prove` failure can't be fixed inside the current slice. Capture the blocker in
`state.md`, decide: route around it (reorder), shrink the slice (reslice), or escalate
to the user (if it changes scope). Don't loop on the same failing approach.

## Always
Update `state.md` (phase, next step) and append a dated line to `decisions.md`
explaining *why* the plan changed. Ask the user before any product-behavior change.
